package aierrors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	openai "github.com/meguminnnnnnnnn/go-openai"
)

func TestClassifyError_AuthError_401(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 401,
		Message:        "Incorrect API key provided",
		Type:           "invalid_request_error",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryAuthError {
		t.Errorf("expected category %q, got %q", CategoryAuthError, ae.Category)
	}
	if ae.UserMsg == "" {
		t.Error("expected non-empty UserMsg")
	}
}

func TestClassifyError_AuthError_403(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 403,
		Message:        "Forbidden",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryAuthError {
		t.Errorf("expected category %q, got %q", CategoryAuthError, ae.Category)
	}
}

func TestClassifyError_RateLimit_429(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 429,
		Message:        "Too Many Requests",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryRateLimit {
		t.Errorf("expected category %q, got %q", CategoryRateLimit, ae.Category)
	}
}

func TestClassifyError_ServerError_500(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 500,
		Message:        "Internal Server Error",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryServerError {
		t.Errorf("expected category %q, got %q", CategoryServerError, ae.Category)
	}
}

func TestClassifyError_ServerError_503(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 503,
		Message:        "Service Unavailable",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryServerError {
		t.Errorf("expected category %q, got %q", CategoryServerError, ae.Category)
	}
}

func TestClassifyError_Timeout_DeadlineExceeded(t *testing.T) {
	err := context.DeadlineExceeded
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryTimeout {
		t.Errorf("expected category %q, got %q", CategoryTimeout, ae.Category)
	}
}

func TestClassifyError_NetworkError_NetOpError(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryNetworkError {
		t.Errorf("expected category %q, got %q", CategoryNetworkError, ae.Category)
	}
}

func TestClassifyError_NetworkError_DNSError(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &net.DNSError{Err: "no such host", Name: "api.example.com"},
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryNetworkError {
		t.Errorf("expected category %q, got %q", CategoryNetworkError, ae.Category)
	}
}

func TestClassifyError_ModelNotFound_404(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 404,
		Message:        "Model not found",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryModelNotFound {
		t.Errorf("expected category %q, got %q", CategoryModelNotFound, ae.Category)
	}
}

func TestClassifyError_QuotaExceeded_402(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 402,
		Message:        "Insufficient balance",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryQuotaExceeded {
		t.Errorf("expected category %q, got %q", CategoryQuotaExceeded, ae.Category)
	}
}

func TestClassifyError_ContextLength_400(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 400,
		Message:        "context_length_exceeded: maximum context length is 8192 tokens",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryContextLength {
		t.Errorf("expected category %q, got %q", CategoryContextLength, ae.Category)
	}
}

func TestClassifyError_ContentFilter_400(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 400,
		Message:        "content_filter policy violation",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryContentFilter {
		t.Errorf("expected category %q, got %q", CategoryContentFilter, ae.Category)
	}
}

