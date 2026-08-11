package mcpserver_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"jot/internal/agent/tools"
	"jot/internal/mcpserver"
)

// TestOpenSessionOverSSE 验证 OpenSession 全链路：
// 内存 SSE MCP 服务器 → Connect 握手 → GetTools 发现工具 → 改名包装 → 调用执行 → 关闭会话。
func TestOpenSessionOverSSE(t *testing.T) {
	// 超时保护：覆盖握手 / 工具发现 / 工具调用等所有网络操作，防止测试挂死
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. 起内存 MCP 服务器并注册两个工具：add（两数求和）与 get_current_time（只读）
	mcpSrv := server.NewMCPServer("test-math-server", "1.0.0")
	mcpSrv.AddTool(mcp.NewTool("add",
		mcp.WithDescription("两数相加"),
		mcp.WithNumber("a", mcp.Required()),
		mcp.WithNumber("b", mcp.Required()),
	), func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// req.Params.Arguments 为 any，先序列化再反序列化为具名结构
		raw, err := json.Marshal(req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		var args struct {
			A float64 `json:"a"`
			B float64 `json:"b"`
		}
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(strconv.FormatFloat(args.A+args.B, 'f', -1, 64)), nil
	})
	mcpSrv.AddTool(mcp.NewTool("get_current_time",
		mcp.WithDescription("只读：返回当前时间"),
	), func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("2026-08-11 12:00:00"), nil
	})

	// 2. 内存 SSE 传输：NewTestServer 基于 httptest 起随机端口的内存服务器，无端口冲突
	ts := server.NewTestServer(mcpSrv)
	defer ts.Close()

	// 3. 构造 MCP 服务器配置（SSE 端点默认 /sse）
	ms := mcpserver.Server{
		Name:      "test-math",
		Transport: "sse",
		URL:       ts.URL + "/sse",
		Enabled:   true,
	}

	// 4. OpenSession：Connect 握手 → GetTools 发现 → 改名包装
	sess, err := mcpserver.OpenSession(ctx, ms)
	if err != nil {
		t.Fatalf("OpenSession 失败: %v", err)
	}
	if sess.ServerName != "test-math" {
		t.Errorf("Session.ServerName = %q, want test-math", sess.ServerName)
	}

	// 5. 断言工具数量与改名前缀 mcp_{服务器名}_{工具名}
	if len(sess.Tools) != 2 {
		t.Fatalf("发现工具数量 = %d, want 2", len(sess.Tools))
	}
	for _, bt := range sess.Tools {
		info, err := bt.Info(ctx)
		if err != nil || info == nil {
			t.Fatalf("工具 Info 失败: %v", err)
		}
		if info.Name == "" || !strings.HasPrefix(info.Name, "mcp_test-math_") {
			t.Errorf("工具名 %q 应带 mcp_test-math_ 前缀", info.Name)
		}
	}

	// 6. 断言工具实现 ActionTextProvider，ActionText 文案含服务器名与工具名
	addTool := findTool(t, ctx, sess.Tools, "_add")
	textProvider, ok := addTool.(tools.ActionTextProvider)
	if !ok {
		t.Fatal("add 工具未实现 ActionTextProvider")
	}
	if text := textProvider.ActionText(`{}`); !strings.Contains(text, "test-math") || !strings.Contains(text, "add") {
		t.Errorf("ActionText = %q, 应包含服务器名与工具名", text)
	}

	// 7. 执行 add(2, 3)，断言返回结果 text 为 5
	out, err := addTool.InvokableRun(ctx, `{"a":2,"b":3}`)
	if err != nil {
		t.Fatalf("InvokableRun(add) 失败: %v", err)
	}
	if text := toolResultText(t, out); text != "5" {
		t.Errorf("add(2,3) 结果 text = %q, want \"5\"（原始输出: %s）", text, out)
	}

	// 8. 执行只读工具 get_current_time
	timeTool := findTool(t, ctx, sess.Tools, "_get_current_time")
	timeOut, err := timeTool.InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("InvokableRun(get_current_time) 失败: %v", err)
	}
	if !strings.Contains(timeOut, "2026-08-11") {
		t.Errorf("get_current_time 结果未包含期望时间, 原始输出: %s", timeOut)
	}

	// 9. 关闭会话无错误
	if err := sess.Close(); err != nil {
		t.Errorf("Session.Close 失败: %v", err)
	}
}

// findTool 按原名后缀（mcp_{服务器名}_{原名}）在会话工具列表中查找可执行工具。
func findTool(t *testing.T, ctx context.Context, list []tool.BaseTool, suffix string) tool.InvokableTool {
	t.Helper()
	for _, bt := range list {
		info, err := bt.Info(ctx)
		if err != nil || info == nil {
			continue
		}
		if strings.HasSuffix(info.Name, suffix) {
			it, ok := bt.(tool.InvokableTool)
			if !ok {
				t.Fatalf("工具 %s 未实现 InvokableTool", info.Name)
			}
			return it
		}
	}
	t.Fatalf("未找到工具 %q", suffix)
	return nil
}

// toolResultText 解析 eino mcp 组件 InvokableRun 返回的 JSON（CallToolResult 序列化），
// 提取 content[0].text 文本内容。
func toolResultText(t *testing.T, out string) string {
	t.Helper()
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("解析工具结果失败: %v，原始输出: %s", err, out)
	}
	if len(res.Content) == 0 {
		t.Fatalf("工具结果无 content，原始输出: %s", out)
	}
	return res.Content[0].Text
}
