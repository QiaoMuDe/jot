package agent

// 本文件定义 internal/agent 模块对外暴露的数据契约。
// 模块职责：基于 cloudwego/eino 的 ChatModelAgent（ReAct 循环）实现 Agent 对话链路，
// 内置本地笔记召回（recall_notes）、读取、管理与交互等工具（联网搜索由 MCP 服务器工具提供），
// 通过事件回调把流式文本与工具状态实时推给调用方（app.go 包装 runtime.EventsEmit）。
// 工具实现在 tools 子包，本文件通过类型别名复用其 EmitFn / Record / Collector 契约。

import "jot/internal/agent/tools"

// Request 一轮 Agent 对话的入参。
// Instruction 由调用方（app.go）组装：baseSystemPrompt + 技能提示词 + 角色扮演 + 引用笔记等，
// 模块内直接透传给 Agent 作为 system 消息，不再重复拼接。
type Request struct {
	SessionID         uint
	UserText          string
	History           []HistoryMessage // 截断后的历史消息（不含当前用户消息）
	Instruction       string           // 系统提示词全文（由调用方组装）
	ThinkingEnabled   bool             // 深度思考开关：开启时 ChatModel 配置 ReasoningEffort=high
	SkillIDs          []string
	RecallNotebookIDs []uint // recall_notes 工具限定的笔记本过滤（构造工具时绑定）
	ReferencedNotes   string // 保留字段：手动引用笔记（已并入 Instruction）
	RoleplayNotes     string // 保留字段：角色扮演设定（已并入 Instruction）
	UserMsgID         uint
	DisabledTools     []string // 禁用工具名集合，装配时按此过滤（注册级，被禁工具模型不可见）
	PlanMode          bool     // Plan 模式：true 时注册 create_plan/update_plan 工具并注入计划约束
}

// ToolMeta 内置工具清单条目（供前端工具清单展示）。
// Enabled 表示当前是否启用（默认全部启用，按禁用集合过滤）。
// PlanOnly 标记该工具仅在 Plan 模式下可用。
// AlwaysOn 标记该工具为常驻/不可禁用，前端需置灰勾选并展示说明。
type ToolMeta struct {
	Name     string // 英文工具名
	Label    string // 一行中文说明
	Enabled  bool   // 当前是否启用
	PlanOnly bool   // 仅 Plan 模式可用
	AlwaysOn bool   // 常驻/不可禁用
}

// HistoryMessage 一条历史消息（DB 中的 user/assistant 消息，已由调用方截断）。
type HistoryMessage struct {
	Role    string // "user" / "assistant"
	Content string
}

// Result 一轮 Agent 对话的结果，供调用方落库（写入 ai_messages 表）。
// SearchSources / RecallCards 分别为搜索类工具、recall_notes 工具执行时经 tools.Collector
// 收集的结构化来源与召回卡片（与历史 chat 消息存 search_sources / recall_cards 的格式一致），
// ToolCalls 为完整工具调用链摘要（tools.Record 序列化：action/name/args/result）。
type Result struct {
	Content          string  // 最终回答全文
	SearchSources    string  // 联网搜索来源 JSON（tools.Collector.Sources）
	RecallCards      string  // 召回卡片 JSON（tools.Collector.Cards）
	ToolCalls        string  // 工具调用链 JSON（[]tools.Record）
	Plan             string  // 执行计划 JSON（*tools.Plan，未使用规划工具时为空）
	ReasoningContent string  // 深度思考链全文（流式 reasoning_content 拼接，供落库与展示）
	ThinkingElapsed  float64 // 思考净时长（秒）：按每轮 assistant 消息独立计时并累加，排除工具执行时间
	PromptTokens     int     // 全部 ReAct 轮次的真实输入 token 累计（provider usage.PromptTokens 求和）
	CompletionTokens int     // 全部 ReAct 轮次的真实输出 token 累计（provider usage.CompletionTokens 求和）
}

// EmitFn 事件回调，由调用方注入（内部封装 runtime.EventsEmit）。
// event 为事件名（如 "ai:stream-chunk" / "ai:tool-status"），data 为事件负载。
type EmitFn = tools.EmitFn
