package tools

// 本文件实现 read_url 网页链接读取工具：模型在 ReAct 循环中发现用户消息
// 包含链接或要求阅读网页时调用，内部基于 eino-ext 官方 URL Document Loader
// 抓取网页并提取正文（默认 HTML 解析器，取 body 内容），按 ai_read_url_max_chars
// 设置截断后返回给模型。仅放行 http/https，避免 file:// 等本地路径读取。
// SSRF 三层防护复用 ssrf.go 的共享客户端（含拨号期 DNS rebinding 校验与
// 响应体 1MB 限长）；isPrivateHost 额外做 inet_aton 数值编码 IP 归一化。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	urlLoader "github.com/cloudwego/eino-ext/components/document/loader/url"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

// readURLTimeout 抓取链接的超时时间（过长则视为失败，避免卡住 ReAct 循环）。
const readURLTimeout = 15 * time.Second

// browserUserAgent 浏览器 UA：多数站点对非浏览器 UA 返回 403，须模拟浏览器请求。
const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// readURLTool 网页链接读取工具。
type readURLTool struct {
	setting *services.SettingService // 读取输出最大字符数设置（复用 ai_read_url_max_chars）
	ctx     *Context                 // 事件发射、日志
}

// 编译期断言：确保 readURLTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*readURLTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 展示被读取的链接（截断防超长），解析失败或为空时回退通用文案。
func (r *readURLTool) ActionText(argumentsInJSON string) string {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "阅读网页链接"
	}
	if args.URL = strings.TrimSpace(args.URL); args.URL != "" {
		return "阅读链接 " + TruncateRunes(args.URL, 30)
	}
	return "阅读网页链接"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (r *readURLTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_url",
		Desc: "读取网页链接（URL）的内容并返回正文。当用户消息中包含链接、或要求阅读/总结/提取某个网页的内容时调用；也可在搜索工具返回结果不够深入时进一步打开搜索结果中的链接。注意：仅支持 http/https 链接；动态渲染（JS）的页面可能只能拿到部分内容。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.String,
				Desc:     "要读取的网页链接，必须是 http/https 开头的完整 URL",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行链接读取：校验 URL → 构建 loader（超时 + 浏览器 UA）→
// 加载并提取正文 → 按设置截断后返回。错误路径（参数缺失 / 非法 scheme /
// 抓取失败 / 空正文 / 用户取消）返回 error 经 WrapWithError 回填模型继续推理。
func (r *readURLTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 read_url 参数失败: %w", err)
	}

	// 1. 校验 URL：仅放行 http/https 完整链接
	target, err := validateHTTPURL(args.URL)
	if err != nil {
		return "", err
	}

	// 2. 构建 loader：默认 HTML 解析器提取正文；复用共享防护客户端（SSRF 三层
	//    防护，含拨号期 DNS rebinding 校验与响应体限长，见 ssrf.go），浏览器 UA
	//    规避 403。
	loader, err := urlLoader.NewLoader(ctx, &urlLoader.LoaderConfig{
		Client: newGuardedHTTPClient(readURLTimeout, true),
		RequestBuilder: func(ctx context.Context, src document.Source, _ ...document.LoaderOption) (*http.Request, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URI, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("User-Agent", browserUserAgent)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			return req, nil
		},
	})
	if err != nil {
		return "", fmt.Errorf("创建 URL Loader 失败: %w", err)
	}

	// 3. 加载文档并提取正文（可能返回多个 document，全部拼接后按上限截断）
	docs, err := loader.Load(ctx, document.Source{URI: target})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("读取链接失败: %w", err)
	}

	maxChars := getIntSetting(r.setting, "ai_read_url_max_chars", 5000, 50000)
	var b strings.Builder
	for _, d := range docs {
		if d == nil || strings.TrimSpace(d.Content) == "" {
			continue
		}
		if b.Len() >= maxChars {
			break
		}
		b.WriteString(strings.TrimSpace(d.Content))
		b.WriteString("\n\n")
	}
	if b.Len() == 0 {
		return "", errors.New("未能从该链接提取到正文内容（可能页面为空或为动态渲染页面）")
	}

	// 4. 按 rune 截断（支持中文），超长时追加截断提示，避免撑爆模型上下文窗口
	text := b.String()
	if runes := []rune(text); len(runes) > maxChars {
		text = string(runes[:maxChars]) + "\n\n（内容过长，已截断）"
	}

	if r.ctx != nil && r.ctx.Logger != nil {
		r.ctx.Logger.Debugw("Agent read_url 调用",
			fastlog.String("url", target),
			fastlog.Int("chars", len([]rune(text))))
	}
	return fmt.Sprintf("以下为链接 %s 的内容：\n%s", target, text), nil
}

