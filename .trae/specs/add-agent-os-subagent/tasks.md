# Tasks

## 阶段一：后端子 Agent 装配
- [x] Task 1: 新增 `internal/agent/subagent.go` 子 Agent 逻辑文件
  - [x] 子 Agent 逻辑与定义集中于此文件编写：`osSubAgentMaxIterations = 20` 常量、内层系统提示词（OS 任务角色 + workspace 边界 + 审批说明 + 收尾摘要要求）
  - [x] 内层 Agent 构造器：`buildOSSubAgent(runCtx, chatModel, innerCtx, disabled)` 用 `adk.NewChatModelAgent` 装配 11 个文件/命令工具白名单，复用会话 ChatModel 客户端，`MaxIterations` 独立
  - [x] `osAgentTool.InvokableRun`：构建内层 Runner（EnableStreaming），消费内层事件——内层工具 start/result/error 经 `emitToolStart`/`emitToolResult` 语义写入父层 `toolRecords` 并发射 `ai:tool-status`（用内层独立 toolByName 映射）；内层流式正文/思考链不转发；返回内层最终摘要文本
  - [x] 内层 `tools.Context` 复用父层 `Emit`/`Records`，注入同一会话 `Approver`/`AskWaiter`
- [x] Task 2: `registry.go`/`agent.go` 装配接线
  - [x] `buildTools` 移除 11 个文件/命令工具条目；新增 `os_agent` 注册（受 `disabled` 过滤、非 PlanOnly）
  - [x] `Run` 中构造 os_agent：chatModel 就绪后构建内层 Context 与子 Agent，追加进 toolList（构造失败不影响其他工具装配，记 Warn）
  - [x] `agent.Request.DisabledTools` 透传不变；`os_agent` 名进入禁用集合即不注册

## 阶段二：元数据与文档
- [x] Task 3: 工具清单与文档收敛
  - [x] `tools/meta.go`：移除 11 个文件/命令条目，新增 `{Name: "os_agent", Label: "操作系统任务（工作区文件读写与命令执行）"}`（普通组）
  - [x] `tools/doc.go`：工具列表同步（移除 11 个，新增 os_agent）
  - [x] `TOOLS.md`：更新工具清单说明、子 Agent 内层工具白名单与边界、审批语义沿用
  - [x] `EVENTS.md`：补充"内层步骤经 ai:tool-status 实时下发、记录顺序与分组边界"说明

## 阶段三：后端测试
- [x] Task 4: 单元测试
  - [x] 内层装配：白名单恰好 11 工具、禁用过滤生效、`os_agent` 禁用后父层无文件能力
  - [x] 委托工具事件转发：用 stub/fake 内层 Agent（实现 `adk.Agent` 接口返回预设事件流）驱动 `InvokableRun`，断言 toolRecords 顺序 = os_agent start → 内层 start/result → os_agent result，且 `ai:tool-status` 事件按序发射
  - [x] 审批共享：内层工具注入的 `Approver` 与会话实例一致（直接断言指针或行为）
  - [x] `go build ./... && go vet ./... && go test ./internal/agent/...` 全绿

## 阶段四：前端渲染
- [x] Task 5: 前端工具列表收敛核查
  - [x] 排查 `main.js`/`ai-chat.js` 中硬编码的 11 个旧工具名引用（如特殊样式/文案），消除残留
  - [x] 验证设置页与 AI 助手下拉自动显示单条 `os_agent`（数据源 `GetAgentTools()` 自动生效），勾选禁用/全选/计数逻辑正常
- [x] Task 6: 子步骤实时渲染 + 执行中动画
  - [x] `os_agent` tool_start：主行 + 执行中动画（脉动点/齿轮，新 CSS class）
  - [x] 内层记录渲染为缩进子步骤（引导线、弱化色、压缩行高），按 `os_agent` call_id 打开/关闭分组（同一轮多次调用正确嵌套）
  - [x] 历史回放（buildToolRecords）与实时共用同一渲染函数族，结构完全一致
  - [x] `ai-chat.css` 新增 `.is-substep`/执行中动画等样式，并覆盖深色主题

## 阶段五：验收
- [x] Task 7: 构建与验收
  - [x] `npm run build` 通过；`wails build`（或 dev）验证
  - [x] 手动验收：Agent 对话中让子 Agent 执行多步文件任务（写文件→读回→执行命令），确认实时子步骤动画、审批弹窗、完成摘要、历史回放一致
  - [x] 设置页禁用 `os_agent` 后模型不再具备文件能力；旧禁用残留（如 `["run_command"]`）保存后清理
  - [x] AGENTS.md 记忆点收尾（临时记忆 + 长期记忆工具族条目更新）

# Task Dependencies
- Task 1 ← Task 2（装配接线依赖委托工具实现）
- Task 3 独立可与 Task 1/2 并行（仅文档/元数据）
- Task 4 依赖 Task 1/2（测试对象就绪）
- Task 5/6 依赖 Task 2（后端装配完成后方可前端联调），Task 5 可先于 Task 6
- Task 7 依赖 Task 1,2,3,4,5,6
- 建议执行顺序：Task 1 → Task 2 → Task 4（后端闭环）→ Task 3（可与后端并行）→ Task 5 → Task 6 → Task 7
