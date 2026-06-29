# Tasks

- [x] Task 1: 自实现 AI 对话 UI — 替换 QuikChat，纯基础对话功能
  - [x] 修改 `index.html`：`#aiChatContainer`（QuikChat 挂载点）拆分为 `#aiChatMessages`（消息列表）+ `#aiChatInput`（textarea）+ `#aiChatSendBtn`（发送按钮），保留空状态 `#aiChatEmpty`
  - [x] 重写 `ai-chat.js`：移除 QuikChat 导入。实现 `addMessage(msg, role)` → 根据 role 创建 `.ai-msg-user` 或 `.ai-msg-assistant` 气泡 DOM，AI 回复使用 `marked.parse()` + `hljs.highlightElement()` 渲染 Markdown + 代码高亮
  - [x] 实现打字指示器：`#aiChatTyping` 三圆点 bounce 动画，发送时显示，收到回复后隐藏并替换为内容
  - [x] 输入框支持 Enter 发送 + Shift+Enter 换行
  - [x] 清空对话：确认后清空 `#aiChatMessages` innerHTML
  - [x] 页面激活时检查配置，未配置显示空状态 + 「前往设置」按钮
  - [x] 重写 `ai-chat.css`：移除所有 `.quikchat-*` 引用，保留 `.ai-chat-*` + 新增 `.ai-msg-user` / `.ai-msg-assistant` / `.ai-msg-typing` 样式，完整使用 Jot 主题变量（`--bg`/`--accent`/`--text-primary`/`--border` 等）
  - [x] 移除 `quikchat` 依赖（`package.json` + `ai-chat.js` import）

- [x] Task 2: 构建验证
  - [x] npm install（确认无 quikchat 残留 — 1 package removed）
  - [x] Wails dev 编译通过，零错误
  - [x] AI 对话页面正常加载（已配置/未配置两种状态）
  - [x] 发送消息 → 打字指示器 → AI 回复 → Markdown + 代码高亮渲染正确
  - [x] 清空对话、空状态、Enter 发送/Shift+Enter 换行功能正常

# Task Dependencies
- [Task 1] 依赖无
- [Task 2] 依赖 [Task 1]
