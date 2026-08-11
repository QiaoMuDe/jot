// Package tools 提供 Agent 工具实现与共享上下文。
//
// 职责：
//   - 工具实现（web_search / recall_notes / refine_search_query / get_current_time / manage_todo /
//     manage_notebook / manage_tag / manage_note / get_stats）每文件一个，
//     均提供导出构造器（NewWebSearch / NewRecallNotes / NewRefineSearchQuery / NewGetCurrentTime /
//     NewManageTodo / NewManageNotebook / NewManageTag / NewManageNote / NewGetStats），
//     由父包 agent 的 registry.go 统一装配与注册。
//   - 共享上下文类型（EmitFn / Record / Collector / Context / WrapWithError）定义于
//     context.go：工具通过注入的 Context 发射事件、登记调用记录、收集结构化结果
//     （搜索来源 / 召回卡片）与日志，通过 WrapWithError 统一包装失败行为。
//   - 本子包不感知父包 agent 的事件循环细节，也不 import 父包（避免循环依赖）。
package tools
