# Checklist

- [x] `AISessionConfig` 模型新增 `PlanMode` 列，AutoMigrate 自动创建
- [x] `SessionConfig` 结构体新增 `PlanMode` 字段，`SaveSessionConfig` / `LoadSessionConfig` / `CreateDefaultSessionConfig` 读写正确
- [x] `ToolMeta` 新增 `PlanOnly` 字段，`create_plan` / `update_plan` 的 `PlanOnly=true`
- [x] `agent.Request` 新增 `PlanMode` 字段
- [x] `buildTools` 按 `planMode` 过滤：Agent 模式不注册 create_plan / update_plan
- [x] `agent.Run` 将 `req.PlanMode` 传入 `buildTools`
- [x] `genPlanHint` 在 `planMode=false` 时跳过计划提示注入
- [x] 结果兜底逻辑在 `planMode=false` 时跳过（不补建计划、不补标步骤）
- [x] `CallAIAgentStream` 读取 `SessionConfig.PlanMode` 传入 `Request.PlanMode`
- [x] `GetAgentTools` 返回的工具列表包含 `plan_only` 属性
- [x] 工具栏模型选择器左侧显示 Agent/Plan pill 切换按钮，默认 Agent 激活
- [x] 点击切换按钮即时保存 `plan_mode` 并通知"已切换到 X 模式"
- [x] 切换会话时按钮 active 态跟随 `plan_mode` 同步
- [x] 设置页工具列表中 `create_plan` / `update_plan` 显示为禁用样式（灰色 + disabled checkbox）
- [x] 点击禁用工具条目触发 shake 抖动动画 + Toast 通知
- [x] Go 编译通过（`go build ./...`）
- [x] 前端无 JS 语法错误
