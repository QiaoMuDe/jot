# Checklist

- [x] 预览模式下表格 hover 时显示复制按钮（右侧垂直居中，与代码块按钮一致）
- [x] 点击复制按钮将表格内容以 Markdown 格式复制到剪贴板
- [x] 复制成功显示 "✓ 已复制" 1.5s 后恢复
- [x] 复制失败显示 "✗ 复制失败" 1s 后恢复
- [x] 多次调用 `updatePreview()` 不产生重复按钮（通过 table-wrapper 类检测）
- [x] 表格上下外边距与代码块一致（1.2em）
- [x] 复制按钮于 table wrapper 内垂直居中，z-index 确保无遮挡
