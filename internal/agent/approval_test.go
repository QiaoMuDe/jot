package agent

// 本文件测试会话级 Agent 交互状态中"审批暂停/续跑"的核心协调逻辑：
// RequestApproval 抢占互斥、ApproveToolCall 批准/拒绝投递、取消解锁、
// 取消竞态残留决定排空（drainApproval）、以及审批模式门控（confirm_every/review/auto
// 与 critical 的交互）。不依赖真实模型，通过注入 loadApprovalMode 与构造 client ctx 驱动。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"jot/internal/agent/tools"
)

// waitApprovalPending 轮询等待 RequestApproval 进入等待状态（approvePending=true）。
func waitApprovalPending(t *testing.T, sess *agentSession) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		sess.approveMu.Lock()
		p := sess.approvePending
		sess.approveMu.Unlock()
		if p {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("RequestApproval 未进入等待状态（approvePending 未置位）")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// setTestEmit 为无需真实事件通道的审批测试注入占位发射函数（emit==nil 时
// RequestApproval 会因"缺少事件通道"直接返回错误而非阻塞，故需注入非 nil）。
func setTestEmit(sess *agentSession) {
	sess.emit = func(string, string) {}
}

// TestRequestApprovalApprovedDelivers 验证审批-批准路径：RequestApproval 阻塞后，
// ApproveToolCall(approved=true) 解锁返回 nil；同时校验 ai:tool-approval 事件负载
// 携带 approval_id 与 critical。无等待审批时再次投递返回错误。
func TestRequestApprovalApprovedDelivers(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(31)
	sess := svc.getOrCreateSession(sid)
	ctx := context.Background()

	// 捕获审批事件负载，校验前端回调契约（approval_id / critical）
	eventCh := make(chan string, 1)
	sess.emit = func(event, data string) {
		if event == "ai:tool-approval" {
			eventCh <- data
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var gotErr error
	go func() {
		defer wg.Done()
		gotErr = sess.RequestApproval(ctx, "write_file", "覆盖文件：a.txt", false)
	}()
	waitApprovalPending(t, sess)

	// 事件应在投递前已发射且含正确 approval_id
	sess.approveMu.Lock()
	curID := sess.pendingApprovalID
	sess.approveMu.Unlock()
	select {
	case gotEvent := <-eventCh:
		var ev struct {
			Tool       string `json:"tool"`
			Summary    string `json:"summary"`
			ApprovalID uint64 `json:"approval_id"`
			Critical   bool   `json:"critical"`
		}
		if err := json.Unmarshal([]byte(gotEvent), &ev); err != nil {
			t.Fatalf("审批事件 JSON 解析失败: %v", err)
		}
		if ev.Tool != "write_file" || ev.ApprovalID != curID || ev.Critical {
			t.Fatalf("审批事件负载不符：%+v（期望 tool=write_file, id=%d, critical=false）", ev, curID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("应发射 ai:tool-approval 事件")
	}

	if err := svc.ApproveToolCall(sid, curID, true); err != nil {
		t.Fatalf("ApproveToolCall 应成功，got %v", err)
	}
	wg.Wait()
	if gotErr != nil {
		t.Fatalf("批准后 RequestApproval 应返回 nil，got %v", gotErr)
	}
	sess.approveMu.Lock()
	pending := sess.approvePending
	sess.approveMu.Unlock()
	if pending {
		t.Fatal("收到决定后 approvePending 应清除")
	}

	// 无等待审批时再次投递应报错
	if err := svc.ApproveToolCall(sid, curID, true); err == nil {
		t.Fatal("无等待审批时 ApproveToolCall 应返回错误")
	}
}

// TestRequestApprovalRejectedDelivers 验证审批-拒绝路径：ApproveToolCall(approved=false)
// 解锁返回中文拒绝错误文本。
func TestRequestApprovalRejectedDelivers(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(32)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()

	var wg sync.WaitGroup
	var gotErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		gotErr = sess.RequestApproval(ctx, "run_command", "执行命令：rm -rf x", true)
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	curID := sess.pendingApprovalID
	sess.approveMu.Unlock()

	if err := svc.ApproveToolCall(sid, curID, false); err != nil {
		t.Fatalf("ApproveToolCall 应成功，got %v", err)
	}
	wg.Wait()
	if gotErr == nil {
		t.Fatal("拒绝后 RequestApproval 应返回非 nil 错误")
	}
	if !strings.Contains(gotErr.Error(), "用户拒绝") {
		t.Fatalf("拒绝错误应含'用户拒绝'，实际: %v", gotErr)
	}
}

// TestRequestApprovalRejectsMismatchID 验证 approval_id 不匹配时 ApproveToolCall 报错：
// 防前端串审（提交与当前待审批 id 不一致）。
func TestRequestApprovalRejectsMismatchID(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(33)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = sess.RequestApproval(ctx, "write_file", "x", false)
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	curID := sess.pendingApprovalID
	sess.approveMu.Unlock()

	// 错误的审批 id 应被拒绝，且审批仍处于等待可继续正确投递
	if err := svc.ApproveToolCall(sid, curID+999, true); err == nil {
		t.Fatal("approval_id 不匹配应返回错误")
	}
	if err := svc.ApproveToolCall(sid, curID, true); err != nil {
		t.Fatalf("正确 id 投递应成功，got %v", err)
	}
	wg.Wait()
}

// TestRequestApprovalRejectsParallel 验证并行危险工具防挂起：
// 并发调用 RequestApproval（critical=true）应恰有一个进入等待，其余返回错误，
// 不会多个等待者共抢一个通道导致整轮挂起。
func TestRequestApprovalRejectsParallel(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(34)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()

	results := make([]error, 3)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = sess.RequestApproval(ctx, "run_command", "执行命令：rm -rf", true)
		}(i)
	}
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	curID := sess.pendingApprovalID
	sess.approveMu.Unlock()
	if err := svc.ApproveToolCall(sid, curID, true); err != nil {
		t.Fatalf("ApproveToolCall 应成功，got %v", err)
	}
	wg.Wait()

	nilCount := 0
	for _, err := range results {
		if err == nil {
			nilCount++
		}
	}
	if nilCount != 1 {
		t.Fatalf("并行 RequestApproval 应恰有一个成功，got %d 个成功（结果 %v）", nilCount, results)
	}
}

// TestRequestApprovalCancel 验证 ctx 取消（停止/会话释放）解锁审批等待并清除标记。
func TestRequestApprovalCancel(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(35)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var gotErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		gotErr = sess.RequestApproval(ctx, "write_file", "x", false)
	}()
	waitApprovalPending(t, sess)

	cancel()
	wg.Wait()
	if !errors.Is(gotErr, context.Canceled) {
		t.Fatalf("取消后应返回 context.Canceled，got %v", gotErr)
	}
	sess.approveMu.Lock()
	pending := sess.approvePending
	sess.approveMu.Unlock()
	if pending {
		t.Fatal("取消后 approvePending 应清除")
	}
}

// TestDrainApprovalRemovesStale 验证取消竞态残留的审批决定被排空：
// 决定已投递但未被消费（如取消竞态），drainApproval 应将其清空，防止下一轮审批
// 消费到陈旧决定。
func TestDrainApprovalRemovesStale(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(36)
	sess := svc.getOrCreateSession(sid)

	// 构造"决定已投递但未被消费"的残留状态：抢占审批名额 + ApproveToolCall（无消费方）
	if err := sess.claimApproval(); err != nil {
		t.Fatalf("claimApproval 应成功，got %v", err)
	}
	sess.approveMu.Lock()
	id := sess.pendingApprovalID
	sess.approveMu.Unlock()
	if err := svc.ApproveToolCall(sid, id, true); err != nil {
		t.Fatalf("ApproveToolCall 应成功，got %v", err)
	}
	if len(sess.approveCh) != 1 {
		t.Fatalf("通道应恰好残留 1 条决定（前置条件），got len=%d", len(sess.approveCh))
	}

	// 排空后通道应无残留
	sess.drainApproval()
	if len(sess.approveCh) != 0 {
		t.Fatalf("drainApproval 后通道应清空，got len=%d", len(sess.approveCh))
	}
}

// TestRequestApprovalModeGating 验证审批模式门控：
// review/auto + critical=false 直接放行（不阻塞）；confirm_every + critical=false 会阻塞；
// critical=true：confirm_every / review 阻塞，auto（完全访问）自动放行且仅留痕；
// 非法模式按 confirm_every 处理（阻塞）。
// 通过可注入的 loadApprovalMode 驱动。
func TestRequestApprovalModeGating(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(40)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()
	unlock := func() {
		sess.approveMu.Lock()
		id := sess.pendingApprovalID
		sess.approveMu.Unlock()
		if err := svc.ApproveToolCall(sid, id, false); err != nil {
			t.Fatalf("ApproveToolCall 解锁失败: %v", err)
		}
	}

	// review / auto + critical=false：自动放行，不阻塞
	for _, mode := range []string{"review", "auto"} {
		sess.loadApprovalMode = func() string { return mode }
		if err := sess.RequestApproval(ctx, "write_file", "覆盖文件：x", false); err != nil {
			t.Fatalf("mode=%s + critical=false 应自动放行，got %v", mode, err)
		}
	}

	// confirm_every + critical=false：阻塞确认
	sess.loadApprovalMode = func() string { return "confirm_every" }
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = sess.RequestApproval(ctx, "write_file", "覆盖文件：x", false)
	}()
	waitApprovalPending(t, sess)
	unlock()
	wg.Wait()

	// critical=true：confirm_every / review 阻塞确认（不可绕过的最后防线）；
	// auto（完全访问）不再阻塞，改为自动放行并发非阻塞告警事件。
	for _, mode := range []string{"review", "confirm_every"} {
		sess.loadApprovalMode = func() string { return mode }
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sess.RequestApproval(ctx, "run_command", "rm -rf", true)
		}()
		waitApprovalPending(t, sess)
		unlock()
		wg.Wait()
	}

	// auto + critical=true：不阻塞直接放行；发射 ai:tool-status(tool_auto_approval)
	// 审计事件，并追加一条 tool_auto_approval 留痕记录（明确"未经人工确认自动放行"）。
	sess.loadApprovalMode = func() string { return "auto" }
	statusCh := make(chan string, 1)
	sess.emit = func(event, data string) {
		if event == "ai:tool-status" {
			statusCh <- data
		}
	}
	var records []tools.Record
	sess.appendRecord = func(rec tools.Record) { records = append(records, rec) }
	if err := sess.RequestApproval(ctx, "run_command", "rm -rf", true); err != nil {
		t.Fatalf("auto + critical=true 应自动放行，got %v", err)
	}
	// 应发射 ai:tool-status 的 tool_auto_approval 审计事件
	select {
	case data := <-statusCh:
		if !strings.Contains(data, "tool_auto_approval") {
			t.Fatalf("自动放行应发射 tool_auto_approval 审计事件，got：%s", data)
		}
	default:
		t.Fatal("auto 自动放行应发射 tool_auto_approval 审计事件")
	}
	// 应追加一条 tool_auto_approval 的详细留痕记录（明确未经人工确认放行）
	if len(records) != 1 {
		t.Fatalf("auto 自动放行应写入 1 条留痕记录，got %d", len(records))
	}
	if records[0].Action != "tool_auto_approval" {
		t.Fatalf("留痕记录 action 不符：%+v", records[0])
	}
	if !strings.Contains(records[0].Result, "自动放行") {
		t.Fatalf("留痕记录未标注自动放行：%+v", records[0])
	}

	// 非法模式：按 confirm_every 处理（critical=false 仍会阻塞）
	sess.loadApprovalMode = func() string { return "bogus" }
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = sess.RequestApproval(ctx, "write_file", "覆盖文件：x", false)
	}()
	waitApprovalPending(t, sess)
	unlock()
	wg.Wait()
}

