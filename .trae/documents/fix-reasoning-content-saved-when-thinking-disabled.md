# 修复：关闭深度思考后思维链被错误保存到数据库

## 问题描述

用户关闭"深度思考"后进行对话，AI 回复正常（没有显示思维链）。但重启程序后，那些本应没有思维链的消息却显示了思维链内容。

## 根因分析

完整数据流链路：

1. 用户发送消息，`thinkingEnabled=false`
2. `OpenAI/Ollama` API 发送 `enable_thinking: false` 参数
3. **部分推理模型可能忽略此参数**，仍然返回 `reasoning_content`/`thinking` 字段
4. `client.go` 行 50 无条件累积 `fullThinking.WriteString(text)`
5. `app.go` **行 863 无条件保存** `ReasoningContent: fullThinking.String()`，未检查 `thinkingEnabled`
6. 数据写入 DB，`reasoning_content` 字段非空
7. 应用重启 → `LoadAISessionMessages` 原样返回 `ReasoningContent` → `addMessage()` 检测到非空 → 渲染思维链 DOM

**问题本质**：后端在 `app.go` 的 done 回调中缺少对 `thinkingEnabled` 的检查，无论 API 是否返回了思维链内容，都原样存入数据库。

## 修改方案

### 文件：`d:\资源池\下水道\Dev\本地项目\jot\app.go`

**位置 1**（第 863 行）：构建 `assistantMsg` 时，`ReasoningContent` 根据 `thinkingEnabled` 决定是否保存

```go
// 修改前：
ReasoningContent: fullThinking.String(),

// 修改后：
ReasoningContent: func() string {
    if !thinkingEnabled { return "" }
    return fullThinking.String()
}(),
```

**位置 2**（第 864 行）：`ThinkingElapsed` 同步根据 `thinkingEnabled` 清空

```go
// 修改前：
ThinkingElapsed: elapsedThinking,

// 修改后：
ThinkingElapsed: func() float64 {
    if !thinkingEnabled { return 0 }
    return elapsedThinking
}(),
```

**位置 3**（第 891 行）：`stream-thinking-done` 事件也受 `thinkingEnabled` 控制（虽然前端无监听，但保持一致性）

```go
// 修改前：
if fullThinking.Len() > 0 {
    runtime.EventsEmit(a.ctx, "ai:stream-thinking-done", fullThinking.String())
}

// 修改后：
if thinkingEnabled && fullThinking.Len() > 0 {
    runtime.EventsEmit(a.ctx, "ai:stream-thinking-done", fullThinking.String())
}
```

## 影响范围

* 仅修改 `app.go` 一个文件

* 仅影响新发消息的保存逻辑，不影响已有消息

* 前端无需修改（前端加载时 `reasoningContent` 为空就不会渲染思维链）

* 无向后兼容性问题

## 验证步骤

1. 关闭深度思考，发送一条消息 → 确认回复没有思维链
2. 重启程序 → 确认该消息仍然没有思维链
3. 开启深度思考，发送一条消息 → 确认思维链正常显示
4. 重启程序 → 确认思维链仍然正常显示

