# Tasks

- [x] Task 1: 设置页添加快速笔记开关 UI — 在 index.html 添加开关 HTML，在 style.css 添加开关样式
  - [x] SubTask 1.1: index.html: 在主题设置下方添加「快速笔记」开关行（标签 + 切换条 + 说明文字）
  - [x] SubTask 1.2: style.css: 添加 iOS 风格开关样式（`.toggle-switch`/`.toggle-track`/`.toggle-thumb`）
- [x] Task 2: 快速笔记开关交互与持久化 — 在 main.js 添加设置加载/保存/切换逻辑
  - [x] SubTask 2.1: `init()` 中加载 `quick_note_enabled` 设置
  - [x] SubTask 2.2: 添加开关 change 事件监听，持久化到后端
  - [x] SubTask 2.3: `init()` 最后判断若启用则打开全屏编辑器

# Task Dependencies
- Task 2 依赖 Task 1（需要 DOM 元素存在才能绑定事件）
