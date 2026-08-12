# Tasks

## 阶段一：后端（数据层 + 方法）

- [x] Task 1: 新增工具元数据清单（`internal/agent/tools/meta.go`）
  - 定义 `ToolMeta{Name, Label string}` 与导出 `BuiltinTools() []ToolMeta`，覆盖全部 10 个内置工具（refine_search_query / web_search / recall_notes / get_current_time / manage_todo / manage_notebook / manage_tag / manage_note / get_stats / ask_user），`Label` 为一行中文说明
  - 在 [tools/doc.go](internal/agent/tools/doc.go) 注释中补充该清单为工具展示文案的权威来源
  - 验证：`go build ./...`、`go vet ./internal/agent/...` 通过

- [x] Task 2: `agent.Request` 新增 `DisabledTools []string` 字段（[types.go](internal/agent/types.go)）
  - 注释说明：禁用工具名集合，装配时按此过滤（注册级，模型不可见）
  - 验证：`go build ./...` 通过

- [x] Task 3: `buildTools` 支持禁用过滤（[registry.go](internal/agent/registry.go)）
  - 签名改为接收禁用集合（如 `buildTools(p BuildParams, disabled map[string]bool)`），构造后过滤；`BuildParams` 或参数按需调整
  - [agent.go](internal/agent/agent.go) 的 `Run` 将 `req.DisabledTools` 转为 map 传入；被禁工具不注册进 toolList，不参与 toolByName 索引
  - 验证：`go build ./...`、`go vet ./internal/agent/...` 通过

- [x] Task 4: `SettingsConfig` 新增 `AIAgentToolsDisabled string` 字段（[internal/services/types.go](internal/services/types.go)）
  - json tag `ai_agent_tools_disabled`；`GetAllSettings` 映射 `s.Get("ai_agent_tools_disabled")`，`SaveAllSettings` 映射 `s.Set("ai_agent_tools_disabled", cfg.AIAgentToolsDisabled)`
  - 验证：`go build ./...` 通过

- [x] Task 5: `app.go` 新增 `GetAgentTools()` 绑定 + `CallAIAgentStream` 读取设置
  - `GetAgentTools() []agent.ToolMeta`：基于 `tools.BuiltinTools()` 返回清单并标注启用状态（不在 `ai_agent_tools_disabled` 禁用列表即启用）
  - `CallAIAgentStream` 读取 `ai_agent_tools_disabled`，JSON 解析为 `[]string` 传入 `Request.DisabledTools`（解析失败按空列表处理，记 Warn 日志）
  - 验证：`go build ./...`、`wails build` 编译通过（或 `go vet`）

## 阶段二：前端（设置项 + 下拉多选）

- [x] Task 6: 「对话与搜索」面板新增「Agent 工具」设置项（[frontend/index.html](frontend/index.html)）
  - 面板尾部追加 `ai-setting-item`：标签「Agent 工具」+ 描述「控制 Agent 模式可调用的内置工具」+ 右侧按钮（`#aiAgentToolsBtn`，显示「已启用 N/10」，含下拉箭头）
  - 追加浮层容器（`#aiAgentToolsPopover`），内含多选条目列表容器（`#aiAgentToolsList`）
  - 验证：构建前端资源 `npm run build` 无报错

- [x] Task 7: 浮层渲染与交互（[frontend/src/main.js](frontend/src/main.js)）
  - 设置页加载时调用 `GetAgentTools()` 渲染条目（checkbox + 英文名 + 中文说明），按启用状态初始化勾选，更新按钮文案
  - 按钮点击切换浮层显隐；点击外部 / ESC 关闭；勾选条目即时更新 `ai_agent_tools_disabled` 并调用 `saveSettings()`，同步按钮文案
  - `saveSettings()` 的 cfg 新增 `ai_agent_tools_disabled`（序列化当前未勾选工具名数组为 JSON 字符串）
  - 验证：`npm run build` 通过；手动验证勾选/保存/回填

- [x] Task 8: 浮层样式（[frontend/src/css/components/settings-panel.css](frontend/src/css/components/settings-panel.css)）
  - 浮层绝对定位悬浮于设置项下方（面板内 `position:relative` 上下文），z-index 高于面板内容；条目 hover 高亮、checkbox 主题适配（14 主题）
  - 验证：`npm run build` 通过；重启应用（`wails build`）检查 14 主题下样式正常

# Task Dependencies
- Task 2 依赖 Task 1（过滤需要工具名清单校验可选）
- Task 3 依赖 Task 2（Request 字段）
- Task 5 依赖 Task 1 / Task 4 / Task 3（清单、设置字段、过滤入口）
- Task 6 依赖 Task 5（前端需要 GetAgentTools 绑定）
- Task 7 / Task 8 依赖 Task 6
