# 空回复标记字段 checklist

## Go 模型层
- [x] AIMessage 结构体新增 `IsEmptyResponse bool` 字段（ai_message.go）
- [x] Message 结构体新增 `IsEmptyResponse bool` 字段（ai_service.go）

## Go 服务层
- [x] SaveAIMessages 中传递 IsEmptyResponse 字段（ai_service.go）
- [x] LoadAISessionMessages 中传递 IsEmptyResponse 字段（ai_service.go）

## 前端 JS
- [x] 空回复时保存消息带 `is_empty_response: true`（ai-chat.js）
- [x] 正常回复时保存消息带 `is_empty_response: false`（ai-chat.js）
- [x] addMessage 新增 `isEmptyResponse` 参数并按标记渲染占位样式（ai-chat.js）
- [x] switchSession 加载消息时传递 `is_empty_response` 给 addMessage（ai-chat.js）

## 验证
- [x] `wails build` 编译通过
- [ ] 空回复气泡渲染占位样式（琥珀色图标 + 灰色提示文字）
- [ ] 切换会话后空回复气泡样式保持
- [ ] 正常 AI 回复不受影响
