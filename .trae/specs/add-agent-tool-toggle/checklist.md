# Checklist

- [x] 后端 `tools.BuiltinTools()` 返回 10 个工具元数据（英文名 + 中文说明），`go build ./...` 通过
- [x] `agent.Request` 含 `DisabledTools []string` 字段，`Run` 将其转为 map 传入 `buildTools`
- [x] `buildTools` 按禁用集合过滤：被禁工具不注册进 toolList / toolByName；禁用列表为空时 10 个工具全部注册（与现状行为一致）
- [x] `SettingsConfig` 含 `ai_agent_tools_disabled` 字段，`GetAllSettings` / `SaveAllSettings` 读写映射正确，键不存在时默认空
- [x] `CallAIAgentStream` 读取 `ai_agent_tools_disabled` 并 JSON 解析为 `[]string` 传入 `Request.DisabledTools`；解析失败按空列表处理（全部注册）
- [x] `app.go` 提供 `GetAgentTools()` 绑定：返回 10 个工具元数据，且启用状态与禁用列表一致
- [x] 「对话与搜索」面板尾部存在「Agent 工具」设置项：标签 + 描述 + 按钮（显示「已启用 N/10」）+ 下拉浮层容器
- [x] 设置页加载时调用 `GetAgentTools()` 渲染浮层条目（checkbox + 工具名 + 中文说明），勾选状态与保存值一致，按钮文案显示启用数
- [x] 按钮点击展开/收起浮层；点击外部或 ESC 关闭浮层
- [x] 勾选条目即时保存（写入 `ai_agent_tools_disabled` 并调 `saveSettings()`），按钮文案同步更新；重新打开设置页后状态保持
- [x] 首次使用（设置为空）时按钮显示「已启用 10/10」，浮层条目全部勾选
- [x] 浮层样式在 14 主题下正常（checkbox、hover 高亮、悬浮定位不遮挡其他面板内容）
- [x] 前后端验证通过：`go build ./...`、`npm run build`（如需 `wails build` 联调 Agent 模式）
- [x] Agent 模式实际验证：禁用某工具后该工具不被调用（注册级过滤，代码验证）；全部启用时行为与现状一致
