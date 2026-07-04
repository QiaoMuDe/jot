# Tasks

- [ ] Task 1: 后端 — DataStats 新增 AI 性能字段 + AIService 聚合查询方法
  - `internal/services/types.go`: DataStats 新增 `TotalTokens int64`、`AvgResponseTime float64`、`AvgThinkingTime float64`、`MaxResponseTime float64`
  - `internal/services/ai_service.go`: 新增 `SumTokens()`、`AvgResponseTime()`、`AvgThinkingTime()`、`MaxResponseTime()` 四个方法
  - 使用 GORM 原生 SQL 或 `Select` 聚合，COALESCE 确保空表返回零值

- [ ] Task 2: 后端 — GetDataStats 接入 AI 性能统计
  - `app.go` 中 `GetDataStats()` 在现有 AI 统计代码后调用 4 个新方法并赋值

- [ ] Task 3: 前端 — HTML 结构调整为两个分组 Section
  - `frontend/index.html`: 将现有 7 张 `.stat-card` 拆分为两个 `.data-section`:
    - Section 1 "笔记与存储": 笔记总数、标签总数、回收站、笔记本数、数据库大小（5 张卡）
    - Section 2 "AI 统计数据": AI 会话、AI 消息 + 新增 4 张 AI 性能卡（6 张卡）
  - 每个 Section 前添加 `<h3 class="data-section-title">` 标题

- [ ] Task 4: 前端 — 新增 DOM 引用 + loadDataStats 适配
  - `frontend/src/main.js`: els 新增 `statTotalTokens`、`statAvgResponseTime`、`statAvgThinkingTime`、`statMaxResponseTime` 四个 DOM 引用
  - `frontend/src/js/data-management.js`: `loadDataStats()` 中:
    - 从 stats 对象读取 4 个新字段
    - 重置阶段将 4 个新元素置初始值
    - 入场动画计算卡数量时考虑两组卡片的总和
    - 数字递增动画阶段处理 Token 的千分位格式和时间值的 `X.Xs` 格式
    - 两组卡片各自触发交错入场动画（`cardEnter`），取最晚结束时间启动 count-up

- [ ] Task 5: 前端 — CSS 分组网格布局
  - `frontend/src/css/components/data-view.css`:
    - 移除旧的 `.data-stats` 的 `grid-template-columns: repeat(7, 1fr)`
    - 笔记与存储网格: `.data-section-note-stats .data-stats { grid-template-columns: repeat(5, 1fr); }`
    - AI 统计网格: `.data-section-ai-stats .data-stats { grid-template-columns: repeat(3, 1fr); }`
    - 窄屏响应式 ≤640px 两组都切为 `repeat(2, 1fr)`

# Task Dependencies

- [Task 1] 必须先于 [Task 2]
- [Task 2] 必须先于 [Task 3]（前端需要确认后端接口字段名）
- [Task 4] 依赖 [Task 3]（HTML DOM 结构必须先就位）
- [Task 5] 无依赖，可与其他任务并行