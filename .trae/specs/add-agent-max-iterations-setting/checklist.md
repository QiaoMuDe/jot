# Checklist

- [x] SettingsConfig 包含 `AIAgentMaxIterations`（json: `ai_agent_max_iterations`），`GetAllSettings` 未配置时默认 20
- [x] `SaveAllSettings` 对 `ai_agent_max_iterations` 做范围校验（<1 → 20，>100 → 100）并持久化
- [x] internal/agent 不再硬编码迭代次数：`Run` 从设置读取配置，缺失/非法回退 `DefaultMaxIterations`（20），且 `ChatModelAgentConfig.MaxIterations` 与日志均使用解析后的值
- [x] `internal/agent.MaxIterations` 已重命名为 `DefaultMaxIterations`，无残留引用（含 playground 注释）
- [x] 设置页「对话与搜索」面板存在「Agent 运行上限」输入项（id: `aiAgentMaxIterations`，min 1 / max 100 / 默认 20）
- [x] `loadSettings` 回填该输入框；`saveSettings` 收集该字段；change 越界时重置并提示
- [x] 后端 `go build ./...` 与 `go vet ./...` 通过
- [x] 前端 lint / HTML 校验不引入新错误
