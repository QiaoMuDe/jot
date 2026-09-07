# MCP 服务器管理列表：折叠/展开 + 入场/新增/删除动画

## Summary

给 MCP 服务器配置区新增一个「管理」按钮，用于展开/折叠服务器列表；并复用预设配置（`#presetMgrBtn` / `.preset-mgr-list`）那一套交互与动画：列表默认折叠，点「管理」展开（行内滑入 + 条目逐行入场），再点或用关闭按钮折叠移除；新增服务器时在展开列表里播「插入动画」，删除时播「滑出动画」。

## Current State Analysis

**预设配置管理（参考实现）：**
- HTML：[index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L436-L437) 的 `#presetMgrBtn`「管理」按钮
- JS：[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3316) `renderPresetMgrList`（动态创建 `.preset-mgr-list` 容器 + header 标题/关闭 + 逐行 `preset-row-enter` + stagger）；[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3427) `closePresetMgrList`（Web Animations API 折叠后移除）；新增动画 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3257-L3282)（`preset-row-insert`）；删除动画 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3290-L3313)（`preset-delete-out`）
- CSS：[settings-panel.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L1151-L1189)（`mgrSlideDown`/`mgrSlideUp`）、[settings-panel.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L1298-L1358)（`preset-row-enter` / `preset-row-insert` / `preset-delete-out` 三个 keyframes）

**MCP 服务器区（现状）：**
- HTML：[index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L647-L664)：`.mcp-server-head`（desc + actions：分享/导入/添加）→ `.mcp-server-list#mcpServerList`（常驻显示）→ `#mcpServerEmpty` 空态
- JS：[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10496) `loadMCPServers` → [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10508) `renderMCPServerList`（全量 innerHTML 渲染）；[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10526) `buildMCPServerItem`（条目含启用/分享/编辑/测试/删除）；表单保存 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10960-L10965)（保存后 `loadMCPServers` 全量刷新）；删除 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10675-L10687)（确认后 API + 全量刷新）
- CSS：[settings-panel.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L1773-L1806) `.mcp-server-list`（`max-height:320px; overflow-y:auto; input-bg`）+ `.mcp-server-item`（无背景 + 底部 `border-bottom` 分隔线，已与预设条目对齐）

**差异点：** 预设列表默认不渲染、点「管理」显隐并带动画；MCP 列表当前常驻渲染、无展开折叠、无动画。

## Proposed Changes

采用「默认折叠 + 管理按钮展开」模式，复用预设动画效果。展开区沿用现有 `.mcp-server-list` 容器（其条目样式已与预设对齐），不新增独立动态容器，改动最小。

### 1) HTML — [index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html)
- 在 `.mcp-server-head-actions` 内、`#mcpServerShareAllBtn` 之前插入「管理」按钮：
  ```html
  <button id="mcpServerMgrBtn" class="btn btn-sm btn-save" title="展开/折叠服务器列表">管理</button>
  ```
- 列表容器 `#mcpServerList` 默认折叠：增加 `hidden` 属性（`<div class="mcp-server-list" id="mcpServerList" hidden>`），由「管理」按钮展开。

### 2) CSS — [settings-panel.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css)
- 新增 `.mcp-server-list.open` 展开动画（参照对应预设 `.preset-mgr-list.open`），keyframe 用本容器实际高度 `max-height:320px`：
  ```css
  .mcp-server-list.open { animation: mcpMgrSlideDown 250ms ease-out both; }
  @keyframes mcpMgrSlideDown {
    from { opacity: 0; transform: scaleY(0.95); max-height: 0; }
    to   { opacity: 1; transform: scaleY(1); max-height: 320px; }
  }
  ```
- 移除 `#mcpServerList` 的 `gap:0` 覆盖规则不受影响（`gap` 依旧与条目分隔线配套）。
- 折叠动画由 JS 的 Web Animations API 实现（与 `closePresetMgrList` 一致），CSS 无需额外 keyframe。

