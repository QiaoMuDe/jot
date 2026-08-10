# Tasks

> **签名契约（各任务共同遵守）**
> - `TestAIBaseURL(baseURL, apiKey)` / `TestAIConnection(baseURL, apiKey)` / `FetchAIModels(baseURL, apiKey)`
> - `CreateProfile(name, baseURL, apiKey)` / `UpdateProfile(id, name, baseURL, apiKey)`
> - `GetEmbedConfig() (baseURL, apiKey, model string, err error)`；`agent.Deps.GetEmbedConfig` 同签名
> - `aicli.Config` 不再有 Provider 字段；`services.AIConfig`、`SettingsConfig`、`models.APIProfile` 不再有 Provider 字段

- [x] Task 1: aicli 模块移除 Provider 与 Ollama 实现
  - [x] `internal/aicli/types.go`：`Config` 删除 `Provider` 字段，更新注释（去掉 ollama 描述）
  - [x] `internal/aicli/client.go`：`Client` 删除 `Provider` 字段；`NewClient` 去掉 `Provider: cfg.Provider`；`Stream`（L60-65）、`Chat`（L93-100）、`Embed`（L104-114）、`EmbedWithProgress`（L119-152）删除 `switch c.Provider`，直接调用 openai 实现（ollama 分支整段删除），更新注释
  - [x] 删除 `internal/aicli/ollama.go` 整个文件
  - [x] `internal/aicli/openai.go`：L39 注释"如 Qwen3 / Ollama OpenAI 兼容接口"去掉 Ollama 字样
- [x] Task 2: services 层移除 Provider
  - [x] `internal/services/ai_service.go`：`AIConfig` 删 `Provider` 字段；`GetConfig` 删 `svc.Get("ai_provider")`；`SaveConfig` 删 `svc.Set("ai_provider", ...)`；`CallAI`/`CallAIStream` 构造 `aicli.Config` 去掉 `Provider` 行；`TestConnection` 直接 `return testOpenAIConnection(cfg)`，删除 `testOllamaConnection` 与 `testGenericConnection`（`testGenericConnection` 不再需要）；`FetchModels` 直接 `return fetchOpenAIModels(cfg)`，删除 `fetchOllamaModels` 与 `ollamaTagResponse`；同步更新相关注释
  - [x] `internal/services/types.go`：`SettingsConfig` 删 `AIProvider`/`AIEmbedProvider` 字段；`GetAllSettings` 删 L106/L110 两行；`SaveAllSettings` 删 `"ai_provider"`/`"ai_embed_provider"` 两行（约 L213/L217）
  - [x] `internal/services/profile_service.go`：`CreateProfile` 签名 `(name, baseURL, apiKey string, isDefault ...bool)`，结构体删 Provider 行；`UpdateProfile` 签名 `(id uint, name, baseURL, apiKey string)`，Updates map 删 "provider" 键；`SwitchProfile` 删 `svc.Set(prefix+"provider", ...)` 与相关注释；顶部函数注释更新
- [x] Task 3: models 与 database 移除 Provider
  - [x] `internal/models/api_profile.go`：`APIProfile` 删除 `Provider` 字段（gorm tag 与 json tag 一并删）
  - [x] `internal/database/db.go`：defaults 删除 `{Key: "ai_provider", ...}`（L533）与 `{Key: "ai_embed_provider", ...}`（L537）；`ai_embed_base_url` 默认值 `http://localhost:11434` 改为 `""`
  - [x] `internal/database/builtin_profiles.go`：删除所有 `Provider: "openai"` 行与"Ollama"预设的 `Provider: "ollama"` 行；"Ollama"预设 BaseURL 改为 `http://localhost:11434/v1`；注释示例改为 `{Name: "XX", BaseURL: "https://api.xxx.com"}`
- [x] Task 4: agent 模块签名同步
  - [x] `internal/agent/agent.go`：`Deps.GetEmbedConfig` 签名改为 `func() (baseURL, apiKey, model string, err error)`；L84 注释去掉 Ollama 字样
  - [x] `internal/agent/tools.go`：`recallNotesTool.getEmbedConfig` 字段签名同步；`InvokableRun` 中 `provider, baseURL, apiKey, model, err := r.getEmbedConfig()` 改为 `baseURL, apiKey, model, err := ...`，删除 `provider == ""` 判断；`aicli.NewClient` 构造去掉 `Provider` 行