func TestClassifyError_RequestError(t *testing.T) {
	err := &openai.RequestError{
		HTTPStatusCode: 401,
		Err:            errors.New("401 Unauthorized"),
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryAuthError {
		t.Errorf("expected category %q, got %q", CategoryAuthError, ae.Category)
	}
}

func TestClassifyError_NilError(t *testing.T) {
	ae := ClassifyError(nil)
	if ae != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestClassifyError_Canceled(t *testing.T) {
	err := context.Canceled
	ae := ClassifyError(err)
	if ae != nil {
		t.Fatal("expected nil for context.Canceled")
	}
}

func TestAIError_ToJSON(t *testing.T) {
	ae := NewAIError(CategoryAuthError, "test raw")
	jsonStr := ae.ToJSON()
	if jsonStr == "" {
		t.Fatal("expected non-empty JSON")
	}
}

func TestClassifyByText_RateLimit(t *testing.T) {
	err := errors.New("rate limit exceeded, too many requests")
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryRateLimit {
		t.Errorf("expected category %q, got %q", CategoryRateLimit, ae.Category)
	}
}

func TestClassifyByText_NetworkError(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "https://api.example.com",
		Err: errors.New("no such host"),
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryNetworkError {
		t.Errorf("expected category %q, got %q", CategoryNetworkError, ae.Category)
	}
}

func TestClassifyError_EinoAPIError_AuthError_401(t *testing.T) {
	err := &einoopenai.APIError{
		HTTPStatusCode: 401,
		Message:        "Incorrect API key provided",
		Type:           "invalid_request_error",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryAuthError {
		t.Errorf("expected category %q, got %q", CategoryAuthError, ae.Category)
	}
	if ae.UserMsg == "" {
		t.Error("expected non-empty UserMsg")
	}
}

func TestClassifyError_EinoAPIError_RateLimit_429(t *testing.T) {
	err := &einoopenai.APIError{
		HTTPStatusCode: 429,
		Message:        "Too Many Requests",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryRateLimit {
		t.Errorf("expected category %q, got %q", CategoryRateLimit, ae.Category)
	}
}

func TestClassifyError_EinoAPIError_ContextLength_400(t *testing.T) {
	err := &einoopenai.APIError{
		HTTPStatusCode: 400,
		Message:        "context_length_exceeded: maximum context length is 8192 tokens",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryContextLength {
		t.Errorf("expected category %q, got %q", CategoryContextLength, ae.Category)
	}
}

// TestClassifyError_NotSupportThinking_Ollama 覆盖 Ollama 返回
// `"model:tag" does not support thinking` 的 400 报错（日志中的真实场景）。
func TestClassifyError_NotSupportThinking_Ollama(t *testing.T) {
	err := &einoopenai.APIError{
		HTTPStatusCode: 400,
		Message:        "\"jewelzufo/MiniCPM5-1B:latest\" does not support thinking",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryModelNotSupportThinking {
		t.Errorf("expected category %q, got %q", CategoryModelNotSupportThinking, ae.Category)
	}
}

// TestClassifyError_NotSupportThinking_Reasoning 覆盖 "does not support reasoning" 类报错。
func TestClassifyError_NotSupportThinking_Reasoning(t *testing.T) {
	err := &einoopenai.APIError{
		HTTPStatusCode: 400,
		Message:        "model qwen3 does not support reasoning",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryModelNotSupportThinking {
		t.Errorf("expected category %q, got %q", CategoryModelNotSupportThinking, ae.Category)
	}
}

// TestClassifyError_NotSupportTools 覆盖 Ollama 返回
// `... does not support tools` 的 400 报错（日志中的真实场景）。
func TestClassifyError_NotSupportTools(t *testing.T) {
	err := &einoopenai.APIError{
		HTTPStatusCode: 400,
		Message:        "registry.ollama.ai/jewelzufo/MiniCPM5-1B:latest does not support tools",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryModelNotSupportTools {
		t.Errorf("expected category %q, got %q", CategoryModelNotSupportTools, ae.Category)
	}
}

// TestClassifyError_NotSupportThinking_Wrapped 覆盖 NodeRunError 链：
// eino 的 internalError 用 %w 包装底层 APIError，errors.As 应能穿透。
func TestClassifyError_NotSupportThinking_Wrapped(t *testing.T) {
	apiErr := &einoopenai.APIError{
		HTTPStatusCode: 400,
		Message:        "\"jewelzufo/MiniCPM5-1B:latest\" does not support thinking",
	}
	err := fmt.Errorf("[NodeRunError] %w", apiErr)
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryModelNotSupportThinking {
		t.Errorf("expected category %q, got %q", CategoryModelNotSupportThinking, ae.Category)
	}
}

// TestClassifyError_NotSupportTools_TextFallback 覆盖无 APIError 类型、
// 仅剩错误文本的兜底分类路径。
func TestClassifyError_NotSupportTools_TextFallback(t *testing.T) {
	err := errors.New("registry.ollama.ai/jewelzufo/MiniCPM5-1B:latest does not support tools")
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryModelNotSupportTools {
		t.Errorf("expected category %q, got %q", CategoryModelNotSupportTools, ae.Category)
	}
}

// TestClassifyError_WrapperPreserved 覆盖二次分类场景：
// 上层对已包装的 AIErrorWrapper 再次调用 ClassifyError，应直接还原原分类，
// 而不是按 wrapper 的 JSON 文本重新匹配导致误判。
func TestClassifyError_WrapperPreserved(t *testing.T) {
	inner := NewAIError(CategoryModelNotSupportThinking, "raw thinking error")
	wrapped := &AIErrorWrapper{Err: inner}
	ae := ClassifyError(wrapped)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryModelNotSupportThinking {
		t.Errorf("expected category %q, got %q", CategoryModelNotSupportThinking, ae.Category)
	}
	if ae.UserMsg == "" {
		t.Error("expected non-empty UserMsg")
	}
}

// TestClassifyError_WrapperPreserved_Wrapped 覆盖 AIErrorWrapper 被 %w 包装后的还原。
func TestClassifyError_WrapperPreserved_Wrapped(t *testing.T) {
	inner := NewAIError(CategoryAuthError, "raw auth error")
	wrapped := &AIErrorWrapper{Err: inner}
	err := fmt.Errorf("调用 AI 失败: %w", wrapped)
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryAuthError {
		t.Errorf("expected category %q, got %q", CategoryAuthError, ae.Category)
	}
}

// TestClassifyError_InvalidToken_NotContextLength 覆盖 400 "invalid token"
// 不应被宽泛的 token 关键词误判为上下文超长。
func TestClassifyError_InvalidToken_NotContextLength(t *testing.T) {
	err := &openai.APIError{
		HTTPStatusCode: 400,
		Message:        "invalid token: authentication token is malformed",
	}
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category == CategoryContextLength {
		t.Error("expected category NOT to be context_length for invalid token")
	}
}

// TestClassifyError_ResponseFormat_InvalidJSON 覆盖非 JSON 响应（本地模型常见）的分类。
func TestClassifyError_ResponseFormat_InvalidJSON(t *testing.T) {
	err := errors.New("invalid character '<' looking for beginning of value")
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryResponseFormat {
		t.Errorf("expected category %q, got %q", CategoryResponseFormat, ae.Category)
	}
}

// TestClassifyError_ServerError_Text 覆盖 OpenAI 常见 500 文本
// "The server had an error while processing your request"（不含 "server error" 连写）。
func TestClassifyError_ServerError_Text(t *testing.T) {
	err := errors.New("The server had an error while processing your request. Sorry about that!")
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category != CategoryServerError {
		t.Errorf("expected category %q, got %q", CategoryServerError, ae.Category)
	}
}

// TestClassifyError_ModelBusy_NotModelNotFound 覆盖含 "model" 但并非"模型不存在"
// 的错误不再被宽泛关键词误判。
func TestClassifyError_ModelBusy_NotModelNotFound(t *testing.T) {
	err := errors.New("model is currently busy, please try again later")
	ae := ClassifyError(err)
	if ae == nil {
		t.Fatal("expected non-nil AIError")
	}
	if ae.Category == CategoryModelNotFound {
		t.Error("expected category NOT to be model_not_found for model busy error")
	}
}
