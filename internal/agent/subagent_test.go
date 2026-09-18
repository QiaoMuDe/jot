package agent

// 本文件覆盖 os_agent 子 Agent 委托工具的单元测试：
//  1. 内层白名单装配（12 个文件/命令工具）；
//  2. chatModel 为 nil / disabled 禁用时的装配跳过；
//  3. 内层 Context 与父层同一指针（审批共享）；
//  4. InvokableRun 参数错误分支；
//  5. 核心：内层每步工具调用的事件转发顺序
//     （os_agent start → 内层 start/result → os_agent result，记录与事件均按序）。

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/agent/tools"
)

// fakeApprover 测试用审批器 stub（实现 tools.Approver，断言注入同一指针）。
type fakeApprover struct{}

func (f *fakeApprover) RequestApproval(context.Context, string, string, bool) error { return nil }

// fakeAgent 测试用内层 Agent stub：实现 adk.Agent 接口，Run 返回预设事件流。
type fakeAgent struct {
	name   string
	desc   string
	events []*adk.AgentEvent
}

func (f *fakeAgent) Name(context.Context) string        { return f.name }
func (f *fakeAgent) Description(context.Context) string { return f.desc }

func (f *fakeAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		for _, ev := range f.events {
			gen.Send(ev)
		}
	}()
	return iter
}

// newTestInnerCtx 构造测试用内层工具上下文：收集 ai:tool-status 事件负载与调用记录。
func newTestInnerCtx() (*tools.Context, *[]tools.Record, *[]string) {
	records := &[]tools.Record{}
	emitted := &[]string{}
	ctx := &tools.Context{
		Emit:    func(_ string, data string) { *emitted = append(*emitted, data) },
		Records: records,
	}
	return ctx, records, emitted
}

// TestBuildOSSubAgentNilChatModel chatModel 为 nil 时构造返回 nil（buildTools 过滤循环跳过）。
func TestBuildOSSubAgentNilChatModel(t *testing.T) {
	innerCtx := &tools.Context{}
	if oa := buildOSSubAgent(context.Background(), nil, innerCtx); oa != nil {
		t.Errorf("chatModel 为 nil 时应返回 nil，实际返回 %#v", oa)
	}
}

// TestBuildOSSubAgentInnerTools 内层白名单恰好 12 个，名称集合与 osSubAgentToolNames 一致。
func TestBuildOSSubAgentInnerTools(t *testing.T) {
	innerCtx, _, _ := newTestInnerCtx()
	oa := buildOSSubAgent(context.Background(), &openai.ChatModel{}, innerCtx)
	if oa == nil {
		t.Fatal("chatModel 非 nil 时应构造成功")
	}
	got := oa.InnerTools()
	if len(got) != len(osSubAgentToolNames) {
		t.Fatalf("内层白名单应为 %d 个工具，实际 %d 个", len(osSubAgentToolNames), len(got))
	}
	names := make(map[string]bool, len(got))
	for _, it := range got {
		info, err := it.Info(context.Background())
		if err != nil || info == nil {
			t.Fatalf("内层工具 Info 失败: %v", err)
		}
		names[info.Name] = true
	}
	for _, want := range osSubAgentToolNames {
		if !names[want] {
			t.Errorf("内层白名单缺少工具 %q", want)
		}
	}
}

// TestBuildOSSubAgentPlatformNoteInjected 平台片段已注入内层提示词（osSubAgentInstruction 初始化时）。
func TestBuildOSSubAgentPlatformNoteInjected(t *testing.T) {
	innerCtx, _, _ := newTestInnerCtx()
	oa := buildOSSubAgent(context.Background(), &openai.ChatModel{}, innerCtx)
	if oa == nil {
		t.Fatal("chatModel 非 nil 时应构造成功")
	}
	wantNote := "当前平台: " + runtime.GOOS
	if !strings.Contains(oa.cfg.instruction, wantNote) {
		t.Errorf("内层提示词应包含平台片段 %q，实际:\n%s", wantNote, oa.cfg.instruction)
	}
	// osSubAgentInstruction 在初始化时即已注入平台片段（非纯原始提示词）
	if !strings.Contains(osSubAgentInstruction, wantNote) {
		t.Errorf("osSubAgentInstruction 应已注入平台片段 %q", wantNote)
	}
}

// TestBuildOSSubAgentApproverShared 内层 Context 与传入 ctx 为同一指针（同一会话 Approver）。
func TestBuildOSSubAgentApproverShared(t *testing.T) {
	approver := &fakeApprover{}
	innerCtx := &tools.Context{Approver: approver}
	oa := buildOSSubAgent(context.Background(), &openai.ChatModel{}, innerCtx)
	if oa == nil {
		t.Fatal("chatModel 非 nil 时应构造成功")
	}
	if oa.ctx != innerCtx {
		t.Errorf("osAgentTool.ctx 应为传入的同一指针（Approver 共享）")
	}
	if innerCtx.Approver != approver {
		t.Errorf("Approver 未注入内层 Context")
	}
}

