# Tasks

- [x] Task 1: 后端 NoteService 新增按标题匹配查询方法
  - [x] SubTask 1.1: 在 `note_service.go` 中新增 `FindByTitleAndExt(title, fileExt string, notebookID uint) (*models.Note, error)` 方法，查询条件：`title = ? AND file_ext = ? AND notebook_id = ? AND deleted_at IS NULL`，返回匹配的第一条笔记
  - [x] SubTask 1.2: 编写单元测试验证查询逻辑（有匹配/无匹配/不同笔记本不匹配）

- [x] Task 2: 后端 FileImportResult 扩展状态字段
  - [x] SubTask 2.1: 在 `app.go` 的 `FileImportResult` 结构体中新增 `Status string`、`FileTime int64`、`NoteTime int64` 字段
  - [x] SubTask 2.2: 在现有导入逻辑中为"新建"场景设置 `Status: "created"`

- [x] Task 3: 后端 processImportFile 改造导入逻辑
  - [x] SubTask 3.1: 在 `processImportFile` 中，文件读取成功后、创建笔记前，调用 `FindByTitleAndExt` 查找已有笔记
  - [x] SubTask 3.2: 找到匹配笔记时，用 `info.ModTime()` 与笔记 `UpdatedAt` 对比（精确到秒）
  - [x] SubTask 3.3: 文件更新 → 调用现有 `Update` 方法覆盖，返回 `status: "updated"`
  - [x] SubTask 3.4: 笔记更新 → 返回 `status: "conflict"`，附带 `FileTime` 和 `NoteTime`
  - [x] SubTask 3.5: 时间相同 → 返回 `status: "skipped"`

- [x] Task 4: 后端新增 ResolveImportConflict 方法
  - [x] SubTask 4.1: 在 `app.go` 中新增 `ResolveImportConflict(noteID uint, overwrite bool, title, content, fileExt string) FileImportResult`
  - [x] SubTask 4.2: `overwrite=true` 时调用 `noteService.Update` 覆盖内容
  - [x] SubTask 4.3: `overwrite=false` 时直接返回成功（跳过）

- [x] Task 5: 前端新增冲突解决弹窗 `showImportConflictDialog`
  - [x] SubTask 5.1: 新增 `showImportConflictDialog(conflicts, onComplete)` 函数，接收冲突结果列表和完成回调
  - [x] SubTask 5.2: 弹窗顶部显示"发现 N 个冲突文件"，提供"全部覆盖"和"全部保留"快捷按钮
  - [x] SubTask 5.3: 列表每项显示：文件名、笔记标题、笔记时间、文件时间，以及"覆盖"和"保留"两个操作按钮
  - [x] SubTask 5.4: 用户点击单个项的按钮后，调用 `App.ResolveImportConflict`，该项从列表中移除
  - [x] SubTask 5.5: 点击"全部覆盖"时，批量调用所有冲突项的 `ResolveImportConflict(noteID, true)`
  - [x] SubTask 5.6: 点击"全部保留"时，批量调用所有冲突项的 `ResolveImportConflict(noteID, false)`
  - [x] SubTask 5.7: 全部项处理完后弹窗自动关闭，调用 `onComplete` 回调

- [x] Task 6: 前端导入结果处理和流程适配
  - [x] SubTask 6.1: 改造 `showImportResults`，将 `status === "conflict"` 的结果收集为冲突列表
  - [x] SubTask 6.2: 有冲突时调用 `showImportConflictDialog`，无冲突时直接显示结果通知
  - [x] SubTask 6.3: 冲突弹窗关闭后，统一刷新笔记列表和侧边栏
  - [x] SubTask 6.4: 处理 `status === "updated"` 和 `status === "skipped"` 的通知文案

# Task Dependencies

- Task 2 依赖 Task 1（需要查询方法来判断状态）
- Task 3 依赖 Task 1 和 Task 2
- Task 4 依赖 Task 1
- Task 5 依赖 Task 2 和 Task 4
- Task 6 依赖 Task 5
