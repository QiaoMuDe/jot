# Checklist

- [x] data-management.js 文件已创建，包含 10 个函数
- [x] 所有 10 个函数通过 ES module `export` 暴露（main.js 通过 `import` 引用）
- [x] data-management.js 通过函数内 `const { ... } = window;` 解构引用外部依赖（nm, els, SVGS, state, showConfirmDialog, loadNotes, loadTags, loadNotebooks, switchView）
- [x] main.js 中已移除数据管理函数定义，替换为注释
- [x] main.js 中事件绑定中的函数引用保持不变（exportData, importData, backupToDir, restoreFromDir, resetDatabase, vacuumDatabase, openDataDir）
- [x] main.js 中 switchView 的 case 'data' 调用 loadDataStats() 保持不变
- [x] 通过 ES module `import` 机制自动处理加载顺序（data-management.js 作为依赖先于 main.js 执行）
- [x] main.js 底部添加了 window 导出（els, nm, SVGS, state, showConfirmDialog, loadNotes, loadTags, loadNotebooks, switchView, updateSidebarMenuItem）
- [ ] 应用能正常启动，数据管理页面功能正常（需运行时验证）
- [ ] 导出数据功能正常（需运行时验证）
- [ ] 导入数据功能正常（需运行时验证）
- [ ] 备份/还原功能正常（需运行时验证）
- [ ] 数据库瘦身功能正常（需运行时验证）
- [ ] 打开数据目录功能正常（需运行时验证）
- [ ] 恢复出厂设置功能正常（需运行时验证）
- [ ] 统计概览加载正常（需运行时验证）
