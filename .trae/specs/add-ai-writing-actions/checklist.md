# Checklist

- [x] 后端 `AITextOperation` 绑定正确实现，支持 8 种 operation
- [x] AI 未配置时返回友好错误提示
- [x] `executeAction` 改为 async 后，同步操作（格式化/文本转换等）行为不变
- [x] AI 操作正确调用后端并写回编辑器
- [x] 无选中文本时 AI 操作提示"请先选择要处理的文本"
- [x] AI 处理中编辑器面板顶部出现 `#aiStatusBar`，含 spinner + "AI 处理中..." + 取消按钮
- [x] `#aiStatusBar` 的取消按钮可中止 AI 操作，编辑器内容不变
- [x] AI 处理期间 `editorActionsBtn` 禁用，完成后恢复
- [x] AI 处理失败时通知显示"AI 处理失败: 原因"
- [x] 8 个 AI 写作操作项在菜单中正确显示且可点击
- [x] 构建通过，无编译错误