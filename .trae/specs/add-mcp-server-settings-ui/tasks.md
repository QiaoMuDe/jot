# Tasks

- [x] Task 1: index.html 新增导航项与面板骨架
  - [x] 1.1 `.settings-nav` 中新增 `data-panel="mcp-server"` 导航按钮（server 图标 + 「MCP 服务器」文案），插在「对话与搜索」之后
  - [x] 1.2 `.settings-panels` 新增 `<div class="settings-section settings-panel" data-panel="mcp-server">`：组标题（ai-group-header + server 图标 + 「MCP 服务器」）、「添加服务器」按钮、列表容器（`id="mcpServerList"`）、空态容器（`id="mcpServerEmpty"`）
  - [x] 1.3 新增表单对话框骨架（`id="mcpServerFormDialog"`，backdrop + dialog）：标题、名称输入、传输方式下拉、stdio 组（命令/参数/环境变量文本域）、sse-http 组（URL）、启用开关、「取消/保存」按钮；对话框层级低于 confirmDialog（复用项目业务 modal 层级约定）
  - [x] 验证：前端页面结构完整（静态检查）

- [x] Task 2: main.js 数据加载与列表渲染
  - [x] 2.1 实现 `loadMCPServers()`：调用 `window.go.main.App.GetMCPServers()`，结果缓存到模块级 `let mcpServers = []`，失败 `nm.show` 中文错误
  - [x] 2.2 实现 `renderMCPServerList()`：按列表渲染条目（名称、传输徽标、stdio 显示命令 / sse-http 显示 URL 摘要、启用开关、编辑/删除按钮），空列表显示空态隐藏列表
  - [x] 2.3 在 `loadSettings()` 中联动调用 `loadMCPServers()`；「添加服务器」按钮绑定打开表单逻辑
  - [x] 验证：`npm run build` 通过

- [x] Task 3: main.js 表单模态（新增/编辑）
  - [x] 3.1 实现打开表单：新增传空记录、编辑以记录当前值预填充；对话框显示/隐藏与焦点管理（Esc 关闭、点 backdrop 关闭）
  - [x] 3.2 传输方式下拉切换时动态显隐 stdio 组 / sse-http 组字段
  - [x] 3.3 保存：前端校验（名称非空、stdio 命令必填、sse/http URL 必填、环境变量行需含 `=`），参数文本域每行转数组、环境变量文本域每行 `KEY=VALUE` 转对象；调用 `window.go.main.App.SaveMCPServer({...})`
  - [x] 3.4 成功：关闭对话框、重新加载列表、`nm.show('已保存...','success')`；失败：保留对话框、`nm.show(错误信息,'error')`
  - [x] 验证：`npm run build` 通过

- [x] Task 4: main.js 行内操作（启用开关/删除）
  - [x] 4.1 启用开关切换：翻转 enabled 后调用 `SaveMCPServer`，成功 `nm.show` 成功提示；失败恢复原状态并 `nm.show` 中文错误
  - [x] 4.2 删除：`showConfirmDialog('确定删除 MCP 服务器「xx」？', '删除')` 确认后调用 `window.go.main.App.DeleteMCPServer(id)`，成功刷新列表并提示；取消无操作
  - [x] 验证：`npm run build` 通过

- [x] Task 5: settings-panel.css 样式
  - [x] 5.1 列表条目（卡片式行：名称 + 徽标 + 摘要 + 操作区）、传输类型徽标（stdio/sse/http 差异化色）、空态样式
  - [x] 5.2 表单对话框样式（backdrop、卡片、字段间距、启用开关复用 `ai-chat-toggle-switch` 既有样式）
  - [x] 5.3 全部使用 `--bg`/`--card-bg`/`--accent` 等主题变量，14 主题自适应；`prefers-reduced-motion` 下无位移动画
  - [x] 验证：`npm run build` 通过

- [x] Task 6: 绑定生成与构建验证
  - [x] 6.1 运行 `wails generate module`（或 `wails build`）确保 `frontend/wailsjs/go/main/App.js` 生成 `GetMCPServers` / `SaveMCPServer` / `DeleteMCPServer` 绑定
  - [x] 6.2 `npm run build` 通过；确认前端资源编译后页面生效（Wails 前端资源需重新构建编译）
  - [x] 6.3 代码审查：导航切换、列表渲染、表单校验、开关持久化、删除确认路径均符合 spec
  - [x] 验证：`npm run build` 与 `wails build` 均成功

# Task Dependencies

- Task 2/3/4 依赖 Task 1（DOM 骨架先行）
- Task 3 依赖 Task 2（表单保存成功后刷新列表）
- Task 4 依赖 Task 2（行内操作复用列表渲染与数据缓存）
- Task 5 可与 Task 2-4 并行
- Task 6 依赖 Task 1-5（最终构建与审查）
