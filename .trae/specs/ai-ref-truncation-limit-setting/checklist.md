# Checklist

- [x] NoteService 新增 settingService 字段，结构正确
- [x] NewNoteService 接收 *SettingService 参数
- [x] app.go 中所有 NewNoteService 调用处已传入 settingService
- [x] BuildNoteRefContext 从数据库读取 ai_ref_max_chars，空值时使用 1000
- [x] BuildNoteRefContext 中无效值（非数字、≤0）时使用默认值 1000
- [x] 后端 GetAIRefMaxChars 绑定返回 int，空值时返回 1000
- [x] 后端 SetAIRefMaxChars 绑定含范围校验（1-50000）
- [x] 前端 HTML 新增引用截断字数输入框（id: aiRefMaxChars）
- [x] 前端 loadAISettings 中调用 GetAIRefMaxChars 并填入输入框
- [x] 前端输入框 change 事件自动调用 SetAIRefMaxChars 保存
- [ ] ~~修改值后重新发送 AI 引用消息，截断按新值生效~~（需运行时验证，代码逻辑已确认正确）
