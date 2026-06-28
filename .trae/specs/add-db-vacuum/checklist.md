# Checklist

- [x] `note_service.go` 中 `Vacuum()` 方法已添加，执行 `s.db.Exec("VACUUM")`
- [x] `app.go` 中 `VacuumDatabase()` 绑定方法已添加，执行前后计算释放空间
- [x] `index.html` 中"数据库瘦身"按钮已添加，包含正确图标和描述文本
- [x] `main.js` 中 `vacuumDatabase()` 函数已添加，包含 Wails 调用、通知、刷新统计
- [x] `main.js` 中 `els.vacuumDbBtn` 引用和事件监听已注册
- [x] 构建通过，无编译错误
- [ ] 功能验证：创建笔记 → 删除笔记 → 清空回收站 → 点击瘦身 → 数据库大小减小