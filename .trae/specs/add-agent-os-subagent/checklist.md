# Checklist

- [x] 新增 `internal/agent/subagent.go` 子 Agent 逻辑文件：os_agent 委托工具存在，内层为独立 ChatModelAgent，白名单恰为 11 个文件/命令工具，MaxIterations 独立（20），内层 Context 复用父层 Emit/Records 且注入同一会话 Approver/AskWaiter
- [x] 内层系统提示词约束"仅操作 ~/.jot/workspace、文件之外诉求回告主 Agent、危险操作需审批、收尾给摘要"
- [x] 父层 buildTools 不再注册 11 个文件/命令工具，仅注册 os_agent；os_agent 受 ai_agent_tools_disabled 过滤且非 PlanOnly
- [x] os_agent 被禁用后，模型不可见任何文件/命令能力
- [x] 内层每步工具调用以 ai:tool-status 实时下发，toolRecords 顺序 = os_agent start → 内层 start/result/error → os_agent result
- [x] 内层流式正文/思考链未转发（主消息气泡不被污染）
- [x] 子 Agent 内层审批弹窗照常工作（工具名 + 完整命令；批准继续、拒绝回填继续推理）
- [x] BuiltinTools() 移除 11 个文件/命令条目、新增单条 os_agent（Label 中文说明）；doc.go/TOOLS.md/EVENTS.md 已同步
- [x] 设置页与 AI 助手下拉的工具列表仅显示单条 os_agent；勾选禁用/全选/计数正常（注：旧禁用残留清理逻辑已按用户决策删除（2026-09-17），旧禁用名由后端静默忽略，该项验收不适用）
- [x] 前端 os_agent 行有执行中动画；内层记录以缩进子步骤渲染（引导线/弱化色/压缩行高）；按 call_id 分组边界正确（同轮多次调用正确嵌套）
- [x] 历史回放与实时渲染结构一致（共用渲染函数族）
- [x] 后端单测覆盖内层装配/事件转发/审批共享；go build、go vet、go test 全绿
- [ ] npm run build 通过（已确认）；手动验收（写文件→读回→执行命令→审批→回放）**待用户实操**；AGENTS.md 记忆点已更新
