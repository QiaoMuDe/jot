# Checklist

## 后端
- [x] ask_user 工具存在（`internal/agent/tools/ask_user.go`），Schema 含 question/options/reason，构造器 `NewAskUser` 导出
- [x] ask_user 执行时不执行业务，发射 `ai:ask-user` 事件（JSON 负载含 question/options），返回问句文本给模型
- [x] ask_user 实现 `ActionTextProvider`，文案为 `向用户提问：{question}`（截断 30 字符）
- [x] registry.go buildTools 注册 ask_user（经 WrapWithError 包装）
- [x] agent.go：ask_user 调用轮正文作为 finalContent 兜底（最终轮无输出时），其他工具行为不变
- [x] app.go Instruction 追加 ask_user 使用约束（仅 Agent 流生效）

## 前端
- [x] startStreaming 事件清理列表含 `ai:ask-user`
- [x] `ai:ask-user` 监听注册，解析 question/options 并在 streamingEl 内渲染问题卡片
- [x] 卡片含选项按钮列表（options 非空时）与自定义输入框 + 提交按钮
- [x] 点击选项 / 提交自定义输入 → 保存 user 消息 → 渲染用户气泡 → startStreaming 续流
- [x] 卡片在流结束后保留，不被 stream-done 清空逻辑移除
- [x] ai-chat.css 卡片样式主题自适应，hover/active 反馈正常

## 历史回放
- [x] ask_user 交互轮次 assistant 消息落库正文为问句（可读）
- [x] 切换会话回放显示问句文本 + 工具调用链折叠，无交互控件

## 构建验证
- [x] `go build ./...` 通过
- [x] 前端 `npm run build` 通过
- [x] 文档（doc.go ×2 / TOOLS.md / AGENTS.md）已更新且无遗漏
