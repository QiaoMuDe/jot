# Checklist

- [x] `package.json` 中已移除 `quikchat` 依赖
- [x] `index.html` 中 AI 对话页面使用自定义消息列表 + 输入区，无 QuikChat 挂载点残留
- [x] `ai-chat.js` 无 QuikChat 导入，使用 `marked` + `highlight.js` 渲染消息
- [x] 用户消息右对齐圆形气泡（`.ai-msg-user`），AI 回复左对齐圆角气泡（`.ai-msg-assistant`）
- [x] AI 回复中的 Markdown 正确渲染（标题/加粗/列表/表格/任务列表）
- [x] AI 回复中的代码块使用 highlight.js 高亮
- [x] 打字指示器（三圆点弹跳动画）在等待响应时显示
- [x] 发送消息后正常收到 AI 回复并渲染
- [x] 调用失败时显示错误提示
- [x] 按 Enter 发送消息，Shift+Enter 换行
- [x] 未配置 AI 时页面显示空状态提示和「前往设置」链接
- [x] 「清空对话」按钮清除所有消息
- [x] Wails dev 构建无错误
