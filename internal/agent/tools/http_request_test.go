package tools

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHTTPRequestDialGuard 验证拨号期 DNS 解析校验（DNS rebinding 防护）：
// guardDial=true 时 127.0.0.1 命中回环黑名单被拒（错误文案含"内网"），
// guardDial=false（测试注入缝）时可正常访问 httptest 本机服务器。
func TestHTTPRequestDialGuard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	h := &httpRequestTool{}
	ctx := context.Background()

	// guardDial=true：拨号前解析 IP 并逐个校验，127.0.0.1 为回环地址 → 拒绝连接
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	resp, err := h.buildClient(true).Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("guardDial=true 访问 127.0.0.1 应被拒绝")
	}
	if !strings.Contains(err.Error(), "内网") {
		t.Errorf("错误信息应包含\"内网\"，实际: %v", err)
	}

	// guardDial=false：无防护拨号，可正常访问本机 httptest 服务器（测试注入缝）
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	resp2, err := h.buildClient(false).Do(req2)
	if err != nil {
		t.Fatalf("guardDial=false 访问本机服务器应成功，实际错误: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("状态码 = %d, want %d", resp2.StatusCode, http.StatusOK)
	}
}

// TestHTTPRequestMethodValidation 验证 method 白名单校验与缺省 GET。
func TestHTTPRequestMethodValidation(t *testing.T) {
	h := &httpRequestTool{}

	// 非法 method：校验在触网前返回，错误信息含支持的方法列表
	_, err := h.InvokableRun(context.Background(), `{"url":"https://example.com","method":"TRACE"}`)
	if err == nil {
		t.Fatal("TRACE 应被拒绝")
	}
	if !strings.Contains(err.Error(), "GET/POST/PUT/DELETE/PATCH") {
		t.Errorf("错误信息应包含 GET/POST/PUT/DELETE/PATCH，实际: %v", err)
	}

	// 缺省 method：经 httptest 验证服务端实际收到 GET
	// （skipURLGuard 注入缝跳过内网拒绝，使 httptest 本机地址可达）
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ht := &httpRequestTool{skipURLGuard: true}
	if _, err := ht.invoke(context.Background(), ht.buildClient(false), httpRequestArgs{URL: srv.URL}); err != nil {
		t.Fatalf("缺省 method 请求失败: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("服务端收到 %s, want GET", gotMethod)
	}
}

// TestHTTPRequestURLGuard 验证 SSRF 第一层（公开 InvokableRun 路径）：
// 内网/本机 URL 在触网前即被 validateHTTPURL 拒绝。
func TestHTTPRequestURLGuard(t *testing.T) {
	h := &httpRequestTool{}
	_, err := h.InvokableRun(context.Background(), `{"url":"http://127.0.0.1:8080/api"}`)
	if err == nil {
		t.Fatal("内网地址应被拒绝")
	}
	if !strings.Contains(err.Error(), "内网") {
		t.Errorf("错误信息应包含\"内网\"，实际: %v", err)
	}
}

// TestHTTPRequestActionText 验证 tool_start 动作文案：展示方法与 URL（截断
// 30 字符），method 缺省展示 GET，非法 JSON 回退通用文案。
func TestHTTPRequestActionText(t *testing.T) {
	h := &httpRequestTool{}

	// URL 超过 30 字符：截断到 30 rune 后追加省略号
	got := h.ActionText(`{"url":"https://api.example.com/data/list?foo=bar","method":"GET"}`)
	want := "请求 GET https://api.example.com/data/l..."
	if got != want {
		t.Errorf("ActionText = %q, want %q", got, want)
	}

	// method 缺省展示 GET
	if got := h.ActionText(`{"url":"https://api.example.com"}`); got != "请求 GET https://api.example.com" {
		t.Errorf("ActionText = %q, want %q", got, "请求 GET https://api.example.com")
	}

	// 非法 JSON：回退通用文案
	if got := h.ActionText(`{invalid`); got != "发起 HTTP 请求" {
		t.Errorf("非法 JSON 应回退通用文案，实际: %q", got)
	}

	// URL 为空：回退通用文案
	if got := h.ActionText(`{"method":"POST"}`); got != "发起 HTTP 请求" {
		t.Errorf("URL 为空应回退通用文案，实际: %q", got)
	}
}

// TestHTTPRequestSuccessPath 验证 GET/POST 成功路径（httptest + 无防护拨号的
// 测试注入 client）：自定义头与 body 如实到达服务端，Content-Type 缺省按 JSON，
// 响应输出首行为状态行且含响应体文本。
func TestHTTPRequestSuccessPath(t *testing.T) {
	// skipURLGuard 注入缝：跳过内网 URL 拒绝以访问 httptest 本机服务器
	h := &httpRequestTool{skipURLGuard: true}
	ctx := context.Background()

	t.Run("GET 成功", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hello API"))
		}))
		defer srv.Close()

		out, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{URL: srv.URL})
		if err != nil {
			t.Fatalf("GET 请求失败: %v", err)
		}
		if !strings.HasPrefix(out, "HTTP/1.1 200 OK") {
			t.Errorf("输出首行应为状态行，实际:\n%s", out)
		}
		if !strings.Contains(out, "hello API") {
			t.Errorf("输出应包含响应体文本，实际:\n%s", out)
		}
	})

	t.Run("POST 带 headers/body", func(t *testing.T) {
		var gotMethod, gotCT, gotToken, gotUA, gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotCT = r.Header.Get("Content-Type")
			gotToken = r.Header.Get("X-Token")
			gotUA = r.Header.Get("User-Agent")
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer srv.Close()

		out, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{
			URL:     srv.URL,
			Method:  "POST",
			Headers: map[string]string{"X-Token": "abc"},
			Body:    `{"query":"笔记"}`,
		})
		if err != nil {
			t.Fatalf("POST 请求失败: %v", err)
		}
		if gotMethod != http.MethodPost {
			t.Errorf("服务端收到 %s, want POST", gotMethod)
		}
		// 未显式指定 Content-Type 时缺省按 JSON
		if gotCT != "application/json" {
			t.Errorf("服务端 Content-Type = %q, want application/json", gotCT)
		}
		if gotToken != "abc" {
			t.Errorf("服务端 X-Token = %q, want abc", gotToken)
		}
		// 未指定 User-Agent 时使用浏览器 UA
		if gotUA != browserUserAgent {
			t.Errorf("服务端 User-Agent = %q, want 浏览器 UA", gotUA)
		}
		if gotBody != `{"query":"笔记"}` {
			t.Errorf("服务端 body = %q, want %q", gotBody, `{"query":"笔记"}`)
		}
		if !strings.Contains(out, "200 OK") || !strings.Contains(out, `{"ok":true}`) {
			t.Errorf("输出应包含 200 OK 与响应体，实际:\n%s", out)
		}
	})

	t.Run("POST 显式 Content-Type 优先", func(t *testing.T) {
		var gotCT string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotCT = r.Header.Get("Content-Type")
			_, _ = w.Write(nil)
		}))
		defer srv.Close()

		if _, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{
			URL:     srv.URL,
			Method:  "POST",
			Headers: map[string]string{"Content-Type": "text/plain"},
			Body:    "raw text",
		}); err != nil {
			t.Fatalf("POST 请求失败: %v", err)
		}
		if gotCT != "text/plain" {
			t.Errorf("显式 Content-Type 应优先，服务端收到 %q, want text/plain", gotCT)
		}
	})
}

