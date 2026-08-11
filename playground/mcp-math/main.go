// mcp-math：数学计算 MCP 服务器（Jot Agent 外部 MCP 测试用）。
// 基于 mark3labs/mcp-go 的 stdio 传输，提供 add / multiply / sqrt 三个工具。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	srv := server.NewMCPServer("mcp-math", "1.0.0")

	// add：两数相加
	srv.AddTool(mcp.NewTool("add",
		mcp.WithDescription("计算两个数字之和"),
		mcp.WithNumber("a", mcp.Required(), mcp.Description("第一个加数")),
		mcp.WithNumber("b", mcp.Required(), mcp.Description("第二个加数")),
	), func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		a, b, err := parsePair(req)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(fmt.Sprintf("%v", a+b)), nil
	})

	// multiply：两数相乘
	srv.AddTool(mcp.NewTool("multiply",
		mcp.WithDescription("计算两个数字之积"),
		mcp.WithNumber("a", mcp.Required(), mcp.Description("第一个乘数")),
		mcp.WithNumber("b", mcp.Required(), mcp.Description("第二个乘数")),
	), func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		a, b, err := parsePair(req)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(fmt.Sprintf("%v", a*b)), nil
	})

	// sqrt：平方根
	srv.AddTool(mcp.NewTool("sqrt",
		mcp.WithDescription("计算一个非负数字的平方根"),
		mcp.WithNumber("a", mcp.Required(), mcp.Description("被开方数（须非负）")),
	), func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		a, err := parseSingle(req, "a")
		if err != nil {
			return nil, err
		}
		if a < 0 {
			return nil, fmt.Errorf("sqrt 参数 a 必须为非负数，收到 %v", a)
		}
		return mcp.NewToolResultText(fmt.Sprintf("%v", math.Sqrt(a))), nil
	})

	// stdio 传输：从 stdin 读 JSON-RPC，结果写 stdout
	stdio := server.NewStdioServer(srv)
	if err := stdio.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mcp-math 服务器异常退出:", err)
		os.Exit(1)
	}
}

// parsePair 解析工具参数中的 a / b 两个数字字段。
func parsePair(req mcp.CallToolRequest) (float64, float64, error) {
	a, err := parseSingle(req, "a")
	if err != nil {
		return 0, 0, err
	}
	b, err := parseSingle(req, "b")
	if err != nil {
		return 0, 0, err
	}
	return a, b, nil
}

// parseSingle 从工具参数中解析指定键的 float64 字段。
// req.Params.Arguments 为 any 类型：兼容 map（object）、string（JSON 文本）、
// json.RawMessage 等由不同客户端传入的形态。
func parseSingle(req mcp.CallToolRequest, key string) (float64, error) {
	args, err := argsMap(req)
	if err != nil {
		return 0, err
	}
	v, ok := args[key].(float64)
	if !ok {
		return 0, fmt.Errorf("缺少数字参数 %q", key)
	}
	return v, nil
}

// argsMap 把工具参数统一规整为 map[string]any。
func argsMap(req mcp.CallToolRequest) (map[string]any, error) {
	switch v := req.Params.Arguments.(type) {
	case map[string]any:
		return v, nil
	case string: // 客户端传入的是 JSON 文本字符串
		var m map[string]any
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, fmt.Errorf("解析参数失败: %w", err)
		}
		return m, nil
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("解析参数失败: %w", err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("解析参数失败: %w", err)
		}
		return m, nil
	}
}
