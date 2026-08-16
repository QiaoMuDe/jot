package agent

// 本文件测试会话级 Agent 交互状态的核心协调逻辑：
// ask_user 同轮等待（ClaimAsk 抢占 + WaitForAnswer 阻塞）与 AnswerAskUser 投递答案、
// 并行 ask_user 防挂起（ClaimAsk 互斥）、取消竞态残留答案排空（drainAsk）、
// 取消/会话释放/全量释放（ReleaseAll）时解锁并清理注册表。

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// waitAskPending 轮询等待 WaitForAnswer 进入等待状态（askPending=true）。
func waitAskPending(t *testing.T, sess *agentSession) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		sess.askMu.Lock()
		p := sess.askPending
		sess.askMu.Unlock()
		if p {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("WaitForAnswer 未进入等待状态（askPending 未置位）")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestAnswerAskUserSameRound 验证同轮传输核心（模拟 ask_user 工具真实流程）：
// ClaimAsk 抢占反问名额 → 阻塞等待 → 答案经 AnswerAskUser 投递后返回，且标记清除。
func TestAnswerAskUserSameRound(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(7)
	sess := svc.getOrCreateSession(sid)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 模拟 ask_user 工具：先 ClaimAsk 再阻塞等待
	if err := sess.ClaimAsk(); err != nil {
		t.Fatalf("ClaimAsk 应成功，got %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var got string
	var gotErr error
	go func() {
		defer wg.Done()
		got, gotErr = sess.WaitForAnswer(ctx)
	}()
	waitAskPending(t, sess)

	if err := svc.AnswerAskUser(sid, "选择第一个方案"); err != nil {
		t.Fatalf("AnswerAskUser 应成功，got %v", err)
	}
	wg.Wait()
	if gotErr != nil {
		t.Fatalf("WaitForAnswer 不应报错，got %v", gotErr)
	}
	if got != "选择第一个方案" {
		t.Fatalf("答案投递不一致：got %q", got)
	}
	sess.askMu.Lock()
	pending := sess.askPending
	sess.askMu.Unlock()
	if pending {
		t.Fatal("收到答案后 askPending 应清除")
	}

	// 无等待中的反问时，再次投递应报错
	if err := svc.AnswerAskUser(sid, "多余的答案"); err == nil {
		t.Fatal("无等待反问时 AnswerAskUser 应返回错误")
	}
}

// TestClaimAskRejectsParallelAskUser 验证并行 ask_user 防护：
// 模型同条消息并行发出多条 ask_user 时，仅一条抢占成功，其余 ClaimAsk 报错，
// 不会出现多个等待者共抢一个通道导致整轮挂起。
func TestClaimAskRejectsParallelAskUser(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(8)
	sess := svc.getOrCreateSession(sid)

	// 并发抢占：两个 goroutine 同时 ClaimAsk，恰好一个成功
	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = sess.ClaimAsk()
		}(i)
	}
	wg.Wait()

	okCount := 0
	for _, err := range results {
		if err == nil {
			okCount++
		}
	}
	if okCount != 1 {
		t.Fatalf("并行 ClaimAsk 应恰好一个成功，got %d 个成功（结果 %v）", okCount, results)
	}

	// 抢占成功后清理，避免污染后续用例状态
	sess.askMu.Lock()
	sess.askPending = false
	sess.askMu.Unlock()
}

// TestDrainAskRemovesStaleAnswer 验证取消竞态残留答案被排空：
// 用户提交答案的同时取消（工具 select 抢到 ctx.Done 而非通道），
// 已投递的答案残留在通道；Run 结束/会话释放时 drainAsk 应将其清空，
// 防止下一轮 ask_user 消费到陈旧答案。
func TestDrainAskRemovesStaleAnswer(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(12)
	sess := svc.getOrCreateSession(sid)

	// 构造"答案已投递但未被消费"的残留状态：ClaimAsk + AnswerAskUser
	if err := sess.ClaimAsk(); err != nil {
		t.Fatalf("ClaimAsk 应成功，got %v", err)
	}
	if err := svc.AnswerAskUser(sid, "残留的旧答案"); err != nil {
		t.Fatalf("AnswerAskUser 应成功，got %v", err)
	}
	if len(sess.askCh) != 1 {
		t.Fatalf("通道应恰好残留 1 条答案（前置条件），got len=%d", len(sess.askCh))
	}

	// 排空后通道应无残留，且不影响下一次正常提问
	sess.drainAsk()
	if len(sess.askCh) != 0 {
		t.Fatalf("drainAsk 后通道应清空，got len=%d", len(sess.askCh))
	}
	sess.askMu.Lock()
	pending := sess.askPending
	sess.askMu.Unlock()
	if pending {
		t.Fatal("残留场景下 askPending 应已被 AnswerAskUser 清除")
	}
}

// TestReleaseAllCancelsAll 验证 ReleaseAll（清空所有会话/工厂重置）：
// 取消全部等待中的 run 并清空注册表。
func TestReleaseAllCancelsAll(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sidA = uint(21)
	const sidB = uint(22)
	sessA := svc.getOrCreateSession(sidA)
	sessB := svc.getOrCreateSession(sidB)

	runCtxA, cancelA := context.WithCancel(context.Background())
	sessA.setRunCancel(cancelA)
	runCtxB, cancelB := context.WithCancel(context.Background())
	sessB.setRunCancel(cancelB)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, sess := range []*agentSession{sessA, sessB} {
		wg.Add(1)
		go func(idx int, s *agentSession, ctx context.Context) {
			defer wg.Done()
			_, errs[idx] = s.WaitForAnswer(ctx)
		}(i, sess, []context.Context{runCtxA, runCtxB}[i])
	}
	waitAskPending(t, sessA)
	waitAskPending(t, sessB)

	svc.ReleaseAll()
	wg.Wait()
	for i, err := range errs {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("会话 %d 应被取消，got %v", i, err)
		}
	}
	svc.mu.Lock()
	n := len(svc.sessions)
	svc.mu.Unlock()
	if n != 0 {
		t.Fatalf("ReleaseAll 后注册表应清空，got %d 项", n)
	}
}

