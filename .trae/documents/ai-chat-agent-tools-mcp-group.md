# AI 助手工具浮层：MCP 工具分组显示

## 一、概述

在 AI 助手输入框「Agent 工具」浮层（`renderChatAgentToolsList`）中，将 MCP 服务器扩展工具从内置普通工具中拆出，形成独立的「MCP 扩展」分组；顶部普通内置组加「内置」组标签。最终分组顺序：**内置 → MCP 扩展 → 仅 Plan 模式 → 常驻**。数据层在后端 `ToolMeta` 增加 `MCPServer` 字段用于区分来源。

## 二、现状分析（Phase 1 探查事实）

1. **数据契约无来源标识**：`GetAgentTools()`（[app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2707-L2744)）先返回内置工具（`tools.BuiltinTools()`），再追加 MCP 工具；`agent.ToolMeta`（[types.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/types.go#L33-L39)）仅有 `Name/Label/Enabled/PlanOnly/AlwaysOn`，前端无法可靠区分 MCP 工具（MCP 工具名 `mcp_{server}_{tool}` 前缀判断不可靠）。
2. **MCP 工具元信息已有服务器名**：`SessionToolMeta`（[pool.go](file:///d:/峡谷/Dev/本地项目/jot/internal/mcpserver/pool.go#L40-L44)）含 `ServerName`，`ListToolMetas()` 已返回，`GetAgentTools` 组装时丢弃了该信息。
3. **聊天浮层现有分组**（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10072-L10082)）：三组 —— `normal`（无标签，内置普通工具与 MCP 工具混在一起）、`plan`（仅 Plan 模式）、`always`（常驻）。`selectable = groups[0].tools` 只含 normal 组；`normalInputs` 数组按索引与 `selectable` 对齐（[L10148-L10150](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10148-L10150)、[L10159](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10159)）。
4. **组标签样式已存在**：`.ai-chat-agent-tools-group`（[ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1800-L1806)），组渲染逻辑通用（`group.label` 非空即渲染标签，空组跳过），无需新增 CSS。
5. **元信息刷新时机**：`refreshAgentToolsMeta()`（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L11478-L11489)）在 MCP 预热/变更后调用，目前只重渲染设置页面板，不重渲染聊天浮层。
6. **Wails 绑定需再生成**：修改 `app.go`/`types.go` 后需执行 `wails generate module`（AGENTS.md 约定），影响 `frontend/wailsjs/go/models.ts` 的 `ToolMeta` 类与 `App.d.ts`。
7. **设置页「Agent 工具」面板**（`renderAgentToolsMgrList`，[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10380-L10444)）为平铺列表 —— 用户明确本次只改输入框浮层，设置页保持不动。

## 三、变更方案

### 1. `internal/agent/types.go` —— ToolMeta 增加来源字段

```go
type ToolMeta struct {
    Name      string // 英文工具名
    Label     string // 一行中文说明
    Enabled   bool   // 当前是否启用
    PlanOnly  bool   // 仅 Plan 模式可用
    AlwaysOn  bool   // 常驻/不可禁用
    MCPServer string // 所属 MCP 服务器名；内置工具为空字符串（前端据此分区展示）
}
```

### 2. `app.go` —— GetAgentTools 填充 MCPServer

MCP 工具组装处（[app.go L2736-L2741](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2736-L2741)）增加一行：

```go
result = append(result, agent.ToolMeta{
    Name:      mt.FullName,
    Label:     label,
    Enabled:   !disabledSet[mt.FullName],
    MCPServer: mt.ServerName, // 新增
})
```

内置工具循环保持零值 `""`，不改。

### 3. `frontend/src/main.js` —— renderChatAgentToolsList 四组重构

a. **分组结构**（替换 [L10072-L10083](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10072-L10083)）：

```js
const groups = [
    { key: 'normal', label: '内置', tools: [] },
    { key: 'mcp', label: 'MCP 扩展', tools: [] },
    { key: 'plan', label: '仅 Plan 模式', tools: [] },
    { key: 'always', label: '常驻', tools: [] },
];
agentToolsMeta.forEach((tool) => {
    if (tool.PlanOnly) groups[2].tools.push(tool);
    else if (tool.AlwaysOn) groups[3].tools.push(tool);
    else if (tool.MCPServer) groups[1].tools.push(tool);
    else groups[0].tools.push(tool);
});
// 后端 pool 以 map 遍历返回、顺序不稳定，组内按工具名排序保证展示稳定
groups[1].tools.sort((a, b) => a.Name.localeCompare(b.Name));
const selectable = [...groups[0].tools, ...groups[1].tools]; // 全选范围 = 内置普通 + MCP
```