### 3) JS — [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js)
- 新增状态 `let mcpMgrExpanded = false;`（放在 `mcpServers` 声明附近，[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10195)）。
- 新增 `toggleMCPServerMgr()`：
  - **展开**（`mcpMgrExpanded` 为 false）：移除 `#mcpServerList` 的 `hidden` → `renderMCPServerList()` → `requestAnimationFrame` 加 `.open` 类；将按钮文案改为「收起」；`mcpMgrExpanded = true`。
  - **折叠**：对 `#mcpServerList` 用 `.animate([{scaleY:1,maxHeight:'320px'},…], {duration:280, ease-in-out, fill:'both'})`，`onfinish` 时加回 `hidden`、移除 `open` 类、按钮文案回「管理」；`mcpMgrExpanded = false`。
- 改造 `renderMCPServerList(animateMode)`：
  - 空列表 → 显示 `#mcpServerEmpty`，返回。
  - 渲染每条 `buildMCPServerItem(srv)`；默认每条加 `.preset-row-enter` 并设 `animationDelay = index * 50ms`（stagger，与预设一致）；`animateMode === 'insert'` 时对目标条目走 `.preset-row-insert` 的初始态 + 下一帧加类逻辑（复用 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3257-L3282) 的实现）。
- `loadMCPServers()`：
  - 仅更新 `mcpServers` 缓存；若 `mcpMgrExpanded` 为 true 才调用 `renderMCPServerList()`（折叠态无需渲染）。
- **新增动画**：表单保存成功回调（[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10960)）create 分支处，记录新增身份（`name`），在 `loadMCPServers()` 后调用 `renderMCPServerList('insert')`，渲染时对匹配 `name` 的行应用 insert 动画。折叠态下仅更新缓存，下次展开自然包含新行。
- **删除动画**：`deleteMCPServer`（[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L10675)）改造为：确认 → 行加 `.preset-delete-out` 并等待 `animationend` → 调 `App.DeleteMCPServer` → 从 `mcpServers` 过滤并移除该行 DOM → 若列表空显示空态 → `warmupMCPServers()`（不复用全量 `loadMCPServers`，避免闪烁）。
- 在 `initMCPServerSettings` 中为 `#mcpServerMgrBtn` 绑定 `click → toggleMCPServerMgr`（已存在该初始化函数，添加按钮绑定即可）。

### 需改动文件清单
| 文件 | 改动 |
|------|------|
| `frontend/index.html` | head 加「管理」按钮；`#mcpServerList` 默认 `hidden` |
| `frontend/src/css/components/settings-panel.css` | 新增 `.mcp-server-list.open` 展开动画 keyframe |
| `frontend/src/main.js` | `mcpMgrExpanded` 状态、`toggleMCPServerMgr`、改造 `renderMCPServerList`/`loadMCPServers`、保存回调 insert 动画、`deleteMCPServer` 删除动画、按钮绑定 |

## Assumptions & Decisions

- **默认折叠**：按用户选择，MCP 列表默认隐藏，点「管理」展开。
- **展开区沿用现有 `.mcp-server-list`**，不新增独立动态容器，避免冗余与改动风险；其条目样式已与预设对齐。
- **不额外注入"标题+关闭" header**：展开区沿用已有 `.mcp-server-head`（含 desc 与「管理」按钮），「管理」按钮作为展开/折叠切换，折叠也可重新点「管理」。若需严格对齐预设的标题+关闭样式，可作为后续微调。
- **新增后折叠态行为**：保存成功但当前折叠时不强制展开，仅更新缓存；下次展开自然包含。展开态则以插入动画即时呈现。
- 展开动画高度用 MCP 实际 `max-height:320px`，避免与预设 500px 动画造成跳动。
- 其余逻辑（预热池、空态、启用开关）保持不变。

## Verification

1. `cd frontend && npm run build` 构建前端资源。
2. `wails build`（需在可访问用户数据目录的环境执行）重新编译 `build/bin/jot.exe` 并运行。
3. 手动验证路径：
   - 打开设置 → AI 设置 → MCP 服务器：列表默认折叠，仅见「管理」按钮。
   - 点「管理」：列表滑入展开，条目逐行入场；按钮变为「收起」。
   - 再点「管理」（或点收起）：列表折叠移除，按钮回「管理」。
   - 展开状态下点「添加」保存：新行以插入动画出现在列表。
   - 删除某行：该行以滑出动画移除，列表为空时显示空态。
   - 启用开关、分享、导入、编辑仍工作正常。