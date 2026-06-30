# Tasks

- [x] Task 1: HTML 结构 — 在 `index.html` 的 AI 输入区工具栏中添加 "更多技能" 按钮、下拉菜单、技能指示条
  - [x] 在 `.ai-chat-toolbar` 中添加 `#aiChatMoreSkillsBtn` 按钮（图标 + "更多技能" 文字 + 下拉箭头）
  - [x] 添加 `.ai-chat-skills-dropdown` 下拉菜单容器，包含 "翻译" 菜单项
  - [x] 在 "翻译" 菜单项内添加 `.ai-chat-skills-options` 方向选择区，包含 "翻译为中文"（默认选中）和 "翻译为英文" 两个 radio 选项
  - [x] 在工具栏与输入框之间添加 `#aiChatSkillBar` 技能指示条容器

- [x] Task 2: CSS 样式 — 在 `ai-chat.css` 中新增技能按钮、下拉菜单、chip 指示器样式
  - [x] `.ai-chat-skills-btn` 按钮样式
  - [x] `.ai-chat-skills-dropdown` 菜单样式（位置、阴影、动画）
  - [x] `.ai-chat-skills-item` 菜单项样式（hover、active 态）
  - [x] `.ai-chat-skills-options` 方向选择区样式（radio 列表）
  - [x] `.ai-chat-skill-bar` 和 `.ai-chat-skill-chip` 指示条样式（含叉号按钮）

- [x] Task 3: JS 逻辑 — 在 `ai-chat.js` 中实现技能状态管理、事件绑定、system prompt 注入
  - [x] 初始化全局 `activeSkills` 对象，管理技能激活状态及配置（翻译方向默认 `to_chinese`）
  - [x] 绑定更多技能按钮点击事件（展开/收起下拉菜单）
  - [x] 绑定菜单项点击事件（点击"翻译"展开方向选择区）
  - [x] 绑定方向选择 radio 点击事件（选择后技能激活，菜单关闭）
  - [x] 实现技能 chip 渲染（激活时显示，叉号取消）
  - [x] 修改 `startStreaming()` 函数，在构建 `messages` 时拼入技能 system prompt
  - [x] 点击外部区域关闭所有菜单

## Task Dependencies

- [Task 1] 无依赖
- [Task 2] 依赖 [Task 1]（CSS 选择器基于 HTML 结构）
- [Task 3] 依赖 [Task 1] 和 [Task 2]（JS 操作 DOM 元素需 HTML/CSS 就绪）
