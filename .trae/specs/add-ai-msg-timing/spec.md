# AI 回复耗时记录 Spec

## Why

AI 回复目前没有耗时展示，用户无法感知回复耗时（尤其是深度思考模型的思考时长），缺少直观的反馈体验。
记录两个独立耗时字段（思维链耗时 + 总耗时）并持久化到数据库，历史消息也能查看。

## What Changes

### 后端 — 数据模型
- `AIMessage` 模型新增 `thinking_elapsed`（思维链耗时，float64，秒）和 `total_elapsed`（总耗时，float64，秒）字段
- `services.Message` 结构体同步新增这两个字段
- `CallAIStream` 的 `onDone` 回调增加 `elapsedThinking float64, elapsedTotal float64` 参数
- `SaveAIMessages` 方法写入这两个字段
- `LoadAISessionMessages` 返回这两个字段

### 后端 — app.go
- `CallAIStream` 绑定方法中：记录流开始时间、首个 reasoning_content 时间、首个 content 时间；计算两个耗时后传给 `onDone`；新增 `ai:stream-done` 事件携带耗时参数

### 前端 — 思维链耗时
- 流式完成时，思维链 summary 文字从 `💭 已思考` 变为 `💭 已思考 X.X 秒`
- 加载历史消息时从数据库读取 `thinking_elapsed` 显示

### 前端 — 回复耗时
- 在每条 AI 回复消息内容下方添加耗时标签 `⏱ 总耗时 X.X 秒`
- 加载历史消息时从数据库读取 `total_elapsed` 显示

## Impact

- Affected specs: AI 助手 — 消息渲染 + 数据持久化
- Affected code:
  - Added: `internal/models/ai_message.go` 新增字段
  - Modified: `internal/services/ai_service.go`（Message 结构体 + CallAIStream 回调 + SaveAILoad）
  - Modified: `app.go`（CallAIStream 绑定 + onDone 参数变更）
  - Modified: `frontend/src/js/ai-chat.js`（startStreaming、addMessage）
  - Modified: `frontend/src/css/components/ai-chat.css`（新增耗时显示样式）

## ADDED Requirements

### Requirement: 数据模型

#### Scenario: 存储耗时字段
- **WHEN** AI 流式输出结束并保存消息到数据库
- **THEN** `AIMessage` 表中 `thinking_elapsed` 和 `total_elapsed` 字段被写入对应值
- **THEN** 两个字段类型为 `REAL`（float64），可空，默认为 0

### Requirement: 后端计时逻辑

#### Scenario: 正常完成流式输出时计算耗时
- **WHEN** `CallAIStream` 开始执行
- **THEN** 记录 `streamStart = time.Now()`
- **THEN** 首次收到 `reasoning_content` 时记录 `thinkingStart = time.Now()`
- **THEN** 首次收到 `content` 时记录 `thinkingEnd = time.Now()`（若之前有 thinking 数据）
- **THEN** 流式结束时计算：
  - `thinkingElapsed = thinkingEnd - thinkingStart`（若无 thinking 则为 0）
  - `totalElapsed = time.Since(streamStart)`
- **THEN** 两个耗时通过 `onDone` 回调传给前端

#### Scenario: 深度思考模型 — thinking_elapsed 计算
- **WHEN** 模型返回了 `reasoning_content` 且后续返回了 `content`
- **THEN** `thinkingElapsed = thinkingEnd.Sub(thinkingStart).Seconds()`
- **THEN** `thinkingElapsed` 保留一位小数

#### Scenario: 非深度思考模型
- **WHEN** 模型没有返回 `reasoning_content`
- **THEN** `thinkingElapsed = 0`

#### Scenario: 流式被取消
- **WHEN** 用户点击停止按钮取消流式
- **THEN** 不保存消息，不计算耗时

### Requirement: 思维链摘要显示耗时

#### Scenario: 流式输出完成时更新思考耗时
- **WHEN** AI 流式输出结束（`ai:stream-done` 事件触发，含耗时参数）
- **THEN** 思维链 `<summary>` 文字变为 `💭 已思考 X.X 秒`
- **THEN** 仅在 `thinkingElapsed > 0` 时更新

#### Scenario: 加载历史消息时显示存储的耗时
- **WHEN** 从数据库加载已有会话消息且 `thinkingElapsed > 0`
- **THEN** 思维链摘要显示 `💭 已思考 X.X 秒`

#### Scenario: 加载历史消息且无耗时数据
- **WHEN** `thinkingElapsed == 0`
- **THEN** 思维链摘要保持 `💭 已思考`

### Requirement: 回复消息底部显示耗时

#### Scenario: 正常完成流式输出后显示总耗时
- **WHEN** AI 流式输出正常结束
- **THEN** 在消息内容（`.msg-content`）下方、操作按钮（`.ai-msg-actions`）上方，插入耗时元素 `<div class="ai-msg-time">⏱ 总耗时 X.X 秒</div>`
- **THEN** 字体小号 0.75rem、颜色 `var(--text-muted)`、顶部间距 6px、居右对齐

#### Scenario: 加载历史消息时显示总耗时
- **WHEN** 从数据库加载已有会话消息且 `totalElapsed > 0`
- **THEN** 在消息内容下方显示 `⏱ 总耗时 X.X 秒`

#### Scenario: 流式被用户取消时不显示耗时
- **WHEN** 用户点击停止按钮取消流式
- **THEN** 不保存消息，不显示耗时标签

#### Scenario: 重新生成时重新计算耗时
- **WHEN** 用户点击重新生成按钮
- **THEN** 重新从头计时，新的回复显示新的耗时