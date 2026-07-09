# AI 聊天文件拖拽上传 - 验证检查清单

- [x] 后端 `readAIChatFiles(paths)` 内部方法提取完成，`SelectAIChatFiles` 复用该方法
- [x] 后端 `ReadAIChatFiles(paths)` 公开方法可被前端调用
- [x] 拖拽上传和按钮上传共享同一套校验逻辑（目录拒绝/10MB 限制/二进制检测/截断）
- [x] HTML 中 `#aiChatDropOverlay` 遮罩元素存在，含上传 SVG 图标和"释放以上传文件"文字
- [x] CSS `.ai-chat-drop-overlay` 样式正确：`position: absolute` 锚定、初始隐藏、`.active` 显示
- [x] 全局 `dragenter` 在 AI 聊天区内不递增 `_dragCounter`、不显示全局遮罩
- [x] 全局 `dragleave` 在 AI 聊天区内不递减 `_dragCounter`
- [x] 全局 `drop` 在 AI 聊天区内不重置遮罩
- [x] `OnFileDrop` 回调中 AI 聊天区判断优先级高于编辑器判断
- [x] AI 聊天 `dragenter` 事件正确触发遮罩显示
- [x] AI 聊天 `drageleave` 事件正确触发遮罩隐藏
- [x] AI 聊天 `dragover` 事件 `preventDefault()` 确保 drop 可触发
- [x] AI `_aiDragCounter` 计数器正确控制遮罩显隐（进入+1，离开-1，为0时隐藏）
- [x] 拖拽文件到 AI 聊天消息区，文件成功上传并显示在 chips 中
- [x] 拖拽文件到 AI 聊天输入区，文件成功上传并显示在 chips 中
- [x] 拖拽失败的文件显示错误通知
- [x] 拖拽完成后遮罩自动隐藏
- [x] 拖拽非 AI 聊天区（笔记列表等）仍走原有全局逻辑（导入笔记/编辑器插入）
