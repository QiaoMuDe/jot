package tools

// 本文件实现 read_url 网页链接读取工具：模型在 ReAct 循环中发现用户消息
// 包含链接或要求阅读网页时调用，内部基于 eino-ext 官方 URL Document Loader
// 抓取网页并提取正文（默认 HTML 解析器，取 body 内容），支持按 offset/length
// 分页读取长网页：拼接全文后按 rune 偏移切片返回，并携带起止位置与总字符数，
// 模型可依据返回的结尾位置继续翻页直至读完。仅放行 http/https，避免 file://
// 等本地路径读取。
// SSRF 三层防护复用 ssrf.go 的共享客户端（含拨号期 DNS rebinding 校验与
// 响应体 1MB 限长）；isPrivateHost 额外做 inet_aton 数值编码 IP 归一化。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

	// skipURLGuard 测试注入缝：true 时跳过 validateHTTPURL 的内网/本机拒绝，
	// 仅供同包测试经 InvokableRun 访问 httptest 本机服务器（零值为 false，生产
	// 构造器不设置，内网防护不受影响）。
	skipURLGuard bool
}

// 编译期断言：确保 readURLTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*readURLTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 展示被读取的链接与起始位置（截断防超长），解析失败或为空时回退通用文案。
func (r *readURLTool) ActionText(argumentsInJSON string) string {
	var args struct {
		URL    string  `json:"url"`
		Offset float64 `json:"offset"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "阅读网页链接"
	}
	if args.URL = strings.TrimSpace(args.URL); args.URL != "" {
		if args.Offset > 0 {
			return fmt.Sprintf("阅读链接 %s 第 %d 字符起", TruncateRunes(args.URL, 30), int(args.Offset))
		}
		return "阅读链接 " + TruncateRunes(args.URL, 30)
	}
	return "阅读网页链接"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (r *readURLTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_url",
		Desc: "读取网页链接（URL）的内容并返回正文。当用户消息中包含链接、或要求阅读/总结/提取某个网页的内容时调用；也可在搜索工具返回结果不够深入时进一步打开搜索结果中的链接。注意：仅支持 http/https 链接；动态渲染（JS）的页面可能只能拿到部分内容。长网页可分页读取：首次调用省略 offset 从开头读，返回含\"第 X-Y 字符（共 N 字符）\"，续读时以上一段结尾位置 Y 作为 offset 调用；offset 超出内容范围表示已全部读完。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.String,
				Desc:     "要读取的网页链接，必须是 http/https 开头的完整 URL",
				Required: true,
			},
			"offset": {
				Type:     schema.Number,
				Desc:     "起始字符位置，从 0 开始；首次阅读可省略，续读时传上一段返回的结尾位置（必须小于内容总字符数）",
				Required: false,
			},
			"length": {
				Type:     schema.Number,
				Desc:     "本次读取的字符数，可选；缺省取 ai_read_url_max_chars 设置，上限 100000",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行链接读取：校验 URL → 校验 offset/length → 构建 loader
// （超时 + 浏览器 UA）→ 加载并拼接全部正文 → 按 rune 切片返回（携带起止位置
// 与总字符数，模型据此翻页）。错误路径（参数缺失 / 非法 scheme / 抓取失败 /
// offset 越界 / 空正文 / 用户取消）返回 error 经 WrapWithError 回填模型继续推理。
func (r *readURLTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		URL    string  `json:"url"`
		Offset float64 `json:"offset"`
		Length float64 `json:"length"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 read_url 参数失败: %w", err)
	}

	// 1. 校验 URL：仅放行 http/https 完整链接（测试注入缝 skipURLGuard 跳过内网拒绝）
	target := strings.TrimSpace(args.URL)
	if !r.skipURLGuard {
		var err error
		if target, err = validateHTTPURL(args.URL); err != nil {
			return "", err
		}
	}

	// 2. 校验 offset/length（对齐 read_note_section）：offset 默认 0，须为 >=0
	//    整数；length 缺省取设置，须为 >=0 整数，上限 maxSectionLen。放在抓取
	//    之前，非法参数直接报错，避免白费一次整页抓取
	if args.Offset < 0 {
		return "", errors.New("read_url 的 offset 须为 >=0 的整数")
	}
	if args.Offset != math.Trunc(args.Offset) {
		return "", errors.New("read_url 的 offset 须为整数")
	}
	if args.Length < 0 {
		return "", errors.New("read_url 的 length 须为 >=0 的整数")
	}
	if args.Length != math.Trunc(args.Length) {
		return "", errors.New("read_url 的 length 须为整数")
	}
	length := int(args.Length)
	if length <= 0 {
		length = getIntSetting(r.setting, "ai_read_url_max_chars", 10000, 50000)
	}
	if length > maxSectionLen {
		length = maxSectionLen
	}

	// 3. 构建 loader：默认 HTML 解析器提取正文；复用共享防护客户端（SSRF 三层
	//    防护，含拨号期 DNS rebinding 校验与响应体限长，见 ssrf.go），浏览器 UA
	//    规避 403。测试经 skipURLGuard 一并跳过拨号期校验，放行本机地址
	loader, err := urlLoader.NewLoader(ctx, &urlLoader.LoaderConfig{
		Client: newGuardedHTTPClient(readURLTimeout, !r.skipURLGuard),
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

	// 4. 加载文档并提取正文：可能返回多个 document，全部拼接成全文（分页切片源）
	docs, err := loader.Load(ctx, document.Source{URI: target})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("读取链接失败: %w", err)
	}

	var b strings.Builder
	for _, d := range docs {
		if d == nil {
			continue
		}
		trimmed := strings.TrimSpace(d.Content)
		if trimmed == "" {
			continue
		}
		// 分隔符仅在文档之间插入，避免正文尾部残留空行污染偏移语义
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(trimmed)
	}
	if b.Len() == 0 {
		return "", errors.New("未能从该链接提取到正文内容（可能页面为空或为动态渲染页面）")
	}

	// 5. 按 rune 偏移切片（支持中文）；offset 越界视为已读完，回填模型停止翻页。
	//    巨型 offset 经 int 转换可能溢出为负，一并按越界处理
	runes := []rune(b.String())
	total := len(runes)
	offset := int(args.Offset)
	if offset < 0 || offset >= total {
		return "", fmt.Errorf("read_url 的 offset 超出内容范围（共 %d 字符，已全部读取完毕）", total)
	}
	end := offset + length
	if end > total {
		end = total
	}
	section := string(runes[offset:end])

	// 6. 组织返回：携带起止位置与总字符数，未读完时提示续读 offset
	msg := fmt.Sprintf("以下为链接 %s 第 %d-%d 字符的内容（共 %d 字符）：\n%s",
		target, offset+1, end, total, section)
	if end < total {
		msg += fmt.Sprintf("\n（内容未完，如需继续请以 offset=%d 调用）", end)
	}

	if r.ctx != nil && r.ctx.Logger != nil {
		r.ctx.Logger.Debugw("Agent read_url 调用",
			fastlog.String("url", target),
			fastlog.Int("offset", offset),
			fastlog.Int("end", end),
			fastlog.Int("total", total),
			fastlog.Int("chars", len([]rune(section))))
	}
	return msg, nil
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
