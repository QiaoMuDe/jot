package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestWrapConnectError 验证 wrapConnectError 两个分支：
// DeadlineExceeded → 「连接超时」文案；其余错误 → 「连接失败」文案；均附带服务器名。
func TestWrapConnectError(t *testing.T) {
	// 超时分支：使用已过期 deadline 的 ctx，ctx.Err() 为 DeadlineExceeded
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if ctx.Err() == nil {
		t.Fatal("预置 deadline 过期失败")
	}
	err := wrapConnectError("slow-server", ctx.Err())
	if !strings.Contains(err.Error(), "连接超时") || !strings.Contains(err.Error(), "slow-server") {
		t.Errorf("超时错误文案异常: %v", err)
	}

	// 普通错误分支
	err2 := wrapConnectError("bad-server", errors.New("boom"))
	if !strings.Contains(err2.Error(), "连接失败") || !strings.Contains(err2.Error(), "bad-server") {
		t.Errorf("普通错误文案异常: %v", err2)
	}
}
