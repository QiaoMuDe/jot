# 设置页分组重构计划

## Summary
将设置页侧边导航从当前的 **10 个扁平面板**收敛为 **5 个面板**，把强相关的设置项合并进同一个面板，面板内部用现有的 `ai-group-header` 子标题分节。**不引入大类↔子导航的深层级结构**，仅做相对现有的合并。所有设置项与字段一条不丢，后端零改动。

## Current State Analysis
- 设置页 HTML 位于 `frontend/index.html`，侧边导航 `.settings-nav` 下共 10 个 `.settings-nav-item`（L246–L305），对应 `.settings-panels` 下 10 个 `.settings-panel[data-panel=…]`（L310–L896）。
- 切换逻辑 `switchSettingsTab(panelName)`（`frontend/src/main.js` L11376）按 `data-panel` 在导航与面板容器间**一对一匹配**并切换 `active` class。
- 导航事件绑定 `initSettingsSidebarNav`（`main.js` L9706–L9717）：点击 nav item → 读取其 `data-panel` → 调 `switchSettingsTab`。**无其他代码硬编码引用旧的 `api-connection`/`dialog-search`/`mcp-server` 等面板名**（唯一硬编码是 L781 的默认面板 `switchSettingsTab('appearance')`）。
- 所有设置项控件通过**全局唯一 id** 绑定（`saveSettings` L11313–11367 按 `getElementById`/`els.xxx` 读取；`loadSettings` L11117 起按 id 同步）。移动 HTML 位置**不破坏**这些读写逻辑。
- 部分面板含动态生成的对话框/列表，合并时必须一起搬移：MCP 表单/导入对话框（`mcpServerFormDialog`、`mcpServerImportDialog`，index.html L684/L739）、MCP 服务器列表（`mcpServerList` L673）、预设管理与 Agent 工具列表（多处通过 JS 动态渲染）。
- 项目硬约束（见 project memory）：修改 CSS/HTML 后需 `npm run build` 重新构建前端资源，再 `wails build` 重新编译，更新 `build/bin/jot.exe`。

### 现状面板与归属
| 现有 data-panel | 面板标题 | 归并去向 |
|---|---|---|
| appearance | 外观 | 保留独立 |
| editor | 编辑器 | 保留独立 |
| api-connection | API 连接 | → AI 设置 |
| dialog-search | 对话与搜索 | → AI 设置 |
| mcp-server | MCP 服务器 | → AI 设置 |
| tag-management | 标签管理 | → 笔记与标签 |
| note-list | 笔记列表 | → 笔记与标签 |
| log-settings | 日志设置 | → 通用系统 |
| screen-lock | 锁屏密码 | → 通用系统 |
| trash-clean | 回收站清理 | → 通用系统 |

## Proposed Changes

### 文件 1：`frontend/index.html`（唯一改动文件）

**1. 侧边导航 `.settings-nav`（L246–L305）从 10 项改为 5 项：**
- 保留：`appearance`（外观）、`editor`（编辑器）
- `api-connection` / `dialog-search` / `mcp-server` → 合并为 1 项 `data-panel="ai"`，文字 **AI 设置**（图标复用原 API 连接的对勾/箭头图标）
- `tag-management` / `note-list` → 合并为 1 项 `data-panel="notes"`，文字 **笔记与标签**（图标复用标签图标）
- `log-settings` / `screen-lock` / `trash-clean` → 合并为 1 项 `data-panel="system"`，文字 **通用（系统）**（图标复用日志/工具图标）

**2. 重构 `.settings-panel` 各面板（L310–L896）：**
- **appearance / editor**：内容原样保留，不改。
- **新建 `data-panel="ai"` 面板**：把原 `api-connection`（L448–581）、`dialog-search`（L584–655）、`mcp-server`（L658–760）三块的内容合并进**一个** `.settings-section.settings-panel` 根节点，内部用 `ai-group-header` 分 5 节：
  1. `对话连接`（原对话预设/地址/Key/模型）
  2. `向量嵌入连接`（原向量预设/地址/Key/模型）
  3. `对话增强`（深度思考、上传导入限制、卡片召回数）
  4. `Agent 与 MCP`（Agent 工具、Agent 运行上限 + 完整 MCP 区块：`mcp-server-head`、`mcpServerList`、空态、`mcpServerFormDialog`、`mcpServerImportDialog`）
  5. `上下文`（摘要压缩预算、压缩触发比例）
