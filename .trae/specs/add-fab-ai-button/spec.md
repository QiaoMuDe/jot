# 首页 FAB 区新增 AI 按钮与滚动交互 Spec

## Why
当前笔记首页右下角只有「新建笔记」和「回到顶部」两个浮动按钮，用户需要先打开菜单或按快捷键才能进入 AI 助手，操作路径较长。在 FAB 区直接放置 AI 按钮可一键跳转，提升使用效率。同时优化滚动交互：滚动时让主按钮上移并为回到顶部按钮腾出空间，视觉层次更清晰。

## What Changes
- 在 FAB 组中新增「AI 助手」圆形按钮（位于新建按钮下方）
- FAB 组布局从 `column-reverse` 改为 `column`，DOM 顺序重排为：`fabNewNote → fabAI → backToTopBtn`
- 新增 `.fab-group.scrolled` 状态类，滚动超过阈值时抬高 FAB 组位置，使主按钮上移、回到顶部按钮出现
- JS 滚动监听逻辑重写：单阈值控制「FAB 组上移 + 回到顶部按钮显隐」
- AI 按钮点击跳转到 `switchView('ai-chat')`

## Impact
- 影响 HTML：`frontend/index.html` — FAB 组 DOM 结构调整
- 影响 CSS：`frontend/src/css/components/main-content.css` — FAB 组样式重写
- 影响 JS：`frontend/src/main.js` — DOM 引用、滚动事件、事件绑定
- 不涉及后端、AI 对话逻辑、其他视图

## ADDED Requirements

### Requirement: FAB 区 AI 按钮
The system SHALL provide an AI button in the FAB group.

#### Scenario: AI 按钮显示与点击
- **WHEN** 用户在网格视图（首页）
- **THEN** FAB 组显示新建(+)按钮和 AI(🤖)按钮，AI 按钮位于新建按钮下方
- **WHEN** 用户点击 AI 按钮
- **THEN** 系统调用 `switchView('ai-chat')` 跳转到 AI 助手页面

#### Scenario: 非网格视图隐藏
- **WHEN** 用户切换到非 grid 视图
- **THEN** 整个 FAB 组（含 AI 按钮）隐藏，行为与现有新建按钮一致

### Requirement: 滚动交互（上移 + 回到顶部）
The system SHALL adjust FAB group position and show/hide back-to-top button based on scroll position.

#### Scenario: 页面在顶部
- **WHEN** `mainContent.scrollTop <= 300`
- **THEN** FAB 组处于默认位置（`bottom: 28px`），回到顶部按钮隐藏（`opacity: 0, pointer-events: none`）

#### Scenario: 滚动超过阈值
- **WHEN** `mainContent.scrollTop > 300`
- **THEN** FAB 组获得 `.scrolled` 类，`bottom` 值增大（如 `90px`），使新建和 AI 按钮上移；回到顶部按钮显示（`opacity: 1, pointer-events: auto`）

#### Scenario: 回到顶部按钮点击
- **WHEN** 用户点击回到顶部按钮
- **THEN** `mainContent` 平滑滚动到顶部（`scrollTo({ top: 0, behavior: 'smooth' })`）

## MODIFIED Requirements

### Requirement: FAB 组布局（原实现）
The system SHALL display the FAB group with proper layout.

**修改内容**：
- 布局方向从 `column-reverse` 改为 `column`
- DOM 中按钮顺序改为：新建 → AI → 回到顶部
- 回到顶部按钮在布局中位于最下方（视觉上在新建和 AI 之下）

### Requirement: 回到顶部按钮样式（原实现）
**修改内容**：
- `.fab-top` 不再需要 `transform: translateY(10px)` 初始位置偏移
- 显隐控制通过 `.visible` 类切换，不再依赖其他状态

## REMOVED Requirements
无
