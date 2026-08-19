# AI 聊天用户消息 Meta Chip 显示 Spec

## Why

当前 AI 聊天的用户消息气泡只显示用户键入的纯文本，丢失了"当时用了哪些引用笔记 / 上传了哪些文件 / 激活了哪些技能"这一关键上下文。回看历史时无法还原操作现场。需要在不污染 LLM 输入的前提下，把这些"当时做了什么"在历史消息气泡中以 chip 形式可视化呈现。

## What Changes

* 在 `AIMessage` 表新增 `Meta` 字段（TEXT，存 JSON 数组），存每个用户消息发送时附带的引用笔记 / 上传文件 / 技能信息

* 后端 `App.SaveAIMessage` Wails 绑定签名新增 `meta string` 参数

* 前端 `sendUserText` 在调用 `SaveAIMessage` 时把 `referencedNotes` / `uploadedFiles` / `activeSkills` 序列化为 JSON 一并传入

* 前端 `addMessage` 用户分支改为：原文本段用 `textContent` 写入，meta 段解析为真实 DOM chip 元素（图标 + 标签）追加到文本后

* 新增 CSS `.ai-msg-user .ai-msg-chip` 样式（适配 accent 底色 + 白字）

* 编辑 / 复制 / 保存为笔记 / 重新生成 等操作无需特殊处理（Content 永远保持纯文本，Meta 字段独立存储不影响 LLM 上下文）

* 历史消息无 Meta 字段（NULL）时，渲染行为与现状完全一致

## Impact

* Affected specs:

  * 与 `add-ai-assistant`、`add-card-recall`、`add-ai-chat-file-upload`、`add-ai-skill-*` 等"AI 助手"系列 spec 关联，本次新增依赖其已有能力

* Affected code:

  * `internal/models/ai_message.go` — 新增 `Meta` 字段

  * `internal/services/ai_service.go` — `Message` 结构体新增 `Meta`，`SaveAIMessage` 透传

  * `app.go` — `App.SaveAIMessage` Wails 绑定增加 `meta` 参数

  * `frontend/src/js/ai-chat.js` — `sendUserText` 拼装 meta JSON；`addMessage` 用户分支改用 `renderUserMessageWithChips`

  * `frontend/src/css/components/ai-chat.css` — 新增 `.ai-msg-user .ai-msg-chip` 等样式

## ADDED Requirements

### Requirement: AIMessage 持久化 Meta 信息

系统 SHALL 在 `AIMessage` 表中持久化用户消息的附加上下文（引用笔记 / 上传文件 / 激活技能），且不污染发送给 LLM 的 `Content` 字段。

#### Scenario: 发送带引用的消息

* **WHEN** 用户在 `referencedNotes` 中有 1 条笔记、`uploadedFiles` 中有 1 个文件、`activeSkills` 中激活了 1 个技能时点击发送

* **THEN** 后端 `AIMessage` 行的 `Content` 字段仅存用户键入的纯文本

* **AND** `Meta` 字段存 JSON 数组 `[{type:"ref",id,title,notebook}, {type:"file",name,truncated}, {type:"skill",id,label}]`

* **AND** LLM 收到的用户消息中不包含 Meta 任何字符

#### Scenario: 重新生成历史消息

* **WHEN** 用户对历史某条带 Meta 的用户消息点击"重新生成"

* **THEN** 后端从 DB 读取的 `Content` 字段为纯文本，Meta 字段不会被发送给 LLM

* **AND** AI 回复内容中不会回显 `[{` 之类的 JSON 片段

#### Scenario: 加载历史会话
- **WHEN** 切换到历史会话并加载消息列表
- **THEN** 每条 user 消息如带 Meta 字段，气泡末尾追加渲染对应类型的 chip
- **AND** Meta 为 NULL 或空数组的旧消息气泡保持纯文本（与现状一致）

