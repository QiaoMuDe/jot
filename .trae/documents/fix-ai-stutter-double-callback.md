# 修复 AI 回复结巴（字符重复）问题

## 问题现象

连接到 Jan 本地 API（OpenAI 兼容）时，AI 回复出现字符重复，例如：
```
输入：用一句话介绍自己
输出：哈哈喽喽！！我是我是通通义义千千问问
```
（预期应为：大家好！我是通义千问）

## 根因分析

### LangChainGo 双回调冲突

当「深度思考」开关打开时（`thinkingEnabled = true`），后端 `CallAIStream` 同时设置了 **两个 streaming callback**：

1. `WithStreamingFunc` → 处理普通内容 chunk
2. `WithStreamingReasoningFunc` → 处理 reasoning_content + content

LangChainGo v0.1.14 的 OpenAI provider 在 [chat.go#L652-L663](file:///d:/AppData/gopath/pkg/mod/github.com/tmc/langchaingo@v0.1.14/llms/openai/internal/openaiclient/chat.go) 中对 **每个 SSE chunk 依次调用两个回调**：

```go
if payload.StreamingFunc != nil {
    payload.StreamingFunc(ctx, chunk)  // 第1次：StreamingFunc 处理
}
if payload.StreamingReasoningFunc != nil {
    payload.StreamingReasoningFunc(ctx, reasoningChunk, chunk)  // 第2次：ReasoningFunc 又处理一遍
}
```

因此每个 content chunk 被 `onChunk` 调用了两次 → 每个字出现两次 → 结巴。

### 触发条件

- 仅在 `thinkingEnabled == true`（深度思考开启）时触发
- 非深度思考模型（如 Jan 的 Qwen 模型）没有 `reasoning_content`，但仍会触发双回调
- 开启深度思考时，content chunk 通过两个路径各处理一次

## 修复方案

### 修改文件

**`internal/services/ai_service.go`** — `CallAIStream` 方法

### 修改策略

当 `thinkingEnabled == true` 时，**仅使用 `WithStreamingReasoningFunc`**，不再同时设置 `WithStreamingFunc`。

因为 `WithStreamingReasoningFunc` 的回调签名是 `(ctx, reasoningChunk, chunk []byte)`：
- `reasoningChunk` 非空 → 这是思维链内容
- `chunk` 非空 → 这是正文内容
- 两者可能同时出现，也可能只出现一个

**改动要点：**

1. 将 `WithStreamingFunc` 从选项列表中移除，改为仅在有 content 时在 `WithStreamingReasoningFunc` 内部处理
2. 当 `thinkingEnabled == false` 时，保持原有逻辑（仅 `WithStreamingFunc`）
3. 当 `thinkingEnabled == true` 时，只用 `WithStreamingReasoningFunc`，不设 `WithStreamingFunc`

### 具体代码变化

```go
// 修改前（thinkingEnabled=true 时同时设两个回调）
opts := []llms.CallOption{
    llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
        // ← 这个在 thinking 模式下会导致重复
        text := string(chunk)
        if text != "" {
            fullContent.WriteString(text)
            onChunk(text)
        }
        return nil
    }),
}
if thinkingEnabled {
    opts = append(opts,
        llms.WithStreamingReasoningFunc(...),
    )
}

// 修改后（thinkingEnabled 时只设 reasoning func）
var opts []llms.CallOption
if thinkingEnabled {
    opts = append(opts,
        llms.WithStreamingReasoningFunc(func(_ context.Context, reasoningChunk, chunk []byte) error {
            // 处理思维链
            rText := string(reasoningChunk)
            if rText != "" {
                fullThinking.WriteString(rText)
                if onThinking != nil {
                    onThinking(rText)
                }
            }
            // 处理正文（只在这里处理一次，不再有 StreamingFunc 重复处理）
            if len(chunk) > 0 {
                text := string(chunk)
                if text != "" {
                    fullContent.WriteString(text)
                    onChunk(text)
                }
            }
            return nil
        }),
        llms.WithThinking(...),
    )
} else {
    // 非 thinking 模式：保持原有 WithStreamingFunc
    opts = append(opts,
        llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
            text := string(chunk)
            if text != "" {
                fullContent.WriteString(text)
                onChunk(text)
            }
            return nil
        }),
    )
}
```

### 影响范围

- **深度思考模型（DeepSeek R1 等）**：`reasoning_content` 走 `WithStreamingReasoningFunc` 正常处理，正文也走同一回调，一次处理
- **非深度思考模型（Qwen、GPT、Ollama 等）**：`reasoningChunk` 为空，正文 chunk 仅由 reasoning func 的 `chunk` 参数处理，一次处理
- **OpenAI 兼容 API**：请求体中仍通过 `WithThinking` 发送 `thinking` 参数（API 会忽略或处理，不影响）
- **Ollama Provider**：`ollama.New()` 不影响，ollama provider 不使用 `WithStreamingReasoningFunc`

## 验证步骤

1. 连接 Jan 本地 API（OpenAI 兼容模式），开启深度思考
2. 发送消息，确认回复不再结巴
3. 关闭深度思考，确认回复正常
4. 切换回 OpenAI（DeepSeek），确认思维链功能正常
5. 切换回 Ollama，确认流式输出正常
