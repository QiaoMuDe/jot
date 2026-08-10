# 抽取独立 AI 错误包（aierrors）Spec

## Why

当前 AI 错误分类（12 类常量、AIError、AIErrorWrapper、ClassifyError）全部内聚在 `internal/aicli/errors.go`，且 `ClassifyError` 的 `errors.As` 只认 `sashabaranov/go-openai` 的类型。为后续 chat 模式平替 eino 铺路，需要把错误分类抽成独立包，让当前 aicli（sashabaranov 类型错误）与 eino（`eino-ext/components/model/openai` 的 `*APIError`）能同时使用。

## What Changes

- 新建 `internal/aierrors/` 包（package `aierrors`）：
  - 迁移 12 个错误分类常量（`CategoryAuthError` … `CategoryModelNotSupportThinking`）
  - 迁移 `AIError` 结构、`userMessages` 中文提示映射、`NewAIError`、`ToJSON`、`AIErrorWrapper`
  - 增强 `ClassifyError`：`errors.As` 同时支持两种错误族
    - eino 转换后的 `*openai.APIError`（`github.com/cloudwego/eino-ext/components/model/openai`）
    - 当前 aicli 使用的 `*APIError` / `*RequestError`（`github.com/sashabaranov/go-openai`）
  - 将状态码/错误码分类逻辑抽成共享函数（`classifyByStatus(statusCode, code, message, raw)`），避免两套类型重复代码
  - context 超时/取消、`net.OpError`、文本 fallback 分类逻辑保持不变
- 迁移并扩展测试：`internal/aierrors/errors_test.go` 覆盖原全部用例，并新增 eino `*APIError` 分类用例
- 删除 `internal/aicli/errors.go` 与 `internal/aicli/errors_test.go`
- aicli 内部（`openai.go`、`client.go`）改为引用 `aierrors`
- 三个调用方改引用 `aierrors`：`ai_service.go`（2 处）、`vector_service.go`（1 处）、`app.go`（3 处）

## Impact

- Affected specs: add-ai-error-wrapper（原始错误包装 spec，已完结）
- Affected code:
  - `internal/aierrors/`（新增）
  - `internal/aicli/errors.go`（删除）、`internal/aicli/errors_test.go`（删除）、`internal/aicli/openai.go`、`internal/aicli/client.go`
  - `internal/services/ai_service.go`、`internal/services/vector_service.go`
  - `app.go`

## ADDED Requirements

### Requirement: 独立错误包 aierrors
系统 SHALL 提供 `internal/aierrors` 包，包含：12 个错误分类常量、`AIError` 结构、`NewAIError`、`ToJSON`、`AIErrorWrapper`、`ClassifyError`。外部通过 `aierrors.X` 访问，aicli 与 eino 均可使用。

#### Scenario: aicli 错误可分类（sashabaranov 类型）
- **WHEN** aicli 调用返回 `sashabaranov/go-openai` 的 `*APIError`（如 HTTPStatusCode=401）
- **THEN** `aierrors.ClassifyError` 返回 `Category=CategoryAuthError` 的 `AIError`，`UserMsg` 非空

#### Scenario: eino 错误可分类（eino-ext 类型）
- **WHEN** eino 组件返回 `eino-ext/components/model/openai` 的 `*APIError`（如 HTTPStatusCode=429）
- **THEN** `aierrors.ClassifyError` 返回 `Category=CategoryRateLimit` 的 `AIError`

#### Scenario: 用户取消不报错
- **WHEN** 错误为 `context.Canceled`
- **THEN** `aierrors.ClassifyError` 返回 nil（保持原有行为）

## MODIFIED Requirements

### Requirement: aicli 错误分类
原 `internal/aicli/errors.go` 中的全部错误分类能力迁移至 `aierrors`；aicli 不再自带分类逻辑，`openai.go`/`client.go` 改为引用 `aierrors.ClassifyError` 与 `aierrors.AIErrorWrapper`。

## REMOVED Requirements

### Requirement: aicli 包内错误分类
**Reason**: 需要被 aicli 与 eino 共用，独立成包便于后续 eino 平替时直接复用。
**Migration**: 所有引用方改用 `internal/aierrors`；`internal/aicli/errors.go`、`internal/aicli/errors_test.go` 删除。
