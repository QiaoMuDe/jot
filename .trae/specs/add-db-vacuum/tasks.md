# Tasks

- [x] Task 1: 后端 `note_service.go` 添加 `Vacuum()` 方法
  - 在 `internal/services/note_service.go` 中新增 `Vacuum()` 方法，调用 `s.db.Exec("VACUUM")`
  - 位于 `ExportBackup` 方法附近，保持代码组织一致
- [x] Task 2: 后端 `app.go` 添加 `VacuumDatabase()` 绑定方法
  - 在 `app.go` 中新增 `VacuumDatabase()` 方法
  - 执行前读取数据库文件大小，执行 `a.noteService.Vacuum()`，再读取执行后大小
  - 计算释放空间并格式化返回提示消息
- [x] Task 3: 前端 `index.html` 添加"数据库瘦身"按钮
  - 在"数据操作"区域的 `data-actions` div 内，`resetAllBtn` 按钮之后插入新按钮
  - 按钮 id 为 `vacuumDbBtn`，使用压缩/清理相关 SVG 图标
  - 文本："数据库瘦身"，描述："回收已删除数据占用的磁盘空间"
- [x] Task 4: 前端 `main.js` 添加函数和事件绑定
  - 在 `els` 对象中添加 `vacuumDbBtn: $('vacuumDbBtn')`
  - 新增 `vacuumDatabase()` 异步函数
  - 在数据管理按钮事件绑定区域添加 `els.vacuumDbBtn.addEventListener('click', vacuumDatabase)`

# Task Dependencies

- [Task 1] → [Task 2]: App 的 VacuumDatabase 依赖 NoteService 的 Vacuum 方法
- [Task 2] → [Task 3, Task 4]: 前端按钮和事件必须先有后端方法可用
- [Task 3] → [Task 4]: 事件绑定需要 DOM 元素存在