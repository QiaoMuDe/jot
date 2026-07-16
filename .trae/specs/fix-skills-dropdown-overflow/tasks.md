# Tasks

- [x] Task 1: 修改 CSS 为下拉菜单添加固定高度和滚动支持
  - 在 `.ai-chat-skills-dropdown` 中添加 `max-height: 300px` 和 `overflow-y: auto`
  - 移除 `overflow: visible`
  - 添加 `scrollbar-width: thin` 保持滚动条风格一致
  - 调整或缩短逐项动画的 `transition-delay`，避免滚动容器中动画异常
  - 验证"翻译"技能的 `max-height` 子选项（`.ai-chat-skills-options`）在 `overflow-y: auto` 父容器中正常展开/收起，且 `overflow: hidden` 不与父容器滚动冲突

# Task Dependencies

无依赖，单文件修改。