# AI 搜索来源与召回卡片前端预览截断 Spec

## Why

切换 AI 会话时，历史消息中的召回卡片携带笔记全文，联网搜索来源携带搜索结果全文（默认 5000 字/条），这些数据通过 Wails 桥全量传输到前端。当会话包含多条带召回/搜索的消息时，Wails 桥传输 + 前端 DOM 渲染阻塞主线程，导致切换卡顿数秒。前端折叠面板仅需展示内容片段用于预览，无需全文。

## What Changes

1. **新增 `truncateRecallCardsPreview` 辅助函数**：将 `[]RecallCard` 中每条 card 的 `Content` 截断到 200 字
2. **新增 `truncateSearchSourcesPreview` 辅助函数**：将 `[]SearchSource` 中每条 source 的 `Content` 截断到 200 字
3. **修改 `app.go` `CallAIStream`**：发射 `ai:recall-cards` 和 `ai:search-sources` 事件前，对数据副本做截断，DB 存储保持全量
4. **修改 `LoadAISessionMessagesPaginated`**：返回前解析 `RecallCards`/`SearchSources` JSON，截断每条内容的 `Content` 后重新序列化
5. **前端无需改动**： `renderRecallCards`/`renderMultiSourcesPanel` 收到的就是截断后的数据

## Impact

- Affected specs: 联网搜索、卡片召回、AI 消息持久化
- Affected code: `internal/services/ai_service.go`、`app.go`、`internal/services/recall_service.go`

## ADDED Requirements

### Requirement: 前端预览截断

系统 SHALL 在数据离开 Go 后端进入前端之前，将 `RecallCard.Content` 和 `SearchSource.Content` 截断到 200 字符，减小 Wails 桥传输量和前端 DOM 渲染开销。

#### Scenario: 流式发射时截断
- **GIVEN** AI 回复完成，后端准备发射 `ai:recall-cards` 和 `ai:search-sources` 事件
- **WHEN** 构建事件 payload
- **THEN** 对 `recallResult.Cards` 和搜索结果 `Sources` 的副本做 Content 截断（200 字）
- **AND** `recallCardsJSON` / `searchSourcesJSON`（用于 DB 存储）保持全量不变

#### Scenario: 切换会话加载时截断
- **GIVEN** 用户切换到有历史消息的 AI 会话
- **WHEN** `LoadAISessionMessagesPaginated` 从数据库加载消息
- **THEN** 将每条消息的 `RecallCards` 和 `SearchSources` JSON 字段解析后，对其中每个元素的 `Content` 截断到 200 字
- **AND** 截断后再序列化返回给前端

#### Scenario: 内部调用不走截断
- **GIVEN** `CallAIStream` 内部调用 `LoadAISessionMessages` 加载历史消息（用于构建 AI 上下文）
- **WHEN** 该函数不经过截断路径
- **AND** AI 收到的上下文仍包含全文内容

## MODIFIED Requirements

### Requirement: LoadAISessionMessagesPaginated 返回值截断

**变更**：在返回 `[]Message` 前，对每条消息的 `RecallCards` 和 `SearchSources` JSON 字符串做解析→截断→重序列化。

### Requirement: CallAIStream 事件发射截断

**变更**：发射 `ai:recall-cards` 和 `ai:search-sources` 事件前，先对数据做一份副本，截断副本中每个元素的 Content，用副本构建事件 payload。原始数据保持全量用于 DB 持久化。
