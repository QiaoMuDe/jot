// mcp-text：文本处理 MCP 服务器（Jot Agent 外部 MCP 测试用）。
// 基于 mark3labs/mcp-go 的 stdio 传输，提供 to_uppercase / to_lowercase / word_count 三个工具。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	srv := server.NewMCPServer("mcp-text", "1.0.0")

	// to_uppercase：文本转大写
	srv.AddTool(mcp.NewTool("to_uppercase",
		mcp.WithDescription("把一段文本转换为大写"),
		mcp.WithString("text", mcp.Required(), mcp.Description("待转换的文本")),
	), func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, err := parseText(req)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(strings.ToUpper(text)), nil
	})

	// to_lowercase：文本转小写
	srv.AddTool(mcp.NewTool("to_lowercase",
		mcp.WithDescription("把一段文本转换为小写"),
		mcp.WithString("text", mcp.Required(), mcp.Description("待转换的文本")),
	), func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, err := parseText(req)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(strings.ToLower(text)), nil
	})

	// word_count：统计单词数
	srv.AddTool(mcp.NewTool("word_count",
		mcp.WithDescription("统计一段文本中由空白分隔的单词数量"),
		mcp.WithString("text", mcp.Required(), mcp.Description("待统计的文本")),
	), func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, err := parseText(req)
		if err != nil {
			return nil, err
		}
		count := len(strings.Fields(text))
		return mcp.NewToolResultText(fmt.Sprintf("%d", count)), nil
	})

	// stdio 传输：从 stdin 读 JSON-RPC，结果写 stdout
	stdio := server.NewStdioServer(srv)
	if err := stdio.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mcp-text 服务器异常退出:", err)
		os.Exit(1)
	}
}

// parseText 解析工具参数中的 text 字符串字段。
// req.Params.Arguments 为 any 类型：兼容 map（object）、string（JSON 文本）、
// json.RawMessage 等由不同客户端传入的形态。
func parseText(req mcp.CallToolRequest) (string, error) {
	var args map[string]any
	switch v := req.Params.Arguments.(type) {
	case map[string]any:
		args = v
	case string: // 客户端传入的是 JSON 文本字符串
		if err := json.Unmarshal([]byte(v), &args); err != nil {
			return "", fmt.Errorf("解析参数失败: %w", err)
		}
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("解析参数失败: %w", err)
		}
		if err := json.Unmarshal(raw, &args); err != nil {
			return "", fmt.Errorf("解析参数失败: %w", err)
		}
	}
	text, ok := args["text"].(string)
	if !ok {
		return "", fmt.Errorf("缺少字符串参数 %q", "text")
	}
	return text, nil
}
