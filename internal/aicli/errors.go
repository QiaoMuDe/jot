package aicli

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"

	openai "github.com/sashabaranov/go-openai"
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
	CategoryUnknown:                 "AI 调用出错，请稍后重试",
}

// NewAIError 创建 AIError
func NewAIError(category, raw string) *AIError {
	return &AIError{
		Category: category,
		UserMsg:  userMessages[category],
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
func ClassifyError(err error) *AIError {
	if err == nil {
		return nil
	}

	raw := err.Error()

	// 检测 OpenAI API 错误
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		return classifyOpenAIAPIError(apiErr, raw)
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

// classifyOpenAIAPIError 分类 OpenAI APIError
func classifyOpenAIAPIError(apiErr *openai.APIError, raw string) *AIError {
	statusCode := apiErr.HTTPStatusCode
	msg := apiErr.Message
	code := apiErr.Code

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

// classifyOpenAIRequestError 分类 OpenAI RequestError
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
	case strings.Contains(lower, "content_filter"):
		return NewAIError(CategoryContentFilter, rawMsg(msg))
	case strings.Contains(lower, "context_length") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "maximum context"):
		return NewAIError(CategoryContextLength, rawMsg(msg))
	case strings.Contains(lower, "enable_thinking") ||
		(strings.Contains(lower, "reasoning") && strings.Contains(lower, "not supported")):
		return NewAIError(CategoryModelNotSupportThinking, rawMsg(msg))
	default:
		return NewAIError(CategoryInvalidRequest, rawMsg(msg))
	}
}

// classifyByText 根据错误文本内容进行 fallback 分类
func classifyByText(raw string) *AIError {
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many requests"):
		return NewAIError(CategoryRateLimit, raw)
	case strings.Contains(lower, "auth") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "api key"):
		return NewAIError(CategoryAuthError, raw)
	case strings.Contains(lower, "5") && strings.Contains(lower, "server error"):
		return NewAIError(CategoryServerError, raw)
	case strings.Contains(lower, "quota") ||
		strings.Contains(lower, "insufficient"):
		return NewAIError(CategoryQuotaExceeded, raw)
	case strings.Contains(lower, "not found") ||
		strings.Contains(lower, "model"):
		return NewAIError(CategoryModelNotFound, raw)
	case strings.Contains(lower, "deadline") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "timed out"):
		return NewAIError(CategoryTimeout, raw)
	case strings.Contains(lower, "dns") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "eof"):
		return NewAIError(CategoryNetworkError, raw)
	default:
		return NewAIError(CategoryUnknown, raw)
	}
}

// rawMsg 辅助函数：将错误消息包装为 raw 字符串
func rawMsg(msg string) string {
	return msg
}