// TestBuildToolsNoChatModelSkipsOSAgent chatModel 为 nil 时 buildTools 不含 os_agent 且无文件工具。
func TestBuildToolsNoChatModelSkipsOSAgent(t *testing.T) {
	p := BuildParams{deps: Deps{}, req: Request{}, ctx: &tools.Context{}, runCtx: context.Background(), chatModel: nil}
	toolList := buildTools(p, nil, false)
	names := toolNamesOf(t, toolList)
	if names["os_agent"] {
		t.Errorf("chatModel 为 nil 时不应注册 os_agent")
	}
	for _, ft := range osSubAgentToolNames {
		if names[ft] {
			t.Errorf("文件/命令工具 %q 不应直接注册在父层（已封装进 os_agent）", ft)
		}
	}
}

// TestBuildToolsDisabledOSAgent disabled 含 os_agent 时 buildTools 跳过该工具，其余工具正常装配。
func TestBuildToolsDisabledOSAgent(t *testing.T) {
	p := BuildParams{deps: Deps{}, req: Request{}, ctx: &tools.Context{}, runCtx: context.Background(), chatModel: &openai.ChatModel{}}
	toolList := buildTools(p, map[string]bool{"os_agent": true}, false)
	names := toolNamesOf(t, toolList)
	if names["os_agent"] {
		t.Errorf("disabled 含 os_agent 时不应注册 os_agent")
	}
	if len(toolList) == 0 {
		t.Errorf("禁用 os_agent 不应影响其余工具装配")
	}
	if !names["read_url"] {
		t.Errorf("其余工具（如 read_url）应正常装配")
	}
}

// TestOSAgentInvokableRunEmptyRequest request 为空 / 参数非法时报中文错误。
func TestOSAgentInvokableRunEmptyRequest(t *testing.T) {
	innerCtx, _, _ := newTestInnerCtx()
	oa := buildOSSubAgent(context.Background(), &openai.ChatModel{}, innerCtx)
	if oa == nil {
		t.Fatal("chatModel 非 nil 时应构造成功")
	}
	ctx := context.Background()
	if _, err := oa.InvokableRun(ctx, `{"request":""}`); err == nil {
		t.Errorf("request 为空应报错")
	} else if !strings.Contains(err.Error(), "request") {
		t.Errorf("错误信息应提示缺少 request，实际: %v", err)
	}
	if _, err := oa.InvokableRun(ctx, `not-json`); err == nil {
		t.Errorf("非法 JSON 应报错")
	}
}

// TestOSAgentEventForwarding 核心：事件转发顺序 =
// os_agent start → 内层 read_file start/result → os_agent result，记录与事件均按序发射。
func TestOSAgentEventForwarding(t *testing.T) {
	innerCtx, records, emitted := newTestInnerCtx()
	oa := buildOSSubAgent(context.Background(), &openai.ChatModel{}, innerCtx)
	if oa == nil {
		t.Fatal("chatModel 非 nil 时应构造成功")
	}
	// 用 fake 内层 Agent 替换：预设事件流 = assistant(调 read_file) → tool(read_file 结果) → assistant(最终文本)
	oa.agent = &fakeAgent{
		name: "os_agent",
		desc: "test",
		events: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{{
				ID:       "call_1",
				Function: schema.FunctionCall{Name: "read_file", Arguments: `{"path":"a.txt"}`},
			}}), nil, schema.Assistant, ""),
			adk.EventFromMessage(schema.ToolMessage("文件内容：hello", "call_1", schema.WithToolName("read_file")), nil, schema.Tool, "read_file"),
			adk.EventFromMessage(schema.AssistantMessage("读取完成，内容为 hello", nil), nil, schema.Assistant, ""),
		},
	}

	// 模拟父层：os_agent start（父层 Run 循环对委托工具同样走 emitToolStart/emitToolResult）
	emitToolStart(innerCtx.Emit, innerCtx.Records, schema.ToolCall{
		ID:       "call_os",
		Function: schema.FunctionCall{Name: "os_agent", Arguments: `{"request":"读取 a.txt"}`},
	}, nil)

	out, err := oa.InvokableRun(context.Background(), `{"request":"读取 a.txt"}`)
	if err != nil {
		t.Fatalf("InvokableRun 失败: %v", err)
	}
	if out != "读取完成，内容为 hello" {
		t.Errorf("最终文本应为内层最后一条正文，实际: %q", out)
	}

	// 模拟父层：os_agent result
	emitToolResult(innerCtx.Emit, innerCtx.Records, "os_agent", "call_os", out)

	// 断言 toolRecords 顺序：os_agent start → read_file start → read_file result → os_agent result
	wantActions := []string{"tool_start", "tool_start", "tool_result", "tool_result"}
	wantNames := []string{"os_agent", "read_file", "read_file", "os_agent"}
	if len(*records) != len(wantActions) {
		t.Fatalf("toolRecords 应为 %d 条，实际 %d 条: %+v", len(wantActions), len(*records), *records)
	}
	for i, rec := range *records {
		if rec.Action != wantActions[i] {
			t.Errorf("记录[%d] action 应为 %q，实际 %q", i, wantActions[i], rec.Action)
		}
		if rec.Name != wantNames[i] {
			t.Errorf("记录[%d] name 应为 %q，实际 %q", i, wantNames[i], rec.Name)
		}
	}

	// 断言 ai:tool-status 事件按序发射（每条负载为 Record JSON，顺序与记录一致）
	if len(*emitted) != len(wantActions) {
		t.Fatalf("ai:tool-status 事件应为 %d 条，实际 %d 条", len(wantActions), len(*emitted))
	}
	for i, data := range *emitted {
		var rec tools.Record
		if err := json.Unmarshal([]byte(data), &rec); err != nil {
			t.Fatalf("事件[%d] 负载解析失败: %v", i, err)
		}
		if rec.Action != wantActions[i] || rec.Name != wantNames[i] {
			t.Errorf("事件[%d] 应为 %s(%s)，实际 %s(%s)", i, wantActions[i], wantNames[i], rec.Action, rec.Name)
		}
	}
}

