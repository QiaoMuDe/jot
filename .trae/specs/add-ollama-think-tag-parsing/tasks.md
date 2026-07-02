# Tasks

- [x] Task 1: 实现 Ollama `<think>` 标签实时解析状态机
  - [x] 1.1 在 `CallAIStream` 的 `WithStreamingFunc` 回调中新增 `buffer` 累加器（`strings.Builder`）
  - [x] 1.2 实现状态机逻辑：检测 ``/`` 边界，分流至 `onThinking`/`onChunk`
  - [x] 1.3 处理跨 chunk 标签边界（全文缓冲区 + `strings.Contains`）
  - [x] 1.4 处理无标签、标签不完整、标签在任意位置的边界情况
  - [x] 1.5 确保 `fullContent` 仅累加非思维链文本，`fullThinking` 累加思维链文本
  - [x] 1.6 `hasThinking` 和 `elapsedThinking` 正确计算
  - [x] 1.7 `go build ./internal/services/` 编译通过

- [x] Task 2: 更新 AGENTS.md
  - [x] 2.1 在章节八十一新增 Ollama `<think>` 标签解析记忆点

# Task Dependencies

无。
