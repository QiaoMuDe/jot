# Check List

## 后端
- [ ] 后端 `LoadAISessionMessagesPaginated` 分页查询已实现
  - [ ] `cursorID=0` 返回最新 `limit` 条
  - [ ] `cursorID>0` 返回更早的消息
  - [ ] 正确返回 `hasMore` 标记
  - [ ] 已暴露为 Wails 绑定方法
- [ ] 后端 `CallAIStream` 签名已更新（删除 `messages`，新增 `content`）
- [ ] 后端新增 `SaveUserMessage(sessionID, content) (uint, error)` 方法
- [ ] 后端新增 `SaveAssistantMessage(...) (uint, error)` 方法
- [ ] 后端发送模式下先调用 `SaveUserMessage` 保存 user 消息到 DB，再从 DB 加载全部消息
- [ ] 后端再生模式下直接从 DB 加载全部消息（不新增 user 消息）
- [ ] AI 调用失败时回滚已保存的 user 消息（调用 `DeleteAIMessage(userMessageID)`）
- [ ] `ai:stream-done` 事件新增 `userMessageID` 和 `assistantMessageID` 字段
- [ ] `ai:stream-error` 事件新增 `userMessageID` 字段（发送模式）

## 前端
- [ ] 前端 `chatHistory` 已重命名为 `displayedMessages`
- [ ] 切换会话/首次加载时仅加载最近 6 条消息
- [ ] `switchSession()` 保留了现有副作用逻辑（取消流、清空引用/技能/文件状态、加载 config、更新侧栏）
- [ ] 滚动到顶部触发懒加载，加载更早消息
- [ ] 懒加载消息正确 `unshift` 到头部，滚动位置正确（scrollHeight 恢复）
- [ ] 自动滚动策略：仅在用户位于底部时自动滚动，否则显示"[新消息]"提示按钮
- [ ] 发送消息时只传 `sessionID` + `content`，不再传历史消息
- [ ] 流完成后前端正确更新 `displayedMessages`
- [ ] 流失败时前端根据 `userMessageID` 正确移除 user 消息占位
- [ ] 重新生成（Regenerate）正常工作（从 displayedMessages 取最后一条 user content）
- [ ] 连续多次 Regenerate 正常工作
- [ ] 编辑重发（Resend）正常工作
- [ ] 删除消息正常工作

## 功能回归
- [ ] 笔记引用正常
- [ ] 角色扮演正常
- [ ] 追问引用正常
- [ ] 上传文件正常
- [ ] 联网搜索 + 卡片召回 + 技能正常
- [ ] 短对话（不足 6 条）正常显示，无"加载更多"
- [ ] 长对话反复懒加载直到全部加载
- [ ] 网络错误 / API 失败时回滚和提示正常
