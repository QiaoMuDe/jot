# 设置页 Agent 工具列表分组显示改造

## Summary

将「设置页 → AI 设置 → Agent 工具」管理面板的平铺工具列表，改为分组显示，顺序与 AI 助手浮层一致：**内置 → MCP 扩展 → 仅 Plan 模式 → 常驻**。保持当前**单行**行样式（checkbox + 名称 + 描述同行）不变，并让「内置」「MCP 扩展」两个可勾选组支持点击组标签**全选/取消全选**组内工具。

## Current State Analysis

* 设置页渲染入口：[renderAgentToolsMgrList](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10395-L10459)

  * 容器 `.agent-tools-mgr-list`，头部含全局全选 checkbox（`agentToolsSelectAllCheckbox`，绑 `toggleSelectAllTools`）+ 标题 + 关闭按钮

  * 工具行代码：`agentToolsMeta.forEach((tool, index) => { const row = createAgentToolRow(tool); ... })` **平铺直接 append，无分组**

* 单行行构造：[createAgentToolRow](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L10464-L10557)

  * 返回 `label.ai-agent-tools-item`：`checkbox + span.ai-agent-tools-name + span.ai-agent-tools-desc`，单行（flex-start 对齐）

  * Plan/AlwaysOn：checkbox disabled、行置灰、追加 `plan-only-hint`/`always-on-hint` 提示、点击抖动+通知

  * checkbox change：改 `agentToolsDisabled`/`agentToolsChanges` → `updateAgentToolsButtonText()` + `updateSelectAllCheckboxState()` + `saveSettings()`

* 设置页 CSS：[settings-panel.css L1615-L1745](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L1615-L1745)

  * `.agent-tools-mgr-list`（面板：padding 12、border、max-height 360、overflow-y auto）

  * `.ai-agent-tools-item` 单行样式；`.is-plan-only`/`.is-always-on` 置灰；`.plan-only-hint`/`.always-on-hint` 提示

* 可复用的分组经验（AI 浮层，勿照搬 DOM，只复用逻辑/配色）：

  * 分组判定：PlanOnly→`plan`；AlwaysOn→`always`；MCPServer→`mcp`；否则→`normal`。顺序：plan/always 优先于 mcp（else-if 链）

  * 组标签盲切：`tools.every(isEnabled)` 判 `allEnabled` → `tools.forEach(applyTool(t, !allEnabled))` → save + 联动；`.is-selectable` 标签加 `role=button`/`tabIndex` / click+Enter+空格

  * 配色：`.is-selectable:hover` 用 `--hover-bg`；`:active` 用 `color-mix(var(--hover-bg) 80%, var(--accent))`；`--accent` 为 --text-primary 强调色

## Proposed Changes

### 文件 1：`frontend/src/main.js` — `renderAgentToolsMgrList`（主改动）

将 `agentToolsMeta.forEach` 平铺改为四分组渲染：

1. 在 `agentToolsMgrContainer.innerHTML = ''` 头部拼接后、行循环前，构造分组数组（判定逻辑与 AI 浮层保持一致）：

   ```js
   const groups = [
       { key: 'normal', label: '内置', tools: [], rows: [] }, // rows: [{tool, checkbox}] 供盲切后同步勾选态
       { key: 'mcp', label: 'MCP 扩展', tools: [], rows: [] },
       { key: 'plan', label: '仅 Plan 模式', tools: [], rows: [] },
       { key: 'always', label: '常驻', tools: [], rows: [] },
   ];
   agentToolsMeta.forEach((tool) => {
       if (tool.PlanOnly) groups[2].tools.push(tool);
       else if (tool.AlwaysOn) groups[3].tools.push(tool);
       else if (tool.MCPServer) groups[1].tools.push(tool);
       else groups[0].tools.push(tool);
   });
   ```

2. `groups.forEach((group, gi) => { ... })` 渲染：

   * 空组跳过（`if (group.tools.length === 0) return;`）

   * 结算根是 group.key 可勾选 的组标签 div（class `.agent-tools-mgr-group` + 可勾选时 `is-selectable`，并复用 AI 浮层的 role/tabIndex/click/keydown 盲切），append 到容器

   * 组内 `tools.forEach((tool, ti))`：`createAgentToolRow(tool)`，push 该行的 `{tool, checkbox}` 引用到 `group.rows`，`animationDelay = (gi * 7 + ti) * 30ms`（略微拉开不同组间的 stagger，具体系数可调）

   * 可勾选组（normal/mcp）盲切 handler：

     ```js
     const toggleGroup = () => {
         const allEnabled = group.tools.length > 0 && group.tools.every(t => agentToolsDisabled.indexOf(t.Name) === -1);
         group.tools.forEach(t => applyToolForGroup(t, !allEnabled));
         // 同步该组每行 checkbox 勾选态
         group.rows.forEach(({ checkbox }) => { checkbox.checked = agentToolsDisabled.indexOf(/* 该行 tool.Name */ ) === -1; });
         updateAgentToolsButtonText();
         updateSelectAllCheckboxState();
         saveSettings();
     };
     ```

