# Tasks

- [ ] Task 1: 创建 Base64 编解码工具函数文件 `internal/services/crypto.go`
  - 定义前缀常量 `const encodedPrefix = "(zk)"`
  - 实现 `isEncoded(s string) bool` — 检查是否以 `(zk)` 开头
  - 实现 `EncodeB64(s string) string` — 空字符串返回空，否则 `(zk)` + 标准 Base64 编码
  - 实现 `DecodeB64(s string) string` — 空字符串返回空；有前缀则剥离前缀并解码；无前缀则原样返回（兼容存量明文）
- [ ] Task 2: 修改 `AIService.SaveConfig()` — 写入 DB 前对 `APIKey`、`TavilyAPIKey`、`ZhihuAccessSecret` 三个字段做 `EncodeB64`
- [ ] Task 3: 修改 `AIService.GetConfig()` — 从 DB 读取后对 `APIKey`、`TavilyAPIKey`、`ZhihuAccessSecret` 三个字段做 `DecodeB64`
- [ ] Task 4: 修改 `ProfileService` — 对 `api_key` 做编解码
  - `CreateProfile()`：写入前 `EncodeB64(apiKey)`
  - `UpdateProfile()`：写入前 `EncodeB64(apiKey)`
  - `SwitchProfile()`：写入 settings 表前 `EncodeB64(profile.APIKey)`
  - `ListProfiles()`：返回前对每个 profile 的 `APIKey` 做 `DecodeB64`
- [ ] Task 5: 在 `app.go` 的 `startup()` 中添加 `migrateSensitiveKeys()` 迁移逻辑
  - 扫描 settings 表中 `ai_api_key`、`tavily_api_key`、`zhihu_access_secret` 三个 key
  - 对每个非空值，检查是否以 `(zk)` 开头
  - 无前缀 → 执行 `EncodeB64()` 后写回（明文 → 编码）
  - 已有前缀 → 跳过（幂等安全）
  - 同样方式扫描 api_profiles 表中 `api_key` 字段
  - **位置**：放在 `startup()` 中确保默认笔记本之后、现有预设迁移之前

## Task Dependencies

- [Task 1] 无依赖，可最先完成
- [Task 2, 3, 4, 5] 依赖 [Task 1]
