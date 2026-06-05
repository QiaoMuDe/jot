# 杂项优化 Spec

## Why
提升视觉品质（美化滚动条）和开箱即用体验（默认标签+快捷键说明）。

## What Changes
1. 字体族下拉菜单的 `.font-family-options` 滚动条美化，与全局滚动条风格统一
2. 首次初始化数据库时插入默认标签（待办、工作、生活、个人、学习、重要）
3. 新增快捷键说明页（快捷键一览），入口在关于页面，用 `?` 按钮标识

## Impact
- Affected specs: 设置页面（字体下拉滚动条）、标签系统（种子数据）、导航视图（新增快捷键页）
- Affected code:
  - `frontend/src/style.css` — 美化滚动条
  - `services/tag_service.go` — 初始化默认标签
  - `database/db.go` — 初始化时调用种子数据
  - `frontend/index.html` — 新增快捷键视图
  - `frontend/src/main.js` — 新增视图渲染与导航
  - `frontend/src/app.css` — 全局样式

## ADDED Requirements

### Requirement: 字体下拉滚动条美化
The system SHALL style the scrollbar within `.font-family-options` to match the application's warm minimal design.

#### Scenario: 显示美化滚动条
- **WHEN** 用户打开字体族下拉菜单且选项超出可见区域
- **THEN** 滚动条使用圆角细条样式，颜色匹配 `--divider` 和 `--accent`（hover 时）

### Requirement: 默认标签种子数据
The system SHALL pre-populate default tags on first database initialization.

#### Scenario: 首次启动
- **WHEN** 应用首次初始化数据库且标签表为空
- **THEN** 插入默认标签：`待办`、`工作`、`生活`、`个人`、`学习`、`重要`，使用系统默认颜色

### Requirement: 快捷键说明页面
The system SHALL provide a keyboard shortcuts reference page accessible from a `?` button on the about page.

#### Scenario: 打开快捷键页面
- **WHEN** 用户点击关于页面中的 `?` 按钮
- **THEN** 显示快捷键说明视图，列出所有可用快捷键及其功能说明

#### Scenario: 关闭快捷键页面
- **WHEN** 用户点击关闭按钮或按 Escape
- **THEN** 关闭快捷键说明视图，返回之前页面

### Requirement: 快捷键内容
The shortcuts page SHALL include:

| 快捷键 | 功能 |
|--------|------|
| Ctrl + N | 新建笔记 |
| Ctrl + F | 聚焦搜索框 |
| PgUp | 上翻一页 |
| PgDn | 下翻一页 |
| Ctrl + Home | 回到顶部 |
| Ctrl + End | 滚到底部 |
| Escape | 关闭弹窗/编辑器 |
| Enter（下拉菜单中）| 选中当前项 |
| ↑/↓（下拉菜单中）| 上下导航 |

