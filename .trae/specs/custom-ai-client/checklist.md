# Checklist

- [x] `internal/aicli/openai.go` 使用 go-openai 库实现流式 SSE 调用，正确提取 `Content` 和 `ReasoningContent`
- [x] `internal/aicli/ollama.go` 使用 ollama/ollama/api 库实现流式调用，正确提取 `Content` 和 `Thinking`，`thinkingEnabled=true` 时设置 `Think` 参数
- [x] `internal/aicli/client.go` 统一入口根据 Provider 自动选择后端，`Stream()` / `Chat()` 签名不变
- [x] `internal/aicli/extract.go` 已删除（gjson 字段提取被库的类型化结构体替代）
- [x] `internal/aicli/types.go` 精简为仅保留 `Message`、`StreamCallbacks`、`Config` 等对外类型
- [x] `ai_service.go` 无需改动或仅微调 import，外部行为完全一致
- [ ] `go get github.com/sashabaranov/go-openai` 和 `go get github.com/ollama/ollama/api` 运行成功
- [ ] `go build ./...` 编译通过，无报错
- [ ] `wails build` 编译通过（如需要）
- [ ] Ollama（qwen3:0.6b）深度思考开启时，思维链正确显示在前端（`Message.Thinking`）
- [ ] Ollama 深度思考关闭时，思维链内容不显示（`Think = false`）
- [ ] OpenAI 兼容 API 深度思考开启时，思维链正确显示（`Delta.ReasoningContent`）
- [ ] 非流式 `CallAI` 调用正常
- [ ] AGENTS.md 已更新
