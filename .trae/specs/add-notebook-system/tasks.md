# Tasks - 笔记本层级分组系统

- [x] Task 1: Notebook 数据模型 + 数据库迁移
  - 新建 `internal/models/notebook.go`（Notebook 结构体）
  - `internal/models/note.go` 新增 `NotebookID uint` 字段 (`gorm:"default:0;index"`)
  - `database/db.go` AutoMigrate 追加 `&models.Notebook{}`

- [x] Task 2: NotebookService CRUD + 默认笔记本初始化
  - 新建 `internal/services/notebook_service.go`
  - `Create(name string) (*models.Notebook, error)` — 创建
  - `Update(id uint, name string) (*models.Notebook, error)` — 重命名
  - `Delete(id uint) error` — 删除，将其下笔记 notebook_id 改为默认笔记本 id
  - `GetAll() ([]models.Notebook, error)` — 全部列表（不含已删除）
  - `GetAllNotesCount(db *gorm.DB) (map[uint]int, error)` — 各笔记本笔记数
  - `EnsureDefaultNotebook() error` — 首次启动检查，空表则创建「默认笔记本」+ 迁移存量笔记

- [x] Task 3: NoteService 扩展笔记本筛选
  - `GetByNotebook(notebookID uint, page, pageSize int, sortBy string) ([]models.Note, int64, error)` — 按笔记本分页查询
  - `GetAllNoteIDsByNotebook(notebookID uint) ([]uint, error)` — 笔记本内全部 ID
  - `MigrateOrphanNotesToDefault() error` — 将 notebook_id=0 的笔记迁移到默认笔记本

- [x] Task 4: App.go 绑定 + NotebookService 初始化
  - 新增 `NotebookService` 字段 + 初始化
  - 新增 6 个绑定方法: `CreateNotebook`, `RenameNotebook`, `DeleteNotebook`, `GetAllNotebooks`, `GetNotebookNoteCounts`
  - 应用启动时调用 `NotebookService.EnsureDefaultNotebook()` + `MigrateOrphanNotesToDefault()`
  - CreateNote 存储 notebookID（来自前端 activeNotebookId）
  - GetNotes/Search 支持 notebookID 筛选参数

- [x] Task 5: 前端笔记本侧栏 HTML + CSS
  - `index.html` 新增 `.notebook-sidebar` 面板（标题/笔记本列表/新建按钮/折叠按钮）
  - `style.css` 侧栏样式（展开 200px / 折叠 44px + 高亮激活态 + 笔记数 badge + 右键菜单）
  - 主内容区 `.main-layout` 调整为 `display: flex` 配合侧栏

- [x] Task 6: 前端笔记本状态管理 + 内容筛选
  - `main.js` 新增 `state.activeNotebookId`（当前笔记本 id）
  - `state.notebooks` 数组存储笔记本列表
  - `loadNotes()`/`searchNotes()` 始终传参 `notebookID` 筛选（无「全部」模式）
  - 侧栏点击切换 → 重置页码 → 重新加载笔记网格
  - 新建笔记 `createNote()` 传入当前 `activeNotebookId`
  - 应用初始化时加载笔记本列表 + 默认选中默认笔记本
  - ESC 返回首页不切换笔记本

- [x] Task 7: 前端笔记本 CRUD 交互
  - 新建笔记本按钮 → 弹窗 → 调用后端 → 刷新侧栏 → 自动切换到该笔记本
  - 右键笔记本菜单（重命名/删除）
  - 重命名：内联编辑（input）→ 回车/失焦保存
  - 删除：确认弹窗 → 后端删除 → 若删除的是当前激活笔记本则自动切到默认笔记本 → 刷新侧栏
  - 「默认笔记本」右键菜单无「删除」选项

- [x] Task 8: 侧栏折叠状态
  - 折叠/展开切换 + `localStorage` 持久化

- [x] Task 9: Seed 工具更新
  - `tools/seed/main.go` 创建 5 个笔记本，笔记按 notebook_id 分配到各笔记本

## Task Dependencies
- Task 1 → Task 2 → Task 4
- Task 1 → Task 3 → Task 4
- Task 4 → Task 5, Task 6, Task 7
- Task 6, Task 7 可并行实现
- Task 8 无依赖
- Task 9 依赖 Task 1
