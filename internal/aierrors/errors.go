package aierrors

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	openai "github.com/meguminnnnnnnnn/go-openai"
)

// 错误分类常量
const (
	CategoryAuthError               = "auth_error"
	CategoryRateLimit               = "rate_limit"
	CategoryServerError             = "server_error"
	CategoryQuotaExceeded           = "quota_exceeded"
	CategoryModelNotFound           = "model_not_found"
	CategoryContextLength           = "context_length"
	CategoryTimeout                 = "timeout"
	CategoryInvalidRequest          = "invalid_request"
	CategoryContentFilter           = "content_filter"
	CategoryNetworkError            = "network_error"
	CategoryUnknown                 = "unknown"
	CategoryModelNotSupportThinking = "model_not_support_thinking"
	CategoryModelNotSupportTools    = "model_not_support_tools"
	CategoryResponseFormat          = "response_format"
)

// AIError 结构化错误信息
type AIError struct {
	Category string `json:"category"`
	UserMsg  string `json:"user_msg"`
	Raw      string `json:"raw"`
}

// userMessages 中文提示映射
var userMessages = map[string]string{
	CategoryAuthError:               "API 密钥无效或权限不足，请检查 API 配置",
	CategoryRateLimit:               "请求过于频繁，已达速率限制，请稍后重试",
	CategoryServerError:             "AI 服务暂时不可用，请稍后重试",
	CategoryQuotaExceeded:           "API 额度已用尽，请检查账户余额",
	CategoryModelNotFound:           "模型不存在或已弃用，请更换模型名称",
	CategoryContextLength:           "上下文长度超限，请缩短对话历史或笔记内容",
	CategoryTimeout:                 "请求超时，请检查网络连接或稍后重试",
	CategoryInvalidRequest:          "请求参数有误，请检查输入内容",
	CategoryContentFilter:           "内容被安全策略拦截，请调整输入后重试",
	CategoryNetworkError:            "网络连接失败，请检查网络设置或 API 地址",
	CategoryModelNotSupportThinking: "当前模型不支持「深度思考」功能，请在输入框上方关闭深度思考开关后重试",
	CategoryModelNotSupportTools:    "当前模型不支持工具调用，无法使用 Agent 模式，请更换支持 tool calling 的模型后重试",
	CategoryResponseFormat:          "AI 服务返回了无法解析的响应，请检查服务状态或更换模型",
	CategoryUnknown:                 "AI 调用出错，请稍后重试",
}

// NewAIError 创建 AIError
func NewAIError(category, raw string) *AIError {
	msg, ok := userMessages[category]
	if !ok {
		// 未知分类兜底到通用文案，避免 UserMsg 为空导致前端展示原始错误
		msg = userMessages[CategoryUnknown]
	}
	return &AIError{
		Category: category,
		UserMsg:  msg,
		Raw:      raw,
	}
}

// ToJSON 将 AIError 序列化为 JSON 字符串
func (e *AIError) ToJSON() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// AIErrorWrapper 包装分类后的 AIError，使其可以通过 error 接口传递
type AIErrorWrapper struct {
	Err *AIError
}

func (e *AIErrorWrapper) Error() string {
	return e.Err.ToJSON()
}

