# AI 会话配置与全局配置分离 & 返回按钮刷新笔记列表 Spec

## Why

### 问题 1：会话操作栏与全局配置耦合
在 AI 会话操作栏中编辑深度思考、搜索源、卡片召回等开关时，当前代码会同步更新设置页（全局配置）的对应 toggle 并调用 `window.saveSettings()` 保存到全局配置。这导致：
- 会话级的操作不应该影响全局设置
- 会话配置与会话配置之间也不应该相互影响
- 只有新建会话（无会话配置时）才应从全局配置复制初始值

### 问题 2：返回按钮未刷新笔记列表
各页面（回收站、数据管理、MD 语法手册、待办、AI 助手、设置）的返回按钮，以及 `moreMenu` 菜单中的「首页」选项，点击回到首页时仅调用 `switchView('grid')` 切换视图，未调用 `loadNotes()` 刷新笔记列表，导致用户看到的是旧缓存数据。

## What Changes

### 问题 1（会话配置分离）
- 移除会话工具栏 toggle 变更时同步设置页 toggle 的代码
- 移除会话工具栏 toggle 变更时调用 `window.saveSettings()` 的代码
- 保留 `saveCurrentSessionConfig()` 调用，确保只保存到会话配置
- 新建会话时「从全局配置创建默认配置」的行为保持不变

### 问题 2（返回按钮刷新）
- 所有返回按钮的点击处理中，在 `switchView('grid')` 后追加 `loadNotes()` 调用
- 涉及：回收站、数据管理、MD 语法手册、待办清单、AI 助手、设置页的返回按钮，以及 `moreMenu` 菜单中的「首页」选项

## Impact
- Affected specs: 无
- Affected code:
  - `frontend/src/js/ai-chat.js` — 移除会话工具栏同步全局配置的代码
  - `frontend/src/main.js` — 返回按钮追加 `loadNotes()` 调用

## ADDED Requirements
### Requirement: 会话操作栏只影响会话配置
- **WHEN** 用户在 AI 会话操作栏切换深度思考开关
- **THEN** 只更新 `enableThinking` 变量并保存到会话配置（`saveCurrentSessionConfig()`）
- **AND** 不同步更新设置页 toggle，不调用 `window.saveSettings()`

- **WHEN** 用户在 AI 会话操作栏切换搜索源
- **THEN** 只更新 `searchSources` 变量并保存到会话配置
- **AND** 不同步更新设置页 toggle，不调用 `window.saveSettings()`

- **WHEN** 用户在 AI 会话操作栏切换卡片召回开关
- **THEN** 只更新 `enableCardRecall` 变量并保存到会话配置
- **AND** 不同步更新设置页 toggle，不调用 `window.saveSettings()`

### Requirement: 返回按钮刷新笔记列表
- **WHEN** 用户点击回收站/数据管理/MD语法手册/待办/AI助手/设置页的返回按钮
- **THEN** 切换回笔记首页视图后，自动调用 `loadNotes()` 获取最新笔记列表

- **WHEN** 用户在 `moreMenu` 菜单中选择「首页」
- **THEN** 切换回笔记首页视图后，自动调用 `loadNotes()` 获取最新笔记列表