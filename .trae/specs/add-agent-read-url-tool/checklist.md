# Checklist

- [x] `go.mod` / `go.sum` 包含 `github.com/cloudwego/eino-ext/components/document/loader/url` 依赖
- [x] `internal/agent/tools/web_search.go` 中 `intSetting` 已改为包级函数 `getIntSetting`，web_search 调用点同步更新且行为不变
- [x] `internal/agent/tools/read_url.go` 存在，`readURLTool` 实现 `tool.InvokableTool` 与 `ActionTextProvider`（有编译期断言）
- [x] `read_url` 工具 `Info()` 名称/描述/参数（url 必填）正确，Desc 说明何时调用
- [x] `InvokableRun()` 校验 URL（仅 http/https）、设置 15s 超时与浏览器 UA、按 `ai_web_search_max_chars` 截断、空正文/抓取失败/ctx 取消错误路径正确
- [x] `internal/agent/registry.go` `buildTools` 注册 `read_url`（`WrapWithError` 包装）
- [x] `internal/agent/tools/meta.go` `BuiltinTools` 包含 `read_url` 展示文案
- [x] `internal/agent/tools/doc.go` 与 `internal/agent/doc.go` 清单同步包含 `read_url` / `NewReadURL`
- [x] `go build ./...` 通过
- [x] `go vet ./internal/agent/...` 通过