#### Scenario: 编辑用户消息
- **WHEN** 用户右键带 Meta 的消息选择"编辑"并保存
- **THEN** textarea 只填充 `Content` 纯文本，不显示 Meta 相关内容
- **AND** 保存时重新从当前 `referencedNotes` / `uploadedFiles` / `activeSkills` 工具栏状态派生新的 Meta
- **AND** 当前工具栏有引用/文件/技能 → DB 中该消息的 Meta 字段更新为新派生的 JSON
- **AND** 当前工具栏为空 → DB 中该消息的 Meta 字段被清空（空字符串或空数组）

#### Scenario: 重新生成 AI 回复
- **WHEN** 用户对某条带 Meta 的用户消息点击"重新生成"
- **THEN** 重新从当前工具栏状态派生 Meta 并更新 DB 中该用户消息的 Meta 字段
- **AND** 若当前工具栏为空 → Meta 字段被清空
- **AND** AI 重新生成时 LLM 拿到的上下文（refs/files/skills 参数）与最新 Meta 一致

#### Scenario: 无附加上下文的消息
- **WHEN** 用户没有引用笔记、未上传文件、未激活技能时发送消息
- **THEN** `Meta` 字段存 `""` 或 `[]`
- **AND** 气泡渲染时不追加任何 chip，行为与现状完全一致

### Requirement: 用户消息气泡渲染 Meta Chip

前端 SHALL 在用户消息气泡内，将 Meta 数组解析为对应的 chip DOM 元素并追加到纯文本之后。

#### Scenario: 渲染 ref chip

* **WHEN** Meta 包含 `{type:"ref", title:"Golang 学习笔记", notebook:"技术"}`

* **THEN** 气泡末尾追加一个 chip DOM，结构为 `<span class="ai-msg-chip ai-msg-chip-ref"><svg/><span class="ai-msg-chip-label">Golang 学习笔记</span></span>`

* **AND** chip 在 `.ai-msg-user`（accent 底 + 白字）的视觉下显示为半透明白色圆角胶囊

#### Scenario: 渲染 file chip

* **WHEN** Meta 包含 `{type:"file", name:"需求.pdf", truncated:false}`

* **THEN** 气泡末尾追加一个 file 类型 chip，显示名称"需求.pdf"

* **AND** `truncated:true` 时 chip 角标显示"截断"提示

#### Scenario: 渲染 skill chip

* **WHEN** Meta 包含 `{type:"skill", id:"coding", label:"编程开发"}`

* **THEN** 气泡末尾追加一个 skill 类型 chip，显示标签"编程开发"

#### Scenario: 渲染多 chip

* **WHEN** Meta 数组长度 > 1

* **THEN** 多个 chip 在气泡末尾横向排列，过长自动折行（受 `white-space` 控制）

* **AND** 文本段与 chip 段之间有视觉间距（CSS 控制）

### Requirement: LLM 上下文零污染

系统 SHALL 保证 LLM 永远只看到用户的纯文本输入。

#### Scenario: 文本段 XSS 安全

* **WHEN** 渲染用户消息文本段

* **THEN** 使用 `document.createTextNode` 写入（自动转义）

* **AND** 禁止 `innerHTML` 直接拼用户输入

#### Scenario: Meta 字段不流向 LLM

* **WHEN** 任何路径调用 LLM（新发 / 重新生成 / 多轮历史加载）

* **THEN** LLM 输入的消息列表中每条 user 消息的 `Content` 字段都不包含 Meta JSON 内容

* **AND** 验证方式：日志记录 LLM 输入 + 多轮对话测试不出现 `[{type` 片段

## MODIFIED Requirements

无（本功能为新增，不修改现有需求）

## REMOVED Requirements

无

## Migration

* GORM AutoMigrate 自动给 `AIMessage` 表加 `Meta TEXT` 列（旧行 NULL，零破坏）

* 旧消息 `Meta` 为 NULL → 渲染跳过 chip，行为完全等同现状

* 旧消息不补回 Meta（按项目约定，新能力向前兼容，不回填历史）

* `App.SaveAIMessage` Wails 绑定增加第 4 个参数 `meta string`，前端为唯一调用方，调用处同步加参数即可

