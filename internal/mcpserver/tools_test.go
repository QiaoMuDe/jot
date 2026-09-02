package mcpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"jot/internal/agent/tools"
	"jot/internal/mcpserver"
)

// TestOpenSessionOverSSE 验证 OpenSession 全链路：
// 内存 SSE MCP 服务器 → Connect 握手 → ListTools 发现工具 → 改名包装 → 调用执行 → 关闭会话。
func TestOpenSessionOverSSE(t *testing.T) {
	// 超时保护：覆盖握手 / 工具发现 / 工具调用等所有网络操作，防止测试挂死
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. 起内存 MCP 服务器并注册两个工具：add（两数求和）与 ping（只读）
	srv := mcp.NewServer(&mcp.Implementation{Name: "test-math-server", Version: "1.0.0"}, nil)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "add",
		Description: "两数相加",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "number"},
				"b": map[string]any{"type": "number"},
			},
			"required": []string{"a", "b"},
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: strconv.FormatFloat(a+b, 'f', -1, 64)}},
		}, nil, nil
	})
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "ping",
		Description: "只读：返回 pong",
		InputSchema: map[string]any{"type": "object"},
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ map[string]any) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "pong"}},
		}, nil, nil
	})

	// 2. 内存 SSE 传输：httptest 起随机端口的内存服务器，无端口冲突
	handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 3. 构造 MCP 服务器配置（SSE 端点默认 /sse）
	ms := mcpserver.Server{
		Name:      "test-math",
		Transport: "sse",
		URL:       ts.URL + "/sse",
		Enabled:   true,
	}

	// 4. OpenSession：Connect 握手 → ListTools 发现 → 改名包装
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

	// 8. 执行只读工具 ping
	pingTool := findTool(t, ctx, sess.Tools, "_ping")
	pingOut, err := pingTool.InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("InvokableRun(ping) 失败: %v", err)
	}
	if !strings.Contains(pingOut, "pong") {
		t.Errorf("ping 结果未包含期望内容, 原始输出: %s", pingOut)
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

// toolResultText 解析 InvokableRun 返回的 JSON（CallToolResult 序列化），
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