// toolNamesOf 提取工具列表名称集合（测试辅助）。
func toolNamesOf(t *testing.T, toolList []tool.BaseTool) map[string]bool {
	t.Helper()
	names := make(map[string]bool, len(toolList))
	for _, tl := range toolList {
		info, err := tl.Info(context.Background())
		if err != nil || info == nil {
			t.Fatalf("工具 Info 失败: %v", err)
		}
		names[info.Name] = true
	}
	return names
}

// TestOSAgentTwoCallsForwarding 同一轮父模型多次调用 os_agent（不同 call_id）时，
// 事件转发顺序与 call_id 配对正确：两轮各自保持
// os_agent start → 内层 read_file start/result → os_agent result 的完整分组，
// 不跨轮错配（对应前端 osAgentGroupClose 按 call_id 配对的输入契约）。
func TestOSAgentTwoCallsForwarding(t *testing.T) {
	innerCtx, records, _ := newTestInnerCtx()
	oa := buildOSSubAgent(context.Background(), &openai.ChatModel{}, innerCtx)
	if oa == nil {
		t.Fatal("chatModel 非 nil 时应构造成功")
	}
	oa.agent = &fakeAgent{
		name: "os_agent",
		desc: "test",
		events: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{{
				ID:       "inner_1",
				Function: schema.FunctionCall{Name: "read_file", Arguments: `{"path":"a.txt"}`},
			}}), nil, schema.Assistant, ""),
			adk.EventFromMessage(schema.ToolMessage("内容A", "inner_1", schema.WithToolName("read_file")), nil, schema.Tool, "read_file"),
			adk.EventFromMessage(schema.AssistantMessage("完成A", nil), nil, schema.Assistant, ""),
		},
	}

	// 第一轮：os_agent start(call_os1) → 内层 read_file → os_agent result(call_os1)
	emitToolStart(innerCtx.Emit, innerCtx.Records, schema.ToolCall{
		ID:       "call_os1",
		Function: schema.FunctionCall{Name: "os_agent", Arguments: `{"request":"任务A"}`},
	}, nil)
	if _, err := oa.InvokableRun(context.Background(), `{"request":"任务A"}`); err != nil {
		t.Fatalf("第一轮 InvokableRun 失败: %v", err)
	}
	emitToolResult(innerCtx.Emit, innerCtx.Records, "os_agent", "call_os1", "完成A")

	// 第二轮：os_agent start(call_os2) → 内层 read_file → os_agent result(call_os2)
	emitToolStart(innerCtx.Emit, innerCtx.Records, schema.ToolCall{
		ID:       "call_os2",
		Function: schema.FunctionCall{Name: "os_agent", Arguments: `{"request":"任务B"}`},
	}, nil)
	if _, err := oa.InvokableRun(context.Background(), `{"request":"任务B"}`); err != nil {
		t.Fatalf("第二轮 InvokableRun 失败: %v", err)
	}
	emitToolResult(innerCtx.Emit, innerCtx.Records, "os_agent", "call_os2", "完成B")

	// 断言记录顺序与 call_id 配对（共 8 条，两轮完整分组、互不串扰）
	want := []struct {
		action string
		name   string
		callID string
	}{
		{"tool_start", "os_agent", "call_os1"},
		{"tool_start", "read_file", "inner_1"},
		{"tool_result", "read_file", "inner_1"},
		{"tool_result", "os_agent", "call_os1"},
		{"tool_start", "os_agent", "call_os2"},
		{"tool_start", "read_file", "inner_1"},
		{"tool_result", "read_file", "inner_1"},
		{"tool_result", "os_agent", "call_os2"},
	}
	if len(*records) != len(want) {
		t.Fatalf("toolRecords 应为 %d 条，实际 %d 条: %+v", len(want), len(*records), *records)
	}
	for i, rec := range *records {
		if rec.Action != want[i].action || rec.Name != want[i].name || rec.CallID != want[i].callID {
			t.Errorf("记录[%d] 应为 %s(%s, callID=%s)，实际 %s(%s, callID=%s)",
				i, want[i].action, want[i].name, want[i].callID, rec.Action, rec.Name, rec.CallID)
		}
	}
}

