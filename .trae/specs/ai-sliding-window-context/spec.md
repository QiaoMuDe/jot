# AI 滑动窗口消息截断 Spec

## Why

每次 AI 对话请求都将全部历史消息发送给 LLM，对话轮次增多时 token 消耗线性增长，导致响应变慢、成本增加，并可能触及模型上下文窗口上限。

## What Changes

- 在 `CallAIStream` 和 `CallAIStreamRegenerate` 中，加载全量消息后、context 注入前，对 user/assistant 消息做滑动窗口截断，只保留最后 N 条消息（默认 20 条 = 10 轮对话）
- 数据库中新增设置项 `ai_context_window_size`，默认值为 `20`
- system 消息（含注入的 context）不受窗口影响
- 前端 `chatHistory` 渲染缓冲区保持不变，用户仍可看到全部历史消息
- 不添加设置页 UI

## Impact

- Affected specs: AI 对话相关
- Affected code:
  - `app.go` — `CallAIStream`、`CallAIStreamRegenerate`
  - `internal/services/ai_service.go` — 可能新增带窗口的消息加载方法
  - `internal/models/` — 无需改动

## ADDED Requirements

### Requirement: 滑动窗口截断

The system SHALL limit the number of user/assistant messages sent to the LLM per request.

#### Scenario: 长对话截断
- **WHEN** 会话中的 user/assistant 消息超过 `ai_context_window_size` 条
- **THEN** 只保留最近的 N 条 user/assistant 消息发送给 LLM，较早的消息被丢弃

#### Scenario: 短对话不截断
- **WHEN** 会话中的 user/assistant 消息不超过 N 条
- **THEN** 全部发送，不做截断

#### Scenario: system 消息不受影响
- **WHEN** 执行截断后
- **THEN** system 消息（含注入的角色扮演、笔记引用、搜索、召回、技能等 context）始终保留

#### Scenario: 默认值
- **WHEN** `ai_context_window_size` 设置项不存在或非法
- **THEN** 使用默认值 20

### Requirement: 双入口统一逻辑

The system SHALL将截断逻辑抽取为公共函数，供 `CallAIStream` 和 `CallAIStreamRegenerate` 统一调用。

#### Scenario: 正常发送
- **WHEN** 前端调用 `CallAIStream`
- **THEN** 加载全量消息 → 截断 user/assistant 消息 → 注入 context → 发送给 LLM

#### Scenario: 重新生成
- **WHEN** 前端调用 `CallAIStreamRegenerate`
- **THEN** 加载全量消息 → 截断 user/assistant 消息 → 注入 context → 发送给 LLM
