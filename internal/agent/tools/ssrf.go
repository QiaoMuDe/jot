package tools

// 本文件实现联网工具共享的 SSRF 防护 HTTP 客户端构造，read_url 与 http_request
// 共用同一套三层防护：
// ① 调用方先用 validateHTTPURL 校验初始 URL（仅放行 http/https 公网地址）；
// ② CheckRedirect 对每个重定向目标逐跳 isPrivateHost 校验（Go 默认跟随重定向，
//    公网 URL 可 302 跳转到 127.0.0.1 / 169.254.169.254 等内网地址被读取）；
// ③ DialContext 在拨号前解析出全部 IP 逐个黑名单校验并直连已校验 IP（DNS
//    rebinding 防护），随后对通过校验的公网 IP 依次尝试拨号（多 A 记录容灾）。
// 另外传输层统一对响应体限长（1MB），防止超大响应撑爆内存。
// 已知权衡：用户配置系统代理时，请求经代理转发，第③层校验的是代理服务器地址
// 而非目标主机（代理视为可信基础设施，内网可达性由代理策略决定）。

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// httpMaxBodyBytes 响应体读取上限（1MB），防止超大响应撑爆内存。
const httpMaxBodyBytes = 1 << 20 // 1MB

// newGuardedHTTPClient 构造带 SSRF 防护的 HTTP 客户端（三层防护见文件头注释）。
// Transport 以 DefaultTransport 为底座 Clone（保留代理环境变量、HTTP/2、TLS
// 握手超时、连接复用等默认值），guardDial=false 时跳过拨号期 DNS 校验，
// 仅供测试访问 httptest 本机服务器；生产路径一律传 true。
func newGuardedHTTPClient(timeout time.Duration, guardDial bool) *http.Client {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("重定向次数过多（超过 10 次）")
			}
			if isPrivateHost(req.URL.Host) {
				return fmt.Errorf("拒绝跟随重定向到内网/本机地址 %s", req.URL.Host)
			}
			return nil
		},
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if guardDial {
		transport.DialContext = guardedDialContext
	}
	// 响应体限长：传输层统一截断（对透明解压后的字节流生效），调用方无需自防超大响应
	client.Transport = &limitBodyTransport{base: transport}
	return client
}

// guardedDialer 防护拨号使用的底层拨号器（无状态，可并发复用）。
var guardedDialer net.Dialer

// guardedDialContext 防护拨号：第一步解析出全部 IP 并逐个黑名单校验
// （isPrivateHost 对 IP 字面量可直接判定），任一命中即整体拒绝（防 DNS
// rebinding）；第二步对通过校验的公网 IP 依次尝试拨号，首个成功即返回；
// 全程直连已校验的 IP 而非原 addr，杜绝校验与连接之间 DNS 变化被利用。
func guardedDialContext(ctx context.Context, _ string, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("解析拨号地址失败: %w", err)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("域名解析失败: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("域名 %s 未解析到任何 IP", host)
	}
	// 黑名单校验：任一解析 IP 命中内网即整体拒绝（防 DNS rebinding 语义不变）
	allowed := make([]net.IPAddr, 0, len(ips))
	for _, ip := range ips {
		if isPrivateHost(ip.String()) {
			return nil, fmt.Errorf("域名 %s 解析到内网/本机地址 %s，拒绝连接（DNS rebinding 防护）", host, ip.String())
		}
		allowed = append(allowed, ip)
	}
	// 逐个尝试拨号：首个 IP 不可达时自动换下一个（多入口域名容灾）
	var lastErr error
	for _, ip := range allowed {
		conn, dialErr := guardedDialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	return nil, lastErr
}

// limitBodyTransport 包装 RoundTripper：对响应体（透明解压后的字节流）施加
// httpMaxBodyBytes 读取上限，防止超大响应撑爆内存。
type limitBodyTransport struct {
	base http.RoundTripper
}

// RoundTrip 执行底层请求并将响应体包装为限长 Reader（保留 Close 语义）。
func (t *limitBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		resp.Body = &limitReadCloser{r: io.LimitReader(resp.Body, httpMaxBodyBytes), c: resp.Body}
	}
	return resp, nil
}

// limitReadCloser 组合限长 Reader 与原始 Closer，保持响应体可关闭。
type limitReadCloser struct {
	r io.Reader
	c io.Closer
}

func (l *limitReadCloser) Read(p []byte) (int, error) { return l.r.Read(p) }

// Close 关闭底层连接（读取被截断时也须释放连接资源）。
func (l *limitReadCloser) Close() error { return l.c.Close() }
