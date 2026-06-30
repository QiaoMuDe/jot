# Tasks

- [x] Task 1: 后端 - NotebookService 新增回收站方法
  - [x] SubTask 1.1: 修改 `DeleteWithNotes` 将笔记软删除（进回收站）而非硬删除
  - [x] SubTask 1.2: 新增 `GetTrash(page, pageSize)` 获取回收站中已删除的笔记本列表
  - [x] SubTask 1.3: 新增 `RestoreFromTrash(id)` 恢复回收站中指定笔记本
  - [x] SubTask 1.4: 新增 `PermanentDeleteFromTrash(id)` 永久删除回收站中指定笔记本（同时将引用此 notebook_id 的回收站笔记迁到默认笔记本）
  - [x] SubTask 1.5: 新增 `RestoreAllFromTrash()` 恢复回收站中所有笔记本
  - [x] SubTask 1.6: 新增 `EmptyTrash()` 清空回收站中所有笔记本（先迁移引用此 notebook_id 的回收站笔记到默认笔记本，再硬删除笔记本）

- [x] Task 2: 后端 - App 新增绑定方法
  - [x] SubTask 2.1: 新增 `GetTrashNotebooks(page, pageSize)` 绑定
  - [x] SubTask 2.2: 新增 `RestoreTrashNotebook(id)` 绑定
  - [x] SubTask 2.3: 新增 `PermanentDeleteTrashNotebook(id)` 绑定
  - [x] SubTask 2.4: 新增 `RestoreAllTrashNotebooks()` 绑定
  - [x] SubTask 2.5: 新增 `EmptyTrashNotebooks()` 绑定

- [x] Task 3: 前端 - 重构回收站页面渲染逻辑
  - [x] SubTask 3.1: 修改 `loadTrashNotes()` 改为同时加载笔记和笔记本，管理两种条目的状态
  - [x] SubTask 3.2: 修改 `renderTrashList()` 混合渲染笔记和笔记本条目，区分图标和操作
  - [x] SubTask 3.3: 新增 `restoreNotebook(id)` 恢复笔记本
  - [x] SubTask 3.4: 新增 `permanentDeleteNotebook(id)` 永久删除笔记本
  - [x] SubTask 3.5: 修改 `restoreAllNotes()` → `restoreAllTrashItems()` 同时恢复笔记和笔记本
  - [x] SubTask 3.6: 修改 `emptyTrash()` 同时清空笔记和笔记本

- [x] Task 4: 前端 - 笔记本侧栏删除对话框文案更新
  - [x] SubTask 4.1: 将 "同时永久删除该笔记本中的 N 条笔记（不进回收站）" 改为 "同时将该笔记本中的 N 条笔记移入回收站"

- [x] Task 5: 前端 - 回收站条目 CSS 样式
  - [x] SubTask 5.1: 新增笔记本条目的样式（不同图标/徽章、笔记本名称样式）

# Task Dependencies
- Task 1, Task 2 → Task 3 (后端完成后再改前端)
- Task 1, Task 2 可并行
- Task 4 无需依赖
- Task 5 与 Task 3 可并行