# 移除更多菜单 Ctrl+1~8 快捷键绑定 Spec

## Why

更多菜单中的 Ctrl+1~8 快捷键（笔记首页/侧栏/批量管理/数据管理/回收站/待办清单/设置/AI 助手）在 title 属性中以 tooltip 形式展示，实用性低（用户极少通过悬停发现这些快捷键），且快捷键存在与用户预期不符的冲突风险（如 Ctrl+3 批量管理在非输入框处意外触发）。移除后可简化代码，消除误触可能。

## What Changes

- 从 `frontend/index.html` 更多菜单的 `.dropdown-item` 中移除 `title` 属性
- 从 `frontend/src/main.js` 的全局 `keydown` 监听中移除 Ctrl+1~8 的 switch case 分支
- 从 `frontend/src/main.js` 的 `renderShortcutsPage()` 快捷键说明列表中移除 Ctrl+1~8 条目
- 从 `frontend/src/main.js` 的 `updateSidebarMenuItem()` 中移除动态设置 `title` 的逻辑
- **保留** Ctrl+0 锁屏快捷键（不在更多菜单中，属于安全功能）
- **保留** Ctrl+J AI 侧栏折叠快捷键（不在更多菜单中）

## Impact

- Affected specs: 快捷键说明页面、更多菜单
- Affected code:
  - `frontend/index.html`（更多菜单 HTML 结构）
  - `frontend/src/main.js`（keydown 事件处理、快捷键说明渲染、侧栏菜单更新）

## MODIFIED Requirements

### Requirement: 更多菜单

The system SHALL display 更多菜单项不含 title 属性快捷键提示。

#### Scenario: 悬停更多菜单项
- **WHEN** 用户悬停在菜单项上
- **THEN** 不显示快捷键 tooltip

### Requirement: 全局键盘快捷键

The system SHALL NOT 对 Ctrl+1~8 注册全局键盘监听。

#### Scenario: 按下 Ctrl+1~8
- **WHEN** 用户在非输入框区域按下 Ctrl+1~8
- **THEN** 不触发视图切换/操作

#### Scenario: 按下 Ctrl+0
- **WHEN** 用户在非输入框区域按下 Ctrl+0
- **THEN** 仍然触发锁屏操作

### Requirement: 快捷键说明页面

The system SHALL 在快捷键说明页中移除 Ctrl+1~8 条目。

#### Scenario: 打开快捷键说明
- **WHEN** 用户打开快捷键说明页面
- **THEN** 列表中不包含 Ctrl+1~8 相关条目

## REMOVED Requirements

### Requirement: 更多菜单快捷键 tooltip

**Reason**: 实用性低，仅在悬停 tooltip 中展示，发现成本高
**Migration**: 无迁移需求，功能本身不变（菜单项点击仍可用）

### Requirement: Ctrl+1~8 全局快捷键导航

**Reason**: 存在误触风险，用户可改用菜单点击完成相同操作
**Migration**: 用户可通过更多菜单直接点击访问对应页面