// ClassifyError 对错误进行分类，返回结构化错误信息
// 同时兼容两类错误来源：
//   - eino：eino-ext components/model/openai 转换后的 *APIError
//   - go-openai 家族（sashabaranov 等）：*APIError / *RequestError
func ClassifyError(err error) *AIError {
	if err == nil {
		return nil
	}

	raw := err.Error()

	// 已分类过的 AIErrorWrapper 直接还原，避免二次分类退化为通用/误判分类
	// （wrapper 的 Error() 只是 JSON 文本，按文本匹配会得到错误结果）
	var wrapped *AIErrorWrapper
	if errors.As(err, &wrapped) {
		return wrapped.Err
	}

	// 检测 eino 转换后的错误类型
	var einoErr *einoopenai.APIError
	if errors.As(err, &einoErr) {
		return classifyAPIErrorLike(einoErr.HTTPStatusCode, einoErr.Code, einoErr.Message, raw)
	}

	// 检测 go-openai 家族 API 错误
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		return classifyAPIErrorLike(apiErr.HTTPStatusCode, apiErr.Code, apiErr.Message, raw)
	}

	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) {
		return classifyOpenAIRequestError(reqErr, raw)
	}

	// 检测 context 超时 / 取消
	if errors.Is(err, context.DeadlineExceeded) {
		return NewAIError(CategoryTimeout, raw)
	}

	// 检测网络错误
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return NewAIError(CategoryNetworkError, raw)
	}
	if errors.Is(err, context.Canceled) {
		// 用户主动取消不属于错误，返回 nil
		// 但有些情况下 context.Canceled 可能不是用户主动取消
		return nil
	}

	// 通用 fallback：根据错误文本匹配
	return classifyByText(raw)
}

// classifyAPIErrorLike 按 HTTP 状态码与错误码分类（eino 与 go-openai 的 APIError 字段一致，共用此函数）
func classifyAPIErrorLike(statusCode int, code any, msg, raw string) *AIError {
	switch statusCode {
	case 401, 403:
		return NewAIError(CategoryAuthError, raw)
	case 402:
		return NewAIError(CategoryQuotaExceeded, raw)
	case 429:
		if codeStr, ok := code.(string); ok && strings.Contains(codeStr, "insufficient_quota") {
			return NewAIError(CategoryQuotaExceeded, raw)
		}
		return NewAIError(CategoryRateLimit, raw)
	case 404:
		return NewAIError(CategoryModelNotFound, raw)
	case 500, 502, 503:
		return NewAIError(CategoryServerError, raw)
	case 400:
		return classifyBadRequest(msg)
	default:
		return NewAIError(CategoryUnknown, raw)
	}
}

// classifyOpenAIRequestError 分类 go-openai RequestError
func classifyOpenAIRequestError(reqErr *openai.RequestError, raw string) *AIError {
	// RequestError 通常包含 HTTP 状态码
	if reqErr.HTTPStatusCode != 0 {
		switch reqErr.HTTPStatusCode {
		case 401, 403:
			return NewAIError(CategoryAuthError, raw)
		case 429:
			return NewAIError(CategoryRateLimit, raw)
		default:
			if reqErr.HTTPStatusCode >= 500 {
				return NewAIError(CategoryServerError, raw)
			}
		}
	}

	// 如果请求错误包含底层错误，尝试递归分类
	if reqErr.Err != nil {
		if classified := ClassifyError(reqErr.Err); classified != nil {
			return classified
		}
	}

	return NewAIError(CategoryNetworkError, raw)
}

// classifyBadRequest 分类 400 Bad Request 的具体原因
func classifyBadRequest(msg string) *AIError {
	lower := strings.ToLower(msg)
	switch {
	// 内容安全拦截（OpenAI content_filter / 通义、DeepSeek 的 data inspection 等）
	case strings.Contains(lower, "content_filter") ||
		strings.Contains(lower, "content filter") ||
		strings.Contains(lower, "was filtered") ||
		strings.Contains(lower, "data inspection") ||
		strings.Contains(lower, "content safety"):
		return NewAIError(CategoryContentFilter, msg)
	// 上下文长度超限：token 需与 exceed/too many/limit 等组合，
	// 避免 "invalid token"、"api token" 等无关错误被误判为上下文超长
	case strings.Contains(lower, "context_length") ||
		strings.Contains(lower, "context length") ||
		strings.Contains(lower, "maximum context") ||
		(strings.Contains(lower, "token") &&
			(strings.Contains(lower, "exceed") || strings.Contains(lower, "too many") ||
				strings.Contains(lower, "too long") || strings.Contains(lower, "limit"))):
		return NewAIError(CategoryContextLength, msg)
	// 深度思考不支持：兼容 OpenAI 的 enable_thinking 参数报错与 Ollama 等
	// "does not support thinking/reasoning" 类报错
	case strings.Contains(lower, "enable_thinking") ||
		(strings.Contains(lower, "thinking") || strings.Contains(lower, "reasoning")) &&
			(strings.Contains(lower, "not support") || strings.Contains(lower, "unsupported")):
		return NewAIError(CategoryModelNotSupportThinking, msg)
	// 工具调用不支持：Ollama 等 "does not support tools" / "tools not supported" 类报错
	case (strings.Contains(lower, "tool") || strings.Contains(lower, "function calling")) &&
		(strings.Contains(lower, "not support") || strings.Contains(lower, "unsupported")):
		return NewAIError(CategoryModelNotSupportTools, msg)
	default:
		return NewAIError(CategoryInvalidRequest, msg)
	}
}

