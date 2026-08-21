package mcpserver

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPoolWarmupOneWaitsForInflight 验证「预热进行中发消息」的并发场景：
// Warmup 建连服务器 A 尚未完成（open 阻塞中），此时 WarmupOne(A)（发消息兜底路径）
// 应等待而非重复建连；预热完成后两者复用同一会话，open 仅调用 1 次。
func TestPoolWarmupOneWaitsForInflight(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	// 覆盖默认 open：第一个建连阻塞在 gate 上，直到测试放行
	gate := make(chan struct{})
	blockedOpen := func(ctx context.Context, s Server) (*Session, error) {
		rec.mu.Lock()
		rec.opened++
		rec.mu.Unlock()
		<-gate // 模拟慢建连（预热未完成）
		rec.mu.Lock()
		rec.openedSess = append(rec.openedSess, &Session{ServerName: s.Name, srv: s})
		rec.mu.Unlock()
		return rec.openedSess[len(rec.openedSess)-1], nil
	}
	p := NewPool()
	p.setOpen(blockedOpen)

	ctx := context.Background()
	srv := baseServer("svr-wait")

	// 1. 预热开始（goroutine 中），建连阻塞在 gate
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Warmup(ctx, []Server{srv})
	}()

	// 等待 Warmup 进入建连（open 被调用即已注册 inflight）
	deadline := time.Now().Add(2 * time.Second)
	for {
		rec.mu.Lock()
		n := rec.opened
		rec.mu.Unlock()
		if n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Warmup 未进入建连状态")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// 2. 预热未完成时，发消息路径 WarmupOne 应等待（不新建、不返回错误）
	done := make(chan struct{})
	var sess2 *Session
	var err2 error
	go func() {
		sess2, err2 = p.WarmupOne(ctx, srv)
		close(done)
	}()
	// 放行建连，让预热完成
	close(gate)

	select {
	case <-done:
		if err2 != nil {
			t.Fatalf("WarmupOne 等待预热后失败: %v", err2)
		}
		if sess2 == nil {
			t.Fatal("WarmupOne 返回 nil 会话")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("WarmupOne 未在预热完成后返回（未等待/死锁）")
	}
	wg.Wait()

	// 3. open 应仅调用 1 次（发消息路径未重复建连）
	rec.mu.Lock()
	opened := rec.opened
	sess1 := rec.openedSess[0]
	rec.mu.Unlock()
	if opened != 1 {
		t.Errorf("open 应仅调用 1 次（发消息路径应等待复用）, got %d", opened)
	}
	if sess1 != sess2 {
		t.Error("WarmupOne 应复用预热创建的同一会话")
	}
}
