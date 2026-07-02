# Checklist

- [x] 1. `internal/aicli/errors.go` 存在，定义了 11 种错误分类常量和对应的中文提示
- [x] 2. `ClassifyError()` 函数对 HTTP 401/403 返回 `auth_error` + 正确中文提示
- [x] 3. `ClassifyError()` 函数对 HTTP 429 返回 `rate_limit` + 正确中文提示
- [x] 4. `ClassifyError()` 函数对 HTTP 500/502/503 返回 `server_error` + 正确中文提示
- [x] 5. `ClassifyError()` 函数对 context.DeadlineExceeded 返回 `timeout` + 正确中文提示
- [x] 6. `ClassifyError()` 函数对网络连接错误（`*net.OpError`）返回 `network_error` + 正确中文提示
- [x] 7. `ClassifyError()` 单元测试覆盖 auth_error/rate_limit/server_error/timeout/network_error 五个核心场景
- [x] 8. `openai.go` 中 `openaiChatStream()` 的流创建错误经过 `ClassifyError()` 处理
- [x] 9. `openai.go` 中 `openaiChat()` 的非流式错误经过 `ClassifyError()` 处理
- [x] 10. `client.go` 的 `Stream()` 方法中错误通过 JSON 字符串传递
- [x] 11. 前端 `ai:stream-error` 事件处理器解析 JSON 并调用 `showNotification()` 展示用户友好消息
- [x] 12. 前端 `CallAI`（优化表达）的 `catch` 块解析 JSON 并调用 `showNotification()`
- [x] 13. 前端 `ai:stream-error` 不再调用 `addErrorMessage()`（降级时除外）
- [x] 14. 前端对非 JSON 格式的错误字符串降级兼容（直接展示原始文本）
- [x] 15. 构建通过，无编译错误