// validateHTTPURL 校验并规范化 URL：仅放行 http/https scheme，其余（file://、
// data: 等）直接拒绝，防止读取本地文件等非预期来源。
func validateHTTPURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("read_url 参数缺少 url")
	}
	if err := validateTextLen("url", raw, maxToolShortText); err != nil {
		return "", err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("URL 格式无效: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("仅支持 http/https 链接，收到 scheme: %s", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("URL 缺少主机名（请提供完整的 http/https 链接）")
	}
	if isPrivateHost(u.Host) {
		return "", fmt.Errorf("拒绝访问内网/本机地址 %s（仅允许公网链接）", u.Host)
	}
	return u.String(), nil
}

// isPrivateHost 判断主机是否指向内网/本机（SSRF 防护）：仅依据 IP 字面量与显式
// 本机 hostname 判断，不做 DNS 解析（避免额外网络 IO 与探测面）。数值编码的 IP
// 字面量（如 0x7f000001）会先经 normalizeIPLiteral 归一化再判定。
func isPrivateHost(host string) bool {
	h := host
	// 去除端口：带方括号的 IPv6（[::1]:8080）剥掉端口与方括号；单冒号视为
	// host:port（IPv4/域名）去除端口；裸 IPv6 字面量（如 ::1，多冒号无方括号）
	// 与无冒号主机直接判定，避免尾部截断破坏 IPv6 形式
	if ipv6Bracket := strings.IndexByte(h, ']'); ipv6Bracket >= 0 {
		h = strings.Trim(h[:ipv6Bracket+1], "[]")
	} else if strings.Count(h, ":") == 1 {
		h = h[:strings.IndexByte(h, ':')]
	}
	h = strings.TrimSpace(h)

	if strings.EqualFold(h, "localhost") ||
		strings.HasSuffix(strings.ToLower(h), ".local") ||
		strings.HasSuffix(strings.ToLower(h), ".internal") {
		return true
	}

	ip := net.ParseIP(h)
	if ip == nil {
		// 标准形式解析失败：尝试 inet_aton 兼容的数值编码形式归一化
		// （0x7f000001 / 2130706433 / 0177.0.0.1 等写法可绕过 ParseIP 判定），
		// 仍失败则按普通域名处理（非 IP 字面量不做 DNS 解析，放行）
		if ip = normalizeIPLiteral(h); ip == nil {
			return false
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast()
}

// normalizeIPLiteral 将 inet_aton 兼容的数值编码 IP 字面量归一化为标准 IPv4，
// 防止 2130706433（十进制整数）、0x7f000001（十六进制）、0177.0.0.1（八进制）、
// 127.1（末段吸收剩余字节）等写法绕过 isPrivateHost 的 IP 判定。
// 解析失败返回 nil（按普通域名处理，交由拨号层失败兜底）。
func normalizeIPLiteral(h string) net.IP {
	// 含冒号视为 IPv6，标准形式 ParseIP 已覆盖，不做数值归一化
	if strings.Contains(h, ":") {
		return nil
	}
	// 无点纯数值：整体为 32 位地址（十进制/0x 十六进制/0 前缀八进制）
	if !strings.Contains(h, ".") {
		v, err := strconv.ParseUint(h, 0, 64)
		if err != nil || v > 0xFFFFFFFF {
			return nil
		}
		return ipv4FromParts(uint32(v)>>24, uint32(v)>>16, uint32(v)>>8, uint32(v))
	}
	// 点分形式：各段允许十进制/0x 十六进制/0 前缀八进制；末段按 inet_aton
	// 语义吸收剩余字节（a.b.c.d / a.b.c / a.b）
	parts := strings.Split(h, ".")
	if len(parts) > 4 {
		return nil
	}
	vals := make([]uint32, 0, 4)
	for i, p := range parts {
		if p == "" {
			return nil
		}
		v, err := strconv.ParseUint(p, 0, 64)
		if err != nil {
			return nil
		}
		// 末段上限：4 段时单字节，3/2 段时吸收剩余 2/3 字节
		limit := uint64(0xFF)
		if i == len(parts)-1 && len(parts) < 4 {
			limit = uint64(1)<<(8*(4-len(parts)+1)) - 1
		}
		if v > limit {
			return nil
		}
		vals = append(vals, uint32(v))
	}
	switch len(vals) {
	case 4:
		return ipv4FromParts(vals[0], vals[1], vals[2], vals[3])
	case 3: // a.b.c → a.b.(c 高字节).(c 低字节)
		return ipv4FromParts(vals[0], vals[1], vals[2]>>8, vals[2])
	case 2: // a.b → a.(b 三字节拆分)
		return ipv4FromParts(vals[0], vals[1]>>16, vals[1]>>8, vals[1])
	default:
		return nil
	}
}

// ipv4FromParts 由四个字节构造标准 IPv4 地址。
func ipv4FromParts(a, b, c, d uint32) net.IP {
	return net.IPv4(byte(a), byte(b), byte(c), byte(d))
}

// NewReadURL 创建网页链接读取工具。
func NewReadURL(setting *services.SettingService, ctx *Context) tool.InvokableTool {
	return &readURLTool{setting: setting, ctx: ctx}
}
