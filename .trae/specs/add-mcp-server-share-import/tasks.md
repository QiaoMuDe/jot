# Tasks

- [x] Task 1: index.html DOM 扩展
  - [x] 1.1 `.mcp-server-head` 中将单按钮改为三按钮组「分享全部」「导入」「添加服务器」（按 10px 间距并列，复用 `btn btn-sm` 现有样式，添加 `id="mcpServerShareAllBtn"` / `id="mcpServerImportBtn"`）
  - [x] 1.2 列表行操作区（`renderMCPServerList` 模板中）追加「分享」按钮（图标 share / 文字，分享图标 SVG），置于「测试」之前
  - [x] 1.3 新增导入对话框骨架 `id="mcpServerImportDialog"`：复用 `.mcp-server-form-dialog` 体系，含 backdrop + 标题「导入 MCP 服务器」+ 多行 textarea（`id="mcpServerImportInput"`，占满宽度，rows=10）+ footer「取消 / 导入」按钮
  - [x] 验证：`npm run build` 通过

- [x] Task 2: main.js 序列化与复制函数
  - [x] 2.1 实现 `buildMCPServersShareJSON(servers)`：将 `MCPServer[]` 序列化为本项目 JSON 数组（每条含 `name / transport`，按 transport 附带 `command / args / env` 或 `url / headers`，加 `enabled`；排除 `id / sort_order / created_at / updated_at`）
  - [x] 2.2 实现 `copyMCPServersShare(text, successMsg, emptyMsg)`：`navigator.clipboard.writeText` 主路径；不可用/失败时降级到隐藏 textarea + `document.execCommand('copy')`；空数据时 `nm.show(emptyMsg, 'info')` 不复制
  - [x] 2.3 「分享」行内按钮渲染：在 `renderMCPServerList` 中为每条追加 share 按钮，事件调 `copyMCPServersShare(JSON.stringify(buildMCPServersShareJSON([srv]), null, 2), '已复制「'+srv.name+'」配置', '')`
  - [x] 2.4 「分享全部」按钮事件：调 `copyMCPServersShare(JSON.stringify(buildMCPServersShareJSON(mcpServers), null, 2), '已复制 '+mcpServers.length+' 条服务器配置', '当前没有可分享的服务器')`
  - [x] 验证：`npm run build` 通过

- [x] Task 3: main.js 导入解析与批量入库
  - [x] 3.1 实现 `parseMCPServersImportJSON(text)`：
    - `JSON.parse` 解析；非合法 JSON 抛 `JSON 解析失败：{err.message}`
    - 容错：数组 → 原样；对象含 `servers` 字段 → 取 `servers`；单个对象 → 包装为 `[obj]`
    - 遍历逐条校验：`name` 非空（否则 `名称不能为空`）、`transport` 必须是 `stdio/sse/http`（否则 `transport 非法`）、stdio 必带 `command`、sse/http 必带 `url`；任一失败抛 `第 N 条：{具体错误}`
    - 返回校验通过的数组
  - [x] 3.2 实现 `openMCPImportDialog` / `closeMCPImportDialog`：与现有 form dialog 模式一致（visible 类淡入、backdrop 关闭、Esc 关闭）；打开时 textarea 聚焦且清空；关闭时清空 textarea
  - [x] 3.3 实现 `handleMCPImport`：
    - 取 textarea 文本，空则 `nm.show('请粘贴 JSON', 'error')` + 抖动
    - 调 `parseMCPServersImportJSON`，失败 → 抖动 textarea + `nm.show(错误, 'error')` + return
    - 成功 → 循环 `await SaveMCPServer({...parsed, id:0})` 累计 success/fail 计数（每条 `id=0`、`enabled=false` 入库）
    - 全部完成后：`await loadMCPServers()` + `await warmupMCPServers()` 刷新列表与池
    - 弹聚合通知 `已导入 N 条，失败 M 条，详情见应用日志`（成功 / 错误按 N/M 决定类型，全成功 success，否则 error）
    - 关闭对话框 + 清空 textarea
  - [x] 3.4 失败详情按项目记忆约定走后端日志（`console.error`），不在 UI 展示
  - [x] 验证：`npm run build` 通过

- [x] Task 4: main.js 事件绑定
  - [x] 4.1 在 `initMCPServerSettings` 末尾追加：
    - 分享全部按钮 click → 2.4
    - 导入按钮 click → 3.2
    - 导入对话框「取消」/「导入」/backdrop click → 3.3
  - [x] 4.2 导入对话框 Esc 关闭复用全局 `handleKeyboardNavigation` 既有逻辑（如不支持则手动绑定 `keydown`）
  - [x] 验证：`npm run build` 通过

- [x] Task 5: settings-panel.css 样式
  - [x] 5.1 `.mcp-server-head` 三按钮组：flex 横向、10px 间距，遵循项目 toolbar 间距约定
  - [x] 5.2 列表行分享按钮：复用 `.mcp-icon-btn` 既有图标按钮样式（如有）或与测试/编辑按钮同级，主题色与 `currentColor` 自适应
  - [x] 5.3 导入对话框复用 `.mcp-server-form-dialog` 体系，textarea 高度自适应、占满宽度，`font-family: var(--font-mono)` 等宽字体展示 JSON
  - [x] 5.4 抖动关键帧复用项目既有 `mcpFormInputError` 关键帧或新增 `mcpImportInputError` 关键帧
  - [x] 5.5 全部使用 `--bg` / `--card-bg` / `--accent` 等主题变量，14 主题自适应；`prefers-reduced-motion` 下抖动降级为 0
  - [x] 验证：`npm run build` 通过

- [x] Task 6: 构建验证与回归
  - [x] 6.1 `npm run build` 通过
  - [ ] 6.2 手动验证流程：单条分享 → 粘贴到导入 → 成功入库；分享全部 → 粘贴 → 全部入库；非法 JSON → 抖动 + 提示；name 重复 → 失败计数 +1、其余继续；空列表分享全部 → info 提示（需 wails build 后运行时验证）
  - [ ] 6.3 回归验证：原有 CRUD（新增/编辑/删除/启用开关/测试）未受影响（需 wails build 后运行时验证）
  - [ ] 6.4 14 主题下视觉一致性验证（需 wails build 后运行时验证）
  - [ ] 6.5 验证：`wails build` 成功后运行（按要求未自动执行）

# Task Dependencies
- Task 1 先行（DOM 骨架）
- Task 2 依赖 Task 1.2（列表行模板就绪）
- Task 3 依赖 Task 1.3（对话框 DOM 节点就绪）
- Task 4 依赖 Task 1-3（事件挂载需要节点 + 函数就绪）
- Task 5 与 Task 2-4 可并行
- Task 6 依赖 Task 1-5
