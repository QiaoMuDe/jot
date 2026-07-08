# 将快捷键说明和 MD 语法移入子菜单 Spec

## Why
「更多」菜单中「快捷键说明」和「MD 语法」属于参考类功能，使用频率较低。将它们归入一个子菜单项，可减少主菜单项数量，使菜单结构更清晰。

## What Changes
- **HTML**: 在「更多」菜单中，用带有右侧箭头图标的「帮助参考」子菜单触发器项替换原有的「快捷键说明」和 `Ctrl+7` 以及「MD 语法」和 `Ctrl+8` 两个独立菜单项；子菜单内包含这两个入口
- **CSS**: 新增子菜单容器定位/显隐样式、子菜单触发器交互样式（hover 状态、右侧箭头图标旋转）
- **JS**: 
  - 「更多」菜单点击事件中移除对 `help` 和 `md-ref` 的直接处理；新增子菜单显示/隐藏逻辑（hover 触发或点击触发）；子菜单内项点击事件仍调用 `openShortcuts()` 和 `switchView('md-ref')`
  - **键盘导航**：移除 `handleKeyboardNavigation` 中对 `Ctrl+7`（快捷键说明）和 `Ctrl+8`（MD 语法）的处理
  - **快捷键页面**：将快捷键说明页面中 `Ctrl+7` 和 `Ctrl+8` 的绑定条目移除或更新为无快捷键状态

## Impact
- Affected specs: 无
- Affected code: [index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html) (更多菜单 HTML 结构), [topbar.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/topbar.css) (子菜单样式), [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js) (菜单交互逻辑)

## MODIFIED Requirements
### Requirement: 更多菜单
The system SHALL provide a "更多" dropdown menu with a submenu for reference features.

#### Scenario: 子菜单触发
- **WHEN** 用户将鼠标悬停在（或点击）子菜单触发器项上
- **THEN** 子菜单展开，显示「快捷键说明」和「MD 语法」两个入口

#### Scenario: 子菜单项操作
- **WHEN** 用户在子菜单中点击「快捷键说明」
- **THEN** 调用 `openShortcuts()` 打开快捷键页面
- **WHEN** 用户在子菜单中点击「MD 语法」
- **THEN** 调用 `switchView('md-ref')` 切换到 MD 语法页面
