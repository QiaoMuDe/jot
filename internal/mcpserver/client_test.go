package mcpserver_test

import (
	"context"
	"strings"
	"testing"

	"jot/internal/mcpserver"
)

// TestConnectWithCanceledCtx 验证 Connect 在 stdio 启动失败时返回含服务器名的包装错误：
// 不存在的命令会立即失败，无需等待 ConnectTimeout。
func TestConnectWithCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消：即使进程能启动，Initialize 也会立即失败
	cli, err := mcpserver.Connect(ctx, mcpserver.Server{
		Name:      "gone-server",
		Transport: "stdio",
		Command:   "definitely-not-exist-cmd-jot-test",
	})
	if err == nil {
		if cli != nil {
			_ = cli.Close()
		}
		t.Fatal("Connect 应返回错误")
	}
	if !strings.Contains(err.Error(), "gone-server") {
		t.Errorf("错误应含服务器名: %v", err)
	}
	if !strings.Contains(err.Error(), "连接失败") {
		t.Errorf("错误应含「连接失败」: %v", err)
	}
}
