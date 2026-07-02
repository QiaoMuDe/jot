# Tasks

- [x] Task 1: 后端 — SearchWeb 接受 maxResults 参数
  - [x] SubTask 1.1: 修改 `SearchWeb` 函数签名，新增 `maxResults int` 参数
  - [x] SubTask 1.2: 使用 `maxResults` 替代硬编码的 `MaxResults: 5`
- [x] Task 2: 后端 — 新增 GetAISearchResultLimit / SetAISearchResultLimit 绑定
  - [x] SubTask 2.1: `app.go` 新增 `GetAISearchResultLimit() int` 绑定，读取 `ai_search_result_limit`，空值时返回 5
  - [x] SubTask 2.2: `app.go` 新增 `SetAISearchResultLimit(limit int) error` 绑定，写入 `ai_search_result_limit`，含范围校验（1-20）
  - [x] SubTask 2.3: 修改 `CallAIStream` 中调用 `SearchWeb` 处，读取 setting 并传入 maxResults
- [x] Task 3: 前端 — 设置页 UI 增加搜索结果数输入
  - [x] SubTask 3.1: `frontend/index.html` AI 设置中联网搜索区域（API Key 下方）新增"搜索结果数"数字输入框，id 为 `aiSearchResultLimit`
  - [x] SubTask 3.2: `frontend/src/main.js` 中 `loadAISettings()` 增加加载该值
  - [x] SubTask 3.3: `frontend/src/main.js` 增加输入框 `change` 事件自动保存

# Task Dependencies

- [Task 1] 无依赖
- [Task 2] 依赖 [Task 1]（SearchWeb 签名变更后需更新调用处）
- [Task 3] 依赖 [Task 2]（前端需要后端绑定可用）
