# Checklist

- [x] `ImportFiles` 在导入前后的三个时机发射进度事件（start/update/complete）
- [x] `readAIChatFiles` 在导入前后的三个时机发射进度事件（start/update/complete），事件名为 `"import:ai-progress"`
- [x] `NotificationManager.showProgress(prefix, total)` 创建持久通知，内容含旋转动画
- [x] `NotificationManager.updateProgress(ctrl, current, total, title)` 更新通知文本
- [x] 进度通知前缀区分：笔记导入用"正在导入"，AI 上传用"正在上传"
- [x] 进度通知无关闭按钮、不自动消失
- [x] 快速批量场景使用 `requestAnimationFrame` 防抖更新
- [x] `handleFileDropPaths` 在调用 RPC 前注册事件监听，完成后清理
- [x] AI 聊天文件上传进度监听注册和清理正确
- [x] `go build ./...` 无错误
- [ ] 前端通知系统测试通过（单文件、多文件、AI 上传）
