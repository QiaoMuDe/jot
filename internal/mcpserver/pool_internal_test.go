package mcpserver

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// fakeSessRecorder 记录 fake open 创建的会话，便于断言关闭行为（同包测试可读 closed 字段）。
type fakeSessRecorder struct {
	mu         sync.Mutex
	opened     int
	openedSess []*Session // 创建的会话引用（按创建顺序）
	failNext   map[string]bool
}

// fakeOpen 构造一个注入用 open 函数：返回带 ServerName 的空会话（Tools 为空），
// 按 failNext 模拟失败；创建的会话记录到 recorder 供断言关闭状态。
func fakeOpen(rec *fakeSessRecorder) func(ctx context.Context, s Server) (*Session, error) {
	return func(_ context.Context, s Server) (*Session, error) {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		if rec.failNext[s.Name] {
			delete(rec.failNext, s.Name)
			return nil, errors.New("fake connect failed: " + s.Name)
		}
		rec.opened++
		sess := &Session{ServerName: s.Name, srv: s}
		rec.openedSess = append(rec.openedSess, sess)
		return sess, nil
	}
}

// closedCount 统计已关闭（closed=true）的会话数。
func (rec *fakeSessRecorder) closedCount() int {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return rec.closedCountLocked()
}

// closedCountLocked 统计已关闭会话数（调用方须已持有 rec.mu）。
func (rec *fakeSessRecorder) closedCountLocked() int {
	n := 0
	for _, s := range rec.openedSess {
		if s.closed {
			n++
		}
	}
	return n
}

// mustPool 构造带 fake open 的 Pool 与 recorder。
func mustPool(rec *fakeSessRecorder) *Pool {
	p := NewPool()
	p.setOpen(fakeOpen(rec))
	return p
}

func baseServer(name string) Server {
	return Server{Name: name, Transport: "http", URL: "http://example.com/" + name, Enabled: true}
}

// TestPoolWarmupReuse 验证幂等：同配置两次 Warmup → open 仅 1 次，第二次 Reused。
func TestPoolWarmupReuse(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()
	srv := baseServer("svr-a")

	res1 := p.Warmup(ctx, []Server{srv})
	if res1.Warmed != 1 || res1.Reused != 0 || res1.Failed != 0 {
		t.Fatalf("首次 Warmup 结果异常: %+v", res1)
	}
	res2 := p.Warmup(ctx, []Server{srv})
	if res2.Warmed != 0 || res2.Reused != 1 {
		t.Fatalf("二次 Warmup 应复用: %+v", res2)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.opened != 1 {
		t.Errorf("open 应仅调用 1 次, got %d", rec.opened)
	}
}

// TestPoolWarmupSkipsDisabled 验证仅预热 enabled 的服务器：disabled 不进入池。
func TestPoolWarmupSkipsDisabled(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()

	enabledHTTP := baseServer("http-a")
	disabledHTTP := baseServer("disabled-a")
	disabledHTTP.Enabled = false
	stdio := baseServer("stdio-a")
	stdio.Transport = "stdio"

	res := p.Warmup(ctx, []Server{enabledHTTP, disabledHTTP, stdio})
	if res.Total != 2 || res.Warmed != 2 {
		t.Fatalf("应预热 2 台 enabled（http + stdio）: %+v", res)
	}
	if p.Session("http-a") == nil {
		t.Error("http-a 应入池")
	}
	if p.Session("stdio-a") == nil {
		t.Error("stdio-a 应入池（stdio 池化）")
	}
	if p.Session("disabled-a") != nil {
		t.Error("disabled 不应入池")
	}
}

// TestPoolWarmupFingerprintChangeReconnect 验证配置指纹变化 → 关闭旧会话并重连。
func TestPoolWarmupFingerprintChangeReconnect(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()

	srv1 := baseServer("svr-b")
	srv1.URL = "http://old.example.com/b"
	srv2 := baseServer("svr-b")
	srv2.URL = "http://new.example.com/b"

	res1 := p.Warmup(ctx, []Server{srv1})
	if res1.Warmed != 1 {
		t.Fatalf("首次 Warmup 结果异常: %+v", res1)
	}
	res2 := p.Warmup(ctx, []Server{srv2})
	if res2.Warmed != 1 || res2.Closed != 1 {
		t.Fatalf("指纹变化应重连(Closed=1, Warmed=1): %+v", res2)
	}
	rec.mu.Lock()
	opened := rec.opened
	closed := rec.closedCountLocked()
	rec.mu.Unlock()
	if opened != 2 {
		t.Errorf("open 应调用 2 次, got %d", opened)
	}
	if closed != 1 {
		t.Errorf("旧会话应关闭 1 个, got %d", closed)
	}
}

// TestPoolWarmupFailureNotCached 验证预热失败不缓存，Session 返回 nil。
func TestPoolWarmupFailureNotCached(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{"svr-c": true}}
	p := mustPool(rec)
	ctx := context.Background()

	srv := baseServer("svr-c")
	res := p.Warmup(ctx, []Server{srv})
	if res.Failed != 1 || res.FailedMsgs == nil {
		t.Fatalf("预热失败应记录: %+v", res)
	}
	if p.Session("svr-c") != nil {
		t.Error("失败服务器不应入池")
	}
}

