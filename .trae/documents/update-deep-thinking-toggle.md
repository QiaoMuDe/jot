# 计划：前端深度思考按钮逻辑与新 aicli 后端对齐

## 概要

前端深度思考（`enableThinking`）按钮当前仅将 boolean 传递到后端，但**前端展示层面未受 toggle 控制**。需改为：前后端以 Toggle 为核心统一控制，后端按 Provider 设置相应参数，前端在 Toggle 关闭时抑制思维链显示，彻底移除依赖旧系统字段结构的残留逻辑。

## 当前状态分析

### 现有流程

```
前端 Toggle（enableThinking boolean）
  → localStorage + ai_thinking_enabled 设置键持久化
  → CallAIStream(myGen, messages, enableThinking, ...)
    → AIService.CallAIStream(ctx, messages, thinkingEnabled, ...)
      → aicli.Client.Stream(ctx, msgs, thinkingEnabled, cb)
        ├─ Ollama: req.Think = &api.ThinkValue{Value: true}
        └─ OpenAI: 无特殊参数传递
```

### 历史消息加载时思维链字段映射路径

```
数据库 ai_messages 表
  └─ reasoning_content（存储思维链内容）
  └─ thinking_elapsed（思考耗时）
  └─ total_elapsed（总耗时）
  └─ is_empty_response（空回复标记）
    → LoadAISessionMessages() → services.Message{ReasoningContent, ...}
    → 前端 addMessage(content, 'assistant', msg.reasoning_content, msg.thinking_elapsed, ...)
```

### 已确认的问题

| 问题                                 | 说明                                                                                                                         |
| ---------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| Toggle 仅控制后端请求，未控制前端展示             | `ai:stream-thinking` 事件处理始终创建 `thinkingDetails` UI 元素，未检查 `enableThinking`                                                 |
| 历史消息中思维链始终显示                       | `addMessage()` 无条件渲染 `reasoningContent`，未检查 Toggle 状态                                                                      |
| `services.Message` 结构体含遗留字段        | `ReasoningContent`、`ThinkingElapsed`、`TotalElapsed` 三个字段在旧系统（LangChainGo tag 解析）时加入，现由 aicli 的 `onThinking` 回调独立传递，结构体字段冗余 |
| 设置键 `ai_thinking_enabled` 名称含旧系统痕迹 | 键名无功能性影响，但为与新系统语义对齐可改                                                                                                      |

## 改动范围

### 文件 1：`frontend/src/js/ai-chat.js`

**1.1** **`ai:stream-thinking`** **事件处理（约 L1789）**

```javascript
// 改前：无条件创建 thinking UI
const unsubThinking = window.runtime.EventsOn('ai:stream-thinking', (streamGen, chunk) => {
    if (streamGen !== myGen) return;
    if (!thinkingDetails) { /* 始终创建 */ }
    // ...
});

// 改后：enableThinking 关闭时跳过创建
const unsubThinking = window.runtime.EventsOn('ai:stream-thinking', (streamGen, chunk) => {
    if (streamGen !== myGen) return;
    if (!enableThinking) return;  // ← 新增：思考关闭时不展示思维链
    if (!thinkingDetails) { /* 创建 UI */ }
    // ...
});
```

**为什么改**：`enableThinking` 关闭时后端不会返回 thinking 内容，但即使后端返回了，前端也不应展示。

**1.2** **`addMessage()`** **函数（约 L2175）**

```javascript
// 改前：无条件渲染 reasoningContent
if (role === 'assistant' && reasoningContent) { /* 始终创建 thinking details */ }

// 改后：启用思考时才渲染
if (role === 'assistant' && reasoningContent && enableThinking) { /* 创建 thinking details */ }
```

**为什么改**：历史消息加载时，如果用户当前关闭了深度思考，不应展示过去的思维链。

**1.3 保存消息时移除冗余字段传递（约 L1918-L1920）**

```javascript
// 改前：saveSessionMessages 传 reasoning_content/thinking_elapsed/total_elapsed
saveSessionMessages([{ role: 'assistant', content: finalContent, reasoning_content: streamingThinking || '', ... }]);

// 改后：这些字段由后端 aicli 回调按需传递，前端不再手动组装
// streamingThinking 依然用于 UI 展示，但保存时不作为 reasoning_content 重复写入
```

**注意**：此项需确认后端是否仍需要 `services.Message.ReasoningContent` 来加载历史。如果保留数据库字段，则前端保存时仍需传递。建议保持传递但标注为只读兼容字段，后续可独立清理。

### 文件 2：`internal/services/ai_service.go`

**2.1** **`Message`** **结构体添加注释标记（约 L19-L26）**

```go
type Message struct {
    Role             string  `json:"role"`
    Content          string  `json:"content"`
    ReasoningContent string  `json:"reasoning_content"`  // 兼容遗留字段，新 aicli 通过 onThinking 回调
    ThinkingElapsed  float64 `json:"thinking_elapsed"`   // 同上
    TotalElapsed     float64 `json:"total_elapsed"`      // 同上
    IsEmptyResponse  bool    `json:"is_empty_response"`
}
```

**不动代码，仅加注释**标明这些字段属于旧系统兼容，未来可从数据库模型中移除。

### 文件 3：`internal/aicli/openai.go`

**3.1** **`openaiChatStream()`** **函数 - 思考兼容性路径（约 L33-40）**

当前 `enableThinking` 对此无影响：

```go
func (c *Client) openaiChatStream(ctx context.Context, messages []Message, callbacks StreamCallbacks) error {
    // 当前未使用 thinkingEnabled 参数
```

**不改。** OpenAI 兼容 API 没有标准 `think` 参数，推理模型自行决定是否输出 `reasoning_content`。前端 Toggle 关闭时跳过展示是合理行为。

### 文件 4：`internal/aicli/ollama.go`

**4.1 确认** **`req.Think`** **参数传递（无需改动）**

已在 v1 实现中正确传递（约 L24-27）：

```go
if thinkingEnabled {
    req.Think = &api.ThinkValue{Value: true}
}
```

## 假设与决策

| 假设/决策                           | 说明                                  |
| ------------------------------- | ----------------------------------- |
| 保留数据库 `reasoning_content` 字段    | 已有数据无需迁移，新消息正常写入，兼容旧记录              |
| 前端 Toggle 为最终判断                 | 即使后端返回 thinking 内容，Toggle 关闭时前端也不展示 |
| OpenAI 后端无 `thinkingEnabled` 效果 | OpenAI 不设 `think` 参数，模型自行决定是否推理     |
| 设置键 `ai_thinking_enabled` 不改名   | 键名不暴露给用户，改名的风险收益比低                  |

## 验证步骤

1. `wails build` 编译通过
2. Ollama（qwen3:0.6b）+ 深度思考开启 → 思维链正常显示
3. Ollama + 深度思考关闭 → 思维链不显示，内容正常
4. 切换会话 → 关闭思考开启时加载的历史消息 → 思维链不被渲染
5. 切换会话 → 开启思考时加载的历史消息 → 思维链正常渲染
6. OpenAI + 深度思考（模型支持）→ 思维链正常
7. OpenAI + 深度思考关闭 → 思维链不展示

## 未涉及范围

* 数据库表结构调整（`reasoning_content`、`thinking_elapsed`、`total_elapsed` 列保留）

* `ai_thinking_enabled` 设置键改名

* 后端 aicli 接口签名变更

* 前端历史消息的数据迁移

