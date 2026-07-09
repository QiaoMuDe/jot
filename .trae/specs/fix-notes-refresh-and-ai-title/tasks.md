# Tasks

## 前置说明
两项任务无依赖关系，可并行执行。

## 任务列表

- [x] Task 1: 笔记首页刷新 — Ctrl+1 快捷键回首页时补全 `loadNotes()` 调用
  - [x] 1.1 在 `main.js` 中 `Ctrl+1` 快捷键处理分支内，在 `switchView('grid')` 后添加 `await loadNotes()`

- [x] Task 2: AI 会话标题首条消息后同步更新 UI
  - [x] 2.1 在 `ai-chat.js` 的 `ai:stream-done` 回调末尾（`unsubs.push(unsubDone)` 之前），添加 `await loadSessionList()` 和 `updateChatTitle()` 调用
  - [x] 2.2 将 `ai:stream-done` 回调改为 async 函数以支持 await

# Task Dependencies
- 无依赖关系，两项任务可并行完成。
