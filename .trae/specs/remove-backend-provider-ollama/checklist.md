# Checklist

- [x] `internal/aicli/ollama.go` 已删除；`types.go`/`client.go` 无 Provider 字段与 switch 分发
- [x] `internal/services/ai_service.go` 无 provider 读取/写入/分支；`testOllamaConnection`、`fetchOllamaModels`、`ollamaTagResponse` 已删除
- [x] `internal/services/types.go` 的 `SettingsConfig` 无 AIProvider/AIEmbedProvider（GetAllSettings/SaveAllSettings 同步）
- [x] `internal/services/profile_service.go`：CreateProfile/UpdateProfile 无 provider 参数；SwitchProfile 不写 `*_provider` 键
- [x] `internal/models/api_profile.go` 无 Provider 字段
- [x] `internal/database/db.go` 默认设置无 `ai_provider`/`ai_embed_provider` 键；`ai_embed_base_url` 默认值为空
- [x] `internal/database/builtin_profiles.go` 无 Provider 字段；"Ollama" 预设 BaseURL 为 `http://localhost:11434/v1`（名称保留为 spec 决策）
- [x] `internal/agent/` 的 `GetEmbedConfig` 签名为 `(baseURL, apiKey, model, err)`；embed client 无 Provider
- [x] `app.go` 五个 Wails 方法签名已按契约变更；GetEmbedConfig 四值返回；各校验函数无 provider 判断
- [x] `app.go` 新增 `migrateProviderRemoval()` 并在 startup 调用：删设置键、迁移 ollama 预设、删 provider 列
- [x] `go mod tidy` 移除 `github.com/ollama/ollama`（根模块）
- [x] `go build ./...`、`go vet ./...`、`go test ./internal/aicli/...` 通过
- [x] 前端 `main.js`/`ai-chat.js`/`wailsjs` 调用签名与后端一致，无 `'openai'` 占位残留
- [x] `npm run build` 通过
- [x] 根模块代码全局搜索 `ollama` 无残留（大小写不敏感；仅迁移代码注释与 "Ollama" 预设名称合规命中）
