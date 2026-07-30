# Tasks

- [x] Task 1: 在 `AIService` 中添加设置项读取方法 `GetContextWindowSize`，读取 `ai_context_window_size`，默认返回 20
- [x] Task 2: 在 `AIService` 中添加截断方法 `TruncateMessagesForLLM`，接收 `[]Message` 和窗口大小，保留 system 消息 + 最后 N 条 user/assistant 消息
- [x] Task 3: 在 `CallAIStream` 中调用 `LoadAISessionMessages` 后、注入 context 前，调用截断方法
- [x] Task 4: 在 `CallAIStreamRegenerate` 中调用 `LoadAISessionMessages` 后、注入 context 前，调用截断方法

# Task Dependencies

- [Task 1] 和 [Task 2] 可并行
- [Task 3] 和 [Task 4] 依赖 [Task 1] 和 [Task 2]
- [Task 3] 和 [Task 4] 可并行
