# Tasks

- [x] Task 1: 修改 `CallAIStream` — 精炼触发条件扩展 + 精炼提前执行
  - [x] 将 `searching` 标志触发条件从 `len(searchSources) > 0` 改为 `len(searchSources) > 0 || cardRecallEnabled`
  - [x] 在 goroutine 内、搜索块之前新增独立精炼步骤，条件 `len(searchSources) > 0 || cardRecallEnabled`
  - [x] 精炼成功后发射 `ai:refined-keywords`
  - [x] 仅当 `len(searchSources) > 0` 时发射 `ai:search-status=searching`
  - [x] 精炼失败时发射 `ai:stream-error` 终止流程

- [x] Task 2: 修改 `CallAIStream` — 联网搜索和卡片召回使用精炼关键词
  - [x] 搜索块直接使用 `refinedQuery`（去除块内重复精炼代码）
  - [x] 卡片召回块使用 `combinedQuery = userText + " " + refinedQuery`
  - [x] `refinedQuery` 为空时回退为原始 query

- [x] Task 3: 同步修改 `CallAIStreamRegenerate`
  - [x] 同上三处变更，但 `userText` 从最后一条 user message 提取

## Task Dependencies

- [Task 1] → [Task 2]
- [Task 1] → [Task 3]（可并行执行，但内容类似）
