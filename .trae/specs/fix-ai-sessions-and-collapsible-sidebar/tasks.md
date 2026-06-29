# Tasks

- [x] Task 1: 后端 — Message 结构体追加 ReasoningContent 字段
  - `services.Message` 新增 `ReasoningContent string \`json:"reasoning_content"\``
  - `LoadAISessionMessages` 拷贝 `ReasoningContent` 字段到返回值

- [x] Task 2: 前端 JS — 修复 switchSession + addMessage 消息错乱问题
  - `addMessage` 新增第三个参数 `reasoningContent`
  - 当 `role === 'assistant'` 且有 `reasoningContent` 时，在气泡内渲染可折叠思考区域
  - `switchSession` 加载消息前先确保清理历史（`messagesEl.innerHTML = ''`）
  - `switchSession` 对每条消息用正确的 role 调用 `addMessage`
  - `onAIChatViewActivated` 仅在 `activeSessionId === null` 时才自动加载

- [x] Task 3: 前端 CSS — 侧栏折叠/展开样式
  - `.ai-session-sidebar.collapsed` 状态：`width: 0; overflow: hidden`
  - `.ai-session-sidebar` 添加 `transition: width 0.25s ease`
  - 折叠按钮 `.ai-sidebar-toggle`：固定在侧栏右边框位置，35×35px 圆形
  - 侧栏展开时按钮 `left: 220px`，折叠时 `left: 0`

- [x] Task 4: 前端 JS — 实现侧栏折叠/展开功能
  - 在侧栏右侧添加折叠/展开按钮
  - 默认折叠（`collapsed = true`），存储在 `localStorage`
  - 点击切换 `collapsed` 状态，更新 CSS class
  - `onAIChatViewActivated` 读取 `localStorage` 恢复折叠状态

# Task Dependencies
- Task 1 → Task 2（前端依赖 ReasoningContent 字段存在）
- Task 3、Task 4 可并行
