# Tasks

- [x] Task 1: 后端设置模型扩展（services/types.go）
  - [x] SubTask 1.1: `SettingsConfig` 新增字段 `AIAgentMaxIterations int`（json: `ai_agent_max_iterations`）
  - [x] SubTask 1.2: `GetAllSettings` 读取 `ai_agent_max_iterations`，默认值 20
  - [x] SubTask 1.3: `SaveAllSettings` 校验（<1 重置 20，>100 重置 100）并写入 `sets` 映射

- [x] Task 2: Agent 装配读取配置（internal/agent/agent.go）
  - [x] SubTask 2.1: 常量 `MaxIterations` 重命名为 `DefaultMaxIterations`，值保持 20
  - [x] SubTask 2.2: `Run` 内新增 `strconv` 导入，从 `s.deps.Setting` 读取 `ai_agent_max_iterations`，非法/缺失时回退 `DefaultMaxIterations`，替换第 247 行与第 422 行的引用
  - [x] SubTask 2.3: 更新 playground/agent-demo/main.go 中引用注释（`internal/agent.MaxIterations` → `DefaultMaxIterations`）

- [x] Task 3: 前端设置项 UI（frontend/index.html）
  - [x] SubTask 3.1: 在「对话与搜索」面板 Agent 工具设置项之后新增「Agent 运行上限」数字输入项（id: `aiAgentMaxIterations`，min 1 / max 100 / 默认 20）

- [x] Task 4: 前端加载/保存/校验（frontend/src/main.js）
  - [x] SubTask 4.1: `loadSettings` 中回填 `aiAgentMaxIterations` 输入框（缺省 20）
  - [x] SubTask 4.2: `saveSettings` 中收集 `ai_agent_max_iterations`（解析失败回退 20）
  - [x] SubTask 4.3: 新增 change 监听：<1 重置 20、>100 重置 100，保存并提示（参照 aiSearchResultLimit 模式）

- [x] Task 5: 验证
  - [x] SubTask 5.1: `go build ./...` 与 `go vet ./...` 通过
  - [x] SubTask 5.2: 前端 `cd frontend && npm run lint` 与 `npm run validate:html` 通过（或确认现有校验不报新错）

# Task Dependencies
- Task 2 依赖 Task 1（读取同一设置 key）
- Task 3、Task 4 依赖 Task 1（前端字段与后端字段对应）
- Task 5 依赖全部
