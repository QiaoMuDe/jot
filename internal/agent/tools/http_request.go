package tools

// 本文件实现 http_request HTTP API 调用工具：模型需要调用第三方 REST API
// （如查询开放接口数据）或获取原始响应体时调用，基于标准库 net/http 发送
// GET/POST/PUT/DELETE 请求，返回状态行、关键响应头与响应体原文（按设置截断）。
// 与 read_url 的分工：read_url 面向网页正文提取（HTML 解析），本工具面向
// API / 原始响应（不做任何解析加工，模型依据原始输出自行推理）。
// 实现：标准库 net/http；SSRF 三层防护——① validateHTTPURL 校验初始 URL 仅放行
// http/https 公网地址；② CheckRedirect 对每个重定向目标逐跳 isPrivateHost 校验；
// ③ DialContext 在拨号前解析出全部 IP 逐个校验并直连已校验 IP（DNS rebinding 防护）。
// 三层防护的共享实现见 ssrf.go（与 read_url 共用 newGuardedHTTPClient）。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

const (
	// httpTimeout 请求超时时间（过长则视为失败，避免卡住 ReAct 循环）。
	httpTimeout = 15 * time.Second
)

// binaryMediaTypes 视为二进制的媒体类型集合（命中则不输出响应体正文）。
var binaryMediaTypes = map[string]bool{
	"application/octet-stream": true,
	"application/pdf":          true,
	"application/zip":          true,
	"application/gzip":         true,
	"application/x-gzip":       true,
	"application/wasm":         true,
}

// httpRequestArgs http_request 工具的调用参数。
type httpRequestArgs struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// httpRequestTool HTTP API 调用工具。
type httpRequestTool struct {
	setting *services.SettingService // 响应体截断上限设置（复用 ai_http_max_chars）
	ctx     *Context                 // 日志

	// skipURLGuard 测试注入缝：true 时跳过 validateHTTPURL 的内网/本机拒绝，
	// 仅供同包测试经 invoke 访问 httptest 本机服务器（零值为 false，生产构造器
	// 不设置，内网防护不受影响）。
	skipURLGuard bool
}

// 编译期断言：确保 httpRequestTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*httpRequestTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 展示请求方法与目标地址（截断防超长），解析失败或为空时回退通用文案。
func (h *httpRequestTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "发起 HTTP 请求"
	}
	if args.URL = strings.TrimSpace(args.URL); args.URL != "" {
		method := strings.ToUpper(strings.TrimSpace(args.Method))
		if method == "" {
			method = http.MethodGet
		}
		return fmt.Sprintf("请求 %s %s", method, TruncateRunes(args.URL, 30))
	}
	return "发起 HTTP 请求"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (h *httpRequestTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "http_request",
		Desc: "调用 HTTP API 并返回原始响应（状态行、关键响应头与响应体）。当需要调用第三方 REST API（如查询天气/翻译/汇率等开放接口）、发送 POST/PUT 请求、或需要自定义请求头与原始响应体时调用；若只是阅读网页正文请改用 read_url。注意：仅支持 http/https 公网地址；4xx/5xx 也会原样返回，请依据状态码推理。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.String,
				Required: true,
				Desc:     "完整的 http/https 请求地址",
			},
			"method": {
				Type: schema.String,
				Enum: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
				Desc: "HTTP 方法，默认 GET",
			},
			"headers": {
				Type: schema.Object,
				Desc: "可选的自定义请求头，string 到 string 的键值对",
			},
			"body": {
				Type: schema.String,
				Desc: "可选的请求体原文（如 JSON 字符串）；GET 请求忽略",
			},
		}),
	}, nil
}

// InvokableRun 执行 HTTP API 调用：解析参数 → 构造带 SSRF 三层防护的 client →
// 校验（method 白名单 / body 长度 / URL）并发送 → 格式化响应输出。
// 4xx/5xx 不作为工具失败，正常返回（模型依据状态码推理）；错误路径（参数缺失 /
// 内网地址 / 请求失败 / 用户取消）返回 error 经 WrapWithError 回填模型继续推理。
func (h *httpRequestTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args httpRequestArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 http_request 参数失败: %w", err)
	}
	// 生产路径：client 含重定向逐跳校验与拨号期 DNS 解析校验（见 buildClient）；
	// URL 校验在 invoke 内完成。测试可注入 buildClient(false) 的 client 访问
	// httptest 本机服务器。
	return h.invoke(ctx, h.buildClient(true), args)
}

