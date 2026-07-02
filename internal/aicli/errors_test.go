package aicli

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	openai "github.com/sashabaranov/go-openai"
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
