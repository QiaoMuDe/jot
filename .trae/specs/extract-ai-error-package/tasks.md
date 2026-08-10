# Tasks

## Task 1: 创建 internal/aierrors 包
- [x] SubTask 1.1: 新建 `internal/aierrors/errors.go`，迁移 12 个分类常量、`AIError`、`userMessages`、`NewAIError`、`ToJSON`、`AIErrorWrapper`（基于原 `internal/aicli/errors.go`）
- [x] SubTask 1.2: 重构 `ClassifyError`：抽共享分类函数 `classifyAPIErrorLike(statusCode, code, message, raw)`；`errors.As` 同时匹配 eino `*openai.APIError`（`github.com/cloudwego/eino-ext/components/model/openai`，import 别名避免与 sashabaranov 冲突）与 sashabaranov `*APIError`/`*RequestError`；context/net/文本 fallback 逻辑保持不变
- [x] SubTask 1.3: 新建 `internal/aierrors/errors_test.go`：迁移原 `aicli/errors_test.go` 全部用例（改为引用 aierrors 自身），并新增 eino `*APIError` 分类用例（401/429/400 context_length 各一）

## Task 2: aicli 内部改用 aierrors
- [x] SubTask 2.1: `internal/aicli/openai.go` 中 4 处 `ClassifyError` → `aierrors.ClassifyError`，4 处 `&AIErrorWrapper{...}` → `&aierrors.AIErrorWrapper{...}`
- [x] SubTask 2.2: `internal/aicli/client.go` 中 `var aiErr *AIErrorWrapper` → `*aierrors.AIErrorWrapper`
- [x] SubTask 2.3: 删除 `internal/aicli/errors.go` 与 `internal/aicli/errors_test.go`

## Task 3: 调用方改用 aierrors
- [x] SubTask 3.1: `internal/services/ai_service.go`：`aicli.ClassifyError` → `aierrors.ClassifyError`、`&aicli.AIErrorWrapper{...}` → `&aierrors.AIErrorWrapper{...}`，补充 import
- [x] SubTask 3.2: `internal/services/vector_service.go`：`aicli.ClassifyError` → `aierrors.ClassifyError`，补充 import（vector_service 仍使用 `*aicli.Client`，aicli import 已保留）
- [x] SubTask 3.3: `app.go`：`aicli.AIErrorWrapper` / `aicli.NewAIError` / `aicli.CategoryUnknown` / `aicli.ClassifyError`（L2106/L2110/L2633）→ `aierrors.*`，补充 import

## Task 4: 构建与测试验证
- [x] SubTask 4.1: `go build ./...` 通过
- [x] SubTask 4.2: `go test ./internal/aierrors/... ./internal/aicli/...` 通过
- [x] SubTask 4.3: `golangci-lint run ./...` 无新增问题（0 issues）

# Task Dependencies
- [Task 2] 依赖 [Task 1]（aicli 需引用已存在的 aierrors）
- [Task 3] 依赖 [Task 1]（调用方需引用已存在的 aierrors）
- [Task 4] 依赖 [Task 1]/[Task 2]/[Task 3]
