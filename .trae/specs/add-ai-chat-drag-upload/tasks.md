# Tasks

- [ ] Task 1: 后端提取 readAIChatFiles 内部方法（app.go）
  - 步骤 1.1: 将 `SelectAIChatFiles` 中的文件遍历/校验/读取/截断逻辑提取为私有方法 `func (a *App) readAIChatFiles(paths []string) []AIChatFileResult`
  - 步骤 1.2: `SelectAIChatFiles` 中先调用 `runtime.OpenMultipleFilesDialog` 获取 paths，再调用 `readAIChatFiles(paths)` 返回结果
  - 步骤 1.3: 新增公开方法 `func (a *App) ReadAIChatFiles(paths []string) []AIChatFileResult`（Waisl 绑定用），直接调用 `readAIChatFiles(paths)`
  - 验证: 前端可调用 `window.go.main.App.ReadAIChatFiles(paths)` 获取文件结果

- [ ] Task 2: 新增 AI 聊天专属遮罩 HTML（index.html）
  - 在 `#aiChatMessages` 和 `#aiChatInputArea` 之间或 `#viewAiChat` 内新增 `#aiChatDropOverlay`，文字"释放以上传文件"
  - 遮罩使用上传 SVG 图标（与全局遮罩相同），`style="display:none"` 初始隐藏

- [ ] Task 3: 新增 AI 聊天遮罩 CSS（ai-chat.css）
  - `.ai-chat-drop-overlay`：`position: absolute; inset: 0` 锚定在 `.ai-chat-content`（已有 `position: relative`）
  - 默认 `opacity: 0; pointer-events: none`，`.active` 时 `opacity: 1`
  - 内容框样式与全局 `.drop-overlay-content` 一致：dashed border、居中、渐变背景等
  - transition 动画与全局遮罩一致

- [ ] Task 4: 全局拖拽屏蔽 AI 聊天区（main.js）
  - `initFileDrop()` 的 `dragenter` 中：当 `e.target.closest('.ai-chat-content')` 存在时，不操作 `_dragCounter` 和全局遮罩
  - `dragleave` 中：相同判断，不减少 `_dragCounter`
  - `dragover` 中：不需要额外处理（AI 聊天区内不涉及编辑器悬停）
  - `drop` 中：相同判断，不重置遮罩
  - 验证: 拖拽文件到 AI 聊天区时全局遮罩不出现

- [ ] Task 5: AI 聊天模块拖拽事件和遮罩控制（ai-chat.js）
  - 获取 `.ai-chat-content` 和 `#aiChatDropOverlay` 的 DOM 引用
  - 在 `.ai-chat-content` 上注册 `dragenter/dragover/dragleave` 事件
  - AI 专属 `_aiDragCounter` 计数器控制遮罩显隐
  - `dragenter`：`e.preventDefault()` + 有文件时 `_aiDragCounter++` → 为 1 时遮罩 `.active`
  - `dragover`：`e.preventDefault()`（必须，否则 drop 不触发）
  - `dragleave`：`_aiDragCounter--` → 为 0 时遮罩移除 `.active`
  - 验证: 拖拽文件进 AI 聊天区显示"释放以上传文件"遮罩，拖出后消失

- [ ] Task 6: OnFileDrop 路由至 AI 聊天处理（main.js + ai-chat.js）
  - main.js `OnFileDrop` 回调中：在 `cmEditor !== null` 判断之前，先用 `elementFromPoint(x, y)` 判断是否在 `.ai-chat-content` 内
  - 如果是：调用 `window.handleAiChatFileDrop(paths)`（挂到 window 上），然后 return
  - ai-chat.js 中：定义 `window.handleAiChatFileDrop` 函数，调用 `window.go.main.App.ReadAIChatFiles(paths)`，处理结果（错误通知 + 成功加入 `uploadedFiles[]` + `renderFileChips()`）
  - 验证: 拖拽文件到 AI 聊天消息区/输入区，文件出现在 chips 中

# Task Dependencies
- Task 1 → Task 6（后端方法需先存在）
- Task 2 → Task 3（HTML 结构先于样式）
- Task 2,3 → Task 5（遮罩 DOM 和样式先于 JS 控制）
- Task 4,5,6 无彼此依赖，可并行实现
