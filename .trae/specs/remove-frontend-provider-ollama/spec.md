# 移除前端服务商（Provider / Ollama）选择与判断 Spec

## Why
AI 模块已确定只使用 OpenAI 兼容格式，不再兼容 Ollama 原生格式。前端目前仍存在"服务商"选择 UI（对话/量化两组分段控件、预设弹窗下拉）以及基于 `provider` 的分支判断逻辑。本次先行删除前端这些代码与组件，后端与库表改造另行进行。

## What Changes
- 删除对话设置页「服务商」分段控件 `aiProviderSegmented`（index.html）
- 删除量化连接「服务商」分段控件 `aiEmbedProviderSegmented`（index.html）
- 删除预设弹窗「服务商」下拉 `presetModalProvider*`（index.html）
- 删除 `main.js` 中全部 provider 相关状态、辅助函数、事件绑定、badge 渲染与判断逻辑
- 简化 `ai-chat.js` 中基于 provider 的空态判断（`hasRequired`）与模型获取调用
- 更新 `data-management.js` 中"openai 需 key"的分支注释
- 清理 `settings-panel.css` 中 `.preset-provider-badge` 样式
- **兼容策略（BREAKING 前的过渡态）**：后端 Wails 方法签名本次不改，前端对 `TestAIBaseURL` / `TestAIConnection` / `FetchAIModels` / `CreateProfile` / `UpdateProfile` 的调用保留原参数个数，`provider` 位置固定传字面量 `"openai"`，保证后端未改造时前端可正常运行；后端改造完成后此占位随签名一并移除。
- 前端 `saveSettings` 中 `ai_provider` / `ai_embed_provider` 固定提交 `"openai"`，保持设置表数据与最终目标一致。

## Impact
- 受影响能力：AI 连接配置（对话 + 量化）、配置预设管理、AI 对话空态判断
- 受影响代码（仅前端）：
  - `frontend/index.html`
  - `frontend/src/main.js`
  - `frontend/src/js/ai-chat.js`
  - `frontend/src/js/data-management.js`
  - `frontend/src/css/components/settings-panel.css`
- 明确不动：后端 Go（`internal/`）、数据库、Wails 绑定方法签名

## REMOVED Requirements

### Requirement: 服务商选择 UI
**Reason**: 不再兼容 Ollama 原生格式，服务商选择失去意义，只保留 OpenAI 兼容一种格式。
**Migration**: 设置页不再展示「服务商」项；预设弹窗不再展示「服务商」下拉；后端 `ai_provider` / `ai_embed_provider` 键由前端固定提交 `"openai"`（后端改造后整体移除）。

#### Scenario: 打开 AI 设置页
- **WHEN** 用户打开 AI 设置页（对话连接 / 量化连接）
- **THEN** 不再显示「服务商」行与 OpenAI/Ollama 分段按钮，直接显示 API 地址、API Key、模型等配置项

#### Scenario: 打开 / 编辑预设弹窗
- **WHEN** 用户新增或编辑 API 配置预设
- **THEN** 弹窗不再显示「服务商」下拉，保存与测试连接时 provider 固定为 `"openai"`

### Requirement: 前端 provider 分支判断
**Reason**: provider 概念从 UI 与逻辑层移除。
**Migration**: 删除 `main.js` 中 `getActiveProvider` / `setActiveProvider` / 分段控件辅助函数 / `AI_DEFAULT_URLS.ollama` / badge 渲染 / `canFetch` 判断等；删除 `ai-chat.js` 中 `provider === 'ollama'` 分支。

## MODIFIED Requirements

### Requirement: AI 连接测试与模型获取
不再依赖服务商选择：测试连接、获取模型列表时 provider 固定 `"openai"`，前端校验仅要求 API 地址与 API Key。

#### Scenario: 测试连接
- **WHEN** 用户点击「测试」按钮
- **THEN** 前端仅校验 API 地址（与 API Key）非空，调用 `TestAIBaseURL('openai', url, key)`

#### Scenario: 获取模型列表
- **WHEN** 用户点击「获取」按钮
- **THEN** 前端仅校验 API 地址（与 API Key）非空，调用 `FetchAIModels('openai', url, key)`

### Requirement: AI 对话空态判断
不再区分 Ollama（只需 base_url）与 OpenAI（需 api_key）。

#### Scenario: 激活 AI 对话视图
- **WHEN** 用户切到 AI 对话视图且未配置 API Key
- **THEN** 显示空态引导配置，判断逻辑简化为仅检查 `api_key`
