# Checklist

- [x] `GetContextWindowSize` 方法实现：读取 `ai_context_window_size`，默认返回 20
- [x] `TruncateMessagesForLLM` 方法实现：保留 system 消息 + 最后 N 条 user/assistant 消息
- [x] `CallAIStream` 中加载消息后、注入 context 前调用截断
- [x] `CallAIStreamRegenerate` 中加载消息后、注入 context 前调用截断
- [x] 短对话（< 20 条）不做截断，全部发送
- [x] system 消息不受窗口影响，始终保留
- [x] 前端 `chatHistory` 渲染缓冲区未受影响
- [x] 编辑/重发/删除等消息操作功能正常
