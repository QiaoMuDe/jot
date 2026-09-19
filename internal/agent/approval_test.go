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

	if err := svc.ApproveToolCall(sid, curID, true, false); err != nil {
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
	if err := svc.ApproveToolCall(sid, curID, true, false); err == nil {
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

	if err := svc.ApproveToolCall(sid, curID, false, false); err != nil {
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
	if err := svc.ApproveToolCall(sid, curID+999, true, false); err == nil {
		t.Fatal("approval_id 不匹配应返回错误")
	}
	if err := svc.ApproveToolCall(sid, curID, true, false); err != nil {
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
	if err := svc.ApproveToolCall(sid, curID, true, false); err != nil {
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
	if err := svc.ApproveToolCall(sid, id, true, false); err != nil {
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
		if err := svc.ApproveToolCall(sid, id, false, false); err != nil {
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
	if err := svc.ApproveToolCall(sid, id, true, false); err != nil {
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
	if err := svc.ApproveToolCall(sid, id, false, false); err != nil {
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

// TestRequestApprovalRoundAllow 验证「允许本轮」授权后本轮内一律直接放行：
// 首次普通审批经用户在面板选择「允许本轮」（approved=true, allowRound=true）后置位授权，
// 随后的普通与高危操作均立即返回 nil（不阻塞、不弹窗），且不再发射新的 ai:tool-approval
// 事件、不占用审批名额。
func TestRequestApprovalRoundAllow(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(61)
	sess := svc.getOrCreateSession(sid)
	ctx := context.Background()
	// 最严格模式：未授权时普通操作也会阻塞确认
	sess.loadApprovalMode = func() string { return "confirm_every" }

	// 统计 ai:tool-approval 事件数（emit 与断言可能跨 goroutine，加锁保护）
	var eventsMu sync.Mutex
	approvalEvents := 0
	sess.emit = func(event, data string) {
		if event == "ai:tool-approval" {
			eventsMu.Lock()
			approvalEvents++
			eventsMu.Unlock()
		}
	}

	// 首次普通审批：阻塞等待，用户选择「允许本轮」
	var wg sync.WaitGroup
	var firstErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		firstErr = sess.RequestApproval(ctx, "write_file", "覆盖文件：a.txt", false)
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	id := sess.pendingApprovalID
	sess.approveMu.Unlock()
	if err := svc.ApproveToolCall(sid, id, true, true); err != nil {
		t.Fatalf("批准并授权本轮应成功，got %v", err)
	}
	wg.Wait()
	if firstErr != nil {
		t.Fatalf("首次审批被批准后应返回 nil，got %v", firstErr)
	}
	if !sess.allowRoundAll.Load() {
		t.Fatal("批准并授权本轮后 allowRoundAll 应为 true")
	}
	eventsMu.Lock()
	firstCount := approvalEvents
	eventsMu.Unlock()
	if firstCount != 1 {
		t.Fatalf("首次审批应发射恰 1 次 ai:tool-approval，实际 %d", firstCount)
	}

	// 授权后：普通操作立即放行
	if err := sess.RequestApproval(ctx, "write_file", "覆盖文件：a.txt", false); err != nil {
		t.Fatalf("本轮已授权，普通操作应直接放行，got %v", err)
	}
	// 授权后：高危操作同样立即放行
	if err := sess.RequestApproval(ctx, "run_command", "执行命令：rm -rf x", true); err != nil {
		t.Fatalf("本轮已授权，高危操作也应直接放行，got %v", err)
	}
	if !sess.allowRoundAll.Load() {
		t.Fatal("本轮未结束，allowRoundAll 应保持 true")
	}

	// 两次短路放行不应再发射审批事件（不弹窗）
	eventsMu.Lock()
	afterCount := approvalEvents
	eventsMu.Unlock()
	if afterCount != firstCount {
		t.Fatalf("本轮已授权后不应再发射 ai:tool-approval，期望 %d 次，实际 %d 次", firstCount, afterCount)
	}
	// 短路分支不占用审批名额
	sess.approveMu.Lock()
	pending := sess.approvePending
	sess.approveMu.Unlock()
	if pending {
		t.Fatal("本轮放行短路不应占用审批名额（approvePending 应保持 false）")
	}
}

// TestRequestApprovalRoundAllowAudit 验证本轮授权后的留痕语义：
// 高危操作经 recordAutoApproval 追加恰 1 条 tool_auto_approval 记录，原因文案含
// 「本轮已授权自动放行」且与完全访问模式的「完全访问模式自动放行」区分；
// 普通操作走同一短路分支但不额外留痕；用户点击「允许本轮」当次的 tool_approval
// 记录为第三态（Result 含「已允许本轮」）。
func TestRequestApprovalRoundAllowAudit(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(62)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()
	sess.loadApprovalMode = func() string { return "confirm_every" }

	// 收集留痕记录。此处直接 Store(true) 置位而非走一次完整授权交互：本用例聚焦
	// 短路分支的留痕文案，直接置位与「用户选择允许本轮」后的状态语义等价，可省去交互噪声。
	var records []tools.Record
	sess.appendRecord = func(r tools.Record) { records = append(records, r) }
	sess.allowRoundAll.Store(true)

	// 高危操作：应追加恰 1 条 tool_auto_approval 留痕
	if err := sess.RequestApproval(ctx, "run_command", "执行命令：rm -rf x", true); err != nil {
		t.Fatalf("本轮已授权，高危操作应直接放行，got %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("高危自动放行应追加恰 1 条留痕记录，实际 %d 条（%+v）", len(records), records)
	}
	if records[0].Action != "tool_auto_approval" {
		t.Fatalf("留痕 action 应为 tool_auto_approval，实际 %q", records[0].Action)
	}
	if !strings.Contains(records[0].Result, "本轮已授权自动放行") {
		t.Fatalf("留痕结果应含「本轮已授权自动放行」，实际 %q", records[0].Result)
	}
	if strings.Contains(records[0].Result, "完全访问模式自动放行") {
		t.Fatalf("本轮授权留痕不应与完全访问模式文案混淆，实际 %q", records[0].Result)
	}

	// 普通操作：不额外留痕
	if err := sess.RequestApproval(ctx, "write_file", "覆盖文件：a.txt", false); err != nil {
		t.Fatalf("本轮已授权，普通操作应直接放行，got %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("普通操作不应追加留痕记录，实际 %d 条（%+v）", len(records), records)
	}

	// 「允许本轮」当次的审批记录为第三态：复位授权态后走真实审批路径
	sess.resetRoundAllow()
	records = nil
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = sess.RequestApproval(ctx, "write_file", "覆盖文件：b.txt", false)
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	id := sess.pendingApprovalID
	sess.approveMu.Unlock()
	if err := svc.ApproveToolCall(sid, id, true, true); err != nil {
		t.Fatalf("批准并授权本轮应成功，got %v", err)
	}
	wg.Wait()

	var approvalRec *tools.Record
	for i := range records {
		if records[i].Action == "tool_approval" {
			approvalRec = &records[i]
		}
	}
	if approvalRec == nil {
		t.Fatalf("「允许本轮」应追加 1 条 tool_approval 记录，实际 %+v", records)
	}
	if !strings.Contains(approvalRec.Result, "已允许本轮") {
		t.Fatalf("「允许本轮」留痕应为第三态（含「已允许本轮」），实际 %q", approvalRec.Result)
	}
}

// TestApproveToolCallRoundScope 验证「本轮放行」仅在审批校验通过后置位，防误置位：
// 会话不存在 / 无等待中审批 / approval_id 不匹配时均返回中文错误且不置位本轮授权；
// 用正确 id 但 allowRound=false 批准时不置位（批准本次但不授权本轮）。
func TestApproveToolCallRoundScope(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(63)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()

	// 子场景 A：会话存在但无等待中审批 → 报错且不置位
	err := svc.ApproveToolCall(sid, 1, true, true)
	if err == nil {
		t.Fatal("无等待审批时 ApproveToolCall 应返回错误")
	}
	if !strings.Contains(err.Error(), "审批") && !strings.Contains(err.Error(), "会话") {
		t.Fatalf("错误消息应为中文且含「审批」或「会话」，实际 %q", err.Error())
	}
	if sess.allowRoundAll.Load() {
		t.Fatal("无等待审批时不应置位本轮放行")
	}
	// 会话不存在同样报错且不置位
	notExistErr := svc.ApproveToolCall(sid+1000, 1, true, true)
	if notExistErr == nil {
		t.Fatal("会话不存在时 ApproveToolCall 应返回错误")
	}
	if !strings.Contains(notExistErr.Error(), "审批") && !strings.Contains(notExistErr.Error(), "会话") {
		t.Fatalf("错误消息应为中文且含「审批」或「会话」，实际 %q", notExistErr.Error())
	}

	// 子场景 B：构造等待中审批（默认 confirm_every 会阻塞）
	var wg sync.WaitGroup
	var gotErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		gotErr = sess.RequestApproval(ctx, "write_file", "覆盖文件：a.txt", false)
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	curID := sess.pendingApprovalID
	sess.approveMu.Unlock()

	// 错误 id + allowRound=true → 报错且不置位
	mismatchErr := svc.ApproveToolCall(sid, curID+999, true, true)
	if mismatchErr == nil {
		t.Fatal("approval_id 不匹配应返回错误")
	}
	if !strings.Contains(mismatchErr.Error(), "审批") {
		t.Fatalf("错误消息应为中文且含「审批」，实际 %q", mismatchErr.Error())
	}
	if sess.allowRoundAll.Load() {
		t.Fatal("approval_id 不匹配时不应置位本轮放行")
	}

	// 正确 id + allowRound=false → 批准本次但不授权本轮
	if err := svc.ApproveToolCall(sid, curID, true, false); err != nil {
		t.Fatalf("正确 id 投递应成功，got %v", err)
	}
	wg.Wait()
	if gotErr != nil {
		t.Fatalf("批准后 RequestApproval 应返回 nil，got %v", gotErr)
	}
	if sess.allowRoundAll.Load() {
		t.Fatal("allowRound=false 批准不应置位本轮放行")
	}
}

// TestResetRoundAllow 验证「本轮放行」随本轮结束失效：
// resetRoundAllow（Run 结束时调用）复位授权态，此前被短路的同一操作重新恢复为阻塞等待。
func TestResetRoundAllow(t *testing.T) {
	svc := NewAgentService(Deps{})
	const sid = uint(64)
	sess := svc.getOrCreateSession(sid)
	setTestEmit(sess)
	ctx := context.Background()
	sess.loadApprovalMode = func() string { return "confirm_every" }

	// 置位后高危操作直接放行
	sess.allowRoundAll.Store(true)
	if err := sess.RequestApproval(ctx, "run_command", "rm -rf", true); err != nil {
		t.Fatalf("本轮已授权，高危操作应直接放行，got %v", err)
	}

	// 本轮结束：复位授权态（幂等）
	sess.resetRoundAllow()
	if sess.allowRoundAll.Load() {
		t.Fatal("resetRoundAllow 后 allowRoundAll 应为 false")
	}
	sess.resetRoundAllow() // 再次调用应幂等无副作用
	if sess.allowRoundAll.Load() {
		t.Fatal("resetRoundAllow 重复调用后 allowRoundAll 仍应为 false")
	}

	// 复位后同一操作重新阻塞等待（证明授权不跨轮）
	var wg sync.WaitGroup
	var gotErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		gotErr = sess.RequestApproval(ctx, "run_command", "rm -rf", true)
	}()
	waitApprovalPending(t, sess)
	sess.approveMu.Lock()
	id := sess.pendingApprovalID
	sess.approveMu.Unlock()
	if err := svc.ApproveToolCall(sid, id, false, false); err != nil {
		t.Fatalf("ApproveToolCall 解锁失败: %v", err)
	}
	wg.Wait()
	if gotErr == nil || !strings.Contains(gotErr.Error(), "用户拒绝") {
		t.Fatalf("复位后审批被拒绝应返回含「用户拒绝」的错误，got %v", gotErr)
	}
}
