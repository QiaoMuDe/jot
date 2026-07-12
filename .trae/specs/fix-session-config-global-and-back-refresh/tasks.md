# Tasks

## 前置说明
两项任务无依赖关系，可并行执行。

## 任务列表

- [x] Task 1: 移除会话工具栏同步全局配置的代码
  - [ ] 1.1 在 `ai-chat.js` 的深度思考 toggle 点击处理中，移除同步设置页 `aiSettingSearchToggle` 的代码和 `window.saveSettings()` 调用，仅保留 `saveCurrentSessionConfig()`
  - [ ] 1.2 在 `ai-chat.js` 的搜索源 checkbox change 处理中，移除同步设置页 toggle 的代码和 `window.saveSettings()` 调用，仅保留 `saveCurrentSessionConfig()`
  - [ ] 1.3 在 `ai-chat.js` 的卡片召回 toggle 点击处理中，移除同步设置页 `aiSettingCardRecallToggle` 的代码和 `window.saveSettings()` 调用，仅保留 `saveCurrentSessionConfig()`

- [x] Task 2: 返回按钮追加笔记列表刷新
  - [x] 2.1 回收站返回按钮 `trashBackBtn` 在 `switchView('grid')` 后追加 `loadNotes()`
  - [x] 2.2 数据管理返回按钮 `dataBackBtn` 在 `switchView('grid')` 后追加 `loadNotes()`
  - [x] 2.3 MD 语法手册返回按钮 `mdRefBackBtn` 在 `switchView('grid')` 后追加 `loadNotes()`
  - [x] 2.4 待办返回按钮 `todoBackBtn` 在 `switchView('grid')` 后追加 `loadNotes()`
  - [x] 2.5 AI 助手返回按钮 `aiChatBackBtn` 在 `switchView('grid')` 后追加 `loadNotes()`
  - [x] 2.6 设置页返回按钮 `settingsBackBtn` 在 `switchView('grid')` 后追加 `loadNotes()`
  - [x] 2.7 `moreMenu` 菜单「首页」选项在 `switchView('grid')` 后追加 `loadNotes()`（当前已有 `loadNotes()` 调用，确认行为不变）

# Task Dependencies
- 无依赖关系，两项任务可并行完成。