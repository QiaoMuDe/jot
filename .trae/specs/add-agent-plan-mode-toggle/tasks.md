# Tasks

- [x] Task 1: 后端数据模型 — Session 配置新增 `plan_mode` 字段
  - [x] SubTask 1.1: `internal/models/ai_session_config.go` 的 `AISessionConfig` 结构体新增 `PlanMode bool` 字段（`gorm:"column:plan_mode;default:false" json:"plan_mode"`）
  - [x] SubTask 1.2: `internal/services/ai_service.go` 的 `SessionConfig` 结构体新增 `PlanMode bool` 字段（`json:"plan_mode"`）
  - [x] SubTask 1.3: `SaveSessionConfig` 方法的 `map[string]interface{}` 赋值中追加 `"plan_mode": cfg.PlanMode`
  - [x] SubTask 1.4: `LoadSessionConfig` 方法的返回 `SessionConfig` 中追加 `PlanMode: record.PlanMode`
  - [x] SubTask 1.5: `CreateDefaultSessionConfig` 方法中 `PlanMode` 默认为 `false`（零值，无需显式赋值，确认结构体初始化一致即可）
  - [x] SubTask 1.6: `internal/database/db.go` 的废弃列清理注释中确认 `plan_mode` 不会被误删（AutoMigrate 自动添加新列，无需手动迁移）

- [x] Task 2: 后端工具元数据 — `ToolMeta` 新增 `PlanOnly` 标记
  - [x] SubTask 2.1: `internal/agent/tools/meta.go` 的 `ToolMeta` 结构体新增 `PlanOnly bool` 字段（`json:"plan_only"`）
  - [x] SubTask 2.2: `BuiltinTools()` 中 `create_plan` 和 `update_plan` 两条的 `PlanOnly` 设为 `true`，其余为 `false`（零值不写）

- [x] Task 3: 后端 Agent 请求链路 — 按模式过滤计划工具
  - [x] SubTask 3.1: `internal/agent/types.go` 的 `Request` 结构体新增 `PlanMode bool` 字段
  - [x] SubTask 3.2: `internal/agent/registry.go` 的 `buildTools` 函数签名新增 `planMode bool` 参数；`planMode=false` 时在 `all` 列表构建后跳过 `create_plan` / `update_plan`（在 disabled 过滤之前额外过滤）
  - [x] SubTask 3.3: `internal/agent/agent.go` 的 `Run` 方法中，读取 `req.PlanMode` 传入 `buildTools`

- [x] Task 4: 后端 GenModelInput 钩子 — Agent 模式跳过计划注入
  - [x] SubTask 4.1: `internal/agent/agent.go` 中 `genPlanHint` 调用处，判断 `req.PlanMode` 为 false 时直接返回空字符串，不注入计划提示
  - [x] SubTask 4.2: 结果兜底逻辑（补建计划/补标步骤）包裹在 `req.PlanMode` 判断内，Agent 模式下跳过

- [x] Task 5: 后端绑定方法 — `GetAgentTools` 标注 plan_only + `CallAIAgentStream` 读取配置
  - [x] SubTask 5.1: `app.go` 中 `GetAgentTools` 返回的工具列表，根据 `ToolMeta.PlanOnly` 字段标注 `plan_only` 属性供前端消费
  - [x] SubTask 5.2: `app.go` 中 `CallAIAgentStream` 读取 `SessionConfig.PlanMode`，传入 `agent.Request.PlanMode`

- [x] Task 6: 前端工具栏 — Agent/Plan 模式切换按钮
  - [x] SubTask 6.1: `frontend/index.html` 中 AI 对话工具栏模型选择器左侧新增 pill 按钮组容器（`#aiModeToggle`），包含 Agent / Plan 两个选项
  - [x] SubTask 6.2: `frontend/src/js/ai-chat.js` 中新增模式切换逻辑：点击切换 → 更新 active 态 → 调用 `SaveSessionConfig` 保存 `plan_mode` → 显示通知"已切换到 Plan/Agent 模式"
  - [x] SubTask 6.3: 加载会话时（`loadSessionConfig`）读取 `plan_mode` 同步按钮 active 态
  - [x] SubTask 6.4: 切换会话时按钮状态跟随更新
  - [x] SubTask 6.5: `frontend/src/css/components/ai-chat.css` 新增 pill 按钮组样式（复用现有 pill/toggle 设计语言，accent 高亮激活项）

- [x] Task 7: 前端设置页 — Plan 模式工具禁用展示与交互
  - [x] SubTask 7.1: `frontend/src/main.js` 中渲染 Agent 工具列表时，检查 `plan_only` 属性：若为 `true` 则 checkbox 设为 disabled、整行添加 `is-plan-only` CSS 类、底部追加说明文字"仅 Plan 模式可用"
  - [x] SubTask 7.2: 为 `is-plan-only` 条目绑定点击事件：点击整行或 checkbox 时播放 shake 抖动动画（添加 `shake` class，0.4s 后移除），并调用现有 Toast 通知方法提示"此工具仅在 Plan 模式下可用，请切换到 Plan 模式"
  - [x] SubTask 7.3: `frontend/src/css/components/settings-panel.css` 新增 `is-plan-only` 禁用样式（灰色文字、checkbox 透明度降低、说明文字小号灰色）和 shake 动画 keyframes

- [x] Task 8: 编译验证
  - [x] SubTask 8.1: Go 编译通过（`go build ./...`）
  - [x] SubTask 8.2: 前端构建通过（如使用打包工具则执行构建命令，否则确认无语法错误）

# Task Dependencies

- Task 1（数据模型）无依赖，可先行
- Task 2（元数据）无依赖，可与 Task 1 并行
- Task 3 依赖 Task 1（需要 Request.PlanMode 字段）和 Task 2（需要 ToolMeta.PlanOnly）
- Task 4 依赖 Task 3（需要 req.PlanMode 可用）
- Task 5 依赖 Task 1、Task 2、Task 3
- Task 6 依赖 Task 1（需要 SessionConfig.PlanMode）和 Task 5（需要 SaveSessionConfig 可用）
- Task 7 依赖 Task 2（需要 ToolMeta.PlanOnly）和 Task 5（需要 GetAgentTools 返回 plan_only）
- Task 8 依赖所有前置任务
