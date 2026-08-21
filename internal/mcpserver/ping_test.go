package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"

	"jot/internal/mcpserver"
)

// TestSSEServerPingHandling 模拟知乎 MCP 服务器行为验证 go-sdk 客户端：
// 手写原始 SSE 服务器（不依赖 go-sdk 服务端），连接建立后由服务端主动向 SSE 流
// 发送 JSON-RPC ping 请求（mcp-go 无法处理导致超时的根因场景）。
// 验证：客户端能自动响应 ping（回 pong），且 ping 不阻塞后续 ListTools / CallTool。
func TestSSEServerPingHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var (
		pongsReceived atomic.Int64 // 服务端收到的 pong 响应数
		sseMu         sync.Mutex
		sseWriter     http.ResponseWriter
		sseFlusher    http.Flusher
	)

	// 向 SSE 流写入一个事件（JSON-RPC 消息），服务端→客户端方向
	writeSSE := func(data string) bool {
		sseMu.Lock()
		defer sseMu.Unlock()
		if sseWriter == nil {
			return false
		}
		if _, err := fmt.Fprintf(sseWriter, "data: %s\n\n", data); err != nil {
			return false
		}
		sseFlusher.Flush()
		return true
	}

	// 手写 SSE 服务器：GET /sse 建立流，POST /sse/message 接收客户端消息
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		sseMu.Lock()
		sseWriter = w
		sseFlusher = fl
		sseMu.Unlock()

		// 首个事件：endpoint，告知客户端 POST 地址（flush 与 writeSSE 共用 sseMu 保护）
		sseMu.Lock()
		_, _ = fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", r.URL.Path+"/message")
		fl.Flush()
		sseMu.Unlock()

		// 模拟知乎：连接后服务端持续主动发 ping（JSON-RPC 请求）
		go func() {
			for i := 1; i <= 3; i++ {
				time.Sleep(100 * time.Millisecond)
				writeSSE(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"ping"}`, i))
			}
		}()

		<-r.Context().Done() // 挂起直到客户端断开
	})

	mux.HandleFunc("/sse/message", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		var msg map[string]any
		if err := json.Unmarshal(body, &msg); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		// 有 id 且有 result/error 的是客户端对服务端请求（如 ping）的响应
		if _, hasResult := msg["result"]; hasResult {
			if id, ok := msg["id"]; ok && id != nil {
				pongsReceived.Add(1)
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		// notification（无 id）：不响应
		if id, ok := msg["id"]; !ok || id == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		// 其余是客户端请求：处理并把响应写回 SSE 流
		method, _ := msg["method"].(string)
		var resp map[string]any
		switch method {
		case "initialize":
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      msg["id"],
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "zhihu-sim", "version": "1.0.0"},
				},
			}
		case "tools/list":
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      msg["id"],
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "zhihu_search",
							"description": "搜索知乎内容",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"query": map[string]any{"type": "string"},
								},
								"required": []any{"query"},
							},
						},
					},
				},
			}
		case "tools/call":
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      msg["id"],
				"result": map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": "知乎搜索结果内容"},
					},
				},
			}
		default:
			// server/discover 等未实现方法：返回 method not found，客户端会 fallback 到 legacy initialize
			resp = map[string]any{
				"jsonrpc": "2.0",
				"id":      msg["id"],
				"error": map[string]any{
					"code":    -32601,
					"message": "method not found",
				},
			}
		}
		data, _ := json.Marshal(resp)
		writeSSE(string(data))
		w.WriteHeader(http.StatusAccepted)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ms := mcpserver.Server{
		Name:      "zhihu-sim",
		Transport: "sse",
		URL:       ts.URL + "/sse",
		Enabled:   true,
	}

	sess, err := mcpserver.OpenSession(ctx, ms)
	if err != nil {
		t.Fatalf("OpenSession 失败: %v", err)
	}
	defer func() {
		if err := sess.Close(); err != nil {
			t.Errorf("Session.Close 失败: %v", err)
		}
	}()

	if len(sess.Tools) != 1 {
		t.Fatalf("发现工具数量 = %d, want 1", len(sess.Tools))
	}
	info, err := sess.Tools[0].Info(ctx)
	if err != nil || info == nil {
		t.Fatalf("工具 Info 失败: %v", err)
	}
	if info.Name != "mcp_zhihu-sim_zhihu_search" {
		t.Errorf("工具名 = %q, want mcp_zhihu-sim_zhihu_search", info.Name)
	}

	// 等待服务端 3 个 ping 完成（客户端应已自动响应全部）
	deadline := time.Now().Add(5 * time.Second)
	for pongsReceived.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := pongsReceived.Load(); got < 3 {
		t.Errorf("服务端收到的 pong 响应数 = %d, want >= 3（客户端未自动响应服务端 ping）", got)
	}

	// ping 风暴后调用工具，验证连接仍活跃
	inv, ok := sess.Tools[0].(tool.InvokableTool)
	if !ok {
		t.Fatal("工具未实现 InvokableTool")
	}
	out, err := inv.InvokableRun(ctx, `{"query":"测试"}`)
	if err != nil {
		t.Fatalf("ping 后 InvokableRun 失败: %v", err)
	}
	if !strings.Contains(out, "知乎搜索结果内容") {
		t.Errorf("工具结果未含期望文本, 原始输出: %s", out)
	}
}
