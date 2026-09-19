# Tasks

- [x] Task 1: 后端设置层支持两个迭代上限设置项（主 Agent 默认改 100 / 范围 1-500；新增子 Agent 默认 50 / 范围 1-200）
  - [x] SubTask 1.1: [db.go](file:///d:/峡谷/Dev/本地项目/jot/internal/database/db.go) `InitDefaultSettings` —— `ai_agent_max_iterations` 种子值 `"20"` → `"100"`；新增 `{Key: "ai_sub_agent_max_iterations", Value: "50"}`（加中文注释说明用途，参照 `ai_context_token_budget` 写法）
  - [x] SubTask 1.2: [types.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/types.go) `SettingsConfig` 新增字段 `AISubAgentMaxIterations int \`json:"ai_sub_agent_max_iterations"\``
  - [x] SubTask 1.3: `GetAllSettings()` 读取映射 —— 主 Agent 默认值 `20` → `100`；新增 `AISubAgentMaxIterations: parseIntSetting(s.Get("ai_sub_agent_max_iterations"), 50)`
  - [x] SubTask 1.4: `SaveAllSettings()` clamp 区 —— 主 Agent `< 1 → 100`（上限 500 不变）；新增子 Agent `< 1 → 50`、`> 200 → 200`
  - [x] SubTask 1.5: `SaveAllSettings()` sets map 新增 `"ai_sub_agent_max_iterations": strconv.Itoa(cfg.AISubAgentMaxIterations)`

- [x] Task 2: 后端装配层读取配置并作用于 os_agent 子 Agent
  - [x] SubTask 2.1: [agent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go) `DefaultMaxIterations` 常量 `20` → `100`，注释同步（语义仍为「未配置时的默认值」）
  - [x] SubTask 2.2: [subagent_os.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/subagent_os.go) 常量 `osSubAgentMaxIterations = 20` → `osSubAgentDefaultMaxIterations = 50`，注释改为「未配置时的默认值」
  - [x] SubTask 2.3: subagent_os.go 新增本地读取函数 `subAgentMaxIterations(setting *services.SettingService) int`：`setting == nil` 或解析失败或 `< 1` 回退 50，`> 200` 取 200（语义对齐 tools 包 `getIntSetting`）
  - [x] SubTask 2.4: `buildOSSubAgent` 增加第 4 参 `setting *services.SettingService`；内部**拷贝** `osAgentConfig` 后再覆盖 `maxIterations`（禁止直接改包级 var，避免并发 Run 数据竞争）
  - [x] SubTask 2.5: [registry.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/registry.go) 装配调用点传 `p.deps.Setting`
  - [x] SubTask 2.6: [subagent_test.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/subagent_test.go) 8 处 `buildOSSubAgent(...)` 调用补第 4 参 `nil`（保持既有断言不变）

- [x] Task 3: 前端设置页新增/修正两个数字输入项
  - [x] SubTask 3.1: [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html) 「Agent 运行上限」输入框 `value="20"` → `value="100"`（`min=1 max=500` 保持）
  - [x] SubTask 3.2: index.html 其下方新增「子 Agent 运行上限」`ai-setting-item`（复用 `.ai-setting-item`/`.ai-setting-label`/`.font-setting-desc`/`.ai-setting-control`/`.settings-input`，`id="aiSubAgentMaxIterations"`、`min="1" max="200" value="50"`、`style="width:80px;"`）
  - [x] SubTask 3.3: [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) `loadSettings()` —— 主 Agent 回退值 `|| 20` → `|| 100`；新增子 Agent 回显 `|| 50`
  - [x] SubTask 3.4: main.js `saveSettings()` —— 主 Agent 回退值 `|| 20` → `|| 100`；新增 `ai_sub_agent_max_iterations: parseInt(...) || 50`
  - [x] SubTask 3.5: main.js change 监听 —— 主 Agent 校验 `< 1 → 重置 100`、`> 500 → 重置 500`（含提示文案同步）；新增子 Agent 监听（`< 1 → 50`、`> 200 → 200`，提示文案与相邻项同风格）

- [x] Task 4: 文档同步
  - [x] SubTask 4.1: [SUBAGENTS.md](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/SUBAGENTS.md) —— `osSubAgentMaxIterations` 引用改为 `osSubAgentDefaultMaxIterations`，并说明「迭代上限由设置项 `ai_sub_agent_max_iterations` 提供（默认 50），装配时读取」（另同步了 3 处构造器签名示例：§2 字段要点、§7 注册示例、§5.3 自查清单）
  - [x] SubTask 4.2: [AGENTS.md](file:///d:/峡谷/Dev/本地项目/jot/AGENTS.md) 长期记忆 20 中「`MaxIterations=20`」改为「`MaxIterations` 由设置项 `ai_sub_agent_max_iterations` 提供（默认 50）」
  - [x] SubTask 4.3: AGENTS.md 临时记忆按维护规范三步执行（删最旧的第 1 条、其余顺移重编号、末尾追加新条目为「临时记忆 5」），记录本次两项改动与默认值变更（文件引用须用项目相对路径）

- [x] Task 5: 验证
  - [x] SubTask 5.1: `go build ./...` + `go vet ./...` 通过
  - [x] SubTask 5.2: `go test ./internal/agent/... ./internal/services/...` 全绿
  - [x] SubTask 5.3: `golangci-lint run ./...` 0 issues
  - [x] SubTask 5.4: `npm run build` 通过（仅既有 chunk 体积警告）
  - [x] SubTask 5.5: 修复验证发现的两处陈旧注释 —— `agent.go` `Run` 内「默认 20」改为「未配置时回退 DefaultMaxIterations=100」；`playground/agent-demo/main.go` `MaxIterations: 20` → `100`（与注释声明的「与主项目保持一致」对齐）

# Task Dependencies

- Task 2 依赖 Task 1（设置项字段/clamp 就位后装配层才有值可读）
- Task 3 依赖 Task 1（`SettingsConfig` 字段是前端读写的前提）
- Task 4 依赖 Task 1-3（文档需反映最终实现）
- Task 5 依赖 Task 1-4
- Task 1 内部 SubTask 1.1-1.5 可并行；Task 3 的 SubTask 3.1/3.2（HTML）与 3.3-3.5（JS）可并行
