# Tasks
- [x] Task 1: 数据模型新增耗时字段
  - [x] `internal/models/ai_message.go`: 新增 `ThinkingElapsed float64` 和 `TotalElapsed float64` 字段，GORM 自动迁移
  - [x] `internal/services/ai_service.go`: `Message` 结构体新增 `ThinkingElapsed float64` 和 `TotalElapsed float64`
- [x] Task 2: 后端计时逻辑 + `CallAIStream` 回调变更
  - [x] `internal/services/ai_service.go`: `CallAIStream` 的 `onDone` 回调签名增加 `elapsedThinking float64, elapsedTotal float64` 参数
  - [x] `internal/services/ai_service.go`: `CallAIStream` 中记录 `streamStart`、首个 `reasoning_content` 时间、首个 `content` 时间，流式结束时计算两个耗时传给 `onDone`
  - [x] `app.go`: `CallAIStream` 绑定中适配新的 `onDone` 签名，将耗时通过 `ai:stream-done` 事件额外参数传给前端
- [x] Task 3: `SaveAIMessages` + `LoadAISessionMessages` 支持耗时字段
  - [x] `internal/services/ai_service.go`: `SaveAIMessages` 接收并写入 `ThinkingElapsed` 和 `TotalElapsed`
  - [x] `internal/services/ai_service.go`: `LoadAISessionMessages` 返回 `ThinkingElapsed` 和 `TotalElapsed`
- [x] Task 4: 前端思维链耗时展示
  - [x] `frontend/src/js/ai-chat.js`: `startStreaming()` 中读取 `ai:stream-done` 事件的 `thinkingElapsed` 参数，更新 thinking summary 为 `💭 已思考 X.X 秒`
  - [x] `frontend/src/js/ai-chat.js`: `addMessage()` 中当加载历史消息且 `thinkingElapsed > 0` 时，显示 `💭 已思考 X.X 秒`
- [x] Task 5: 前端回复耗时展示
  - [x] `frontend/src/js/ai-chat.js`: `startStreaming()` 的 `ai:stream-done` 回调中，在 `.msg-content` 后插入 `<div class="ai-msg-time">⏱ 总耗时 X.X 秒</div>`
  - [x] `frontend/src/js/ai-chat.js`: `addMessage()` 中当 `totalElapsed > 0` 时，在 `.msg-content` 后插入耗时元素
  - [x] `frontend/src/css/components/ai-chat.css`: 新增 `.ai-msg-time` 样式（字号 0.75rem、颜色 var(--text-muted)、顶部间距 6px、居右对齐）

# Task Dependencies
- Task 1 无依赖，可最先做
- Task 2 依赖 Task 1（Message 结构体字段）
- Task 3 依赖 Task 1（模型字段）
- Task 4 依赖 Task 2（事件参数变更），可和 Task 5 并行
- Task 5 依赖 Task 2 + Task 3（事件参数 + 历史数据加载），CSS 部分无依赖可提前做