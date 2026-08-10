// Package agent 提供基于 cloudwego/eino 的 Agent 对话链路（ReAct 循环）。
//
// 职责：
//   - AgentService.Run：组装 ChatModelAgent（OpenAI 兼容协议，配置复用现有 AI 设置）、
//     注入只读工具（web_search / recall_notes）、消费事件流，通过 EmitFn 实时推送
//     流式文本与工具状态，返回最终回答与工具调用摘要供调用方落库。
//   - 事件通过回调透出（不依赖 Wails runtime），调用方（app.go）包装 runtime.EventsEmit。
//
// 未来扩展点：
//   - 新增写操作工具（创建 / 保存笔记）时在 tools.go 注册即可，无需改动事件消费逻辑；
//   - 多 Agent 编排（子 Agent、工具 Agent）可基于 adk 的 AgentTool / DeepAgent 扩展；
//   - 记忆 / 会话上下文可在 Request 中扩展字段，由调用方组装进 Instruction。
package agent
