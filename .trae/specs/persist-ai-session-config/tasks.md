# Tasks

- [x] Task 1: 新建 AISessionConfig GORM 模型并注册 AutoMigrate
  - 创建 `internal/models/ai_session_config.go` — AISessionConfig 结构体
  - `database/db.go` 追加 `&models.AISessionConfig{}` 到 AutoMigrate

- [x] Task 2: 新增后端会话配置 CRUD 方法
  - `ai_service.go` 新增：
    - `CreateDefaultSessionConfig(sessionID uint) error` — 从全局设置创建默认配置
    - `SaveSessionConfig(sessionID uint, config SessionConfig) error` — 保存会话配置
    - `LoadSessionConfig(sessionID uint) SessionConfig` — 加载会话配置
  - 新增 `SessionConfig` 结构体用于前端交互
  - `app.go` 新增对应 2 个 Wails 绑定（SaveSessionConfig / LoadSessionConfig）
  - `CreateAISession()` 中调用 `CreateDefaultSessionConfig`

- [x] Task 3: 前端 — 切换会话时恢复配置
  - `switchSession()` 中加载消息后，额外调用 `LoadSessionConfig()`
  - 恢复模型选择器选中项
  - 恢复深度思考 toggle 状态
  - 恢复搜索源 checkbox 的 checked 状态
  - 恢复卡片召回 toggle 状态
  - 恢复笔记引用 chips
  - 恢复技能 chips
  - 配置不存在时按默认值处理

- [x] Task 4: 前端 — 操作栏变更时保存配置
  - 模型选择变更时调用 `SaveSessionConfig()`
  - 深度思考 toggle 切换时调用 `SaveSessionConfig()`
  - 搜索源 checkbox 变更时调用 `SaveSessionConfig()`
  - 卡片召回 toggle 切换时调用 `SaveSessionConfig()`
  - 笔记引用确认时调用 `SaveSessionConfig()`
  - 技能添加/移除时调用 `SaveSessionConfig()`

- [x] Task 5: 移除设置页对操作栏的同步覆盖
  - `main.js` 中移除 `loadSettingToUI()` 里同步 AI 聊天栏 toggle 的代码（第 7689-7705 行）
  - 新建会话时从全局设置复制初始值

# Task Dependencies
- Task 1 → Task 2（模型必须先定义）
- Task 2 → Task 3, Task 4（后端 API 先于前端调用）
- Task 2 → Task 5（创建默认配置依赖后端方法）
