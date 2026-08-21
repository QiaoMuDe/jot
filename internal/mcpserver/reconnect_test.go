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
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"

	"jot/internal/mcpserver"
)

// startSSEServerWithTool 起一个手写 SSE MCP 服务器，注册一个 echo 工具。
// disconnectAfterCalls>0 时：第 N 次工具调用响应后服务端主动断开 SSE 连接
// （模拟服务器重启/连接中断），客户端下一次调用应自动重连（同 URL）。
// 返回服务器实例、关闭函数与工具调用计数读取函数。
func startSSEServerWithTool(disconnectAfterCalls int) (*httptest.Server, func() int, func()) {
	var (
		mu         sync.Mutex
		calls      int
		disconnect chan struct{} // 服务端断开信号（nil 表示不主动断开）
	)
	newDisconnect := func() chan struct{} {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	disconnect = newDisconnect() // 初始无断开

	cancelStream := func() {
		mu.Lock()
		if disconnect != nil {
			select {
			case <-disconnect:
			default:
				close(disconnect)
			}
		}
		mu.Unlock()
	}

	var sseMu sync.Mutex
	var sseWriter http.ResponseWriter
	var sseFlusher http.Flusher

	writeSSE := func(data string) {
		sseMu.Lock()
		defer sseMu.Unlock()
		if sseWriter == nil {
			return
		}
		_, _ = fmt.Fprintf(sseWriter, "data: %s\n\n", data)
		sseFlusher.Flush()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		sseMu.Lock()
		sseWriter = w
		sseFlusher = fl
		sseMu.Unlock()

		// 复用/重建当前断开信号：每次 GET 建立新流时重置
		mu.Lock()
		if disconnect != nil {
			select {
			case <-disconnect:
				disconnect = make(chan struct{})
			default:
			}
		} else {
			disconnect = make(chan struct{})
		}
		myDisconnect := disconnect
		mu.Unlock()

		// endpoint 事件与 flush 同样在 sseMu 保护下，避免与 writeSSE 并发 flush 同一 writer
		sseMu.Lock()
		_, _ = fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", r.URL.Path+"/message")
		fl.Flush()
		sseMu.Unlock()
		select {
		case <-myDisconnect: // 服务端主动断开（模拟重启）
		case <-r.Context().Done(): // 客户端断开 / 服务器 Close
		}
	})
	mux.HandleFunc("/sse/message", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var msg map[string]any
		if err := json.Unmarshal(body, &msg); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if _, hasResult := msg["result"]; hasResult {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if id, ok := msg["id"]; !ok || id == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		var resp map[string]any
		switch msg["method"] {
		case "initialize":
			resp = map[string]any{
				"jsonrpc": "2.0", "id": msg["id"],
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "reconnect-sim", "version": "1.0.0"},
				},
			}
		case "tools/list":
			resp = map[string]any{
				"jsonrpc": "2.0", "id": msg["id"],
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "echo", "description": "回显", "inputSchema": map[string]any{"type": "object"}},
					},
				},
			}
		case "tools/call":
			mu.Lock()
			calls++
			thisCall := calls
			mu.Unlock()
			resp = map[string]any{
				"jsonrpc": "2.0", "id": msg["id"],
				"result": map[string]any{
					"content": []map[string]any{{"type": "text", "text": "echo-ok"}},
				},
			}
			// 达到断开阈值：响应写回后主动断开（模拟服务器重启）
			if disconnectAfterCalls > 0 && thisCall == disconnectAfterCalls {
				// 先写响应再断开，保证本次调用成功返回
				data, _ := json.Marshal(resp)
				writeSSE(string(data))
				cancelStream()
				w.WriteHeader(http.StatusAccepted)
				return
			}
		default:
			resp = map[string]any{
				"jsonrpc": "2.0", "id": msg["id"],
				"error": map[string]any{"code": -32601, "message": "method not found"},
			}
		}
		data, _ := json.Marshal(resp)
		writeSSE(string(data))
		w.WriteHeader(http.StatusAccepted)
	})

	ts := httptest.NewServer(mux)
	getCalls := func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}
	return ts, getCalls, cancelStream
}

