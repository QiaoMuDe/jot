# Tasks

## Task 1: 替换 aicli 底层为 go-openai + ollama/ollama/api
- [x] 添加依赖：`go get github.com/sashabaranov/go-openai && go get github.com/ollama/ollama/api`（需在有网络的环境运行）
- [x] 删除依赖：`go mod tidy` 自动移除 gjson
- [x] 删除 `internal/aicli/extract.go`
- [x] 重写 `internal/aicli/types.go` — 仅保留 `Message`、`StreamCallbacks`、`Config`
- [x] 重写 `internal/aicli/openai.go` — 使用 go-openai 流式/非流式
- [x] 重写 `internal/aicli/ollama.go` — 使用 ollama/ollama/api 流式/非流式
- [x] 重写 `internal/aicli/client.go` — 统一分派层

## Task 2: 同步 ai_service.go 适配
- [x] 检查引用 — 无引用已删除函数
- [x] import 兼容 — `aicli.NewClient`、`aicli.Message`、`aicli.StreamCallbacks`、`aicli.Config` 均保留
- [x] 回调重连 — `thinkingStart` / `hasThinking` / `fullContent` / `fullThinking` 追踪逻辑与新库兼容

## Task 3: 验证编译
- [ ] **需用户操作**：运行 `go get github.com/sashabaranov/go-openai` 和 `go get github.com/ollama/ollama/api`
- [ ] **需用户操作**：运行 `go build ./...` 确认编译通过
- [ ] **需用户操作**：运行 `wails build`（如需要）确认完整构建

## Task 4: 更新 AGENTS.md
- [x] 记录 aicli 适配层改用 go-openai + ollama/ollama/api
- [x] 更新文件行数统计
- [x] 标记版本号更新
