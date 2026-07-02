# Tasks

- [x] Task 1: 修改 `CallAIStream` 中 Ollama `<think>` 标签解析触发条件
  - [x] 1.1 将 `isOllama && thinkingEnabled` 改为 `isOllama`（始终解析标签）
  - [x] 1.2 `thinkingEnabled=false` 时，找到 `</think>` 后不调用 `onThinking`、不记录 `hasThinking`
  - [x] 1.3 `thinkingEnabled=false` 时，静默丢弃标签内内容（`fullThinking` 不写入）
  - [x] 1.4 流结束刷出时，若 `!thinkingEnabled`，剩余内容也静默丢弃（不调用 `onThinking`）
  - [x] 1.5 `go build ./internal/services/` 编译通过

- [x] Task 2: 更新 AGENTS.md 相关记忆点

# Task Dependencies

无。
