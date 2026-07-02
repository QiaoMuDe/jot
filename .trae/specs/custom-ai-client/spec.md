# 自定义 AI API 客户端适配层

## Why

当前使用自研 HTTP + gjson 手写解析调用 AI 接口，虽然解决了 LangChainGo 的思维链问题，但流式解析、连接管理、错误处理等基础功能自己实现不如成熟库稳定。需要将底层替换为 `go-openai`（OpenAI 兼容）和 `ollama/ollama/api`（Ollama 原生），上层保留 `aicli/` 作为统一适配层，`ai_service.go` 无需感知底层变化。

## What Changes

- **保留 `internal/aicli/` 目录**，作为项目统一的 AI 调用适配层
- 底层实现替换为两个库：
  - `github.com/sashabaranov/go-openai` — OpenAI 兼容 API（`/chat/completions`）
  - `github.com/ollama/ollama/api` — Ollama 原生 API（`/api/chat`）
- 删除自研的 HTTP + bufio.Scanner + gjson 解析代码
- `Client.Stream()` / `Client.Chat()` 签名不变，`ai_service.go` **零改动**

## Impact

- 修改的文件：`internal/aicli/openai.go`、`internal/aicli/ollama.go`、`internal/aicli/client.go`、`internal/aicli/types.go`
- 删除 `gjson` 依赖，新增 `go-openai` 和 `ollama/ollama/api` 依赖
- 删除 `extract.go`（gjson 字段提取不再需要，由库的类型化结构体替代）
- `ai_service.go` 无需改动
- app.go 无需改动

## ADDED Requirements

### Requirement: 适配层统一接口不变

`aicli.Client` 对外接口保持不变：

```
type Client struct { Provider, BaseURL, APIKey, Model }
func NewClient(cfg Config) *Client
func (c *Client) Stream(ctx, messages, thinkingEnabled, callbacks)
func (c *Client) Chat(ctx, messages) (content, thinking, error)
```

`StreamCallbacks` 保持不变：`{OnChunk, OnThinking, OnDone, OnError}`

### Requirement: OpenAI 兼容后端使用 go-openai

系统应使用 `github.com/sashabaranov/go-openai` 调用 OpenAI 兼容 API。

#### Scenario: 流式调用

- **WHEN** Provider 为 `openai` 且调用 `Client.Stream()`
- **THEN** 创建 `openai.Client`，设置 BaseURL 和 APIKey
- **AND** 调用 `client.CreateChatCompletionStream(ctx, req)` 发起 SSE 流
- **AND** 循环 `stream.Recv()` 读取流式块
- **AND** 从 `choices[0].delta` 提取 `Content` 和 `ReasoningContent`
- **AND** 通过 `OnChunk` / `OnThinking` 推送

#### Scenario: 深度思考（thinkingEnabled）

- **WHEN** `thinkingEnabled=true` 且 Provider 为 `openai`
- **THEN** 额外传入推理模型的参数（模型自行决定是否输出 `reasoning_content`）

#### Scenario: 非流式调用

- **WHEN** 调用 `Client.Chat()`
- **THEN** 调用 `client.CreateChatCompletion(ctx, req)` 获取完整响应
- **AND** 返回 `content` 和 `reasoning_content`

### Requirement: Ollama 后端使用 ollama/ollama/api

系统应使用 `github.com/ollama/ollama/api` 调用 Ollama 原生 API。

#### Scenario: 流式调用

- **WHEN** Provider 为 `ollama` 且调用 `Client.Stream()`
- **THEN** 创建 `api.Client`，设置 BaseURL（`http://localhost:11434`）
- **AND** 调用 `client.Chat(ctx, req, fn)` 发起流式聊天
- **AND** 回调函数中从 `response.Message` 提取 `Content` 和 `Thinking`
- **AND** 通过 `OnChunk` / `OnThinking` 推送

#### Scenario: 深度思考（thinkingEnabled）

- **WHEN** `thinkingEnabled=true` 且 Provider 为 `ollama`
- **THEN** 设置 `req.Think = &thinkTrue`（`api.ThinkValue` 类型）
- **AND** 模型返回思维链时，`response.Message.Thinking` 非空

#### Scenario: 非流式调用

- **WHEN** 调用 `Client.Chat()`
- **THEN** 调用 `client.Chat(ctx, req, fn)` 获取完整响应
- **AND** 返回 `content` 和 `thinking`

## MODIFIED Requirements

### Requirement: 流式回调接口（不变）

`StreamCallbacks` 签名保持不变：
```go
type StreamCallbacks struct {
    OnChunk    func(text string)
    OnThinking func(text string)
    OnDone     func(fullContent string, thinkingElapsed float64, totalElapsed float64)
    OnError    func(errMsg string)
}
```

### Requirement: aicli 适配层删除自研 HTTP 解析

删除以下自定义代码：
- `openai.go` 中的 `parseOpenAIStream`（SSE bufio 解析）
- `ollama.go` 中的 `parseOllamaStream`（NDJSON bufio 解析）
- `extract.go` 整个文件（gjson 字段提取不再需要）
- `types.go` 中的自研流式结构体

替换为调用 `go-openai` 和 `ollama/ollama/api` 的类型化结构体。

## REMOVED Requirements

### Requirement: gjson 依赖

**Reason**: 底层库的流式响应自带类型化结构体，不再需要 gjson 做 JSON 路径提取。  
**Migration**: 删除 `go mod` 中的 `github.com/tidwall/gjson`，删除 `extract.go` 整个文件。

### Requirement: 自研 SSE / NDJSON 解析

**Reason**: go-openai 和 ollama/ollama/api 各自实现了完善的流式解析（连接管理、重试、超时）。  
**Migration**: 删除 `parseOpenAIStream`、`parseOllamaStream` 函数，改为调用库的流式接口。