3. 复用现有 `createAgentToolRow`，但其内部 checkbox change 已含 `updateAgentToolsButtonText/updateSelectAllCheckboxState/saveSettings`，无需改动。唯一需注意：盲切 handler 内的批量写操作需与 `createAgentToolRow` 的 change 逻辑同构（改 `agentToolsDisabled`/`agentToolsChanges` 去重+互斥）——**不要直接把** **`applyTool`** **从 AI 浮层拷贝**，因设置页 change 逻辑是内联到 row 构造里的；可提取一个模块级小函数 `applyAgentTool(tool, enabled)` 复用于盲切，将 createAgentToolRow 两个分支内的状态改写收敛进去（降低重复）。

   > 决策：新增模块级 `applyAgentTool(tool, enabled)`（等同 AI 浮层 `applyTool` 逻辑），`createAgentToolRow` 的 change 与组盲切统一调用它，保证单入口。

4. 头部全局全选 checkbox（`agentToolsSelectAllCheckbox`/`toggleSelectAllTools`/`updateSelectAllCheckboxState`）**保留不动**，分组后全局全选仍基于 `agentToolsMeta` 遍历，与分组无冲突。

### 文件 2：`frontend/src/css/components/settings-panel.css` — 新增组标签样式

在 `.ai-agent-tools-item` 相关规则前新增：

```css
/* 分组标签：hairline 底边线作组边界 */
.agent-tools-mgr-group {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  border-bottom: 1px solid color-mix(in srgb, var(--border) 35%, transparent);
  margin-top: 4px;               /* 组标签与上一工具行间距 */
}
.agent-tools-mgr-group:first-child {
  margin-top: 0;
}
/* 可勾选组标签盲切（内置/MCP）反馈 */
.agent-tools-mgr-group.is-selectable {
  cursor: pointer;
  user-select: none;
}
.agent-tools-mgr-group.is-selectable:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}
.agent-tools-mgr-group.is-selectable:focus-visible {
  outline: 1px solid color-mix(in srgb, var(--accent) 60%, transparent);
  outline-offset: -1px;
}
.agent-tools-mgr-group.is-selectable:active {
  background: color-mix(in srgb, var(--hover-bg) 80%, var(--accent));
}
```

* 配色与 AI 浮层一致（`--hover-bg`/`--accent`/35% hairline），字号/间距针对设置页更宽面板微调

* 面板 max-height 仍 360px、overflow-y auto，组标签正常随内容滚动，不做 sticky

* 单行行样式 `.ai-agent-tools-item`（含 name/desc/hint 置灰）**不改**

## Assumptions & Decisions

* **保持单行**：用户明确要求设置页不改副标题式，沿用 `.ai-agent-tools-item` 现状。

* **分组顺序**：内置 → MCP 扩展 → 仅 Plan 模式 → 常驻（对应 `normal/mcp/plan/always`）。

* **盲切范围**：仅 `normal`（内置）与 `mcp`（MCP 扩展）组标签可点击全选/取消；`plan`/`always` 组为纯展示标签，不参与点击。

* **判定优先级**：PlanOnly/AlwaysOn 优先于 MCPServer（与 AI 浮层一致），即使工具同时有 MCP Server 标记也归入锁定组。

* **状态一致性**：新增模块级 `applyAgentTool(tool, enabled)` 收敛单行 change 与组盲切的写入逻辑，避免两份不同步的复制代码。

* **全局全选冲突**：设置页头部全局全选 checkbox 保留，与组盲切互补（组盲切针对单组，全局针对全部），无功能冲突。

## Verification

1. `go` 不涉及后端改动，无需 Go 重新编译；前端需在 `wails dev` 下热重载验证。
2. 打开 设置页 → AI 设置 → Agent 工具 → 点「管理」展开面板，确认列出 4 个组标签且顺序为 **内置 → MCP 扩展 → 仅 Plan 模式 → 常驻**。
3. 确认行仍是单行样式（名称+描述同行），Plan/常驻组行置灰并带提示、点击抖动。
4. 点击「内置」组标签：组内工具全部勾选/取消，底部全局 checkbox 与工具栏「已启用 N/M」文案同步更新；再次点击反向。
5. 点击「MCP 扩展」组标签同理（需先预热至少一个 MCP 服务器使该组非空；无 MCP 时该组不显示）。
6. Tab 聚焦到某个可勾选组标签，按 Enter/空格也应触发盲切。
7. 检查空组：无 MCP 服务器时「MCP 扩展」组整体不渲染，无孤立空标签。