// TestPoolReconcileClosesRemoved 验证 Reconcile 关闭不在列表中的条目。
func TestPoolReconcileClosesRemoved(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()

	srvA := baseServer("svr-a")
	srvB := baseServer("svr-b")
	p.Warmup(ctx, []Server{srvA, srvB})

	res := p.Reconcile(ctx, []Server{srvA}) // svr-b 消失
	if res.Closed != 1 {
		t.Fatalf("Reconcile 应关闭 svr-b: %+v", res)
	}
	if p.Session("svr-b") != nil {
		t.Error("svr-b 应已出池")
	}
	if p.Session("svr-a") == nil {
		t.Error("svr-a 应保留")
	}
}

// TestPoolCloseAll 验证 CloseAll 清空并关闭全部条目。
func TestPoolCloseAll(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()

	p.Warmup(ctx, []Server{baseServer("svr-a"), baseServer("svr-b")})
	if p.Count() != 2 {
		t.Fatalf("预热后 Count = %d, want 2", p.Count())
	}
	p.CloseAll()
	if p.Count() != 0 {
		t.Errorf("CloseAll 后 Count = %d, want 0", p.Count())
	}
	if rec.closedCount() != 2 {
		t.Errorf("CloseAll 应关闭 2 个会话, got %d", rec.closedCount())
	}
	// CloseAll 幂等
	p.CloseAll()
}

// TestPoolWarmupOneFallback 验证 WarmupOne 兜底：未预热时现场连接并入池，重复调用复用。
func TestPoolWarmupOneFallback(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()

	srv := baseServer("svr-d")
	sess1, err := p.WarmupOne(ctx, srv)
	if err != nil || sess1 == nil {
		t.Fatalf("WarmupOne 首次失败: %v", err)
	}
	sess2, err := p.WarmupOne(ctx, srv)
	if err != nil {
		t.Fatalf("WarmupOne 二次失败: %v", err)
	}
	if sess1 != sess2 {
		t.Error("WarmupOne 二次应复用同一会话")
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.opened != 1 {
		t.Errorf("open 应仅 1 次, got %d", rec.opened)
	}
}

// TestPoolConcurrentWarmup 验证并发预热无数据竞争（配合 -race 运行）。
func TestPoolConcurrentWarmup(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()

	names := []string{"a", "b", "c", "d", "e", "f"}
	servers := make([]Server, 0, len(names))
	for _, n := range names {
		servers = append(servers, baseServer("svr-"+n))
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Warmup(ctx, servers)
		}()
	}
	wg.Wait()
	if p.Count() != 6 {
		t.Errorf("并发预热后 Count = %d, want 6", p.Count())
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.opened != 6 {
		t.Errorf("open 应恰好 6 次（各台仅一次）, got %d", rec.opened)
	}
}

// TestPoolNilSafe 验证 Pool 空操作安全（nil receiver / 空列表）。
func TestPoolNilSafe(t *testing.T) {
	var p *Pool
	ctx := context.Background()
	p.Warmup(ctx, nil) // nil receiver 安全
	p.Reconcile(ctx, nil)
	p.CloseAll()
	p.Close("anything")
	if p.Session("x") != nil {
		t.Error("nil pool Session 应返回 nil")
	}

	p2 := NewPool()
	if res := p2.Warmup(ctx, nil); res.Total != 0 {
		t.Errorf("空列表 Warmup 应 Total=0: %+v", res)
	}
}

// TestPoolClose 验证 Close 单台（停用/删除时调用）：关闭并出池，不存在时静默。
func TestPoolClose(t *testing.T) {
	rec := &fakeSessRecorder{failNext: map[string]bool{}}
	p := mustPool(rec)
	ctx := context.Background()

	p.Warmup(ctx, []Server{baseServer("svr-a"), baseServer("svr-b")})
	p.Close("svr-a")
	if p.Session("svr-a") != nil {
		t.Error("Close 后 svr-a 应出池")
	}
	if p.Session("svr-b") == nil {
		t.Error("svr-b 应保留")
	}
	if rec.closedCount() != 1 {
		t.Errorf("应关闭 1 个会话, got %d", rec.closedCount())
	}
	p.Close("nonexistent") // 静默
}
