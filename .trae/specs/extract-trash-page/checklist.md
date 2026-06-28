# 提取回收站页面模块 — 验证清单

- [x] `trash-page.js` 文件创建成功，包含所有回收站逻辑
- [x] `window.restoreNote`、`window.permanentDeleteNote`、`window.restoreAllNotes`、`window.emptyTrash` 在全局可访问
- [x] `main.js` 中已移除 `state.trashNotes`、相关 DOM 引用和 6 个回收站函数
- [x] `main.js` 通过 import 引入 `trash-page.js`
- [x] 回收站视图可通过更多菜单正常打开
- [x] 回收站视图可通过 Ctrl+5 快捷键打开
- [x] 单条恢复笔记功能正常
- [x] 全部恢复笔记功能正常
- [x] 单条永久删除功能正常
- [x] 全部清空功能正常
- [x] 返回按钮正常退回网格视图
- [x] 切换视图时（visibilitychange）回收站数据可刷新
- [x] 统计面板 `statTrashedNotes` 值可更新
- [x] 应用无控制台报错（构建成功 ✅）
