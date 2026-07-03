# 技能提示词迁移至后端数据库 Spec

## Why

当前 10 项技能的系统提示词全部硬编码在前端 `ai-chat.js` 的 `SKILL_PROMPTS` 常量中。这导致：
- 调整提示词措辞需要修改 JS 源码 + 重新编译前端
- 无法支持用户自定义提示词
- 后端作为 AI 请求的发送方，完全不感知技能提示词，注入路径绕路（前端拼好 → 传给后端 → 后端透传）

将提示词迁至后端数据库后，后端直接查表注入，为后续提示词管理界面、自定义技能等功能打下基础。

## What Changes

- **新增**: `models.AIPrompt` 模型，含 `Key`/`Name`/`Category`/`Content`/`IsBuiltin` 字段
- **修改**: `db.InitDB()` 中 `AutoMigrate` 增加 `AIPrompt`，并用 GORM 插入 11 条内置提示词（翻译 2 条 + 其他 9 条各 1 条）
- **新增**: `services.AIService.GetSkillPrompts(skillIds []string) (string, error)` 方法，查表拼接多个技能提示词
- **修改**: `app.go` `CallAIStream` 签名增加 `skillIds []string` 参数，后端查表注入技能提示词到 system message
- **修改**: `ai-chat.js` 删除 `SKILL_PROMPTS` 常量、`getSkillSystemPrompts()` 函数；`startStreaming()` 中把 `Object.keys(activeSkills)` 传入 `CallAIStream`
- **保留**: `OPTIMIZE_EXPRESSION_PROMPT`（"优化"按钮专用）保留在前端，因其与技能体系无关，且调用方式独特（非流式 `CallAI`，仅单次润色）

## Impact

- Affected code:
  - `internal/models/ai_prompt.go` — **新增**
  - `internal/database/db.go` — 加 AutoMigrate + 初始化数据
  - `internal/services/ai_service.go` — 新增 `GetSkillPrompts()`
  - `internal/services/ai_service.go` — `CallAIStream()` 内部注入技能提示词逻辑
  - `app.go` — `CallAIStream` 签名增加 `skillIds` + 调用 `GetSkillPrompts`
  - `frontend/src/js/ai-chat.js` — 删除 `SKILL_PROMPTS`、`getSkillSystemPrompts()`；修改 `startStreaming()` 传参
  - 前端无需改 `index.html`（技能菜单项定义不变）

## ADDED Requirements

### Requirement: AIPrompt 数据模型
系统 SHALL 提供 `AIPrompt` 模型存储技能提示词，支持按 key 唯一索引。

| 字段 | 类型 | 说明 |
|------|------|------|
| `ID` | `uint` (PK) | 主键 |
| `Key` | `string(64)` unique | 唯一键，如 `skill_translate_cn`、`skill_coding` |
| `Name` | `string(64)` | 中文名，如 "翻译→中译英"、"编程" |
| `Category` | `string(32)` indexed | 分类：`skill` / `builtin` |
| `Content` | `text` | 提示词正文 |
| `IsBuiltin` | `bool` default true | 内置标记，不可删除 |
| `UpdatedAt` | `time` | 更新时间 |

#### Scenario: 初始化时自动插入内置提示词
- **WHEN** `InitDB()` 执行 `AutoMigrate` 后
- **THEN** 检查 `AIPrompt` 表是否存在 `IsBuiltin = true` 的记录
- **WHEN** 没有内置记录（首次运行）
- **THEN** 用 GORM `Create` 插入 11 条内置提示词

### Requirement: 后端技能提示词注入
系统 SHALL 在 `CallAIStream` 中根据前端传入的 `skillIds`，从数据库查询并注入对应的系统提示词。

#### Scenario: 流式对话携带技能 ID
- **WHEN** 前端调用 `CallAIStream(myGen, messages, thinking, search, cardRecall, sessionID, isRegenerate, skillIds)`
- **THEN** 后端收到 `skillIds`（如 `["translate", "coding"]`）
- **THEN** 后端调用 `GetSkillPrompts(skillIds)` 查询 `AIPrompt` 表
- **THEN** 将拼接好的技能提示词注入到 `messages` 数组的 system message 中（若有多个 system message 则合并）
- **THEN** 将处理后的 `messages` 发送给 AI 客户端

## MODIFIED Requirements

### Requirement: 前端发送消息流程
修改 `startStreaming()` 中 messages 构建逻辑，前端不再处理技能提示词，仅处理笔记引用、卡片召回等系统上下文，将技能 ID 列表传给后端。

#### 旧流程
```
前端: systemContent = noteContext + cardRecall + skillPrompts (拼接)
前端: messages = [system(systemContent), ...history]
前端: CallAIStream(messages, ...)
后端: 透传 messages → AI
```

#### 新流程
```
前端: systemContent = noteContext + cardRecall (无 skillPrompts)
前端: messages = [system(systemContent), ...history]
前端: CallAIStream(messages, ..., skillIds=[...])
后端: skillPrompts = GetSkillPrompts(skillIds)    ← 查表
后端: messages = injectSkillPrompts(messages, skillPrompts)
后端: 注入后的 messages → AI
```

## REMOVED Requirements

### Requirement: 前端 SKILL_PROMPTS 常量
**Reason**: 技能提示词移至后端数据库管理，不再需要前端硬编码。
**Migration**: 删除 `ai-chat.js` 第 114-370 行的 `SKILL_PROMPTS` 常量和第 1845-1861 行的 `getSkillSystemPrompts()` 函数。
