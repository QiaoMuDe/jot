# Tasks

- [x] Task 1: 前端 URL 保存校验补全（`frontend/src/main.js` `initApiConnectionModule`）
  - [x] 1.1 URL `change` 分支：`url` 为空时提示「请先填写 API 地址」且不保存（现有斜杠校验保留）
  - [x] 1.2 URL `input` 分支：处于 `input-error` 状态时修正为非斜杠结尾 → 移除样式并立即 `saveSettings()` 自动保存一次（Toast「AI 配置已保存」+ 触发 `onSettingsSaved`）
- [x] Task 2: `switchProfile` 对齐隐藏模型搜索框（`frontend/src/main.js` L3178 附近）
  - [x] 2.1 清空模型下拉后隐藏 `.ai-model-search-wrap`（`display: none`），与 `switchProfileEmbed` 一致
- [x] Task 3: 移除 kwCards 死代码（`app.go` `CallAIStream` L2165-L2179）
  - [x] 3.1 删除 `kwCards` 声明/反序列化/`MergeRecallCards` 调用，直接 `cardsJSON, _ := json.Marshal(vectorResult.Cards)`
  - [x] 3.2 保留 `recallCardsJSON` 赋值（DB 存储）与 `TruncateRecallCardsPreview`/`ai:recall-cards` 发射逻辑不变
- [x] Task 4: `startVectorIndex` 复用 `GetEmbedConfig` 并补齐校验（`app.go` L1576-L1595）
  - [x] 4.1 改用 `provider, baseURL, apiKey, model, _ := a.GetEmbedConfig()` 读取四键，删除自读重复代码
  - [x] 4.2 校验 `provider == "" || baseURL == "" || model == ""` 时 `release()` 并返回「请先在设置中配置量化连接与量化模型」
- [x] Task 5: 合并 `TestAIBaseURL`/`TestAIConnection` 重复实现（`app.go`）
  - [x] 5.1 抽公共内部方法 `testAIConnection(provider, baseURL, apiKey, logName string)`
  - [x] 5.2 两个 Wails 绑定方法保留签名与各自日志前缀，内部调用公共方法
- [x] Task 6: 抽取 openai/ollama HTTP 请求公共函数（`internal/services/ai_service.go`）
  - [x] 6.1 新增 `httpGetJSON(url, apiKey string, timeout time.Duration) ([]byte, int, error)` 统一 GET+超时+状态码读取
  - [x] 6.2 `testOpenAIConnection`/`testOllamaConnection`/`fetchOpenAIModels`/`fetchOllamaModels` 改用公共函数，行为不变
- [x] Task 7: 构建与静态检查验证
  - [x] 7.1 `go build ./...` 通过
  - [x] 7.2 `golangci-lint run ./...` 通过（0 issues）
  - [x] 7.3 前端 `npm run build` 通过

# Task Dependencies

- [Task 7] depends on [Task 1], [Task 2], [Task 3], [Task 4], [Task 5], [Task 6]
- [Task 3] 与 [Task 4] 均改 `app.go` 但无冲突，可并行；[Task 5] 同文件独立
- [Task 1]、[Task 2] 改 `frontend/src/main.js`，可并行
