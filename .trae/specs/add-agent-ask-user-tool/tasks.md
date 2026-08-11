# Tasks

- [x] Task 1: 后端 ask_user 工具实现
  - 新增 `internal/agent/tools/ask_user.go`：
    - Schema：`question`（string 必填）、`options`（[]string，2-6 项）、`reason`（string 可选）
    - 执行：不执行业务；将 `{question, options, reason}` JSON 序列化，经 ctx.Emit 发射 `ai:ask-user` 事件（Agent 模式单参数形态：data=JSON 字符串）；返回问句文本给模型（"我需要向你确认：{question}，请从上方选项中选择或直接输入你的答案。"）
    - 实现 `ActionTextProvider.ActionText`：`向用户提问：{question}`（question 用 TruncateRunes 截断 30 字符）
    - 提供导出构造器 `NewAskUser(ctx *Context) tool.InvokableTool`
  - `internal/agent/registry.go` buildTools 追加注册：`tools.WrapWithError("ask_user", tools.NewAskUser(p.ctx), p.ctx)`

- [x] Task 2: agent.go 问句正文兜底
  - 在 `Run()` 事件消费循环中：当某轮 assistant 消息含 ToolCalls 且其中存在 `ask_user`、且该轮正文 Content 非空时，将其暂存为 `pendingQuestion`
  - 循环结束后：若 `finalContent == ""` 且 `pendingQuestion != ""`，则 `finalContent = pendingQuestion`（保证落库内容为问句，历史回放可读）
  - 不改变既有行为（其他工具调用轮的正文仍不显示、不落库）

- [x] Task 3: app.go Instruction 注入 ask_user 约束
  - `CallAIAgentStream` 组装 instruction 末尾追加约束文本（Agent 模式专用，仅 Agent 流生效）：
    - 仅在信息不足或需用户在方案间决策时调用 ask_user；一次只问一个问题；严禁无关澄清
    - 调用 ask_user 前先在正文完整写出问题；调用后立即停止生成，不输出任何其他内容
    - 用户回答后（新 user 消息）继续正常回答

- [x] Task 4: 前端 ai-chat.js 事件监听 + 卡片渲染 + 选择发送
  - `startStreaming` 顶部事件清理列表（`['ai:stream-done', ...].forEach(EventsOff)`）追加 `'ai:ask-user'`
  - 注册 `ai:ask-user` 事件监听（Agent 模式单参数 data=JSON，解析 `{question, options}`），在 streamingEl 内 contentDiv 之后渲染问题卡片
  - 卡片结构：问句标题 + 选项按钮列表（options 非空时）+ 自定义输入框与提交按钮
  - 点击选项 / 提交自定义输入：复用发送流程（保存 user 消息 `SaveAIMessage` → `addMessage` 渲染用户气泡 → `startStreaming(text, false, userMsgID)`），输入内容作为新 user 消息发送；发送前校验 isStreaming 与 AI 配置（与 onSend 一致，可抽取公共函数 `sendUserText(text)` 供 onSend 与卡片共用）
  - 卡片保留在气泡中不随流结束移除；`ai:stream-done` 的 `contentDiv.innerHTML=''` 仅清正文，不影响卡片

- [x] Task 5: 前端 ai-chat.css 问题卡片样式
  - 新增 `.ai-ask-card` 系列样式：卡片容器（`--card-bg` 背景、圆角、1px 边框）、问句标题、选项按钮（hover/active 态用 `--accent`）、自定义输入行（输入框 + 提交按钮）
  - 主题自适应（复用现有 CSS 变量），支持 `prefers-reduced-motion`（无强动画，最多淡入）

- [x] Task 6: 文档更新
  - `internal/agent/doc.go`：工具清单只读/交互分类补 `ask_user`
  - `internal/agent/tools/doc.go`：工具实现列表补 `ask_user` + `NewAskUser`
  - `internal/agent/TOOLS.md`：如含工具枚举/说明，补 ask_user 一行（含交互语义说明）
  - `AGENTS.md`：按滚动窗口规范追加记忆点（ask_user 反向提问 + ai:ask-user 事件 + 问句兜底设计）

- [x] Task 7: 构建与验证
  - `go build ./...` 通过
  - 前端 `npm run build` 通过（Vite 语法检查）
  - 检查前端事件名清理列表与监听注册一致；确认无 ESLint/语法错误

# Task Dependencies
- [Task 2] 依赖 [Task 1]（工具名 ask_user 定稿、事件名 `ai:ask-user` 定稿）
- [Task 3] 依赖 [Task 1]（工具语义定稿）
- [Task 4] 依赖 [Task 1]（事件名与负载结构定稿）；可与 [Task 2]/[Task 3] 并行
- [Task 5] 依赖 [Task 4] 的 DOM 结构约定；可与后端任务并行（样式先行）
- [Task 6] 依赖 [Task 1]（工具清单）与 [Task 2]（设计要点）
- [Task 7] 依赖 [Task 1]-[Task 6]
