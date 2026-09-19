// Package tools 提供 Agent 工具实现与共享上下文。
//
// 职责：
//   - 工具实现（read_url / http_request / recall_notes /
//     json_validate / json_format / json_extract / manage_todo /
//     manage_notebook / manage_tag / manage_memory / browse_notes / manage_note / get_stats / ask_user /
//     create_plan / update_plan）每文件一个，
//     均提供导出构造器（NewReadURL / NewHTTP / NewRecallNotes /
//     MustJSONValidate / MustJSONFormat / MustJSONExtract / NewManageTodo /
//     NewManageNotebook / NewManageTag / NewManageMemory / NewBrowseNotes / NewManageNote / NewGetStats / NewAskUser /
//     NewCreatePlan / NewUpdatePlan），
//     由父包 agent 的 registry.go 统一装配与注册。
//   - os_agent 为委托工具（实例实现于 agent/subagent_os.go，通用机制见 agent/subagent.go）：把 read_file / write_file / edit_file /
//     ls_dir / glob / grep_file / copy_item / move_item / delete_item / mkdir_dir / run_command / run_python / transfer_item
//     这 13 个文件/命令工具封装为内层子 Agent，父层仅注册 os_agent 一个入口，内层工具构造器
//     （NewReadFile / NewWriteFile / NewEditFile / NewLsDir / NewGlob / NewGrepFile /
//     NewCopyItem / NewMoveItem / NewDeleteItem / NewMkdirDir / NewRunCommand / NewRunPython / NewTransferItem）由 buildOSSubAgent 装配。
//   - 共享上下文类型（EmitFn / Record / Collector / Context / WrapWithError）定义于
//     context.go：工具通过注入的 Context 发射事件、登记调用记录、收集结构化结果
//     （搜索来源 / 召回卡片）与日志，通过 WrapWithError 统一包装失败行为。
//   - 本子包不感知父包 agent 的事件循环细节，也不 import 父包（避免循环依赖）。
//   - 工具中文展示文案以 meta.go 的 BuiltinTools 为权威来源。
package tools
