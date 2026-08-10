# chat + embedding 全走 eino（移除 aicli）Spec

## Why

自研客户端库 `internal/aicli`（基于 `sashabaranov/go-openai`）目前同时支撑 chat 模式与 embedding 向量化。项目已引入 eino（`eino-ext/components/model/openai` + `eino-ext/libs/acl/openai`），agent 模式已验证可用。目标是让 chat 与 embedding 全部改走 eino，删除 `internal/aicli` 并清理 `sashabaranov/go-openai` 依赖，消除一套重复的 OpenAI 封装。

## What Changes

采用**路线 A（薄适配层）**：新建 `internal/einocli`，公共 API 与 aicli 完全一致，内部改用 eino 实现，使调用方只改 import 路径。

- 新建 `internal/einocli/` 包（package `einocli`），提供与 aicli 一致的：
  - 类型：`Config{BaseURL, APIKey, Model}`、`Message{Role, Content}`、`StreamCallbacks{OnChunk, OnThinking, OnDone(fullContent, thinkingElapsed, totalElapsed), OnError}`、`Client{BaseURL, APIKey, Model}`
  - 方法：`NewClient`、`Chat(ctx, messages, thinkingEnabled) (content, reasoning, err)`、`Stream(ctx, messages, thinkingEnabled, callbacks)`、`Embed(ctx, texts) ([][]float32, err)`、`EmbedWithProgress(ctx, texts, batchSize, cb) ([][]float32, err)`
  - 内部实现：
    - Chat：`openai.NewChatModel` + `Generate`；过滤空 content 消息（防 400）；role 字符串 → `schema.RoleType`；不设置 enable_thinking（与 aicli 现状一致）；错误经 `aierrors.ClassifyError` 包装为 `AIErrorWrapper`
    - Stream：`openai.NewChatModel` + `Stream`，用 `openai.WithExtraFields(map[string]any{"enable_thinking": thinkingEnabled})`（等价 aicli 的 `ChatTemplateKwargs`）；消费 `*schema.StreamReader[*schema.Message]`（参照 `agent.go` 的 `consumeAssistantStream` 模式）：`chunk.Content` → OnChunk、`chunk.ReasoningContent` → OnThinking、`io.EOF` 结束、ctx 取消不报错；统计 thinkingElapsed/totalElapsed 后回调 OnDone；错误分类后回调 OnError（JSON）
    - Embed：`acl/openai.NewEmbeddingClient` + `EmbedStrings`；**float64 → float32 转换**；按响应顺序取值并校验返回数量
    - EmbedWithProgress：分批循环（`batchSize <= 0` 时整批），每批回调 `cb(done, total)`
- 更新 `internal/aierrors`：`ClassifyError` 的 `errors.As` 目标从 `sashabaranov/go-openai` 的 `*APIError/*RequestError` 切换为 `meguminnnnnnnnn/go-openai` 的 `*APIError/*RequestError`（eino acl 的 embedding 实际返回该类型；两者字段结构一致）；保留 eino components 分支；测试同步更新
- 调用方迁移（仅 import 路径/类型路径变化，函数签名不变）：
  - `internal/services/ai_service.go`：`aicli.NewClient` / `aicli.Message` / `aicli.StreamCallbacks` → `einocli.*`
  - `internal/services/vector_service.go`：4 个方法签名 `*aicli.Client` → `*einocli.Client`，`embedClient.Model` 字段访问不变
  - `internal/agent/tools.go`：1 处 `aicli.NewClient(aicli.Config{...})` → `einocli.*`
  - `app.go`：2 处 `aicli.NewClient(aicli.Config{...})`（startVectorIndex、卡片召回）→ `einocli.*`
- 删除 `internal/aicli/` 整个目录（client.go、openai.go、types.go）
- 依赖清理：`go mod tidy` 移除 `github.com/sashabaranov/go-openai`；`meguminnnnnnnnn/go-openai` 由间接依赖转为直接依赖
- 前端零改动

## Impact

- Affected specs: custom-ai-client（aicli 创建，完结）、extract-ai-error-package（错误包，完结）
- Affected code:
  - `internal/einocli/`（新增）
  - `internal/aierrors/errors.go`、`internal/aierrors/errors_test.go`
  - `internal/services/ai_service.go`、`internal/services/vector_service.go`
  - `internal/agent/tools.go`
  - `app.go`
  - `go.mod`、`go.sum`
  - `internal/aicli/`（删除）

## ADDED Requirements

### Requirement: einocli 适配层
系统 SHALL 提供 `internal/einocli` 包，公开 API 与 aicli 一致，内部基于 eino 实现 chat（流式/非流式）与 embedding（含分批进度）。

#### Scenario: 非流式对话（Chat）
- **WHEN** 调用 `Client.Chat(ctx, messages, false)`
- **THEN** 返回 (content, reasoning, nil)；空消息被过滤；错误经 `aierrors.ClassifyError` 包装为 `*AIErrorWrapper`

#### Scenario: 流式对话（Stream）
- **WHEN** 调用 `Client.Stream(ctx, messages, true, callbacks)` 且模型返回 reasoning
- **THEN** content 增量走 `OnChunk`、reasoning 增量走 `OnThinking`；结束后 `OnDone(fullContent, thinkingElapsed, totalElapsed)`；错误走 `OnError`（分类 JSON）；ctx 取消不回调错误

#### Scenario: 向量化（Embed/EmbedWithProgress）
- **WHEN** 调用 `EmbedWithProgress(ctx, texts, 16, cb)`
- **THEN** 返回与 texts 一一对应的 `[][]float32`，每批完成后回调 `cb(done, total)`

### Requirement: 错误分类兼容 eino embedding
`aierrors.ClassifyError` SHALL 能分类 `meguminnnnnnnnn/go-openai` 的 `*APIError/*RequestError`（embedding 经 acl 返回的实际类型），分类逻辑与中文文案不变。

## MODIFIED Requirements

### Requirement: aierrors 错误类型切换
`internal/aierrors` 中 `errors.As` 目标由 `sashabaranov/go-openai` 切换为 `meguminnnnnnnnn/go-openai`；eino components 分支保留；`errors_test.go` 中原 sashabaranov 用例改为 meguminnnnnnnnn 类型。

## REMOVED Requirements

### Requirement: aicli 客户端库
**Reason**: chat 与 embedding 已全部由 eino（经 einocli 适配层）承载，无使用方。
**Migration**: 所有调用方改用 `internal/einocli`；`internal/aicli/` 目录删除；`github.com/sashabaranov/go-openai` 依赖移除。
