package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Connect 按服务器传输类型构建 MCP 客户端并完成握手（Start + Initialize）。
// stdio 客户端构造时已自动启动 transport，无需手动 Start；sse / http 需手动 cli.Start(ctx)。
// 连接 / 握手失败统一包装为含服务器名的错误（Connect 内部负责清理已创建但未就绪的客户端）；
// 连接成功后由调用方（Session）负责最终 cli.Close()。
func Connect(ctx context.Context, s Server) (client.MCPClient, error) {
	var cli client.MCPClient
	var err error
	switch s.Transport {
	case "stdio":
		cli, err = client.NewStdioMCPClient(s.Command, envSlice(s.Env), s.Args...)
	case "sse":
		var c *client.Client
		if c, err = client.NewSSEMCPClient(s.URL); err == nil {
			err = c.Start(ctx)
			cli = c
		}
	case "http":
		var c *client.Client
		if c, err = client.NewStreamableHttpClient(s.URL); err == nil {
			err = c.Start(ctx)
			cli = c
		}
	default:
		return nil, fmt.Errorf("MCP 服务器 %s 连接失败: 不支持的传输类型 %q", s.Name, s.Transport)
	}
	if err != nil {
		if cli != nil {
			_ = cli.Close()
		}
		return nil, fmt.Errorf("MCP 服务器 %s 连接失败: %w", s.Name, err)
	}

	// 标准握手：Initialize（ClientInfo 标识 jot）
	if _, err := cli.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "jot", Version: "0.0.1"},
		},
	}); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("MCP 服务器 %s 连接失败: %w", s.Name, err)
	}
	return cli, nil
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