- [x] Task 5: app.go 签名变更 + 启动迁移
  - [x] `CreateProfile`（L1434）签名 `(name, baseURL, apiKey string)`，日志去掉 provider 字段
  - [x] `UpdateProfile`（L1440）签名 `(id uint, name, baseURL, apiKey string)`，日志去掉 provider 字段
  - [x] `GetEmbedConfig`（L1549）签名 `(baseURL, apiKey, model string, err error)`，删 `a.settingService.Get("ai_embed_provider")` 行
  - [x] `ValidateCardRecall`（L1568-1596）：`provider, baseURL, apiKey, model, _ := a.GetEmbedConfig()` 改四值接收；删 `provider == ""` 判断与 `provider == "openai" && apiKey == ""` 分支（改 `if apiKey == ""`）；注释更新
  - [x] `ValidateVectorIndexConfig`（L1603-1612）：同上处理
  - [x] `startVectorIndex`（L1644-1660）：GetEmbedConfig 四值接收；删 provider 判断；`aicli.NewClient` 去 Provider
  - [x] `testAIConnection`（L1696）：签名 `(baseURL, apiKey, logName string)`，`AIConfig` 去 Provider
  - [x] `TestAIBaseURL`（L1709）/ `TestAIConnection`（L1714）：签名 `(baseURL, apiKey string)`
  - [x] `TestVectorIndexConnection`（L1720-1734）：四值接收；删 provider 判断与 `provider == "openai" &&` 分支；注释更新
  - [x] `FetchAIModels`（L1786）：签名 `(baseURL, apiKey string)`，`AIConfig` 去 Provider，日志去 provider 字段
  - [x] `SaveAIConfig`（L1416）：`CreateProfile("默认配置", cfg.BaseURL, cfg.APIKey, true)`
  - [x] `SaveSettings`（L1267-1271）：`CreateProfile("默认配置", cfg.AIBaseURL, cfg.AIAPIKey, true)`
  - [x] startup 迁移（L215-220）：`CreateProfile("默认配置", baseURL, apiKey, true)`，删 provider 读取
  - [x] L2242 附近 embed client：`embedProvider, embedBaseURL, ...` 改四值接收，`NewClient` 去 Provider
  - [x] 新增 `migrateProviderRemoval()` 方法并在 startup 中调用（在 `migrateSensitiveKeys()` 之前）：删除 settings 表 `ai_provider`/`ai_embed_provider` 键；`api_profiles` 中 `provider='ollama'` 记录改 provider='openai'、base_url 无 `/v1` 后缀时追加 `/v1`；settings 表 `ai_embed_base_url` 值含 `localhost:11434` 且无 `/v1` 时追加 `/v1`；执行 `ALTER TABLE api_profiles DROP COLUMN provider`（失败仅记日志不中断）
- [x] Task 6: 前端联动同步接口签名
  - [x] `frontend/src/main.js`：`fetchModelsAndRender` 内 `FetchAIModels('openai', m.url, m.key)` → `FetchAIModels(m.url, m.key)`；测试按钮事件删 `const provider = 'openai';` 且 `TestAIBaseURL(provider, url, key)` → `TestAIBaseURL(url, key)`（提示文案不再拼 provider）；`testPresetConnection` 删 provider 变量且 `TestAIConnection(baseURL, apiKey)`；`savePresetModal` 中 `UpdateProfile(editingProfileId, name, baseURL, apiKey)` 与 `CreateProfile(name, baseURL, apiKey)`；`saveSettings` 删除 `ai_provider: 'openai'` / `ai_embed_provider: 'openai'` 两行
  - [x] `frontend/src/js/ai-chat.js`：`FetchAIModels('openai', cfg.base_url, cfg.api_key)` → `FetchAIModels(cfg.base_url, cfg.api_key)`
  - [x] `frontend/wailsjs/go/main/App.js`（生成的绑定）：`CreateProfile` 参数 4→3、`FetchAIModels` 参数 3→2、`TestAIBaseURL` 3→2、`TestAIConnection` 3→2、`UpdateProfile` 5→4
- [x] Task 7: 构建与残留验证
  - [x] `go mod tidy`（根模块）确认移除 `github.com/ollama/ollama`
  - [x] `go build ./...` 与 `go vet ./...` 通过（根模块；vec-poc/agent-demo 有独立 go.mod 不参与）
  - [x] `go test ./internal/aicli/...` 通过
  - [x] `npm run build`（frontend）通过
  - [x] 根模块（internal/、app.go）全局搜索 `ollama` 无残留（大小写不敏感；仅迁移代码注释与 "Ollama" 预设名称合规命中）
  - [x] 前端 `main.js` / `ai-chat.js` / `wailsjs` 中无 `'openai'` 占位残留；后端无 `provider` 字段残留（`settingsConfig` 等无关命名除外）

# Task Dependencies
- [Task 7] depends on [Task 1] ~ [Task 6]
- 代码依赖：Task 5 依赖 Task 1/2/3 的签名（aicli.Config、AIConfig、APIProfile 无 Provider）；Task 4 依赖 Task 2/5 的 `GetEmbedConfig` 签名约定
- [Task 1]、[Task 2]、[Task 3]、[Task 6] 文件互不重叠，可并行；[Task 4]、[Task 5] 紧随其后（遵循签名契约可与其他任务并行）
