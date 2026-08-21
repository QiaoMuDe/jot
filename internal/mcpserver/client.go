package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConnectTimeout 单台 MCP 服务器连接 + 握手（Connect + Initialize）的超时上限。
// 调用方 ctx 通常无超时（仅可取消），若无此限制，远程服务器不可达（DNS/TCP/TLS 挂起）
// 时串行装配会无限阻塞后续服务器与整轮对话；超时走现有「连接失败跳过」分支。
const ConnectTimeout = 10 * time.Second

// Connect 按服务器传输类型构建 MCP 客户端并完成握手（Connect + Initialize）。
// go-sdk 的 Client.Connect 内部完成传输连接、协议版本自动协商（含降级到 2024-11-05）
// 与 initialize 握手，无需手动 Start / Initialize。
// 连接 / 握手失败统一包装为含服务器名的错误（Connect 内部负责清理已创建但未就绪的会话）；
// 连接成功后返回会话与 cancel 函数——cancel 必须由调用方在会话不再需要时调用
// （Session.Close 负责），它终止底层传输（SSE/HTTP 的长连接依赖传入 ctx 的生命周期，
// 若在 Connect 返回后立即取消会断开会话）。
// Headers 鉴权：go-sdk transport 无 Headers 字段，通过自定义 http.Client（RoundTripper
// 注入请求头）实现，SSE 的 GET 与 POST 请求均生效。
func Connect(ctx context.Context, s Server) (*mcp.ClientSession, func(), error) {
	// 会话生命周期 ctx：继承调用方可取消语义，成功握手后不取消（由调用方 Close 时取消）。
	// 独立计时器实现握手超时：超时即取消 ctx（连接与未完成握手一并终止）。
	connCtx, cancel := context.WithCancel(ctx)
	var timedOut atomic.Bool
	timer := time.AfterFunc(ConnectTimeout, func() {
		timedOut.Store(true)
		cancel()
	})

	cli := mcp.NewClient(&mcp.Implementation{Name: "jot", Version: "0.0.1"}, nil)

	var tr mcp.Transport
	switch s.Transport {
	case "stdio":
		cmd := exec.Command(s.Command, s.Args...)
		if len(s.Env) > 0 {
			cmd.Env = append(os.Environ(), envSlice(s.Env)...)
		}
		tr = &mcp.CommandTransport{Command: cmd}
	case "sse":
		tr = &mcp.SSEClientTransport{
			Endpoint:   s.URL,
			HTTPClient: httpClientWithHeaders(s.Headers),
		}
	case "http":
		tr = &mcp.StreamableClientTransport{
			Endpoint:   s.URL,
			HTTPClient: httpClientWithHeaders(s.Headers),
		}
	default:
		timer.Stop()
		cancel()
		return nil, nil, fmt.Errorf("MCP 服务器 %s 连接失败: 不支持的传输类型 %q", s.Name, s.Transport)
	}

	cs, err := cli.Connect(connCtx, tr, nil)
	timer.Stop()
	if err != nil {
		cancel()
		if timedOut.Load() {
			return nil, nil, fmt.Errorf("MCP 服务器 %s 连接超时（%s 内未完成连接握手）: %w", s.Name, ConnectTimeout, err)
		}
		return nil, nil, wrapConnectError(s.Name, err)
	}
	return cs, cancel, nil
}

// httpClientWithHeaders 返回注入自定义请求头的 http.Client。
// Headers 为空时返回 nil，transport 使用默认 http.DefaultClient。
func httpClientWithHeaders(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return nil
	}
	return &http.Client{
		Transport: &headerRoundTripper{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}
}

// headerRoundTripper 包装 http.RoundTripper，为每个请求注入配置的 Headers。
// 克隆请求避免污染共享请求对象（连接复用场景安全）。
type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	for k, v := range h.headers {
		cloned.Header.Set(k, v)
	}
	return h.base.RoundTrip(cloned)
}

// wrapConnectError 包装连接/握手错误：超时（含调用方取消触发的 deadline）明确提示，
// 其余保持通用文案；统一附带服务器名便于日志定位。
func wrapConnectError(name string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("MCP 服务器 %s 连接超时（%s 内未完成连接握手）: %w", name, ConnectTimeout, err)
	}
	return fmt.Errorf("MCP 服务器 %s 连接失败: %w", name, err)
}

// envSlice 将配置的 Env map 转为 exec.Cmd.Env 追加所需的 "KEY=VALUE" 字符串切片。
func envSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
