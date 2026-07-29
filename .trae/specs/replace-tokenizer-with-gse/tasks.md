# Tasks

- [ ] Task 1: 添加 gse 依赖
  - 在 `go.mod` 中添加 `github.com/go-ego/gse` 依赖
  - 执行 `go mod tidy` 下载依赖并更新 `go.sum`
  - 验证：`go build ./...` 通过

- [ ] Task 2: 替换分词逻辑
  - 删除 `tokenize2Gram()`、`isCJK()`、`splitWords()` 三个函数
  - 保留 `stopWords` map 和 `isStopWord()` 函数
  - 新增包级全局 `gse.Segmenter` + `sync.Once` 懒初始化
  - 新增 `tokenize(text string) []string` 函数（调用 gse.Cut + 停用词过滤 + 去重）
  - 更新 `CardRecallSearch` 中调用点：`tokenize2Gram(query)` → `tokenize(query)`
  - 更新注释
  - 验证：`go build ./...` 通过

- [ ] Task 3: 回归验证
  - 编译并启动应用
  - 触发卡片召回功能，确认搜索结果无误
  - 无编译错误、无运行时崩溃

# Task Dependencies
- Task 2 依赖 Task 1
