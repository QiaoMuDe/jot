# MCP 服务器设置面板 CRUD（前端）Spec

## Why

后端已将 MCP 服务器配置迁移至 SQLite（`mcp_servers` 表 + `GetMCPServers` / `SaveMCPServer` / `DeleteMCPServer` 绑定，见 `migrate-mcp-config-to-db-backend`），但设置页缺少管理界面，用户无法通过页面新增/编辑/启停 MCP 服务器。本轮在设置页新增「MCP 服务器」面板，提供列表展示、新增/编辑表单与启用开关。

## What Changes

- `frontend/index.html`：设置侧边栏（`.settings-nav`）新增「MCP 服务器」导航项（`data-panel="mcp-server"`）；`.settings-panels` 新增对应 `.settings-section.settings-panel` 面板（组标题 + 列表容器 + 空态 + 「添加服务器」按钮）
- `frontend/src/main.js`：
  - 数据加载：`loadMCPServers()` 调用 `window.go.main.App.GetMCPServers()`，随 `loadSettings()` 联动加载
  - 列表渲染：`renderMCPServerList()` 动态渲染条目（名称、传输类型徽标、命令/URL 摘要、启用开关、编辑/删除按钮），空列表显示空态
  - 表单模态：新增/编辑共用一个表单对话框（名称、传输方式下拉、stdio 组「命令/参数/环境变量」、sse/http 组「URL」、启用开关），传输方式切换动态显隐字段
  - 行内操作：启用开关切换即调用 `SaveMCPServer` 持久化；删除经 `showConfirmDialog` 确认后调用 `DeleteMCPServer`
  - 反馈：操作成功用 `nm.show(msg,'success')`，后端返回的中文错误用 `nm.show(msg,'error')` 提示；保存失败不关闭表单
- `frontend/src/css/components/settings-panel.css`：新增列表条目、传输徽标、空态、表单对话框样式（复用 `--bg`/`--card-bg` 等主题变量，自适应 14 主题；`prefers-reduced-motion` 降级动画）

**BREAKING**：无（纯前端新增，后端绑定已就绪）

## Impact

- Affected specs：`migrate-mcp-config-to-db-backend`（其绑定方法由本面板消费）
- Affected code：
  - `frontend/index.html`（导航项 + 面板 + 表单对话框骨架）
  - `frontend/src/main.js`（加载/渲染/表单/行内操作）
  - `frontend/src/css/components/settings-panel.css`（样式）
  - `frontend/wailsjs/go/main/App.js`（`wails build`/`wails generate module` 后自动补 `GetMCPServers`/`SaveMCPServer`/`DeleteMCPServer` 绑定）

## ADDED Requirements

### Requirement: 设置页 MCP 服务器面板与列表

系统 SHALL 在设置页侧边栏提供「MCP 服务器」导航项，面板内展示服务器列表与空态。

#### Scenario: 展示服务器列表
- **WHEN** 用户打开「MCP 服务器」面板（或应用初始化 `loadSettings`）
- **THEN** 调用 `GetMCPServers()` 渲染列表，每条展示：名称、传输类型徽标（stdio/sse/http）、命令（stdio）或 URL（sse/http）摘要、启用开关、编辑与删除按钮

#### Scenario: 空列表
- **WHEN** 数据库无任何 MCP 服务器记录
- **THEN** 显示空态提示（如「尚未配置 MCP 服务器」）与「添加服务器」按钮

### Requirement: 新增/编辑表单对话框

系统 SHALL 提供模态表单对话框支持新增与编辑 MCP 服务器，按传输方式动态展示字段。

#### Scenario: 新增服务器
- **WHEN** 用户点击「添加服务器」，填写表单并提交
- **THEN** 前端校验通过后调用 `SaveMCPServer`，成功则关闭对话框、刷新列表并提示成功；失败则保留对话框并提示后端中文错误

#### Scenario: 编辑服务器
- **WHEN** 用户点击某条目的「编辑」
- **THEN** 表单以该记录当前值预填充，保存后更新该记录并刷新列表

#### Scenario: 传输方式动态字段
- **WHEN** 传输方式选择 stdio
- **THEN** 显示并校验「命令」必填，展示「参数」（每行一个参数，转为数组）与「环境变量」（每行 `KEY=VALUE`，转为对象）文本域
- **WHEN** 传输方式选择 sse/http
- **THEN** 显示并校验「URL」必填，隐藏命令/参数/环境变量字段

#### Scenario: 前端校验失败
- **WHEN** 名称为空、或 stdio 缺命令、或 sse/http 缺 URL、或环境变量行缺少 `=` 分隔
- **THEN** 前端直接提示错误（`nm.show`），不发起保存

### Requirement: 启用开关

系统 SHALL 支持在列表行内直接切换服务器启用状态并持久化。

#### Scenario: 行内切换启用
- **WHEN** 用户点击某条目的启用开关
- **THEN** 立即调用 `SaveMCPServer`（`enabled` 翻转），成功提示；失败则恢复原状态并提示中文错误

### Requirement: 删除服务器

系统 SHALL 提供删除能力且必须先经用户确认。

#### Scenario: 删除确认
- **WHEN** 用户点击某条目的「删除」
- **THEN** 弹出确认对话框（复用 `showConfirmDialog`，层级高于表单对话框），确认后调用 `DeleteMCPServer` 并刷新列表；取消则无操作

## MODIFIED Requirements

### Requirement: 设置页侧边栏导航扩展

设置侧边栏导航（`.settings-nav`）新增「MCP 服务器」项，随既有 `switchSettingsTab` 机制切换面板；切换面板时关闭可能打开的 MCP 表单相关浮层（若有）。

## REMOVED Requirements

无
