package agent

// 本文件定义 internal/agent 模块对外暴露的数据契约。
// 模块职责：基于 cloudwego/eino 的 ChatModelAgent（ReAct 循环）实现 Agent 对话链路，
// 内置 web_search（多源联网搜索）与 recall_notes（本地笔记向量召回）两个只读工具，
// 通过事件回调把流式文本与工具状态实时推给调用方（app.go 包装 runtime.EventsEmit）。

import "jot/internal/services"

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
}

// HistoryMessage 一条历史消息（DB 中的 user/assistant 消息，已由调用方截断）。
type HistoryMessage struct {
	Role    string // "user" / "assistant"
	Content string
}

// Result 一轮 Agent 对话的结果，供调用方落库（写入 ai_messages 表）。
// SearchSources / RecallCards 分别为 web_search、recall_notes 工具执行时收集的
// 结构化来源与召回卡片（与问答模式存 search_sources / recall_cards 的格式一致），
// ToolCalls 为完整工具调用链摘要（action/name/args/result）。
type Result struct {
	Content          string  // 最终回答全文
	SearchSources    string  // 联网搜索来源 JSON（[]services.SearchSource）
	RecallCards      string  // 召回卡片 JSON（[]services.RecallCard）
	ToolCalls        string  // 工具调用链 JSON（[]toolCallRecord）
	ReasoningContent string  // 深度思考链全文（流式 reasoning_content 拼接，供落库与展示）
	ThinkingElapsed  float64 // 思考净时长（秒）：按每轮 assistant 消息独立计时并累加，排除工具执行时间
	PromptTokens     int     // 全部 ReAct 轮次的真实输入 token 累计（provider usage.PromptTokens 求和）
	CompletionTokens int     // 全部 ReAct 轮次的真实输出 token 累计（provider usage.CompletionTokens 求和）
}

// EmitFn 事件回调，由调用方注入（内部封装 runtime.EventsEmit）。
// event 为事件名（如 "ai:stream-chunk" / "ai:tool-status"），data 为事件负载。
type EmitFn func(event string, data string)

// toolCallRecord 工具调用记录，用于组装 Result.ToolCalls 摘要 JSON。
type toolCallRecord struct {
	Action string `json:"action"`           // "tool_start" / "tool_result" / "tool_error" / "tool_partial"
	Name   string `json:"name"`             // 工具名
	Args   string `json:"args,omitempty"`   // 工具调用参数（JSON，截断后）
	Result string `json:"result,omitempty"` // 工具返回结果摘要（截断后）
}

// sourceError 记录 web_search 单个来源的失败信息，用于部分失败提示。
type sourceError struct {
	Source string // 来源标识（tavily / zhihu_search / zhihu_global）
	Err    string // 失败原因
}

// resultCollector 收集一轮 Agent 对话中工具执行产生的结构化结果：
// web_search 的来源列表与 recall_notes 的召回卡片，供 Run 结束时汇总进 Result。
type resultCollector struct {
	Sources []services.SearchSource
	Cards   []services.RecallCard
	// SourceErrors web_search 部分来源失败信息（agent.go 在工具结果到达时读取并清空，
	// 用于发射 tool_partial 事件；全部来源失败时仍走工具 error 路径，不依赖此字段）
	SourceErrors []sourceError
}