// TestAnswerAskUserCancel 验证 ctx 取消（停止按钮）解锁等待并清除标记。
func TestAnswerAskUserCancel(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(9)
	sess := svc.getOrCreateSession(sid)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	var gotErr error
	go func() {
		defer wg.Done()
		_, gotErr = sess.WaitForAnswer(ctx)
	}()
	waitAskPending(t, sess)

	cancel() // 停止按钮/取消
	wg.Wait()
	if !errors.Is(gotErr, context.Canceled) {
		t.Fatalf("取消后应返回 context.Canceled，got %v", gotErr)
	}
	sess.askMu.Lock()
	pending := sess.askPending
	sess.askMu.Unlock()
	if pending {
		t.Fatal("取消后 askPending 应清除")
	}
}

// TestReleaseSessionCancelsWaitingRun 验证会话释放（清空/删除会话）：
// 取消等待中的 run（WaitForAnswer 解锁）并删除注册表项。
func TestReleaseSessionCancelsWaitingRun(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(11)
	sess := svc.getOrCreateSession(sid)

	runCtx, runCancel := context.WithCancel(context.Background())
	sess.setRunCancel(runCancel)

	var wg sync.WaitGroup
	wg.Add(1)
	var gotErr error
	go func() {
		defer wg.Done()
		_, gotErr = sess.WaitForAnswer(runCtx)
	}()
	waitAskPending(t, sess)

	svc.ReleaseSession(sid)
	wg.Wait()
	if !errors.Is(gotErr, context.Canceled) {
		t.Fatalf("会话释放后应返回 context.Canceled，got %v", gotErr)
	}
	svc.mu.Lock()
	_, ok := svc.sessions[sid]
	svc.mu.Unlock()
	if ok {
		t.Fatal("ReleaseSession 后注册表应删除该会话")
	}
}

// TestSessionRegistryReuse 验证同一会话复用同一实例、不同会话各自独立。
func TestSessionRegistryReuse(t *testing.T) {
	svc := NewAgentService(Deps{})
	a1 := svc.getOrCreateSession(1)
	a2 := svc.getOrCreateSession(1)
	if a1 != a2 {
		t.Fatal("同一会话应复用同一实例")
	}
	b := svc.getOrCreateSession(2)
	if a1 == b {
		t.Fatal("不同会话应各自独立实例")
	}
}
