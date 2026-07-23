# Tasks

- [x] Task 1: 后端新增两个查询接口 — 在 `note_service.go` 实现 `GetByDate` 和 `GetMonthCounts`，在 `app.go` 添加对应 Wails 绑定方法
  - [ ] 1.1 `note_service.go`: 新增 `GetByDate(db, date string)` — 查询指定创建日期的非删除笔记，按 `created_at DESC` 排序
  - [ ] 1.2 `note_service.go`: 新增 `GetMonthCounts(db, year, month int)` — 按天统计某月非删除笔记数，返回 `map[int]int`
  - [ ] 1.3 `app.go`: 新增 `GetNotesByDate(date string)` 绑定方法
  - [ ] 1.4 `app.go`: 新增 `GetMonthNoteCounts(year, month int)` 绑定方法

- [x] Task 2: 前端 HTML 结构 — 在 `index.html` 新增日历视图容器和更多菜单入口
  - [ ] 2.1 新增 `viewCalendar` 容器（含 view-header 返回按钮 + 双栏布局）
  - [ ] 2.2 更多菜单新增"日历"菜单项（待办清单上方）

- [x] Task 3: 前端 JS 视图注册 — 在 `main.js` 注册日历视图
  - [ ] 3.1 `els` 对象新增 `viewCalendar` 引用
  - [ ] 3.2 `viewMap` 新增 `calendar` 映射
  - [ ] 3.3 `switchView` 的更多菜单 handler 新增 `calendar` 分支
  - [ ] 3.4 视图初始化：`init()` 中调用 `initCalendarView()`（从 `calendar.js` import）

- [x] Task 4: 日历 JS 模块（核心） — 创建 `frontend/src/js/calendar.js`
  - [ ] 4.1 月份渲染函数：根据年月生成 6x7 日期网格，标记今天/非本月日期
  - [ ] 4.2 墨水圆点渲染：根据 `GetMonthNoteCounts` 结果在日期格下方绘制圆点
  - [ ] 4.3 月份切换：◀ ▶ 按钮控制月份前后切换，重新请求 `GetMonthNoteCounts`
  - [ ] 4.4 日期点击：选中高亮 + 调用 `GetNotesByDate` 加载右侧列表
  - [ ] 4.5 笔记列表渲染：显示标题、笔记本名、创建时间，点击触发跳转逻辑
  - [ ] 4.6 空状态：无笔记时显示"这天还没有笔记"
  - [ ] 4.7 跳转函数：关闭编辑器 → `switchView('grid')` → 设置笔记本 → `loadNotes()` → `openEditor()`
  - [ ] 4.8 初始化入口：暴露 `initCalendarView()` 函数供 `main.js` 调用

- [x] Task 5: 日历 CSS 样式 — 创建 `frontend/src/css/components/calendar.css`
  - [ ] 5.1 双栏布局：左侧日历 ~280px 固定，右侧笔记列表 `flex: 1`
  - [ ] 5.2 日历网格：CSS Grid 7 列，日期格 36x36px 交互区域
  - [ ] 5.3 墨水圆点：`::after` 伪元素或独立 span 实现，三档大小/透明度
  - [ ] 5.4 选中态：accent 色圆形背景 + 白色文字
  - [ ] 5.5 笔记卡片：极简卡片样式，标题/笔记本/时间信息
  - [ ] 5.6 入场动画：笔记列表 item 逐条 fadeIn + translateY
  - [ ] 5.7 空状态：居中居中，柔和图标

# Task Dependencies

- [Task 1] 无依赖
- [Task 2] 无依赖
- [Task 3] 依赖 [Task 2] (HTML 容器必须先存在)
- [Task 4] 依赖 [Task 1] (后端接口必须先可用)
- [Task 5] 依赖 [Task 2] (HTML 结构必须先存在)
