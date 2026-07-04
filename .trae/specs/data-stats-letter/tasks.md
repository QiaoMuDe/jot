# Tasks

- [x] Task 1: 后端 — DataStats 新增字段 + AIService 聚合查询方法
  - `internal/services/types.go`: DataStats 新增 `TotalTokens int64`、`AvgResponseTime float64`、`AvgThinkingTime float64`、`MaxResponseTime float64`
  - `internal/services/ai_service.go`: 新增 `SumTokens()`、`AvgResponseTime()`、`AvgThinkingTime()`、`MaxResponseTime()` 四个方法，使用 GORM `Select` + `COALESCE` 确保空表返回零值

- [x] Task 2: 后端 — GetDataStats 接入 AI 性能统计
  - `app.go` 中 `GetDataStats()` 在现有 AI 统计代码后调用 4 个新方法并赋值

- [x] Task 3: 前端 — HTML 信纸结构
  - `frontend/index.html`: 移除旧 `.data-stats` 网格 + 7 张 `.stat-card`
  - 替换为 `.data-letter` 信纸容器，包含信头、正文容器、落款

- [x] Task 4: 前端 — CSS 信纸样式
  - `frontend/src/css/components/data-view.css`:
    - 移除 `.data-stats`、`.stat-card`、`.stat-value`、`.stat-label`、`statCardShake`、`cardEnter` 等旧样式
    - 新增 `.data-letter`、`.letter-header`、`.letter-body`、`.letter-divider`、`.letter-footer`、`.letter-stars` 样式
    - 新增 `@keyframes letterReveal` 入场动画
    - 新增 `.data-letter-empty` 空数据占位样式

- [x] Task 5: 前端 — JS 逻辑重写
  - `frontend/src/main.js`: 更新 `els`，移除 7 个旧 stat 引用，新增 `dataLetter`、`letterDate`、`letterBody`、`letterFooter`
  - `frontend/src/js/data-management.js`: `loadDataStats()` 重写为信纸风格（innerHTML 拼接、星级评价、letterReveal 动画），移除 `_statShakeInited` 和抖动事件

# Task Dependencies

- [Task 1] 必须先于 [Task 2]
- [Task 2] 必须先于 [Task 5]（JS 需要确认后端接口字段名）
- [Task 3] 必须先于 [Task 4] 和 [Task 5]（HTML DOM 结构必须先就位）
- [Task 4] 和 [Task 5] 可并行执行（CSS 与 JS 互不依赖，但都依赖 HTML 结构）
