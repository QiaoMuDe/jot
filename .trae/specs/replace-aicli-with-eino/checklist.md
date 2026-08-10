# Checklist

- [x] `internal/einocli/` 已创建，公共类型与方法（`Config`/`Message`/`StreamCallbacks`/`Client`/`NewClient`/`Chat`/`Stream`/`Embed`/`EmbedWithProgress`）与 aicli 完全一致
- [x] `Chat`：空消息过滤、role 转换、错误经 `aierrors` 包装；行为与非流式调用一致
- [x] `Stream`：content/reasoning 分别走 OnChunk/OnThinking，结束回调 OnDone（含思考/总耗时），错误走 OnError（分类 JSON），ctx 取消不报错
- [x] `Embed`/`EmbedWithProgress`：float64→float32 转换、分批与进度回调、返回数量校验
- [x] `internal/aierrors` 的 `errors.As` 已切换为 `meguminnnnnnnnn/go-openai` 类型，eino components 分支保留，测试更新后通过
- [x] 4 个调用方（`ai_service.go`、`vector_service.go`、`agent/tools.go`、`app.go`）已不再引用 `jot/internal/aicli`
- [x] `internal/aicli/` 目录已删除
- [x] `go.mod` 已移除 `github.com/sashabaranov/go-openai`，`meguminnnnnnnnn/go-openai` 为直接依赖
- [x] `go build ./...` 通过
- [x] 相关 `go test`（einocli/aierrors/services/agent）通过
- [x] `golangci-lint run ./...` 无新增问题
