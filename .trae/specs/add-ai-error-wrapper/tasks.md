# Tasks

## Task Dependencies
- [Task 1] 是基础，必须先完成
- [Task 2, 3] 基于 Task 1，可并行
- [Task 4] 基于 Task 2, 3
- [Task 5] 基于 Task 4

---

- [x] Task 1: 创建 `internal/aicli/errors.go` 错误分类层
  - 定义错误分类常量（auth_error, rate_limit, server_error 等 11 种）
  - 定义 `AIError` 结构体（Category, UserMsg, Raw 字段）
  - 定义 `func ClassifyError(err error) *AIError` 函数
  - 定义 `AIErrorWrapper` 包装类型，满足 error 接口
  - 实现 OpenAPI 同态错误检测（`errors.As` 匹配 `*openai.APIError` 和 `*openai.RequestError`）
  - 实现 context 超时/网络错误检测
  - 实现通用 fallback 分类（根据文本关键词匹配）
  - 编写可运行的单元测试覆盖主要分类场景（17 个测试全部通过）

- [x] Task 2: 修改 `internal/aicli/openai.go` 集成错误分类层
  - 在 `openaiChatStream()` 中捕获 `client.CreateChatCompletionStream` 和 `stream.Recv()` 的错误后，先调用 `ClassifyError()` 转换
  - 在 `openaiChat()` 中捕获 `client.CreateChatCompletion` 的错误后，先调用 `ClassifyError()` 转换
  - 将分类后的错误通过 `AIErrorWrapper` 包装返回

- [x] Task 3: 修改 `internal/aicli/ollama.go` 集成错误分类层
  - 在 `ollamaChatStream()` 中捕获 `ollamaClient.Chat()` 的错误后，先调用 `ClassifyError()` 转换
  - 在 `ollamaChat()` 中捕获错误后，先调用 `ClassifyError()` 转换
  - Ollama 错误多为通用错误，依赖分类层的通用检测逻辑

- [x] Task 4: 修改 `internal/aicli/client.go` 的 `OnError` 传递逻辑
  - `Stream()` 方法中的错误处理改为：先检测是否为 `AIErrorWrapper`，若是则传递 JSON 字符串，否则走原逻辑

- [x] Task 5: 修改前端 `frontend/src/js/ai-chat.js` 错误处理
  - 在 `ai:stream-error` 事件处理器中：尝试解析 JSON，成功则调用 `window.showNotification(errData.user_msg, 'error', 5000)`，失败则降级展示原始文本
  - 在 `CallAI`（优化表达）的 `catch` 块中：尝试解析 JSON，成功则调用 `window.showNotification()`，失败则降级展示
  - 在 `CallAIStream` 的 `catch` 块中：同样尝试解析 JSON
