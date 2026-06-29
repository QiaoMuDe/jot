# 修复 AI 消息重复问题（第二版）

## 根因

`startStreaming()` 使用 `EventsOn`（持久监听器）注册流事件。Wails 的事件按**事件名**分发，不区分流来源。当第一轮流完成事件还在传递过程中、第二轮流的监听器已注册时，两套监听器都接收到了同一个完成事件 → `saveSessionMessages` 执行两次 → 消息重复保存。

## 修复方案

### 1. 流 ID 隔离（预防跨流事件干扰）

每个流生成唯一 ID，事件数据中携带该 ID。前端监听器在回调中校验 ID 是否匹配当前流，不匹配则忽略。

涉及文件：

* `internal/services/ai_service.go` — `CallAIStream` 新增 `streamID string` 参数，回调透传

* `app.go` — `CallAIStream` 生成唯一 streamID，`EventsEmit` 将 streamID 作为首个参数传递

* `frontend/src/js/ai-chat.js` — `startStreaming()` 生成 `currentStreamId`，监听器回调中校验

### 2. 重新生成流程修复（预防再生产生的重复）

`handleRegenerate()` 中，清空该会话 DB 消息 → 重新保存截断后的 chatHistory → 流完成时只保存 assistant。

涉及文件：

* `frontend/src/js/ai-chat.js` — `handleRegenerate()` 增加清库+重保存；`startStreaming()` 接受 `isRegenerate` 参数；`unsubDone` 中条件保存

### 3. 废弃的 `EventsOn` 监听器清理增强

在 `startStreaming()` 开始时，主动调用 `window.runtime.EventsOff('ai:stream-done', 'ai:stream-error', 'ai:stream-chunk', 'ai:stream-thinking')` 清除该事件名下所有旧监听器，防止残留。

涉及文件：

* `frontend/src/js/ai-chat.js` — `startStreaming()` 开头增加清理

## 修改清单

| 文件                                | 修改内容                                                   |
| --------------------------------- | ------------------------------------------------------ |
| `internal/services/ai_service.go` | `CallAIStream` 新增 `streamID string` 参数，回调签名增加 streamID |
| `app.go`                          | `CallAIStream` 生成 `streamID`，`EventsEmit` 传 streamID   |
| `frontend/src/js/ai-chat.js`      | 3 处改动：流 ID 隔离 + EventsOff 清理 + regenerate 修复           |

## 验证方式

1. 快速连续发送 2 条消息 → 切换会话回来 → 恰好 4 条，无重复
2. 重新生成 AI 回复 → 切换会话回来 → 无重复
3. 混合正常发送 + 重新生成 → 无重复

