# 持久化 AI 搜索来源与召回卡片 Spec

## Why

目前联网搜索的搜索来源和卡片召回的召回卡片数据仅在运行时通过 Wails Events 传递到前端内存变量中，切换会话或重启程序后结构化数据丢失，只能看到注入到 system message 的纯文本，无法展示可点击链接和可打开笔记的卡片 UI。

## What Changes

1. **扩展 `AIMessage` 模型**：新增两个 TEXT 字段存储 JSON 序列化的结构化数据
2. **修改 `Message` 结构体**：携带搜索来源和召回卡片数据，贯穿保存链路
3. **修改 `CallAIStream`**：将搜索/召回结果传入 `SaveAIMessages`
4. **修改 `SaveAIMessages`**：将结构化数据写入数据库
5. **修改前端 `addMessage()`**：支持从数据库恢复的搜索来源/召回卡片渲染
6. **修改前端 `switchSession()`**：加载消息时同步恢复搜索来源/召回卡片 UI

## Impact

- Affected specs: AI 消息持久化、联网搜索、卡片召回
- Affected code: `internal/models/ai_message.go`, `internal/services/ai_service.go`, `app.go`, `frontend/src/js/ai-chat.js`

## ADDED Requirements

### Requirement: 持久化搜索来源与召回卡片结构化数据

系统 SHALL 将每次 AI 对话中产生的搜索来源和召回卡片结构化数据持久化到 SQLite 数据库，确保切换会话或重启程序后仍然可以渲染对应的 UI 组件。

#### Scenario: 搜索来源持久化
- **GIVEN** 用户开启联网搜索并发送消息
- **WHEN** 后端完成 Tavily 搜索并 AI 回复完成
- **THEN** 搜索来源列表（标题/URL/摘要）以 JSON 格式存入对应 assistant 消息的 `search_sources` 字段
- **AND** 用户切换回该会话时，前端自动渲染搜索来源折叠面板

#### Scenario: 召回卡片持久化
- **GIVEN** 用户开启卡片召回并发送消息
- **WHEN** 后端完成卡片召回搜索并 AI 回复完成
- **THEN** 召回卡片列表（ID/标题/内容/文件后缀）以 JSON 格式存入对应 assistant 消息的 `recall_cards` 字段
- **AND** 用户切换回该会话时，前端自动渲染召回卡片折叠面板

#### Scenario: 无搜索/召回时兼容
- **GIVEN** 用户未开启联网搜索或卡片召回
- **WHEN** AI 回复完成
- **THEN** `search_sources` 和 `recall_cards` 字段为空字符串
- **AND** 前端渲染时跳过这两个面板

#### Scenario: 编辑/重新发送消息
- **GIVEN** 用户编辑并重新发送消息
- **WHEN** 新的 AI 回复完成
- **THEN** 新的回复消息携带新的搜索来源/召回卡片数据
- **AND** 旧的回复消息数据保持不变

## MODIFIED Requirements

### Requirement: AIMessage 数据模型
**变更**：在 `AIMessage` 中新增两个 TEXT 字段
- `search_sources TEXT`：存储搜索来源的 JSON 数组
- `recall_cards TEXT`：存储召回卡片的 JSON 数组

### Requirement: SaveAIMessages 方法
**变更**：`Message` 结构体新增 `SearchSources` 和 `RecallCards` 字段
- 保存 assistant 消息时，如果这两个字段非空，JSON 序列化后写入对应的数据库字段
- 用户消息和普通 AI 回复这两个字段为空

### Requirement: CallAIStream 数据传递
**变更**：在 `CallAIStream` 的 goroutine 内
- 将搜索结果 `Sources` 和召回结果 `Cards` 缓存到局部变量
- 在 `stream-done` 回调中，将缓存数据传入 `SaveAIMessages`

### Requirement: 前端消息渲染
**变更**：`addMessage()` 新增 `searchSources` 和 `recallCards` 参数
- 非空时渲染搜索来源折叠面板（可点击链接，与现有 UI 一致）
- 非空时渲染召回卡片折叠面板（可点击打开笔记，与现有 UI 一致）

### Requirement: 前端会话切换
**变更**：`switchSession()` 加载消息时
- 检查每条 assistant 消息的 `search_sources` 和 `recall_cards` 字段
- 非空时解析 JSON 并传给 `addMessage()` 渲染
