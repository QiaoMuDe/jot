# Tasks

- [ ] Task 1: 后端新增 `LoadAISessionMessagesPaginated` 分页查询方法
  - [ ] 1.1 在 `ai_service.go` 中新增 `LoadAISessionMessagesPaginated(sessionID, cursorID, limit) (messages, hasMore, error)`
  - [ ] 1.2 逻辑：`cursorID=0` 返回最新 `limit` 条；否则返回 `id < cursorID` 的最多 `limit` 条
  - [ ] 1.3 结果按 `created_at DESC` 排序（前端接收后反转）
  - [ ] 1.4 判断是否有更早的消息以设置 `hasMore`
  - [ ] 1.5 在 `app.go` 中新增 Wails 绑定方法暴露给前端

- [ ] Task 2: 修改 `CallAIStream` 签名和后端实现
  - [ ] 2.1 修改 `CallAIStream` 函数签名：删除 `messages []services.Message`，新增 `content string`
  - [ ] 2.2 新增 `loadSessionMessagesForModel(sessionID)` 内部方法：从 DB 加载会话全部消息，构建 `[]services.Message`
  - [ ] 2.3 发送模式（`isRegenerate=false`）：先保存 user 消息到 DB，再从 DB 加载全部消息
  - [ ] 2.4 再生模式（`isRegenerate=true`）：直接从 DB 加载全部消息（不新增 user 消息）
  - [ ] 2.5 现有 system 上下文注入逻辑不变（复用已有代码）
  - [ ] 2.6 流完成后保存逻辑适配：`ai:stream-done` 事件新增 `userMessageID` 和 `assistantMessageID` 字段
  - [ ] 2.7 在 `ai_service.go` 中新增 `SaveUserMessage(sessionID, content) (uint, error)` 和 `SaveAssistantMessage(sessionID, content, reasoningContent, tokens, thinkingElapsed, totalElapsed, searchSources, recallCards) (uint, error)`
  - [ ] 2.8 AI 调用失败时回滚：删除已保存的 user 消息，通过 `ai:stream-error` 通知前端（含 `userMessageID`）
  - [ ] 2.9 `ai:stream-error` 事件新增 `userMessageID` 字段（发送模式）

- [ ] Task 3: 前端重构 `chatHistory` → `displayedMessages` + 懒加载
  - [ ] 3.1 将 `chatHistory` 重命名为 `displayedMessages`
  - [ ] 3.2 `switchSession()` 中改为调用分页接口加载最新 6 条消息（保留现有副作用逻辑：取消流、清空引用/技能/文件状态、加载 session config、更新侧栏）
  - [ ] 3.3 新增滚动监听逻辑：滚动到消息列表顶部时触发懒加载
  - [ ] 3.4 新加载的消息 `unshift` 到 `displayedMessages` 头部
  - [ ] 3.5 维护滚动位置：加载前记录 scrollHeight，DOM 插入后恢复 scrollTop
  - [ ] 3.6 自动滚动策略：流完成时判断用户是否在底部，仅在底部时自动滚动；否则显示"[新消息]"提示按钮

- [ ] Task 4: 前端发送/重发/再生/删除适配新流程
  - [ ] 4.1 `onSend()` 不再将用户消息 `push` 到 `displayedMessages`，改为直接调用 `CallAIStream(sessionID, content, ...)`
  - [ ] 4.2 `startStreaming()` 不再接收/传递 `messages` 参数，简化参数列表
  - [ ] 4.3 `handleRegenerate()` 适配：从 `displayedMessages` 获取最后一条 user 消息 content，调用后端删除最后一条 assistant 消息后，调用 `CallAIStream` 带 `isRegenerate=true`
  - [ ] 4.4 `handleResend()` 适配：调用后端删除之后的消息，再调用 `CallAIStream` 发送新内容
  - [ ] 4.5 `handleDeleteMsg()` 适配：删除后刷新 `displayedMessages`
  - [ ] 4.6 `ai:stream-done` 回调中根据返回的消息 ID 构建消息对象 `push` 到 `displayedMessages`
  - [ ] 4.7 `ai:stream-error` 回调中根据 `userMessageID` 移除 `displayedMessages` 中对应的 user 消息占位

## Task Dependencies

- [Task 1] 后端分页 → 独立，可先做
- [Task 2] 后端发送流程重构 → 独立，可先做
- [Task 3] 前端懒加载 → 依赖 [Task 1] 完成后端分页接口
- [Task 4] 前端发送适配 → 依赖 [Task 2] 完成后端签名变更

## 验证方式

### 功能验证
1. 运行 `wails dev` 启动应用
2. 打开已有长对话会话 → 确认只显示最近 6 条消息，顶部有"加载更多"提示
3. 短对话（不足 6 条）→ 确认正常显示所有消息，无"加载更多"提示
4. 滚动到顶部 → 确认触发懒加载，加载更早 6 条消息，滚动位置保持正确
5. 反复滚动到顶部 → 确认能持续加载直到所有消息显示完毕，`hasMore=false` 后不再触发
6. 发送新消息 → 确认消息正常发送，流式回复正常
7. 发送新消息后用户不在底部 → 确认不自动滚动，显示"新消息"提示
8. 编辑用户消息重发 → 确认正常工作
9. 重新生成 AI 回复 → 确认正常工作
10. 连续多次重新生成 → 确认每次都能正确获取最新 user message content
11. 删除消息 → 确认正常
12. 切换会话 → 确认懒加载正常工作，旧会话状态完全清理

### 错误场景验证
13. 断网后发送消息 → 确认显示错误提示，user 消息不会残留在 displayedMessages 中
14. API key 无效时发送 → 确认正确回滚并提示

### 回归验证
15. 笔记引用 → 确认正常
16. 角色扮演 → 确认正常
17. 追问引用 → 确认正常
18. 上传文件 → 确认正常
19. 联网搜索 + 卡片召回 + 技能 → 确认正常
