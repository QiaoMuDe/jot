# 空回复标记字段 spec

## Why

当前 AI 空回复占位消息用文本内容匹配来识别（判断字符串是否为 "AI 未返回内容，请尝试重新生成"），不严谨且脆弱。切换会话后，历史消息加载走 `addMessage` 路径，无法区分正常消息和占位消息，导致渲染样式丢失。

通过数据库加专用标记字段，后端返回时携带标记，前端据此精确判断是否显示为空回复样式。

## What Changes

### 后端（3 个文件）

| 文件 | 变更 |
|------|------|
| `internal/models/ai_message.go` | AIMessage 结构体新增 `IsEmptyResponse bool` 字段 |
| `internal/services/ai_service.go` | Message 结构体新增 `IsEmptyResponse bool` 字段；`SaveAIMessages` 和 `LoadAISessionMessages` 中传递该字段 |
| `app.go` | 无变更（绑定层无需改，Message 序列化由 json tag 自动处理） |

### 前端（1 个文件）

| 文件 | 变更 |
|------|------|
| `frontend/src/js/ai-chat.js` | 保存时设 `is_empty_response: true`；`addMessage` 和 `switchSession` 加载时根据字段渲染占位样式 |

### 样式（已有）

`ai-chat.css` 中 `.ai-msg-empty` / `.ai-msg-empty-icon` 样式已在之前添加，无需变动。

## Impact

- 不影响已有数据库表结构：GORM AutoMigrate 对新字段 `is_empty_response` 会执行 `ALTER TABLE` 加列，已有数据该字段为默认值 `false`
- 不影响已有消息的正常渲染（`IsEmptyResponse = false` 时走原逻辑）
- **不涉及 BREAKING 变更**

## 数据流

```
前端 ai:stream-done
  → 检测 finalContent 为空
  → 设 is_empty_response = true
  → saveSessionMessages([{..., is_empty_response: true}])
  → Go SaveAIMessages 写入 DB
  → 切换会话时 LoadAISessionMessages 返回 is_empty_response=true
  → addMessage 读取字段，渲染占位样式
```

## ADDED Requirements

### Requirement: 空回复标记持久化

系统 SHALL 在数据库中持久化标记 AI 消息是否为空回复。

#### Scenario: 空回复消息保存
- **WHEN** AI 流式返回结束且 `finalContent` 为空
- **THEN** 该消息的 `is_empty_response` 字段值设为 `true`，保存到 `ai_messages` 表
- **AND** 页面 DOM 中渲染警示占位样式（琥珀色图标 + 灰色提示文字）

#### Scenario: 正常消息保存
- **WHEN** AI 流式返回结束且 `finalContent` 非空
- **THEN** 该消息的 `is_empty_response` 字段值设为 `false`（默认值），行为不变

#### Scenario: 历史消息加载回显
- **WHEN** 切换会话，从数据库加载消息
- **THEN** 当 `is_empty_response === true` 时，`addMessage` 渲染占位样式（`ai-msg-empty`）
- **AND** 当 `is_empty_response === false` 时，走正常 markdown 渲染

## MODIFIED Requirements

无（新增功能，不修改已有需求）。

## REMOVED Requirements

无。
