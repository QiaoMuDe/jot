# Tasks

- [x] Task 1: 扩展 AIMessage 模型 — 在 `AIMessage` 结构体中新增 `SearchSources` 和 `RecallCards` 两个 TEXT 字段
- [x] Task 2: 修改 Message 结构体与保存逻辑 — `services.Message` 新增字段，`SaveAIMessages` 将其写入数据库
  - [x] SubTask 2.1: 在 `internal/services/types.go` 的 `Message` 结构体中添加 `SearchSources` 和 `RecallCards` 字段
  - [x] SubTask 2.2: 修改 `SaveAIMessages` 方法，将结构化数据 JSON 序列化后写入 `AIMessage` 新字段
- [x] Task 3: 修改 CallAIStream — 将搜索/召回结果缓存并传入 `SaveAIMessages`
- [x] Task 4: 修改前端 `addMessage()` — 支持从数据库恢复的搜索来源/召回卡片渲染
- [x] Task 5: 修改前端 `switchSession()` — 加载消息时同步恢复搜索来源/召回卡片 UI

# Task Dependencies

- [Task 1] depends on none
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 2]
- [Task 4] depends on none
- [Task 5] depends on [Task 4]