// classifyByText 根据错误文本内容进行 fallback 分类
func classifyByText(raw string) *AIError {
	lower := strings.ToLower(raw)
	switch {
	// 服务端错误：匹配常见短语（含 OpenAI 的 "server had an error"），
	// 不再依赖 "5" + "server error" 的脆弱组合
	case strings.Contains(lower, "internal server error") ||
		strings.Contains(lower, "server error") ||
		strings.Contains(lower, "bad gateway") ||
		strings.Contains(lower, "service unavailable") ||
		strings.Contains(lower, "server had an error"):
		return NewAIError(CategoryServerError, raw)
	case strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many requests"):
		return NewAIError(CategoryRateLimit, raw)
	case strings.Contains(lower, "auth") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "api key"):
		return NewAIError(CategoryAuthError, raw)
	case strings.Contains(lower, "quota") ||
		strings.Contains(lower, "insufficient"):
		return NewAIError(CategoryQuotaExceeded, raw)
	// 内容安全拦截
	case strings.Contains(lower, "content_filter") ||
		strings.Contains(lower, "content filter") ||
		strings.Contains(lower, "was filtered") ||
		strings.Contains(lower, "data inspection") ||
		strings.Contains(lower, "content safety"):
		return NewAIError(CategoryContentFilter, raw)
	// 响应解析失败：非 JSON 响应、非法 JSON（本地/Ollama 等模型常见）
	case strings.Contains(lower, "invalid character") ||
		strings.Contains(lower, "unexpected end of json") ||
		strings.Contains(lower, "cannot unmarshal"):
		return NewAIError(CategoryResponseFormat, raw)
	// 兜底检测：深度思考/工具调用不支持（置于 model 分支前，避免被误判为模型不存在）
	case (strings.Contains(lower, "thinking") || strings.Contains(lower, "reasoning")) &&
		(strings.Contains(lower, "not support") || strings.Contains(lower, "unsupported")):
		return NewAIError(CategoryModelNotSupportThinking, raw)
	case (strings.Contains(lower, "tool") || strings.Contains(lower, "function calling")) &&
		(strings.Contains(lower, "not support") || strings.Contains(lower, "unsupported")):
		return NewAIError(CategoryModelNotSupportTools, raw)
	// 模型不存在：用精确短语代替宽泛的 "model"，避免 "model overloaded"、"model is busy" 等误判
	case strings.Contains(lower, "not found") ||
		strings.Contains(lower, "does not exist") ||
		strings.Contains(lower, "unknown model"):
		return NewAIError(CategoryModelNotFound, raw)
	case strings.Contains(lower, "deadline") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "timed out"):
		return NewAIError(CategoryTimeout, raw)
	case strings.Contains(lower, "dns") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "eof"):
		return NewAIError(CategoryNetworkError, raw)
	default:
		return NewAIError(CategoryUnknown, raw)
	}
}
