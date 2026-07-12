# 持久化 AI 会话操作栏配置 Spec

## Why

当前 AI 助手的操作栏配置（选择的模型、是否深度思考、搜索源开关、卡片召回、引用的笔记、启用的技能）仅存在于前端内存中，切换会话或重启程序后全部丢失。需要持久化到 SQLite，使得切回会话时能立即恢复上次的操作栏状态。

## What Changes

- **NEW**: `AISessionConfig` GORM 模型，与会话一对一关联，存储所有操作栏配置
- **NEW**: 后端 `SaveSessionConfig` / `LoadSessionConfig` / `CreateDefaultSessionConfig` 方法
- **NEW**: `app.go` 新增 2 个 Wails 绑定（保存/加载会话配置）
- **MODIFIED**: `database/db.go` 新增 `AutoMigrate` 表
- **MODIFIED**: `ai-chat.js` 在切换会话时恢复配置、在变更操作栏时保存配置
- **MODIFIED**: `ai-chat.js` 新建会话时从全局设置复制初始值
- **MODIFIED**: 设置页保存不再同步覆盖操作栏状态（已解耦）

## Impact

- Affected specs: `persist-ai-sessions`（在其基础上扩展）
- Affected code:
  - `internal/models/`：新增 `ai_session_config.go`
  - `internal/database/db.go`：追加 `AutoMigrate`
  - `internal/services/ai_service.go`：新增方法
  - `app.go`：新增绑定
  - `frontend/src/js/ai-chat.js`：状态管理逻辑
  - `frontend/src/main.js`：移除全局设置同步操作栏的逻辑

## ADDED Requirements

### Requirement: 数据模型

系统 SHALL 在 SQLite 中通过 `AISessionConfig` 表持久化每个会话的操作栏配置，与 `AISession` 一对一关联。

| 字段 | 类型 | 说明 | 初始值 |
|------|------|------|--------|
| ID | uint (PK) | 自增主键 | - |
| SessionID | uint (FK, unique) | 关联会话 | - |
| ModelName | string(100) | 选择的模型 | 全局 `ai_model` |
| EnableThinking | bool | 深度思考开关 | 全局 `ai_thinking_enabled` |
| ZhihuSearchEnabled | bool | 知乎搜索开关 | 全局 `zhihu_search_enabled` |
| ZhihuGlobalSearchEnabled | bool | 知乎全局搜索开关 | 全局 `zhihu_global_search_enabled` |
| TavilySearchEnabled | bool | Tavily 搜索开关 | 全局 `tavily_search_enabled` |
| EnableCardRecall | bool | 卡片召回开关 | 全局 `ai_card_recall_enabled` |
| ReferencedNotes | text | 引用的笔记列表 (JSON) | `[]` |
| EnabledSkills | text | 启用的技能列表 (JSON) | `[]` |

### Requirement: 新建会话时创建默认配置

- **WHEN** 用户创建新会话（`CreateAISession`）
- **THEN** 自动创建一条 `AISessionConfig` 记录
- **AND** `ModelName` 从全局设置 `ai_model` 读取
- **AND** `EnableThinking` 从全局 `ai_thinking_enabled` 读取
- **AND** 三个搜索开关分别从全局 `zhihu_search_enabled` / `zhihu_global_search_enabled` / `tavily_search_enabled` 读取
- **AND** `EnableCardRecall` 从全局 `ai_card_recall_enabled` 读取
- **AND** `ReferencedNotes` / `EnabledSkills` 初始化为空 JSON 数组

### Requirement: 切换会话时恢复配置

- **WHEN** 用户切换到某个会话
- **THEN** 加载该会话的 `AISessionConfig`
- **AND** 恢复模型选择器下拉选中状态
- **AND** 恢复深度思考 toggle 状态
- **AND** 恢复三个搜索源 checkbox 的 checked 状态
- **AND** 恢复卡片召回 toggle 状态
- **AND** 恢复笔记引用 chips
- **AND** 恢复技能 chips
- **AND** 如果配置记录不存在（如旧数据），按默认值处理，不报错

### Requirement: 操作栏变更时保存配置

- **WHEN** 用户在操作栏切换模型
- **THEN** 调用 `SaveSessionConfig` 写入数据库（立即保存）
- **WHEN** 用户点击深度思考 toggle
- **THEN** 同上立即保存
- **WHEN** 用户勾选/取消搜索源 checkbox
- **THEN** 同上立即保存
- **WHEN** 用户切换卡片召回 toggle
- **THEN** 同上立即保存
- **WHEN** 用户确认引用笔记
- **THEN** 同上立即保存
- **WHEN** 用户添加/移除技能
- **THEN** 同上立即保存

### Requirement: 设置页与操作栏解耦

- **WHEN** 用户在设置页修改 `ai_model` / `ai_thinking_enabled` / 搜索开关 / `ai_card_recall_enabled`
- **THEN** 只更新全局 settings 表
- **AND** 不再遍历同步到任何已有会话的操作栏
- **AND** 新建会话时仍使用当前全局设置作为初始值
- **NOTE**：搜索源的 `disabled` 状态仍由全局密钥（`tavily_api_key` / `zhihu_access_secret` 是否为空）决定，但 `checked` 状态从会话配置读取

## MODIFIED Requirements

### Requirement: 全局设置同步操作栏（现有）
- **MODIFIED**: 移除 `main.js` 中 `loadSettingToUI()` 里同步 AI 聊天栏 toggle 的代码（第 7689-7705 行）
- 原因：操作栏状态已由会话配置管理，不再由全局设置覆盖

### Requirement: 搜索源禁用逻辑（现有）
- **MODIFIED**: 搜索源 checkbox 的 `disabled` 状态仍由全局密钥决定（与之前一致）
- **AND**: `checked` 状态从会话配置读取（不再从全局设置读取）

## REMOVED Requirements

无