// mustInvokable 断言会话首个工具为 InvokableTool 并返回。
func mustInvokable(t *testing.T, sess *mcpserver.Session) tool.InvokableTool {
	t.Helper()
	if len(sess.Tools) == 0 {
		t.Fatal("会话无工具")
	}
	inv, ok := sess.Tools[0].(tool.InvokableTool)
	if !ok {
		t.Fatal("工具未实现 InvokableTool")
	}
	return inv
}

// TestSessionCallToolAutoReconnect 验证断线自动重连：
// 第 1 次工具调用后服务端主动断开 SSE 连接，第 2 次调用应自动重连（同 URL）并重试成功。
func TestSessionCallToolAutoReconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ts, getCalls, _ := startSSEServerWithTool(1) // 第 1 次调用后断开
	t.Cleanup(func() {
		ts.CloseClientConnections() // 强制断开挂起的 SSE GET，避免 Close 阻塞
		ts.Close()
	})

	ms := mcpserver.Server{
		Name:      "reconnect-sim",
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

	// 第 1 次调用：成功（服务端随后断开）
	out, err := mustInvokable(t, sess).InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("首次 InvokableRun 失败: %v", err)
	}
	if !strings.Contains(out, "echo-ok") {
		t.Fatalf("首次调用结果异常: %s", out)
	}

	// 第 2 次调用：连接已断，应自动重连（同 URL）并重试成功。
	// 先等待客户端感知断开（jsonrpc2 后台读 goroutine 检测 EOF 后关闭连接）
	time.Sleep(500 * time.Millisecond)
	out2, err := mustInvokable(t, sess).InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("断线后 InvokableRun 失败（应自动重连）: %v", err)
	}
	if !strings.Contains(out2, "echo-ok") {
		t.Fatalf("重连后调用结果异常: %s", out2)
	}
	// 服务端应收到 2 次调用（第 1 次 + 重连后第 2 次）
	if c := getCalls(); c != 2 {
		t.Errorf("服务端应收到 2 次调用, got %d", c)
	}
}

// TestSessionCallToolNoReconnectWhenClosed 验证 Close 后调用不重连、直接返回会话已关闭错误。
func TestSessionCallToolNoReconnectWhenClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ts, _, _ := startSSEServerWithTool(0)
	t.Cleanup(func() {
		ts.CloseClientConnections()
		ts.Close()
	})
	ms := mcpserver.Server{
		Name:      "reconnect-sim",
		Transport: "sse",
		URL:       ts.URL + "/sse",
		Enabled:   true,
	}

	sess, err := mcpserver.OpenSession(ctx, ms)
	if err != nil {
		t.Fatalf("OpenSession 失败: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("Close 失败: %v", err)
	}
	if _, err := mustInvokable(t, sess).InvokableRun(ctx, `{}`); err == nil {
		t.Fatal("Close 后调用应返回错误")
	} else if !strings.Contains(err.Error(), "已关闭") {
		t.Errorf("错误应含「已关闭」: %v", err)
	}
}

// TestSessionCloseIdempotent 验证 Close 幂等。
func TestSessionCloseIdempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ts, _, _ := startSSEServerWithTool(0)
	t.Cleanup(func() {
		ts.CloseClientConnections()
		ts.Close()
	})
	ms := mcpserver.Server{
		Name:      "reconnect-sim",
		Transport: "sse",
		URL:       ts.URL + "/sse",
		Enabled:   true,
	}

	sess, err := mcpserver.OpenSession(ctx, ms)
	if err != nil {
		t.Fatalf("OpenSession 失败: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("首次 Close 失败: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("二次 Close 应幂等: %v", err)
	}
}