// TestHTTPRequestTruncation 验证响应体超长时按设置截断（nil setting 回退默认
// 10000），截断提示追加在末尾。
func TestHTTPRequestTruncation(t *testing.T) {
	long := strings.Repeat("好", 12000) // 12000 rune > 默认上限 10000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(long))
	}))
	defer srv.Close()

	// setting 为 nil：getIntSetting 回退默认 10000；skipURLGuard 放行本机地址
	h := &httpRequestTool{skipURLGuard: true}
	out, err := h.invoke(context.Background(), h.buildClient(false), httpRequestArgs{URL: srv.URL})
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if !strings.Contains(out, "（内容过长，已截断）") {
		t.Errorf("输出应包含截断提示，实际:\n%s", out)
	}
	// 截断后保留 10000 rune 前缀，不含第 10001 个字符起的内容
	if !strings.Contains(out, strings.Repeat("好", 10000)) {
		t.Error("输出应包含前 10000 字符正文")
	}
	if strings.Contains(out, strings.Repeat("好", 10001)) {
		t.Error("输出不应包含超过 10000 字符的正文")
	}
}

// TestHTTPRequestApproval 验证 http_request 的审批门控：所有请求（含 GET）执行前
// 经 Approver 审批——GET 为常规操作（critical=false），POST/PUT/DELETE/PATCH 为
// 高危写操作（critical=true）；Approver 未注入时 fail-fast 报错；裸工具（ctx=nil）放行。
func TestHTTPRequestApproval(t *testing.T) {
	// 计数服务端：统计实际到达的请求数，用于断言拒绝路径未触网
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ctx := context.Background()

	// newApprovalHTTPTool 构造带 Approver 的 http_request 工具（skipURLGuard 放行本机地址）
	newApprovalHTTPTool := func(approver Approver) *httpRequestTool {
		return &httpRequestTool{ctx: &Context{Approver: approver}, skipURLGuard: true}
	}

	t.Run("GET 批准放行", func(t *testing.T) {
		hits = 0
		mock := &mockApprover{}
		h := newApprovalHTTPTool(mock)
		out, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{URL: srv.URL})
		if err != nil {
			t.Fatalf("GET 请求失败: %v", err)
		}
		if !strings.Contains(out, "ok") {
			t.Errorf("输出应包含响应体，实际:\n%s", out)
		}
		if mock.gotCritical {
			t.Error("GET 应为常规操作（critical=false）")
		}
		if !strings.Contains(mock.gotSum, "GET") || !strings.Contains(mock.gotSum, srv.URL) {
			t.Errorf("审批摘要应含 GET 与 URL，实际: %q", mock.gotSum)
		}
		if mock.gotTool != "http_request" {
			t.Errorf("审批工具名 = %q, want http_request", mock.gotTool)
		}
		if hits != 1 {
			t.Errorf("批准后请求应发出，hits = %d, want 1", hits)
		}
	})

	t.Run("GET 拒绝", func(t *testing.T) {
		hits = 0
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		h := newApprovalHTTPTool(rej)
		_, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{URL: srv.URL})
		if err == nil || !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Fatalf("GET 应返回拒绝错误，实际: %v", err)
		}
		if rej.gotSum == "" {
			t.Error("审批器应被调用并记录摘要")
		}
		if hits != 0 {
			t.Errorf("拒绝后请求不应发出，hits = %d, want 0", hits)
		}
	})

	t.Run("POST critical=true", func(t *testing.T) {
		hits = 0
		mock := &mockApprover{}
		h := newApprovalHTTPTool(mock)
		if _, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{URL: srv.URL, Method: "POST", Body: `{"a":1}`}); err != nil {
			t.Fatalf("POST 请求失败: %v", err)
		}
		if !mock.gotCritical {
			t.Error("POST 应为高危写操作（critical=true）")
		}
		if !strings.Contains(mock.gotSum, "POST") {
			t.Errorf("审批摘要应含 POST，实际: %q", mock.gotSum)
		}
		if !strings.Contains(mock.gotSum, "请求体") {
			t.Errorf("带 body 的写请求摘要应含请求体，实际: %q", mock.gotSum)
		}
		if hits != 1 {
			t.Errorf("批准后请求应发出，hits = %d, want 1", hits)
		}
	})

	t.Run("POST 拒绝", func(t *testing.T) {
		hits = 0
		rej := &rejectApprover{err: errors.New("用户拒绝了操作")}
		h := newApprovalHTTPTool(rej)
		_, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{URL: srv.URL, Method: "POST"})
		if err == nil || !strings.Contains(err.Error(), "用户拒绝了操作") {
			t.Fatalf("POST 应返回拒绝错误，实际: %v", err)
		}
		if hits != 0 {
			t.Errorf("拒绝后请求不应发出，hits = %d, want 0", hits)
		}
	})

	t.Run("Approver 缺失 fail-fast", func(t *testing.T) {
		h := &httpRequestTool{ctx: &Context{}, skipURLGuard: true}
		_, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{URL: srv.URL, Method: "POST"})
		if err == nil || !strings.Contains(err.Error(), "未配置审批机制") {
			t.Fatalf("Approver 缺失应 fail-fast 报错，实际: %v", err)
		}
	})

	t.Run("裸工具放行", func(t *testing.T) {
		hits = 0
		h := &httpRequestTool{skipURLGuard: true} // ctx 为 nil，审批直接放行
		out, err := h.invoke(ctx, h.buildClient(false), httpRequestArgs{URL: srv.URL})
		if err != nil {
			t.Fatalf("裸工具 GET 请求失败: %v", err)
		}
		if !strings.Contains(out, "ok") {
			t.Errorf("输出应包含响应体，实际:\n%s", out)
		}
		if hits != 1 {
			t.Errorf("裸工具请求应发出，hits = %d, want 1", hits)
		}
	})
}