// TestRequestApprovalRecordsDecision 验证审批留痕：批准/拒绝决定后均追加一条
// tool_approval 记录（经 appendRecord），字段含工具名/摘要/结果。返回语义不受影响。
func TestRequestApprovalRecordsDecision(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(50)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()

	// 批准路径
	var approvedRec, rejectedRec tools.Record
	received := 0
	gotRec := func(r tools.Record) {
		received++
		if r.Action != "tool_approval" || r.Result == "" {
			t.Errorf("留痕记录字段异常: %+v", r)
		}
	}
	sess.appendRecord = func(r tools.Record) {
		gotRec(r)
		approvedRec = r
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sess.RequestApproval(ctx, "run_command", "执行命令：rm x", true); err != nil {
			t.Errorf("批准路径应返回 nil，got %v", err)
		}
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	id := sess.pendingApprovalID
	sess.approveMu.Unlock()
	if err := svc.ApproveToolCall(sid, id, true); err != nil {
		t.Fatalf("ApproveToolCall 批准失败: %v", err)
	}
	wg.Wait()
	if approvedRec.Name != "run_command" || approvedRec.Result != "批准" {
		t.Errorf("批准留痕记录不符：%+v", approvedRec)
	}
	if !strings.Contains(approvedRec.Args, "rm x") {
		t.Errorf("批准留痕摘要应含 rm x，实际: %q", approvedRec.Args)
	}

	// 拒绝路径
	sess.appendRecord = func(r tools.Record) {
		gotRec(r)
		rejectedRec = r
	}
	var gotErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		gotErr = sess.RequestApproval(ctx, "run_command", "执行命令：rm x", true)
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	id = sess.pendingApprovalID
	sess.approveMu.Unlock()
	if err := svc.ApproveToolCall(sid, id, false); err != nil {
		t.Fatalf("ApproveToolCall 拒绝失败: %v", err)
	}
	wg.Wait()
	if gotErr == nil || !strings.Contains(gotErr.Error(), "用户拒绝") {
		t.Fatalf("拒绝路径应返回拒绝错误，got %v", gotErr)
	}
	if rejectedRec.Name != "run_command" || rejectedRec.Result != "已被用户拒绝" {
		t.Errorf("拒绝留痕记录不符：%+v", rejectedRec)
	}
	if received != 2 {
		t.Errorf("应追加 2 条留痕记录（批准+拒绝），实际 %d", received)
	}
}
