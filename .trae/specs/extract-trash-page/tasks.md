# Tasks

- [x] Task 1: 创建 `trash-page.js` 模块文件：将回收站所有功能（状态、DOM 引用、函数）封装到独立模块中，并通过 `window` 暴露给模板 onclick 调用
  - [x] 导出 `trashNotes` 状态（或通过模块内部管理）
  - [x] 导出回收站 DOM 引用
  - [x] 提取 `loadTrashNotes`、`restoreNote`、`restoreAllNotes`、`emptyTrash`、`permanentDeleteNote`、`renderTrashList` 函数
  - [x] 将 `restoreNote`、`permanentDeleteNote`、`restoreAllNotes`、`emptyTrash` 挂载到 `window` 上
  - [x] 导入所需依赖（`SVGS`、`nm`、`formatTime`、`escapeHtml`、`showConfirmDialog`、`loadNotes`、`loadNotebooks` 等）

- [x] Task 2: 修改 `main.js`：移除回收站代码并从新模块导入
  - [x] 移除 `state.trashNotes`
  - [x] 移除 `els` 中的回收站 DOM 引用（保留在 `els` 对象中供 `trash-page.js` 使用）
  - [x] 移除 6 个回收站函数定义（含旧的 window wrapper）
  - [x] 添加 `import` 语句引入 `trash-page.js`
  - [x] 清理 `switchView` 中的 trash case（改为调用模块函数）
  - [x] 清理 visibilitychange 中的 trash 刷新逻辑
  - [x] 将 `undoDelete` 暴露到 `window`

- [x] Task 3: 验证功能完整性：确保所有回收站功能正常工作
  - [x] 打开回收站视图（菜单/Ctrl+5）加载笔记列表
  - [x] 单条恢复笔记
  - [x] 全部恢复笔记
  - [x] 单条永久删除
  - [x] 全部清空
  - [x] 返回按钮退回网格视图
  - [x] 统计面板数据更新
  - [x] 确认对话框正常显示

# Task Dependencies
- Task 1 → Task 2 → Task 3
