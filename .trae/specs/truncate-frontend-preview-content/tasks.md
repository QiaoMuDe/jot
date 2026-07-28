# Tasks

- [x] Task 1: 实现截断函数

  **文件**: `internal/services/recall_service.go`

  - 在 `recall_service.go` 中新增 `TruncateRecallCardsPreview(cards []RecallCard, maxLen int) []RecallCard` 函数：遍历 `cards`，将每条 card 的 `Content` 按 `maxLen` 截断（使用 rune 处理中文），截断后 `Content = string(runes[:maxLen])`，超过时截断不添加标记（纯预览截断，不是语义截断）。
  - 同文件新增 `TruncateSearchSourcesPreview(sources []SearchSource, maxLen int) []SearchSource` 函数：同上逻辑处理每条 source 的 `Content`。

- [x] Task 2: 修改 `CallAIStream` 事件发射

  **文件**: `app.go`

  - 在发射 `ai:recall-cards` 事件前（约第 1862-1864 行），用 `TruncateRecallCardsPreview` 对 `recallResult.Cards` 做副本截断，用截断后的副本构建事件 payload。
  - 在发射 `ai:search-sources` 事件前（约第 1824-1828 行），同样对累积的 `Sources` 做截断。
  - `recallCardsJSON` / `searchSourcesJSON`（用于 DB 存储的第 1944-1945 行）保持全量不变。

- [x] Task 3: 修改 `LoadAISessionMessagesPaginated` 返回值截断

  **文件**: `internal/services/ai_service.go`

  - 在 `LoadAISessionMessagesPaginated` 函数返回前，对 `result` 中每条消息的 `RecallCards` 和 `SearchSources` JSON 字符串做：
    - `json.Unmarshal` 为 `[]RecallCard` / `[]SearchSource`
    - 调用 `TruncateRecallCardsPreview` / `TruncateSearchSourcesPreview` 截断
    - 重新 `json.Marshal` 赋值回原字段
  - JSON 解析失败时静默跳过该字段（保持原样返回，不影响功能）
  - 注意：不要修改 `LoadAISessionMessages`（供 `CallAIStream` 内部用，需要全量数据）

- [x] Task 4: 编译验证

  - 运行 `wails build` 确保项目编译通过

# Task Dependencies

- [Task 1] → [Task 2], [Task 3]
- [Task 2] → [Task 4]
- [Task 3] → [Task 4]
