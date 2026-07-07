# AI 助手多项修复与改进 Spec

## Why

用户反馈了 AI 助手模块的多个问题：联网搜索菜单样式与"更多技能"不统一、会话侧边栏无法切换到空内容会话、数据库瘦身时未清理空 AI 会话、展开/折叠会话列表时数据源不确定。

## What Changes

1. **联网搜索下拉菜单样式统一**：将 `.ai-chat-search-sources-dropdown` 的样式改为与 `.ai-chat-skills-dropdown` 一致，使用 `.open` class 切换显隐，去掉 `style.display` 直接操作，增加菜单项渐入动画。
2. **修复空会话切换 Bug**：`switchSession` 中当会话消息为空时，提前 return 导致高亮更新代码无法执行。修复后将高亮更新移到 return 之前。
3. **数据库瘦身新增清理空 AI 会话**：在 `note_service.go` 的 `Vacuum()`（或 `app.go` 的 `VacuumDatabase()`）中，在 VACUUM 之前先查询并删除没有关联消息的 AI 会话记录。
4. **会话列表展开时刷新数据**：确保每次展开会话侧边栏时调用 `loadSessionList()` 从数据库获取最新列表，而非展示内存中可能过期的 `sessions` 数组。

## Impact

- Affected specs: `add-db-vacuum`（数据库瘦身功能）、`fix-ai-sessions-and-collapsible-sidebar`（侧栏折叠/展开）
- Affected code:
  - `frontend/index.html` — 联网搜索下拉菜单 HTML 结构调整
  - `frontend/src/css/components/ai-chat.css` — 新增搜索菜单样式，统一为技能菜单风格
  - `frontend/src/js/ai-chat.js` — 修复 `switchSession`、联网搜索菜单切换逻辑、侧栏展开时刷新列表
  - `internal/services/note_service.go` — `Vacuum()` 新增清理空 AI 会话逻辑
  - `internal/services/ai_service.go` — 可能新增查询空会话的方法
  - `app.go` — `VacuumDatabase()` 可能调整

## ADDED Requirements

### Requirement: 联网搜索菜单样式统一

联网搜索下拉菜单 SHALL 采用与"更多技能"下拉菜单一致的样式和交互。

- **WHEN** 用户点击"联网搜索"按钮
- **THEN** 下拉菜单以弹性动画展开，菜单项逐个滑入
- **AND** 使用 `.open` class 控制显隐，而非直接操作 `style.display`
- **AND** 菜单的背景色、圆角、阴影、过渡曲线与技能菜单一致

### Requirement: 侧栏展开时刷新会话列表

当会话侧边栏从折叠状态展开时，系统 SHALL 调用 `loadSessionList()` 从数据库获取最新会话列表。

- **WHEN** 用户点击折叠/展开按钮使侧边栏从折叠变为展开
- **THEN** 系统自动调用 `loadSessionList()` 刷新列表
- **AND** 确保列表展示的是数据库中的最新数据

### Requirement: 数据库瘦身清理空 AI 会话

数据库瘦身功能在执行 VACUUM 之前，SHALL 先删除没有关联消息的 AI 会话记录。

- **WHEN** 用户点击"数据库瘦身"按钮
- **THEN** 系统先查询所有没有关联 `ai_messages` 记录的 `ai_sessions` 记录
- **AND** 删除这些空会话记录
- **AND** 在结果提示中告知用户删除了多少条空会话（如果有）

## MODIFIED Requirements

### Requirement: switchSession 空会话高亮修复（现有）

- **MODIFIED**: 当会话消息为空时，`switchSession` 的侧栏高亮更新代码应提前执行，确保在高亮更新后再 return
- **FIX**: 将 `activeSessionId = id` 设置后的侧栏高亮更新代码移到 `if (!msgs || msgs.length === 0)` 条件块内

### Requirement: 联网搜索菜单交互方式（现有）

- **MODIFIED**: 将 `searchSourcesDropdown.style.display = 'none'/'block'` 改为 `searchSourcesDropdown.classList.toggle('open')`
- **AND**: 修改 `showWelcome()` 中的会话标题更新逻辑，确保空会话也能正确显示标题

### Requirement: 数据库瘦身流程（现有）

- **MODIFIED**: `VacuumDatabase()` 中在执行 VACUUM 前先调用清理空 AI 会话的逻辑
- **AND**: 提示信息中包含清理的空会话数量

## REMOVED Requirements

无
