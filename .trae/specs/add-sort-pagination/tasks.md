# Tasks

- [x] Task 1: 后端支持动态排序
  - [x] 修改 `note_service.go:GetAll()` 增加 `sortBy string` 参数
  - [x] 修改 `note_service.go:GetByTag()` 增加 `sortBy string` 参数
  - [x] 修改 `app.go:GetNotes()` 和 `GetNotesByTag()` 透传 `sortBy`
  - [x] 新增 `app.go:GetSortOrder()` / `SetSortOrder()` 绑定方法（读写 Setting `sort_order`）
  - [x] 新增 `app.go:GetPageSize()` / `SetPageSize()` 绑定方法（读写 Setting `page_size`）

- [x] Task 2: 设置页添加排序选项和分页大小滑块
  - [x] `index.html` 设置页新增「笔记排序」分区（三个 radio 选项）+「分页大小」滑块（`input[type=range]`，min=9, max=54, step=9）
  - [x] `style.css` 添加排序选项 + 滑块样式（与现有设置项风格一致）
  - [x] `main.js` 加载已保存排序和分页大小、切换事件、保存设置；滑块拖动时实时显示数值

- [x] Task 3: 前端实现懒加载
  - [x] `main.js` 新增分页状态变量：`page`, `totalNotes`, `loading`, `hasMore`
  - [x] `loadNotes()` 改为加载第 1 页（按用户设置的分页大小），重置所有分页状态
  - [x] 新增 `loadMoreNotes()` 加载下一页并追加到 `state.notes`
  - [x] 卡片网格容器绑定 scroll 事件，检测到距底部 <200px 时触发加载下一页
  - [x] 底部显示加载状态 / 「共 X 条笔记」提示

- [x] Task 4: CRUD 后刷新适配
  - [x] 创建/更新/删除/置顶/恢复/回收站操作后的 `loadNotes()` 保持重置分页行为（Task 3 已覆盖）

- [x] Task 5: 验证
  - [x] `go build ./...` 编译通过
  - [x] 切换排序 → 列表即时刷新
  - [x] 拖动分页大小滑块 → 显示数值 + 列表即时刷新
  - [x] 滚动到底部 → 自动加载更多
  - [x] 全部加载完 → 显示总数提示
  - [x] CRUD 操作后 → 重置分页只加载第 1 页
