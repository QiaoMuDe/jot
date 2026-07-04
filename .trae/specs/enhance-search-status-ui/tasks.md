# Tasks

- [x] Task 1: 后端 — `app.go` 扩展搜索状态事件
  - [x] 初始发射从 `"searching"` 改为 `"refining"`
  - [x] 精炼成功后、搜索前，发射 `ai:refined-keywords` 事件携带关键词字符串
  - [x] 精炼成功后、搜索前，发射 `ai:search-status` = `"searching"`
  - [x] 精炼失败时发射 error 事件终止流程（已有逻辑，不需要修改）

- [x] Task 2: 前端 — 搜索指示器改造 + 关键词下拉菜单
  - [x] 重写 `createAdvancedSearchIndicator`，接受状态参数（`refining` / `searching`）和可选关键词
  - [x] 精炼阶段：显示旋转图标 + "正在优化搜索词..."
  - [x] 搜索阶段：显示旋转图标 + "正在联网搜索..."
  - [x] 指示器可点击，点击展开/收起下拉菜单
  - [x] 下拉菜单中展示关键词标签（搜索阶段）或"正在提取搜索关键词..."（精炼阶段）
  - [x] 点击页面其他区域收起下拉菜单

- [x] Task 3: 前端 — 事件监听改造
  - [x] 新增 `ai:refined-keywords` 事件监听，存储关键词
  - [x] `ai:search-status` 事件处理扩展：支持 `refining` / `searching` / `done` 三态
  - [x] 首条 `ai:stream-chunk` 到达时移除搜索指示器（已有逻辑自动覆盖）

- [x] Task 4: CSS — 下拉菜单和指示器样式
  - [x] 搜索指示器容器样式（可点击、hover 效果）
  - [x] 下拉菜单样式（圆角、阴影、动画展开）
  - [x] 关键词标签样式

## Task Dependencies

- [Task 1] → [Task 2] （前端需要后端新事件才能联调）
- [Task 2] ← → [Task 4] （前端结构和样式可同步进行）
- [Task 3] 与 [Task 2] 合并实现
