# MCP 服务器分享与导入 Spec

## Why

设置页 MCP 服务器面板已支持 CRUD（见 `add-mcp-server-settings-ui`），但用户需要把一套 MCP 配置迁移到另一台设备或与他人共享时只能逐条手动抄录，效率低且易出错。本轮新增「分享 / 分享全部 / 导入」三件套，让用户一键把当前项目格式的 MCP 配置复制到剪贴板，或粘贴 JSON 批量入库。

## What Changes

- `frontend/index.html`：
  - MCP 服务器面板头部 `mcp-server-head`：「添加服务器」旁新增「分享全部」与「导入」两个按钮
  - 列表条目操作区：在「测试 / 编辑 / 删除」之外新增「分享」按钮（与编辑同级）
  - 新增导入对话框骨架（`id="mcpServerImportDialog"`）：含标题、多行 `<textarea id="mcpServerImportInput">`、`取消` / `导入` 两个按钮
- `frontend/src/main.js`：
  - 新增 `buildMCPServersShareJSON(servers)`：将 `MCPServer[]` 序列化为本项目格式 JSON（数组，每条带 `name / transport / command? / args? / env? / url? / headers? / enabled`，不含 `id / sort_order / created_at / updated_at`）
  - 新增 `copyMCPServersShare(servers)`：调用 `navigator.clipboard.writeText`；被拒时降级为 `document.execCommand('copy')` + 隐藏 textarea 方案
  - 新增 `parseMCPServersImportJSON(text)`：解析 JSON、容错接受数组或 `{servers: [...]}` 包装，逐条校验 `name` 非空、transport 合法、stdio 必带 command、sse/http 必带 url，校验失败抛出中文错误
  - 新增 `openMCPImportDialog / closeMCPImportDialog / handleMCPImport`：打开对话框、点「导入」解析 JSON、解析成功直接循环 `SaveMCPServer` 写入数据库（不入表单）、刷新列表与全局池、聚合通知「已导入 N / 失败 M 条，详情见应用日志」；解析失败抖动 textarea + `nm.show` 中文错误
  - 「分享」行内按钮：调 `copyMCPServersShare([srv])` 复制单条，提示「已复制「xxx」配置」
  - 「分享全部」按钮：调 `copyMCPServersShare(mcpServers)` 复制全部，提示「已复制 N 条服务器配置」
- `frontend/src/css/components/settings-panel.css`：导入对话框样式（复用 `.mcp-server-form-dialog` 体系）、分享按钮样式（图标 + 文字，主题自适应）、按钮组间距

**BREAKING**：无（纯前端新增，剪贴板降级方案不影响功能）

## Impact

- Affected specs：`add-mcp-server-settings-ui`（复用其列表渲染、缓存、刷新与全局池预热 `warmupMCPServers`）
- Affected code：
  - `frontend/index.html`（头部按钮组 + 列表行分享按钮 + 导入对话框）
  - `frontend/src/main.js`（解析/复制/导入三件套函数 + 事件绑定）
  - `frontend/src/css/components/settings-panel.css`（导入对话框 + 分享按钮样式）
  - 后端：0 改动

## ADDED Requirements

### Requirement: 单条分享

系统 SHALL 在每条 MCP 服务器列表项的操作区提供「分享」按钮，点击后将该条记录以本项目格式 JSON 写入剪贴板。

#### Scenario: 单条分享成功
- **WHEN** 用户点击某条目的「分享」
- **THEN** 调用 `navigator.clipboard.writeText` 写入该条 JSON，弹 `nm.show('已复制「{name}」配置', 'success')`

#### Scenario: 剪贴板 API 不可用
- **WHEN** `navigator.clipboard` 不可用或用户拒绝授权
- **THEN** 降级使用隐藏 textarea + `document.execCommand('copy')`；成功同样提示；失败提示「复制失败，请手动复制」

### Requirement: 分享全部

系统 SHALL 在面板头部「添加服务器」按钮旁提供「分享全部」按钮，点击后将全部 MCP 服务器配置以 JSON 数组写入剪贴板。

#### Scenario: 分享全部
- **WHEN** 用户点击「分享全部」且 `mcpServers` 非空
- **THEN** 复制全部 JSON 数组，弹 `nm.show('已复制 N 条服务器配置', 'success')`

#### Scenario: 列表为空
- **WHEN** 用户点击「分享全部」但 `mcpServers.length === 0`
- **THEN** 弹 `nm.show('当前没有可分享的服务器', 'info')`，不复制

### Requirement: 导入对话框

系统 SHALL 在面板头部「添加服务器」按钮旁提供「导入」按钮，点击打开导入对话框，对话框内含多行文本输入与「取消 / 导入」按钮。

#### Scenario: 打开导入对话框
- **WHEN** 用户点击「导入」
- **THEN** 弹出对话框，textarea 自动获得焦点，textarea 内容为空（不记忆上次输入）

#### Scenario: 关闭对话框
- **WHEN** 用户点击「取消」、点 backdrop 或按 Esc
- **THEN** 关闭对话框并清空 textarea 内容

### Requirement: JSON 解析与批量入库

系统 SHALL 在用户点「导入」时解析 textarea JSON，校验通过后**直接**循环调用 `SaveMCPServer` 写入数据库（不再弹出添加表单），全部完成后刷新列表与全局 MCP 池。

#### Scenario: 解析成功并批量入库
- **WHEN** textarea 内容是合法 JSON 且每条字段满足项目校验规则
- **THEN** 逐条调 `SaveMCPServer`，成功计数 +1；全部完成后调 `loadMCPServers()` 刷新列表与 `warmupMCPServers()` 同步全局池，弹聚合通知「已导入 N / 失败 M 条，详情见应用日志」，关闭对话框并清空 textarea

#### Scenario: 解析失败
- **WHEN** textarea 内容不是合法 JSON、不是数组/对象包装、或某条缺 `name`/transport 非法/stdio 缺 command/sse-http 缺 url
- **THEN** 抖动 textarea + 调 `nm.show(中文错误, 'error')`，不关闭对话框、不写入数据库

#### Scenario: 容错包装格式
- **WHEN** textarea 是 `[{...}, {...}]`（裸数组）
- **THEN** 视为多条配置
- **WHEN** textarea 是 `{"servers": [{...}, {...}]}`（servers 包装）
- **THEN** 取 `servers` 字段
- **WHEN** textarea 是单个 `{...}` 配置对象
- **THEN** 视为单条配置包装为数组处理

### Requirement: 重复名称处理

系统 SHALL 在导入时让后端 `SaveMCPServer` 的「名称已存在」校验自然拦截，前端不预先改名或跳过。

#### Scenario: 导入与现有重名
- **WHEN** 导入 JSON 中某条 `name` 与库内已有记录重名
- **THEN** 该条 `SaveMCPServer` 返回「名称 xxx 已存在」错误并被计入失败数，其余条目继续；失败详情不展示给用户，按项目记忆中的「导入结果通知聚合」约定写入后端日志

## MODIFIED Requirements

### Requirement: MCP 服务器面板头部按钮组

「MCP 服务器」面板头部 `.mcp-server-head` 由单按钮扩展为三按钮：「分享全部」/「导入」/「添加服务器」，按钮按 10px 间距并列（遵循项目 toolbar spacing 约定），样式与现有 `btn btn-sm btn-save` / `btn-cancel` 一致。

### Requirement: 列表行操作区扩展

每条服务器操作区由「测试 / 编辑 / 删除」三按钮扩展为「分享 / 测试 / 编辑 / 删除」四按钮，分享按钮置于最左（最常用操作优先）。

## REMOVED Requirements

无
