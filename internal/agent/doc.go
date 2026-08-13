// Package agent 提供基于 cloudwego/eino 的 Agent 对话链路（ReAct 循环）。
//
// 职责：
//   - AgentService.Run：组装 ChatModelAgent（OpenAI 兼容协议，配置复用现有 AI 设置）、
//     统一注册工具（只读：web_search / read_url / recall_notes / refine_search_query / get_stats /
//     get_current_time / read_note_section；交互：ask_user；写操作：manage_todo / manage_notebook /
//     manage_tag / manage_note）、
//     消费事件流，
//     通过 EmitFn 实时推送流式文本与工具状态，返回最终回答与工具调用摘要供调用方落库。
//   - 事件通过回调透出（不依赖 Wails runtime），调用方（app.go）包装 runtime.EventsEmit。
//
// 结构说明：
//   - 工具实现在 tools/ 子包（每文件一个工具 + 导出构造器），经 tools.WrapWithError
//     包装（失败回填模型不中断循环），部分失败由工具经 tools.Context 登记、
//     Run 在 tool_result 之后统一 DrainPartials 发射 tool_partial 事件。
//   - 父包 registry.go 的 buildTools 统一装配与注册全部工具（新增工具只需：
//     1) 在 tools/ 子包新增工具文件与导出构造器；2) 在 buildTools 追加一行注册；
//     3) 若引入新服务则扩展 Deps。无需改动 Run() 的事件消费逻辑）。
//
// 未来扩展点：
//   - 新增写操作工具（如笔记更新 / 删除类操作）按上述步骤在 tools/ 子包实现并注册即可；
//   - 多 Agent 编排（子 Agent、工具 Agent）可基于 adk 的 AgentTool / DeepAgent 扩展；
//   - 记忆 / 会话上下文可在 Request 中扩展字段，由调用方组装进 Instruction。
//   - 外部 MCP 服务器工具（测试阶段，数据库驱动）：internal/mcpserver 包从数据库读取
//     MCP 服务器配置（mcpserver.LoadFromDB），连接 enabled 服务器并发现工具（改名 mcp_{服务器名}_{工具名}），
//     在 Run() 中并入 toolList 并同样经 WrapWithError 包装；配置缺失 / 单服务器失败仅记日志跳过。
package agent
