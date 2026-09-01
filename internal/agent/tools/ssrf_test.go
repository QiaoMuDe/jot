package tools

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestGuardedHTTPClientDialGuard 验证共享防护客户端的拨号期 DNS 解析校验
// （DNS rebinding 防护，read_url 经 eino loader 使用同一实现）：
// guardDial=true 时 127.0.0.1 命中回环黑名单被拒（错误文案含"内网"），
// guardDial=false（测试注入缝）时可正常访问 httptest 本机服务器。
func TestGuardedHTTPClientDialGuard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	ctx := context.Background()

	// guardDial=true：拨号前解析 IP 并逐个校验，127.0.0.1 为回环地址 → 拒绝连接
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	resp, err := newGuardedHTTPClient(5*time.Second, true).Do(req)
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
	resp2, err := newGuardedHTTPClient(5*time.Second, false).Do(req2)
	if err != nil {
		t.Fatalf("guardDial=false 访问本机服务器应成功，实际错误: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("状态码 = %d, want %d", resp2.StatusCode, http.StatusOK)
	}
}

// TestGuardedHTTPClientBodyLimit 验证传输层响应体 1MB 限长：服务端写 2MB，
// 客户端读到的字节数不超过上限（read_url 大页面防护路径）。
func TestGuardedHTTPClientBodyLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, 2*httpMaxBodyBytes)) // 2MB > 上限 1MB
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	resp, err := newGuardedHTTPClient(5*time.Second, false).Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}
	if n > httpMaxBodyBytes {
		t.Errorf("响应体读取字节数 = %d, want <= %d", n, httpMaxBodyBytes)
	}
}

// TestIsPrivateHostEncodedIP 验证 inet_aton 数值编码 IP 字面量的归一化判定：
// 十进制/十六进制/八进制/末段吸收写法均应识别为回环或内网地址，普通域名
// 与公网 IP 不受影响。
func TestIsPrivateHostEncodedIP(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"2130706433", true},            // 十进制整数 = 127.0.0.1（回环）
		{"0x7f000001", true},            // 十六进制整数 = 127.0.0.1（回环）
		{"0x7f.0.0.1", true},            // 十六进制点分（回环）
		{"0177.0.0.1", true},            // 八进制点分（回环）
		{"127.1", true},                 // 末段吸收剩余字节 = 127.0.0.1（回环）
		{"3232235521", true},            // 十进制整数 = 192.168.0.1（内网）
		{"0xC0A80001", true},            // 十六进制整数 = 192.168.0.1（内网）
		{"example.com", false},          // 普通域名放行
		{"8.8.8.8", false},              // 公网 IP 放行
		{"2606:4700:4700::1111", false}, // 公网 IPv6 放行
		{"::1", true},                   // IPv6 回环拦截
		{"4294967296", false},           // 超出 32 位，按域名处理放行
	}
	for _, c := range cases {
		if got := isPrivateHost(c.host); got != c.want {
			t.Errorf("isPrivateHost(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}
