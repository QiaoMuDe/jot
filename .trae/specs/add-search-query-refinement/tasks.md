# Tasks

- [ ] Task 1: 新增 `internal/services/query_refiner.go` — Query 精炼服务
  - [ ] 定义 `RefineSearchQuery(ctx, query string, aiService *AIService) (string, error)` 函数
  - [ ] 内部构建 system prompt（固化的精炼指令）+ user message（原始 query）
  - [ ] 调用 `aiService.CallAI()` 进行非流式调用，获取精炼结果
  - [ ] 对返回结果做简单清理（trim 空格、去空）
  - [ ] 返回空串或调用出错时返回 error，由调用方决定是否终止流程

- [ ] Task 2: 修改 `app.go` 中 `CallAIStream` — 集成 query 精炼到搜索流程
  - [ ] 在联网搜索的 goroutine 中，提取 query 之后、调用 `SearchWeb` 之前，插入精炼步骤
  - [ ] 精炼成功时使用精炼后的 query 搜索
  - [ ] 精炼失败时发射 `ai:stream-error` 事件携带错误信息，终止整个流程
  - [ ] 精炼使用独立的 short timeout context（如 10s）

## Task Dependencies

- [Task 1] → [Task 2]
