# Checklist

- [x] `internal/aierrors/` 已创建，包含 12 个分类常量、`AIError`、`NewAIError`、`ToJSON`、`AIErrorWrapper`、`ClassifyError`
- [x] `ClassifyError` 能同时识别 eino `*APIError`（eino-ext components/model/openai）与 sashabaranov `*APIError`/`*RequestError`
- [x] `context.Canceled` 返回 nil、context 超时/网络错误/文本 fallback 分类行为与原实现一致
- [x] `internal/aierrors/errors_test.go` 已覆盖原 aicli 全部用例，并新增 eino 类型用例
- [x] `internal/aicli/errors.go` 与 `internal/aicli/errors_test.go` 已删除
- [x] `internal/aicli/openai.go`、`client.go` 已改为引用 `aierrors`
- [x] `internal/services/ai_service.go`、`internal/services/vector_service.go`、`app.go` 已改为引用 `aierrors`
- [x] `go build ./...` 通过
- [x] `go test ./internal/aierrors/... ./internal/aicli/...` 通过
- [x] `golangci-lint run ./...` 无新增问题
