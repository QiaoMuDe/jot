# Tasks - 笔记跨笔记本迁移

- [x] Task 1: 后端笔记迁移方法
  - `internal/services/note_service.go` 新增：
    - `MoveToNotebook(noteID uint, targetNotebookID uint) error` — 单条迁移，校验 note 和 notebook 存在性，`UpdateColumn("notebook_id", targetNotebookID)` 不更新时间
    - `BatchMoveToNotebook(noteIDs []uint, targetNotebookID uint) error` — 批量迁移，遍历 noteIDs 逐个更新，失败返回错误但不回滚已迁移的
  - `app.go` 新增：
    - `MoveNoteToNotebook(id, targetNotebookID uint) error` — 绑定
    - `BatchMoveNotesToNotebook(noteIDs []uint, targetNotebookID uint) error` — 绑定

- [x] Task 2: 前端选择器弹窗 HTML + CSS
  - `index.html` 新增 `#moveNotebookDialog` 弹窗结构
  - `style.css` 新增弹窗样式（遮罩 blur、卡片圆角阴影、radio 自定义、入场弹簧动画）
  - 使用 CSS 变量 + `color-mix()` 实现 6 主题自适应

- [x] Task 3: 前端选择器逻辑 + 迁移函数
  - `openMoveDialog(noteIds)` — 打开选择器，加载笔记本列表（过滤当前笔记本），渲染 radio 列表
  - `closeMoveDialog()` — 关闭弹窗
  - `confirmMoveNotes()` — 执行迁移，调用后端 API，关闭弹窗，Toast 通知
  - 迁移后：`loadNotes()` + `loadNotebooks()` + `renderNotebookList()`

- [x] Task 4: 右键菜单 + 批量工具栏入口
  - 右键菜单在「编辑」和「置顶」之间新增「移动到...」
  - 批量栏在「-标签」和「批量删除」之间新增「移动到」按钮
  - `handleContextAction` 新增 `'move'` 分支
  - `batchMoveBtn` 事件绑定

- [x] Task 5: 验证编译通过
  - `go vet ./...` 无错误
