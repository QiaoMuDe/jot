# 后端移除服务商（Provider / Ollama）与库表字段变更 Spec

## Why
前端已移除服务商选择与 Ollama 支持（`remove-frontend-provider-ollama` 已完成）。后端仍保留 provider 概念：`aicli` 按 Provider 分发到 OpenAI/Ollama 两套实现，settings 表存在 `ai_provider` / `ai_embed_provider` 键，`api_profiles` 表存在 `provider` 列，内置 Ollama 预设依赖原生协议。本次彻底移除后端 provider 概念，统一按 OpenAI 兼容协议请求，并完成库表数据迁移。

## What Changes
- **BREAKING** `internal/aicli`：删除 `Config.Provider` / `Client.Provider` 字段与 4 处 `switch c.Provider` 分发；**删除整个 `ollama.go`**；统一走 OpenAI 兼容实现
- **BREAKING** `internal/services/ai_service.go`：`AIConfig` 删除 `Provider`；`GetConfig`/`SaveConfig` 不再读写 `ai_provider`；`TestConnection` / `FetchModels` 删除 provider 分支与 Ollama 实现（`testOllamaConnection` / `fetchOllamaModels` / `ollamaTagResponse`）
- **BREAKING** Wails 方法签名变更（前端调用与 `frontend/wailsjs/go/main/App.js` 同步）：
  - `TestAIBaseURL(provider, baseURL, apiKey)` → `TestAIBaseURL(baseURL, apiKey)`
  - `TestAIConnection(provider, baseURL, apiKey)` → `TestAIConnection(baseURL, apiKey)`
  - `FetchAIModels(provider, baseURL, apiKey)` → `FetchAIModels(baseURL, apiKey)`
  - `CreateProfile(name, provider, baseURL, apiKey)` → `CreateProfile(name, baseURL, apiKey)`
  - `UpdateProfile(id, name, provider, baseURL, apiKey)` → `UpdateProfile(id, name, baseURL, apiKey)`
- **BREAKING** `GetEmbedConfig()` 返回 `(provider, baseURL, apiKey, model, err)` → `(baseURL, apiKey, model, err)`；`agent.Deps.GetEmbedConfig` 与 `tools.go` 同步
- `internal/services/types.go`：`SettingsConfig` 删除 `AIProvider` / `AIEmbedProvider`（含 `GetAllSettings` / `SaveAllSettings` 对应行）
- `internal/services/profile_service.go`：`CreateProfile` / `UpdateProfile` 去掉 provider 参数；`SwitchProfile` 不再写入 `*_provider` 键
- `internal/models/api_profile.go`：`APIProfile` 删除 `Provider` 字段
- `internal/database/db.go`：默认设置删除 `ai_provider` / `ai_embed_provider` 键；`ai_embed_base_url` 默认值由 `http://localhost:11434` 改为空字符串
- `internal/database/builtin_profiles.go`：删除全部 `Provider` 字段；内置 "Ollama" 预设 BaseURL 改为 `http://localhost:11434/v1`（OpenAI 兼容端点）
- **新增启动迁移**（`app.go` startup 内调用）：删除 settings 表 `ai_provider` / `ai_embed_provider` 键；`api_profiles` 中 `provider='ollama'` 记录更新为 `provider='openai'` 且 BaseURL 无 `/v1` 后缀时追加；settings 表 `ai_embed_base_url` 若含 ollama 默认地址且无 `/v1` 则追加；尝试 `ALTER TABLE api_profiles DROP COLUMN provider`（失败则忽略，保留列但不再使用）
- `go.mod`：删除 `ollama.go` 后执行 `go mod tidy` 移除 `github.com/ollama/ollama`（vec-poc/agent-demo 为独立模块，不受影响）
- 前端最小联动：`main.js` / `ai-chat.js` 中 `TestAIBaseURL` / `TestAIConnection` / `FetchAIModels` / `CreateProfile` / `UpdateProfile` 调用去掉 `'openai'` 占位参数；`saveSettings` 删除 `ai_provider` / `ai_embed_provider` 字段

## Impact
- 受影响能力：AI 连接配置、配置预设、AI 对话/Agent、向量量化与卡片召回
- 受影响代码：`internal/aicli/`、`internal/services/`、`internal/models/`、`internal/database/`、`internal/agent/`、`app.go`、`go.mod`、前端 `main.js` / `ai-chat.js` / `frontend/wailsjs/go/main/App.js`
- 明确不动：`vec-poc/`、`agent-demo/`（独立 go.mod 的验证项目）、`internal/aicli/errors_test.go`（仅 OpenAI 错误分类测试）

## REMOVED Requirements

### Requirement: Provider 字段与格式分发
**Reason**: 只保留 OpenAI 兼容一种格式。
**Migration**: `aicli.Config`/`Client` 删除 Provider，`Stream`/`Chat`/`Embed`/`EmbedWithProgress` 直接调用 OpenAI 实现；`services.AIConfig`、`SettingsConfig`、`APIProfile` 删除 Provider 字段。

### Requirement: Ollama 原生协议支持
**Reason**: 不再兼容 Ollama 原生 API（/api/chat、/api/tags、/api/embed）。
**Migration**: 删除 `internal/aicli/ollama.go`、`testOllamaConnection`、`fetchOllamaModels`；内置 "Ollama" 预设改为 OpenAI 兼容端点 `http://localhost:11434/v1`（Ollama 服务本身支持该端点）。

### Requirement: 设置键与库表列
**Reason**: provider 概念彻底移除。
**Migration**: 启动迁移删除 settings 表 `ai_provider` / `ai_embed_provider` 键与 `api_profiles.provider` 列；已有 ollama 预设数据改写为 openai + `/v1` 地址。

## MODIFIED Requirements

### Requirement: AI 连接测试与模型获取
不再按 provider 分发，统一走 OpenAI 兼容 `/models` 探测。

#### Scenario: 测试连接 / 获取模型
- **WHEN** 调用 `TestAIBaseURL(baseURL, apiKey)` 或 `FetchAIModels(baseURL, apiKey)`
- **THEN** 后端固定按 OpenAI 兼容协议请求 `{baseURL}/models`

### Requirement: 量化连接校验
`ValidateCardRecall` / `ValidateVectorIndexConfig` / `TestVectorIndexConnection` / `startVectorIndex` 不再判断 provider。

#### Scenario: 校验量化配置
- **WHEN** 量化连接 BaseURL 或 Model 未配置
- **THEN** 返回"请先在设置中配置量化连接与量化模型"
- **WHEN** APIKey 为空
- **THEN** 返回"请先填写量化 API Key"

### Requirement: 启动数据迁移
老用户数据库无缝升级。

#### Scenario: 老库启动
- **WHEN** 应用启动且 settings 表存在 `ai_provider` / `ai_embed_provider` 键
- **THEN** 自动删除该两键
- **WHEN** `api_profiles` 存在 `provider='ollama'` 记录且 BaseURL 无 `/v1`
- **THEN** Provider 改写为 `openai`、BaseURL 追加 `/v1`
- **WHEN** `api_profiles` 存在 `provider` 列且 SQLite 支持删列
- **THEN** 删除该列（不支持时忽略，保留但不使用）
