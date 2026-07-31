# Tasks

## ~~Task 1~~: 新增操作菜单按钮与下拉菜单 DOM
在 `frontend/index.html` 的 `.editor-header-actions` 中（全屏按钮 `editorFullscreenBtn` 左侧）新增：
- 操作菜单按钮 `editorActionsBtn`（复用 `.editor-header-btn` 样式，图标用「斜杠/尖括号」类表示格式化操作，`title="操作"`）
- 下拉菜单容器 `editorActionsMenu`（class 复用 `.dropdown-menu`），内部由 JS 渲染，HTML 中留空容器即可
- 菜单容器相对 `editor-header-actions` 定位，右对齐弹出（CSS 在 Task 2 处理）

## ~~Task 2~~: 菜单样式（editor.css）
在 `frontend/src/css/components/editor.css` 中新增：
- `.editor-header-actions` 设为相对定位容器（若尚未有），菜单绝对定位在其下方右对齐
- 分组标签样式（如 `.dropdown-group-label`：小号、次要色、首字母大写）
- 操作项 hover 态、分隔线复用现有 `.dropdown-item` / `.dropdown-divider` 样式
- 打开/关闭通过 `.open` class 控制（沿用项目现有 dropdown 交互约定）

## ~~Task 3~~: 操作注册表与执行引擎（新文件 editor-actions.js）
新增 `frontend/src/js/editor-actions.js`，提供：
- `EDITOR_ACTIONS` 配置数组（分组驱动），首期条目：
  - `{ group: '格式化', label: 'JSON 格式化', handler }` → `JSON.stringify(parsed, null, 2)`
  - `{ group: '格式化', label: 'JSON 压缩', handler }` → `JSON.stringify(parsed)`
- `initEditorActionsMenu()`：渲染菜单 DOM（按分组聚合，空分组跳过）、绑定按钮点击开合、外部点击/Esc 关闭
- 执行引擎：读取 `cmEditor` 选中范围（`state.selection.main`，`sliceDoc(from, to)`），选中为空则取全文；`JSON.parse` 失败时 `nm.show('不是合法的 JSON', 'warning')` 且不修改内容；成功后 `cmEditor.dispatch({ changes: { from, to, insert } })` 写回并保持焦点（写回后 Ctrl+Z 可撤销，由 CM6 原生撤销栈保证）
- 暴露 `window.initEditorActionsMenu` 供 main.js 调用

## ~~Task 4~~: main.js 集成与模式联动
在 `frontend/src/main.js` 中：
- 引入/调用 `initEditorActionsMenu()`（与其他编辑器初始化函数同处）
- 查看（只读）模式：跟随 `editorTypeToggle` 的显隐逻辑隐藏 `editorActionsBtn`（`switchEditorReadOnly` 中同步）
- 预览模式：执行操作前若 `els.editorOverlay.dataset.mode === 'preview'` 则先 `switchEditorMode('edit')`

## ~~Task 5~~: 验证
- 编辑模式打开菜单，菜单右对齐弹出且含「JSON 格式化」「JSON 压缩」两项
- 选中一段 JSON 执行格式化：仅选中段被美化，其余不变，Ctrl+Z 可撤销
- 无选中执行压缩：全文被压缩
- 对非 JSON 文本操作：内容不变 + 出现「不是合法的 JSON」通知
- 查看模式按钮隐藏；预览模式操作自动切回编辑模式
- 新建笔记（无标题保存场景）下菜单同样可用
- 运行 `npm run dev`（或项目既有的前端构建/启动命令）确认无编译错误、无控制台报错

# Task Dependencies
- Task 2 依赖 Task 1（样式作用于 Task 1 新增的 DOM）
- Task 3 依赖 Task 1 的 DOM 结构（渲染目标容器）
- Task 4 依赖 Task 3（调用其导出函数）
- Task 5 依赖 Task 1-4
