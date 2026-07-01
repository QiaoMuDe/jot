# Tasks

- [x] Task 1: 在 index.html 中新增 AI 消息右键菜单 DOM 元素
  - 在 `#aiSessionContextMenu` 之后新增 `#aiMsgContextMenu` 元素
  - 复用 `.context-menu` 样式，菜单项通过 JS 动态生成

- [x] Task 2: 在 ai-chat.js 中实现右键菜单逻辑
  - 创建 `showAiMsgContextMenu(event, content, role, msgEl)` 函数
  - 创建 `hideAiMsgContextMenu()` 函数
  - 根据 `role` 动态生成菜单项：
    - 用户消息：复制
    - AI 回复：复制、保存为笔记、重新生成、追问此条回复
  - 每个菜单项点击后执行与 `createMsgActions()` 中相同的操作逻辑
  - 菜单定位跟随鼠标坐标，防止溢出视口
  - 点击外部 / Escape 键 / 点击菜单项时关闭菜单

- [x] Task 3: 为消息气泡绑定 contextmenu 事件
  - 在 `addMessage()` 创建消息气泡后，以及流式输出完成的消息上绑定 `contextmenu` 事件
  - 在右键菜单打开时阻止浏览器默认菜单

- [x] Task 4: 构建验证
  - Wails dev 编译无错误
  - 右键用户消息 → 弹出菜单含「复制」
  - 右键 AI 回复 → 弹出菜单含「复制」「保存为笔记」「重新生成」「追问此条回复」
  - 每个菜单项功能与悬浮按钮一致
  - 点击外部 / Escape 关闭菜单

# Task Dependencies
- [Task 4] 依赖 [Task 1] [Task 2] [Task 3]
