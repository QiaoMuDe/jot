# Tasks

## Task 1: 创建 internal/einocli 包（eino 薄适配层）
- [x] SubTask 1.1: 新建 `internal/einocli/types.go`：`Config`、`Message`、`StreamCallbacks`、`Client{BaseURL,APIKey,Model}`、`NewClient`（与 aicli 完全一致）
- [x] SubTask 1.2: 新建 `internal/einocli/chat.go`：`Chat`（`openai.NewChatModel` + `Generate`，空消息过滤，role→`schema.RoleType`，错误经 `aierrors.ClassifyError` 包装）+ `Stream`（`NewChatModel` + `Stream`，`WithExtraFields{"enable_thinking": thinkingEnabled}`，消费 `*schema.StreamReader` 参照 `internal/agent/agent.go` 的 `consumeAssistantStream`，OnChunk/OnThinking/OnDone/OnError 语义与 aicli 一致，ctx 取消不报错，统计 thinking/total 耗时）
- [x] SubTask 1.3: 新建 `internal/einocli/embedding.go`：`Embed`（`acl/openai.NewEmbeddingClient` + `EmbedStrings`，float64→float32 转换，数量校验）+ `EmbedWithProgress`（分批循环 + 进度回调）
- [x] SubTask 1.4: 运行 `go build ./internal/einocli/...` 通过

## Task 2: aierrors 错误类型切换（可并行）
- [x] SubTask 2.1: `internal/aierrors/errors.go`：import 由 `github.com/sashabaranov/go-openai` 改为 `github.com/meguminnnnnnnnn/go-openai`，`errors.As` 目标 `*APIError/*RequestError` 同步切换；eino components 分支与分类逻辑不变
- [x] SubTask 2.2: `internal/aierrors/errors_test.go`：sashabaranov 类型用例改为 meguminnnnnnnnn 类型，eino 用例保留
- [x] SubTask 2.3: 运行 `go build ./internal/aierrors/... && go test ./internal/aierrors/...` 通过

## Task 3: 调用方迁移（依赖 Task 1，可并行）
- [x] SubTask 3.1: `internal/services/ai_service.go`：`aicli.NewClient` / `aicli.Message` / `aicli.StreamCallbacks` → `einocli.*`，import 同步
- [x] SubTask 3.2: `internal/services/vector_service.go`：4 个方法签名 `*aicli.Client` → `*einocli.Client`（`IndexNotes`/`vectorSearch`/`HybridRecall`/`VectorRecall`），import 同步
- [x] SubTask 3.3: `internal/agent/tools.go`：`aicli.NewClient(aicli.Config{...})` → `einocli.NewClient(einocli.Config{...})`，import 同步
- [x] SubTask 3.4: `app.go`：2 处 `aicli.NewClient(aicli.Config{...})`（startVectorIndex ~L1694、卡片召回 ~L2287）→ `einocli.*`，import 同步
- [x] SubTask 3.5: 运行 `go build ./...` 通过（此时 aicli 已无引用，可先不删）

## Task 4: 删除 aicli 与依赖清理（依赖 Task 2/3）
- [x] SubTask 4.1: 删除 `internal/aicli/` 目录（client.go、openai.go、types.go）
- [x] SubTask 4.2: 运行 `go mod tidy` 移除 `github.com/sashabaranov/go-openai`，确认 `meguminnnnnnnnn/go-openai` 转为直接依赖
- [x] SubTask 4.3: 运行 `go build ./...` 通过

## Task 5: 验证
- [x] SubTask 5.1: `go build ./...` 通过
- [x] SubTask 5.2: `go test ./internal/einocli/... ./internal/aierrors/... ./internal/services/... ./internal/agent/...` 通过
- [x] SubTask 5.3: `golangci-lint run ./...` 无新增问题

# Task Dependencies
- [Task 3] 依赖 [Task 1]（调用方需引用已存在的 einocli）
- [Task 4] 依赖 [Task 2] 与 [Task 3]（删 aicli 前需确保无引用、aierrors 不再依赖 sashabaranov）
- [Task 5] 依赖 [Task 1]/[Task 2]/[Task 3]/[Task 4]
- [Task 2] 与 [Task 3] 互不依赖，可并行