b. **checkbox 行为分支**：

* [L10178](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10178) `if (group.key !== 'normal') cb.disabled = true;` → 改为 `if (group.key === 'plan' || group.key === 'always') cb.disabled = true;`（MCP 工具可勾选）。

* [L10208](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10208) `if (group.key === 'normal')` → `if (group.key === 'normal' || group.key === 'mcp')`（勾选变更走 `applyTool` + `saveSettings()` + `refreshSummary()`；else 抖动提示分支仅剩 plan/always）。

* MCP 行不加角标（服务器信息由组标签与描述承载）。

c. **全选/计数同步改为按名索引**：`normalInputs` 数组（[L10159](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10159)）改为 `const inputByName = new Map();`，交互分支中 `inputByName.set(tool.Name, cb)`；`refreshSummary` 内索引对齐循环（[L10148-L10150](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10148-L10150)）改为：

```js
selectable.forEach((t) => {
    const input = inputByName.get(t.Name);
    if (input) input.checked = isEnabled(t);
});
```

### 4. `frontend/src/main.js` —— refreshAgentToolsMeta 联动刷新聊天浮层

在 [L11486-L11488](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L11486-L11488) 设置页面板重渲染之后追加：

```js
if (chatAgentToolsExpanded) {
    renderChatAgentToolsList();
}
```

MCP 预热完成时若浮层正开着，新工具即时出现。

### 5. 再生成 Wails 绑定

项目根目录执行 `wails generate module`，刷新 `frontend/wailsjs/go/models.ts`（ToolMeta 增加 `MCPServer`）与 `App.d.ts`。

### 不改的部分

* 设置页「Agent 工具」面板保持平铺（用户本次只要求输入框浮层）。

* CSS 无需改动：MCP 组标签复用 `.ai-chat-agent-tools-group`；「内置」标签同样复用。

* `updateAgentToolsButtonText` / `toggleSelectAllTools` / `updateSelectAllCheckboxState` 均按 `agentToolsMeta` 过滤统计，天然涵盖 MCP 工具，无需改动。

## 四、假设与决策记录

* **单一 MCP 组**（用户已选）：所有 MCP 工具合入一个「MCP 扩展」组，不按服务器细分。

* **顶部加「内置」标签**（用户已选）：原先无标签的普通组显式标注「内置」，与下方组标签风格一致。

* **组顺序：内置 → MCP 扩展 → 仅 Plan 模式 → 常驻**：可勾选的组在上、锁定展示的组（仅 Plan/常驻）延续现状垫底；「内置」满足"最上面是内置工具"的要求。

* **数据层加** **`MCPServer`** **字段**而非前端解析 `mcp_` 前缀：可靠、语义清晰，未来按服务器细分时无需再改契约。

* **MCP 组内按工具名排序**：后端 pool map 遍历顺序不稳定，前端排序保证每次展开展示一致。

## 五、验证步骤

1. **静态检查**：项目根 `go vet ./...` + `go build .`；`frontend` 目录 `.\node_modules\.bin\eslint.cmd src/main.js`，0 error。
2. **绑定再生成**：`wails generate module` 成功，`models.ts` 中 ToolMeta 出现 `MCPServer` 字段。
3. **wails dev 实测**（Agent 模式展开工具浮层）：

   * 无 MCP 服务器时：仅「内置」「仅 Plan 模式」「常驻」三组，顶部出现「内置」标签。

   * 配置并预热一台 MCP 服务器后：「MCP 扩展」组出现在「内置」之下，组内为该服务器工具，勾选/取消即时生效并计入「已启用 N/M」；关闭浮层弹「工具配置已保存」汇总。

   * 全选/取消全选同时作用于内置普通工具与 MCP 工具。

   * 浮层展开时在设置页完成 MCP 预热 → 浮层自动出现 MCP 扩展组。

   * 「仅 Plan 模式」「常驻」行仍为禁用勾选 + 点击抖动提示。

   * Chat 模式下浮层按钮隐藏、流式期间按钮锁定等既有行为不变。
4. **回归**：设置页「Agent 工具」面板仍为平铺且开关正常；`PlanOnly`/`AlwaysOn` 豁免逻辑不变。