- **新建 `data-panel="notes"` 面板**：合并原 `note-list`（L798–827）+ `tag-management`（L763–796），用 `ai-group-header` 分 2 节（`笔记列表` / `标签管理`）。
- **新建 `data-panel="system"` 面板**：合并原 `log-settings`（L829–855）、`screen-lock`（L857–880）、`trash-clean`（L882–896），用 `ai-group-header` 分 3 节（`日志` / `安全` / `数据维护`）。
- 删除旧的 9 个 `data-panel="api-connection|dialog-search|mcp-server|tag-management|note-list|log-settings|screen-lock|trash-clean"` 面板根节点，内容迁移到新面板中。
- 保留 `appearance` 作为默认 active 面板（`.settings-panel.active` 留在 appearance 上，L310）。

### 文件 2：`frontend/src/main.js`（预期不需要改动）
- `switchSettingsTab` 与 `initSettingsSidebarNav` 均按 `data-panel` 动态匹配，nav 与 panel 的 `data-panel` 同步更新后即自动适配；唯一硬编码 `switchSettingsTab('appearance')` 保留不变，故**无需修改 JS**。
- 若实施中发现某控件 id 在合并后消失，说明迁移遗漏，需核查 id 是否完整搬迁（见验证步骤）。

### 后端：不涉及
- `internal/services/types.go` 的 `SettingsConfig`、`Setting` 键值模型、`GetAllSettings`/`SaveAllSettings` 均不改变；`loadSettings`/`saveSettings` 依赖的控件 id 全部保留，字段无增删。

## Assumptions & Decisions
- **不做大类↔子导航**：仅把现有 10 面板合并为 5 面板，用户已确认此方向。
- **标签管理归"笔记与标签"**：作为同面板内独立子节 `标签管理`。
- **MCP 并入"Agent 与 MCP"子节**：MCP 是 Agent 外部工具，其表单/导入对话框一并搬移。
- **面板 data-panel 命名**：采用新名 `ai` / `notes` / `system`（旧名仅用于 nav 与 panel 匹配，无外部依赖，重命名安全）。
- **保留子节视觉结构**：复用现有 `ai-group-header`/`ai-group-icon` 样式，不新增 CSS。
- **active 默认面板**：仍为 `appearance`。

## Verification
1. `rg "data-panel=\"api-connection\"|data-panel=\"dialog-search\"|data-panel=\"mcp-server\"|data-panel=\"tag-management\"|data-panel=\"note-list\"|data-panel=\"log-settings\"|data-panel=\"screen-lock\"|data-panel=\"trash-clean\"" frontend/index.html` → 确认旧面板根节点已不存在。
2. 检查 `frontend/index.html` 中所有控件 id（`aiBaseURL`、`aiAPIKey`、`aiModelLabel`、`aiEmbedBaseURL`、`aiEmbedAPIKey`、`aiEmbedModelLabel`、`aiSettingSearchToggle`、`aiSettingCardRecallLimit`、`aiAgentToolsBtn`、`aiAgentMaxIterations`、`aiSummaryTokenBudget`、`aiSummaryTriggerRatio`、`mcpServerList`、`mcpServerFormDialog`、`mcpServerImportDialog`、`maxFileSize`、`newTagName`、`tagList`、`sortControl`、`pageSizeControl`、`logLevelControl`、`screenLockToggle`、`trashCleanupRetentionDays` 等）仍唯一存在 → 保证 `loadSettings`/`saveSettings` 正常。
3. 构建验证（遵循项目硬约束）：在 `frontend` 目录执行 `npm run build`，再执行 `wails build`，确认产物 `build/bin/jot.exe` 更新且无编译错误。
4. 运行应用核对：进入设置页，侧边显示 5 项；逐项点击确认界面切换正常、无面板内容丢失或对话框错位；保存设置后重启应用确认配置持久化。