// invoke 发送 HTTP 请求并格式化响应输出：校验（method / body / URL）→ 构造请求
// → 应用请求头 → 发送 → 读取响应体（限 1MB）→ 组织输出。client 由调用方注入，
// 生产为 buildClient(true)，测试注入 buildClient(false) 以访问本机 httptest 服务器。
func (h *httpRequestTool) invoke(ctx context.Context, client *http.Client, args httpRequestArgs) (string, error) {
	// 1. 校验 method：空缺省 GET，仅放行五个常用方法（防止奇怪动词干扰下游）
	method := strings.ToUpper(strings.TrimSpace(args.Method))
	if method == "" {
		method = http.MethodGet
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
	default:
		return "", fmt.Errorf("method 仅支持 GET/POST/PUT/DELETE/PATCH，收到: %s", method)
	}

	// 2. 校验请求体长度：防止超长 body 撑爆模型上下文
	if args.Body != "" {
		if err := validateTextLen("body", args.Body, maxToolLongText); err != nil {
			return "", err
		}
	}

	// 3. 校验 URL：仅放行 http/https 公网地址（复用 read_url 的校验逻辑）；
	// 测试注入缝（skipURLGuard）跳过内网拒绝，使 httptest 本机地址可达
	target := strings.TrimSpace(args.URL)
	if !h.skipURLGuard {
		var err error
		if target, err = validateHTTPURL(args.URL); err != nil {
			return "", err
		}
	}

	// 4. 构造请求：GET 或 body 为空时不携带请求体（GET 的 body 被忽略）
	var bodyReader io.Reader
	if method != http.MethodGet && args.Body != "" {
		bodyReader = strings.NewReader(args.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bodyReader)
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}

	// 5. 应用自定义请求头：Host 由 net/http 依据 URL 管理，跳过避免冲突
	for k, v := range args.Headers {
		if strings.EqualFold(k, "Host") {
			continue
		}
		req.Header.Set(k, v)
	}
	// 实际发送请求体且未显式指定 Content-Type 时默认按 JSON 处理
	// （API 调用最常见的请求体格式，省去模型每次手写该头）
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	// 未指定 User-Agent 时用浏览器 UA（规避部分 API 网关对非浏览器 UA 的拦截）
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", browserUserAgent)
	}

	// 6. 发送请求：用户取消时原样返回取消错误（外层不误报工具失败）
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 7. 读取响应体（限 1MB，防止超大响应撑爆内存）
	body, err := io.ReadAll(io.LimitReader(resp.Body, httpMaxBodyBytes))
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 8. 组织输出：状态行 + 最终地址（重定向时） + 关键响应头 + 响应体
	// （二进制省略、文本按设置截断）
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", resp.Proto, resp.Status)
	// 跟随重定向后最终地址与初始 URL 不同时额外展示，帮助模型理解跳转链
	if resp.Request != nil && resp.Request.URL != nil && resp.Request.URL.String() != target {
		fmt.Fprintf(&b, "\n最终地址（跟随重定向后）: %s", resp.Request.URL)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		fmt.Fprintf(&b, "\nContent-Type: %s", ct)
	}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		fmt.Fprintf(&b, "\nContent-Length: %s", cl)
	}
	b.WriteString("\n\n")

	// 二进制内容（图片/音视频/常见二进制类型）不输出正文，仅提示类型与字节数
	mediaType := strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0])
	if strings.HasPrefix(mediaType, "image/") || strings.HasPrefix(mediaType, "audio/") ||
		strings.HasPrefix(mediaType, "video/") || binaryMediaTypes[mediaType] {
		fmt.Fprintf(&b, "响应体为二进制内容（%s，%d 字节），已省略", mediaType, len(body))
	} else {
		// 文本按 rune 截断（支持中文），超长时追加提示，避免撑爆模型上下文窗口
		text := string(body)
		maxChars := getIntSetting(h.setting, "ai_http_max_chars", 5000, 50000)
		if runes := []rune(text); len(runes) > maxChars {
			text = string(runes[:maxChars]) + "\n\n（内容过长，已截断）"
		}
		b.WriteString(text)
	}

	// 日志仅记录方法、URL、状态码、耗时与响应字节数（禁止输出请求头，避免泄露密钥）
	if h.ctx != nil && h.ctx.Logger != nil {
		h.ctx.Logger.Debugw("Agent http_request 调用",
			fastlog.String("method", method),
			fastlog.String("url", target),
			fastlog.Int("status", resp.StatusCode),
			fastlog.Int("duration_ms", int(time.Since(start).Milliseconds())),
			fastlog.Int("resp_bytes", len(body)))
	}
	return b.String(), nil
}

// buildClient 构造 HTTP 客户端，委托给共享的 newGuardedHTTPClient（SSRF 三层
// 防护实现见 ssrf.go，read_url 与本工具共用）。guardDial=false 时跳过拨号期
// DNS 校验，仅供测试访问 httptest 本机服务器。
func (h *httpRequestTool) buildClient(guardDial bool) *http.Client {
	return newGuardedHTTPClient(httpTimeout, guardDial)
}

// NewHTTP 创建 HTTP API 调用工具。
func NewHTTP(setting *services.SettingService, ctx *Context) tool.InvokableTool {
	return &httpRequestTool{setting: setting, ctx: ctx}
}
