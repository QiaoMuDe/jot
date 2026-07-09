# Checklist

- [x] Go 后端：`CleanupOrphanImages()` API 实现（扫描目录 → 查数据库引用 → 删除孤儿）
- [x] Go 后端：已删除/回收站笔记的图片引用计入保护范围
- [x] Go 后端：目录不存在/空目录时返回 0 不报错
- [x] 前端：数据管理页"数据清理"分组新增"清理未引用图片"按钮 DOM
- [x] 前端：`main.js` 中按钮事件绑定
- [x] 前端：`cleanupOrphanImages()` 函数实现（确认弹窗 → 调用 API → 结果显示）
- [x] 前端：存储优化按钮联动（VACUUM 后自动执行图片清理，合并结果提示）
- [x] 编译验证：Go build + Vite build 通过
