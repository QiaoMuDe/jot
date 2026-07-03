# Tasks

- [x] Task 1: 创建 AIPrompt 模型 + 初始化插入内置提示词
  - [x] 新建 `internal/models/ai_prompt.go`：定义 `AIPrompt` 结构体（ID, Key, Name, Category, Content, IsBuiltin, UpdatedAt）
  - [x] `internal/database/db.go`：在 `AutoMigrate` 中增加 `&models.AIPrompt{}`
  - [x] `db.go`：在 `InitDB()` 中 AutoMigrate 后，检查并插入 11 条内置提示词（翻译 2 条 + 其他 9 条各 1 条），使用 `Where("is_builtin = ?", true).Count()` + `Create`

- [x] Task 2: 后端 GetSkillPrompts + CallAIStream 注入技能提示词
  - [x] `internal/services/ai_service.go`：新增 `GetSkillPrompts(skillIds []string) (string, error)` 方法，查表拼接
  - [x] `app.go`：`CallAIStream` 签名增加 `skillIds []string` 参数
  - [x] 在 `CallAIStream` 中调用 `GetSkillPrompts`，将返回的提示词注入到 `messages` 数组的 system role 中（若有 system message 则合并内容，若无则新建一条 system message）

- [x] Task 3: 前端删除 SKILL_PROMPTS + 适配新流程
  - [x] 删除 `ai-chat.js` 中 `SKILL_PROMPTS` 常量（第 114-370 行）
  - [x] 删除 `getSkillSystemPrompts()` 函数（第 1845-1861 行）
  - [x] 修改 `startStreaming()`：移除 `getSkillSystemPrompts()` 调用，将 `activeSkills` 的 key 映射为 DB key 后传入 `CallAIStream`
  - [x] 确认 `OPTIMIZE_EXPRESSION_PROMPT` 保留不动

# Task Dependencies
- [Task 1] → [Task 2]（Task 2 依赖模型存在）
- [Task 3] 与 [Task 1][Task 2] 可并行（前端后端独立修改）
