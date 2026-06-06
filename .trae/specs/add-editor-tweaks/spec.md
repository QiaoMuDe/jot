# 编辑器快捷与行为优化 Spec

## Why
编辑器交互有 3 个体验问题：切换编辑/预览模式需要点击底部按钮，不够快捷；新建笔记时标题为空需要手动输入；置顶操作会污染笔记的更新时间。

## What Changes
- 编辑器打开时 `Ctrl+L` 切换纯文本/预览模式
- 新建笔记默认标题取当前日期时间 `2026-06-06 14:30`
- 后端置顶 API 不再更新 `updated_at` 字段

## Impact
- Affected specs: 编辑器、快捷键、笔记操作
- Affected code: `main.js`、`app.go`（后端）

## ADDED Requirements

### Requirement: Ctrl+L 切换模式
编辑器（查看/新建/编辑）打开时，按 `Ctrl+L` 切换纯文本/预览模式，与点击底部模式切换按钮行为一致。

#### Scenario: 成功切换
- **WHEN** 编辑器打开，当前为纯文本模式
- **AND** 用户按下 `Ctrl+L`
- **THEN** 切换到预览模式

#### Scenario: 再次切换
- **WHEN** 当前为预览模式
- **AND** 用户按下 `Ctrl+L`
- **THEN** 切换到纯文本模式

### Requirement: 新建笔记默认时间标题
新建笔记时自动生成标题为当前日期时间 + 表情，格式 `YYYY-MM-DD HH:mm ☺️`。

#### Scenario: 新建笔记
- **WHEN** 用户点击新建笔记按钮
- **THEN** 标题输入框自动填入当日日期时间 + 表情，如 `2026-06-06 14:30 ☺️`

### Requirement: 置顶不更新修改时间
点击置顶或取消置顶时，笔记的 `updated_at` 字段保持不变。

#### Scenario: 置顶笔记
- **WHEN** 用户点击笔记的置顶按钮
- **THEN** 笔记的 `updated_at` 不发生改变

#### Scenario: 取消置顶
- **WHEN** 用户点击已置顶笔记的置顶按钮
- **THEN** 笔记的 `updated_at` 不发生改变
