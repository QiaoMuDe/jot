# Tasks

## Task 1: 后端 — 扩展 SearchNotes API 增加筛选参数

在 `internal/services/note_service.go` 和 `app.go` 中修改搜索方法，支持 `scope`、`days`、`tagID` 三个筛选参数。

- [ ] SubTask 1.1: 修改 `NoteService.Search()` 方法，增加 `scope`, `days`, `tagID` 参数
  - scope="title" 时仅匹配 title LIKE
  - scope="content" 时仅匹配 content LIKE
  - scope="all" 时匹配 title + content + tag（原有逻辑）
  - days>0 时增加 `updated_at >= NOW() - days 天` 条件
  - tagID>0 时增加 `JOIN note_tags ON note_tags.note_id = notes.id WHERE note_tags.tag_id = ?` 条件
- [ ] SubTask 1.2: 修改 `NoteService.SearchByNotebook()` 方法，同样增加三个参数
- [ ] SubTask 1.3: 修改 `App.SearchNotes()` 方法签名，暴露 `scope`, `days`, `tagID` 给前端

## Task 2: 后端 — 新增 GetNotebookTags API

新增后端接口，获取指定笔记本下所有笔记关联的标签列表。

- [ ] SubTask 2.1: 在 `NoteService` 中新增 `GetNotebookTags(notebookID uint) ([]models.Tag, error)` 方法
  - 通过 `note_tags` 关联表和 `notes` 表联合查询
  - 返回去重后的标签列表
- [ ] SubTask 2.2: 在 `App` 中暴露 `GetNotebookTags(notebookID uint)` 方法给前端

## Task 3: 前端 — 搜索视图新增筛选栏 HTML

在 `frontend/index.html` 的 `#viewSearch` 中新增筛选栏 HTML 结构。

- [ ] SubTask 3.1: 在 `#viewSearch` 的 `.view-header` 和 `#searchResults` 之间插入 `.search-filter-bar`
  - 包含：搜索范围下拉、时间筛选下拉、标签筛选下拉、重置按钮

## Task 4: 前端 — 筛选栏样式

在 `frontend/src/style.css` 中新增筛选栏相关样式。

- [ ] SubTask 4.1: 编写 `.search-filter-bar` 容器样式（水平排列、间距、背景色、圆角、阴影）
- [ ] SubTask 4.2: 编写 `.filter-group` 标签样式（标签文本+下拉框，紧凑布局）
- [ ] SubTask 4.3: 编写 `.filter-select` 下拉框样式（与温暖极简主题一致）
- [ ] SubTask 4.4: 编写 `.filter-reset-btn` 重置按钮样式

## Task 5: 前端 — 筛选逻辑实现

在 `frontend/src/main.js` 中实现筛选交互逻辑。

- [ ] SubTask 5.1: 在 `state` 中添加 `searchScope`（默认 `'all'`）、`searchDays`（默认 `0`）、`searchTagID`（默认 `0`）
- [ ] SubTask 5.2: 修改 `searchNotes()` 函数，读取 state 中的筛选参数传递给后端 API
- [ ] SubTask 5.3: 绑定三个筛选下拉的 `change` 事件，更新 state 并自动触发搜索（防抖 300ms）
- [ ] SubTask 5.4: 实现重置按钮逻辑（重置三个筛选为默认值并重新搜索）
- [ ] SubTask 5.5: 切换笔记本时重置所有筛选为默认值
- [ ] SubTask 5.6: 退出搜索视图（返回网格）时重置所有筛选为默认值
- [ ] SubTask 5.7: 进入搜索视图时调用 `GetNotebookTags` 加载标签选项（如果当前笔记本有筛选）

## Task 6: 后端 — wails.json 重新绑定

如果后端新增了方法，需要重新生成绑定。

- [ ] SubTask 6.1: 运行 `wails generate module` 或构建项目以确保前端能调用新 API

# Task Dependencies

- [Task 1] 和 [Task 2] 可以并行开发
- [Task 3]、[Task 4]、[Task 5] 依赖 [Task 1] 和 [Task 2] 完成
- [Task 6] 在 [Task 1] 和 [Task 2] 之后执行