package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// ConnectTimeout 单台 MCP 服务器连接 + 握手（Start + Initialize）的超时上限。
// 调用方 ctx 通常无超时（仅可取消），若无此限制，远程服务器不可达（DNS/TCP/TLS 挂起）
// 时串行装配会无限阻塞后续服务器与整轮对话；超时走现有「连接失败跳过」分支。
const ConnectTimeout = 10 * time.Second

// Connect 按服务器传输类型构建 MCP 客户端并完成握手（Start + Initialize）。
// stdio 客户端构造时已自动启动 transport，无需手动 Start；sse / http 需手动 cli.Start(ctx)。
// 连接 / 握手失败统一包装为含服务器名的错误（Connect 内部负责清理已创建但未就绪的客户端）；
// 连接成功后由调用方（Session）负责最终 cli.Close()。
// 每台服务器连接包一层 ConnectTimeout 超时，超时错误文案明确提示。
func Connect(ctx context.Context, s Server) (client.MCPClient, error) {
	// 独立超时：限制单台服务器连接 + 握手总耗时，避免远程挂起阻塞装配
	connCtx, cancel := context.WithTimeout(ctx, ConnectTimeout)
	defer cancel()

	var cli client.MCPClient
	var err error
	switch s.Transport {
	case "stdio":
		// 注意：NewStdioMCPClient 失败时返回 (*Client)(nil)，直接赋给接口 cli 会让
		// cli != nil 误判为真（typed-nil 陷阱），后续 Close 将 panic。因此先判错再赋接口。
		c, cerr := client.NewStdioMCPClient(s.Command, envSlice(s.Env), s.Args...)
		if cerr != nil {
			return nil, wrapConnectError(s.Name, cerr)
		}
		cli = c
	case "sse":
		var c *client.Client
		var opts []transport.ClientOption
		if len(s.Headers) > 0 {
			opts = append(opts, client.WithHeaders(s.Headers))
		}
		if c, err = client.NewSSEMCPClient(s.URL, opts...); err == nil {
			err = c.Start(connCtx)
			cli = c
		}
	case "http":
		var c *client.Client
		var opts []transport.StreamableHTTPCOption
		if len(s.Headers) > 0 {
			opts = append(opts, transport.WithHTTPHeaders(s.Headers))
		}
		if c, err = client.NewStreamableHttpClient(s.URL, opts...); err == nil {
			err = c.Start(connCtx)
			cli = c
		}
	default:
		return nil, fmt.Errorf("MCP 服务器 %s 连接失败: 不支持的传输类型 %q", s.Name, s.Transport)
	}
	if err != nil {
		safeClose(cli)
		return nil, wrapConnectError(s.Name, err)
	}

	// 标准握手：Initialize（ClientInfo 标识 jot）
	if _, err := cli.Initialize(connCtx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "jot", Version: "0.0.1"},
		},
	}); err != nil {
		safeClose(cli)
		return nil, wrapConnectError(s.Name, err)
	}
	return cli, nil
}

// safeClose 关闭 MCP 客户端。接口变量可能装包 typed-nil 指针（如 (*Client)(nil)），
// 此时 cli != nil 但直接调用 Close 会 panic；先校验接口底层指针非 nil 再关闭，防御兜底。
func safeClose(cli client.MCPClient) {
	if cli == nil {
		return
	}
	v := reflect.ValueOf(cli)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return
	}
	_ = cli.Close()
}

// wrapConnectError 包装连接/握手错误：超时（含调用方取消触发的 deadline）明确提示，
// 其余保持通用文案；统一附带服务器名便于日志定位。
func wrapConnectError(name string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("MCP 服务器 %s 连接超时（%s 内未完成连接握手）: %w", name, ConnectTimeout, err)
	}
	return fmt.Errorf("MCP 服务器 %s 连接失败: %w", name, err)
}

// envSlice 将配置的 Env map 转为 mcp-go stdio 客户端要求的 "KEY=VALUE" 字符串切片。
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
