# 联网搜索（Tavily）功能 Spec

## Why

AI 助手目前知识截止于模型训练日期，无法获取实时信息。添加联网搜索后，用户提问时可以获取最新网络信息，提升回答的时效性和准确性。

## What Changes

- **后端新增** `search_service.go` — 封装 Tavily API 调用，搜索互联网并返回结构化结果
- **后端修改** `ai_service.go` — `CallAIStream` 新增 `searchEnabled bool` 参数，启用时先搜索再调 AI
- **后端修改** `app.go` — 新增 `SearchWeb` binding，更新 `CallAIStream` 签名
- **后端修改** `AIConfig` — 新增 `TavilyAPIKey string` 字段
- **前端设置页** — 新增「Tavily API Key」输入框和「联网搜索默认开启」开关
- **前端 AI 聊天** — 新增「联网搜索」开关（复用现有 toggle CSS）
- **go.mod** — 新增 `github.com/hekmon/tavily` 依赖

## Impact

- Affected specs: AI 助手（流式回复/深度思考）
- Affected code: `internal/services/`（新增 + 修改）、`app.go`、`frontend/src/main.js`、`frontend/src/js/ai-chat.js`、`frontend/index.html`

## ADDED Requirements

### Requirement: 联网搜索后端服务

The system SHALL provide a web search service that queries Tavily API and returns structured results.

#### Scenario: 用户启用联网搜索并提问
- **WHEN** 用户在 AI 对话中启用了联网搜索开关并发送消息
- **THEN** 后端先调用 Tavily 搜索获取最新信息，将搜索结果作为上下文注入 system message，再调用 AI 流式回复

#### Scenario: Tavily API Key 未配置
- **WHEN** 用户启用了联网搜索但 Tavily API Key 为空或无效
- **THEN** 搜索步骤静默跳过，仅使用 AI 自身知识回复，不阻塞对话

#### Scenario: Tavily 搜索超时或失败
- **WHEN** Tavily API 调用超时或返回错误
- **THEN** 搜索步骤静默跳过，仅使用 AI 自身知识回复

### Requirement: 前端联网搜索开关

The system SHALL provide a toggle in the AI chat view to enable/disable web search.

#### Scenario: 切换联网搜索
- **WHEN** 用户点击联网搜索开关
- **THEN** 开关状态切换，localStorage 保存状态，设置页同步状态

### Requirement: 搜索上下文注入

The system SHALL format search results and inject them into the AI's system message.

#### Scenario: 搜索结果注入
- **WHEN** 联网搜索完成且有结果返回
- **THEN** 结果按「标题 + 来源 URL + 正文摘要」格式化为文本，追加到 system message 中

## MODIFIED Requirements

### Requirement: CallAIStream 流式调用

The system SHALL support an optional search phase before the AI call.

- `CallAIStream` 签名新增 `searchEnabled bool` 参数
- 当 `searchEnabled = true` 时，先搜索再将结果注入 system message
- 搜索错误不阻塞主流程

### Requirement: AIConfig 配置结构

`AIConfig` 结构体新增 `TavilyAPIKey string \`json:"tavily_api_key"\`` 字段。
前端设置页新增对应输入框，保存时写入 SettingService。