// TestOSAgentEventForwardingStreaming 内层事件流为流式（真实运行 EnableStreaming 的必经路径）时，
// 转发逻辑与顺序与非流式一致：assistant 流式合并 ToolCall、tool 流式合并结果、
// finalContent 取最后一条非工具正文。覆盖 subagent.go 中 mv.IsStreaming 分支
// （consumeAssistantStream/consumeToolStream + startedByCallID 配对）。
func TestOSAgentEventForwardingStreaming(t *testing.T) {
	innerCtx, records, emitted := newTestInnerCtx()
	oa := buildOSSubAgent(context.Background(), &openai.ChatModel{}, innerCtx)
	if oa == nil {
		t.Fatal("chatModel 非 nil 时应构造成功")
	}
	oa.agent = &fakeAgent{
		name: "os_agent",
		desc: "test",
		events: []*adk.AgentEvent{
			// assistant 流式：单个 chunk 携带 read_file 工具调用
			adk.EventFromMessage(nil, schema.StreamReaderFromArray([]*schema.Message{
				schema.AssistantMessage("", []schema.ToolCall{{
					ID:       "inner_1",
					Function: schema.FunctionCall{Name: "read_file", Arguments: `{"path":"a.txt"}`},
				}}),
			}), schema.Assistant, ""),
			// tool 流式：单个 chunk 携带 read_file 结果
			adk.EventFromMessage(nil, schema.StreamReaderFromArray([]*schema.Message{
				schema.ToolMessage("内容A", "inner_1", schema.WithToolName("read_file")),
			}), schema.Tool, "read_file"),
			// assistant 流式：最终正文
			adk.EventFromMessage(nil, schema.StreamReaderFromArray([]*schema.Message{
				schema.AssistantMessage("完成A", nil),
			}), schema.Assistant, ""),
		},
	}

	// 父层 os_agent start(call_os1) → 内层流式 read_file → os_agent result(call_os1)
	emitToolStart(innerCtx.Emit, innerCtx.Records, schema.ToolCall{
		ID:       "call_os1",
		Function: schema.FunctionCall{Name: "os_agent", Arguments: `{"request":"任务A"}`},
	}, nil)
	out, err := oa.InvokableRun(context.Background(), `{"request":"任务A"}`)
	if err != nil {
		t.Fatalf("InvokableRun 失败: %v", err)
	}
	if out != "完成A" {
		t.Errorf("最终文本应为内层最后一条正文，实际 %q", out)
	}
	emitToolResult(innerCtx.Emit, innerCtx.Records, "os_agent", "call_os1", out)

	// 断言记录顺序与 call_id 配对（与 TestOSAgentEventForwarding 非流式场景一致）
	want := []struct {
		action string
		name   string
		callID string
	}{
		{"tool_start", "os_agent", "call_os1"},
		{"tool_start", "read_file", "inner_1"},
		{"tool_result", "read_file", "inner_1"},
		{"tool_result", "os_agent", "call_os1"},
	}
	if len(*records) != len(want) {
		t.Fatalf("toolRecords 应为 %d 条，实际 %d 条: %+v", len(want), len(*records), *records)
	}
	for i, rec := range *records {
		if rec.Action != want[i].action || rec.Name != want[i].name || rec.CallID != want[i].callID {
			t.Errorf("记录[%d] 应为 %s(%s, callID=%s)，实际 %s(%s, callID=%s)",
				i, want[i].action, want[i].name, want[i].callID, rec.Action, rec.Name, rec.CallID)
		}
	}
	// 事件发射（ai:tool-status）数量与记录一致
	if len(*emitted) != len(want) {
		t.Fatalf("emitted 应为 %d 条，实际 %d 条: %v", len(want), len(*emitted), *emitted)
	}
}
