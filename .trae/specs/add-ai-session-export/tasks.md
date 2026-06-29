# Tasks

- [x] Task 1: 后端导出绑定 — `app.go` 新增 `ExportAISessionAsMarkdown(sessionID uint) (string, error)`，加载会话消息、格式化为 Markdown、弹出保存对话框写入文件
  - 使用 `runtime.SaveFileDialog`（参照 `ExportNoteAsMarkdown` 模式）
  - 默认文件名 `{会话标题}.md`，`.md` 文件过滤
  - 格式：标题 + 分隔线 + 每条消息带角色前缀，有 reasoning_content 时追加 > 引用块
  - 返回成功/取消/错误信息

- [x] Task 2: 右键菜单 HTML + CSS — `index.html` 新增会话右键菜单元素，`ai-chat.css` 新增菜单样式
  - HTML：`#aiSessionContextMenu`，含"重命名"和"导出"两个菜单项（可复用 `.context-menu` 样式类）
  - CSS：固定定位、卡片背景、圆角阴影、`opacity` + `scale` 过渡动画、`z-index` 高于侧栏

- [x] Task 3: 前端右键菜单逻辑 — `ai-chat.js` 中实现右键菜单的显示/隐藏/定位
  - `renderSessionList` 中为每个会话项添加 `contextmenu` 事件监听
  - `contextmenu` 事件：`preventDefault()`、定位菜单到鼠标位置、添加 `active` 类
  - 点击会话标题/右侧内容区/按 `Escape` 关闭菜单
  - 点击菜单项后关闭菜单

- [x] Task 4: 提取内联编辑函数 + 右键重命名 — `ai-chat.js` 中将内联编辑逻辑提取为独立函数 `startInlineEdit(titleEl, sessionId)`，在双击和右键菜单中复用
  - 现有 dblclick 逻辑保持不变，改为调用 `startInlineEdit(titleEl, s.id)`
  - 右键"重命名"调用 `startInlineEdit(titleEl, activeSessionId)`（使用当前会话 ID）

- [x] Task 5: 前端导出调用 — `ai-chat.js` 中实现"导出"菜单项逻辑，调用后端绑定并显示通知

# Task Dependencies
- Task 1 独立，可先实施
- Task 2 独立，可先实施
- Task 3 依赖 Task 2（需要有 HTML/CSS 元素）
- Task 4、5 可在 Task 3 后并行实施
