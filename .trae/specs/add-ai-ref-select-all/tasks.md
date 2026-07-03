# Tasks

- [x] Task 1: 新增后端 SearchNoteIDs API — 按筛选条件返回所有匹配笔记 ID（不分页）
  - [x] 在 `internal/services/note_service.go` 新增 `SearchNoteIDs(keyword string, notebookID uint, tagIDs []uint) ([]uint, error)` 方法（跨笔记本搜索）
  - [x] 在 `internal/services/note_service.go` 新增 `SearchNoteIDsByNotebook` 方法（限定笔记本）
  - [x] 在 `app.go` 新增 `SearchNoteIDs(keyword string, notebookID uint, tagIDs []uint) ([]uint, error)` 绑定方法
  - [x] 验证：使用简单条件（无筛选=全部、仅笔记本、仅标签、组合）均能正确返回 ID 数组

- [x] Task 2: 前端 DOM + CSS — 全选按钮 UI
  - [x] 在 `index.html` 的 `.ai-note-ref-footer` 区域、计数文字左侧添加全选按钮 DOM（`.ai-note-ref-select-all`），包含复选框图标 + "全选" 文字
  - [x] 在 `ai-chat.css` 中添加全选按钮样式（inline-flex 布局、选中态/未选中态图标）

- [x] Task 3: 前端全选逻辑 — toggleSelectAll + 与筛选条件联动
  - [x] 实现 `toggleRefSelectAll()` 函数：根据当前筛选条件（`_refTagIds`、notebookId、搜索关键词）调用 `GetAllNoteIDs()` 或 `SearchNoteIDs()` 获取所有匹配 ID
  - [x] 将获取到的 ID 写入 `_refTempSelected`
  - [x] 实现取消全选：清空 `_refTempSelected`
  - [x] 绑定全选按钮点击事件
  - [x] 筛选条件变更时（切换笔记本/标签/搜索），重置全选按钮状态（但不清除已选 ID）

- [x] Task 4: 前端选中态联动 — renderNoteList / appendToList / updateRefCount 修改
  - [x] 在 `renderNoteList()` 和 `appendToList()` 中新增 `_refSelectAll` 状态判断：若为 true 则新渲染/追加的条目自动选中
  - [x] 追加条目时若 `_refSelectAll === true`，自动将新加载笔记的 ID 加入 `_refTempSelected`
  - [x] `updateRefCount()` 中增加全选按钮的勾选状态同步（手动逐条选完所有切换为勾选，手动取消任一切换为未勾选）
  - [x] `toggleNoteSelection()` 中增加全选状态同步：手动取消任一条时取消全选勾选

# Task Dependencies
- [Task 2] 独立
- [Task 3] 依赖 [Task 1]
- [Task 4] 依赖 [Task 2]、[Task 3]
