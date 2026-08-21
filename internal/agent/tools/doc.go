// Package tools 提供 Agent 工具实现与共享上下文。
//
// 职责：
//   - 工具实现（read_url / recall_notes / summarize_text /
//     get_current_time / json_validate / json_format / json_extract / manage_todo /
//     manage_notebook / manage_tag / manage_note / read_note_section / get_stats / ask_user）每文件一个，
//     均提供导出构造器（NewReadURL / NewRecallNotes / NewSummarizeText /
//     NewGetCurrentTime / MustJSONValidate / MustJSONFormat / MustJSONExtract / NewManageTodo /
//     NewManageNotebook / NewManageTag / NewManageNote / NewReadNoteSection / NewGetStats / NewAskUser），
//     由父包 agent 的 registry.go 统一装配与注册。
//   - 共享上下文类型（EmitFn / Record / Collector / Context / WrapWithError）定义于
//     context.go：工具通过注入的 Context 发射事件、登记调用记录、收集结构化结果
//     （搜索来源 / 召回卡片）与日志，通过 WrapWithError 统一包装失败行为。
//   - 本子包不感知父包 agent 的事件循环细节，也不 import 父包（避免循环依赖）。
//   - 工具中文展示文案以 meta.go 的 BuiltinTools 为权威来源。
package tools
