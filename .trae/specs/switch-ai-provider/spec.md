# Switch AI Provider & Migrate to LangChainGo Spec

## Why

当前 AI 服务通过手写 HTTP 客户端仅支持 OpenAI 兼容协议，切换服务商（如 Anthropic Claude、Google Gemini）需要重写大量代码。引入 LangChainGo 统一接口后可零成本切换服务商，同时减少维护代码量。

## What Changes

### Backend — 新增 `provider` 字段
- `services.AIConfig` 结构体新增 `Provider` 字段
- 后端设置存储新增 `ai_provider` key
- 前端设置页面新增「服务商」下拉选择器

### Backend — 替换为 LangChainGo
- 引入 `github.com/tmc/langchaingo` 依赖
- 用 `llms.Model` 接口替换当前 `net/http` 手写客户端
- `CallAIStream` 根据 `provider` 路由到对应的 LangChainGo Provider
- 保持现有事件推送机制不变（`ai:stream-chunk` / `ai:stream-thinking` / `ai:stream-done` / `ai:stream-error`）
- `TestBaseURL` / `FetchModels` 改为 provider 感知

### Frontend — 设置页 「服务商」下拉
- 新增「服务商」选择器，4 个选项：`OpenAI 兼容` / `Anthropic` / `Google Gemini` / `Ollama`
- 根据选择动态显隐 Base URL 字段（Anthropic/Gemini 隐藏，OpenAI/Ollama 显示）
- 模型获取按钮改为 provider 感知（OpenAI 兼容走 `/models`，其余跳过）

### Frontend — 保持兼容
- `AIConfig` 的 `base_url` / `api_key` / `model` 字段不变，前端已有读写代码不需要修改
- 原有 `window.go.main.App.GetAIConfig()` / `SaveAIConfig()` 接口签名不变，只是内部多了 `provider`

## Impact
- Affected specs: AI 助手
- Affected code:
  - `internal/services/ai_service.go` — **重写**（替换 LangChainGo）
  - `internal/services/types.go` — **修改** AIConfig 加 provider
  - `app.go` — **修改** CallAIStream / TestBaseURL / FetchModels
  - `frontend/index.html` — **修改** AI 设置区域加 provider 下拉
  - `frontend/src/main.js` — **修改** load/save 设置逻辑
  - `frontend/wailsjs/go/models.ts` — **自动生成**（Wails rebuild 后）
  - `frontend/wailsjs/go/main/App.d.ts` — **自动生成**
  - `go.mod` / `go.sum` — **新增** langchaingo 依赖

## ADDED Requirements

### Requirement: Provider Selection
The system SHALL allow users to select an AI provider from a dropdown.

#### Scenario: Switch provider
- **WHEN** user selects a different provider in settings
- **THEN** the Base URL field shall show/hide based on provider type
- **AND** the model label shall reset to `-- 请先获取模型列表 --` / `--` (for non-OpenAI providers)
- **AND** the config shall be saved with the new provider value

#### Scenario: Persist provider
- **GIVEN** user has selected a provider and saved
- **WHEN** user reopens settings
- **THEN** the provider dropdown shall show the previously saved value
- **AND** the Base URL field visibility shall match the saved provider

### Requirement: LangChainGo Backend
The system SHALL use LangChainGo for all AI API calls.

#### Scenario: OpenAI-compatible call
- **GIVEN** provider is `openai`
- **WHEN** CallAIStream is invoked
- **THEN** the backend shall use `openai.New()` with the configured BaseURL/Token/Model
- **AND** streaming chunks shall be pushed via `ai:stream-chunk` events
- **AND** thinking/reasoning content (when enabled) shall be pushed via `ai:stream-thinking`

#### Scenario: Anthropic call
- **GIVEN** provider is `anthropic`
- **WHEN** CallAIStream is invoked
- **THEN** the backend shall use `anthropic.New()` with the configured Token/Model
- **AND** Anthropic's `thinking` mode shall be enabled when `thinkingEnabled` is true

#### Scenario: Google Gemini call
- **GIVEN** provider is `google`
- **WHEN** CallAIStream is invoked
- **THEN** the backend shall use `googleai.New()` with the configured APIKey/Model

#### Scenario: Ollama call
- **GIVEN** provider is `ollama`
- **WHEN** CallAIStream is invoked
- **THEN** the backend shall use `ollama.New()` with the configured ServerURL/Model

## MODIFIED Requirements

### Requirement: AI Configuration (modified)
Previously `AIConfig` had 3 fields. Now it has 4:

```go
type AIConfig struct {
    Provider string `json:"provider"`  // "openai" | "anthropic" | "google" | "ollama"
    BaseURL  string `json:"base_url"`
    APIKey   string `json:"api_key"`
    Model    string `json:"model"`
}
```

### Requirement: Fetch Models (modified)
- **WHEN** provider is `openai` and "获取列表" is clicked → call `/models` endpoint as before
- **WHEN** provider is not `openai` → hide/disable "获取列表" button (models are fixed by provider)

### Requirement: Connection Test (modified)
- `TestAIBaseURL` shall accept a provider parameter and adapt test logic accordingly
