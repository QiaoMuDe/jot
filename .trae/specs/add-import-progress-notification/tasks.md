# Tasks

- [x] Task 1: 后端 ImportFiles 发射进度事件
  - 导入前发射 `"import:progress"` + `"start"` 事件
  - 每个文件处理后发射 `"update"` 事件
  - 全部完成后发射 `"complete"` 事件
  - 使用 `sync.Mutex` 安全计数

- [x] Task 2: 后端 readAIChatFiles 发射进度事件
  - 上传前发射 `"import:ai-progress"` + `"start"` 事件
  - 每个文件处理后发射 `"update"` 事件
  - 全部完成后发射 `"complete"` 事件
  - 事件结构与 Task 1 一致，事件名不同

- [x] Task 3: NotificationManager 新增进度通知方法
  - 实现 `showProgress(prefix, total)` — 创建持久通知
  - 实现 `updateProgress(ctrl, current, total, title)` — 更新内容
  - 通知包含旋转动画图标、进度文字、无关闭按钮、不自动关闭
  - 添加函数级注释

- [x] Task 4: 前端 handleFileDropPaths 增加进度事件监听
  - 在调用 `ImportFiles` 前注册 `EventsOn("import:progress", ...)`
  - `start` 事件 → `nm.showProgress("正在导入", total)`
  - `update` 事件 → 防抖更新通知
  - `complete` 事件 → 关闭进度通知 + 显示汇总 + 刷新笔记列表 + 清理监听器
  - RPC 返回后兜底清理监听器

- [x] Task 5: 前端 AI 聊天文件上传增加进度事件监听
  - 在 AI 聊天相关代码中注册 `EventsOn("import:ai-progress", ...)`
  - `start` 事件 → `nm.showProgress("正在上传", total)`
  - `update` 事件 → 防抖更新通知
  - `complete` 事件 → 关闭进度通知 + 显示汇总 + 清理监听器

- [x] Task 6: 新增进度通知 CSS 样式
  - `.notification.progress` — 无关闭按钮、不自动消失
  - 旋转动画图标（@keyframes spin）

- [x] Task 7: 编译验证
  - `go build ./...` 无错误
  - `go vet ./...` 通过

## Task Dependencies
- [Task 3] 依赖于 [Task 6]（CSS 样式先于 JS 方法）
- [Task 4] 依赖于 [Task 1] 和 [Task 3]
- [Task 5] 依赖于 [Task 2] 和 [Task 3]
