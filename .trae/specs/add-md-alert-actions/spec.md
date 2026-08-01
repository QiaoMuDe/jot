# 添加 Alert 引用块插入操作项 Spec

## Why

在"操作按钮"的"MD 语法"菜单中，为 5 种 Alert 扩展语法（NOTE、TIP、IMPORTANT、WARNING、CAUTION）添加插入操作项，方便用户快速插入格式化后的 Alert 引用块模板。

## What Changes

- 在 `md-syntax.js` 的 `块元素` 子菜单中添加 5 个 Alert 插入操作项
- 每个操作项插入对应的 `<br>` + `> [!TYPE]\n> 提示内容` 模板

## Impact

- Affected code:
  - `frontend/src/js/editor-actions/md-syntax.js` — 新增 5 个操作项

## ADDED Requirements

### Requirement: 插入 Alert 引用块

The system SHALL 在"MD 语法 > 块元素"子菜单中提供 5 个 Alert 类型插入项。

#### Scenario: 插入 NOTE 引用块

- **WHEN** 用户点击"MD 语法 > 块元素 > 引用块: NOTE"
- **THEN** 在光标处插入 `\n> [!NOTE]\n> 提示信息\n`，并保持光标位于提示信息位置

#### Scenario: 插入 TIP 引用块

- **WHEN** 用户点击"MD 语法 > 块元素 > 引用块: TIP"
- **THEN** 在光标处插入 `\n> [!TIP]\n> 小技巧\n`

#### Scenario: 插入 IMPORTANT 引用块

- **WHEN** 用户点击"MD 语法 > 块元素 > 引用块: IMPORTANT"
- **THEN** 在光标处插入 `\n> [!IMPORTANT]\n> 重要提醒\n`

#### Scenario: 插入 WARNING 引用块

- **WHEN** 用户点击"MD 语法 > 块元素 > 引用块: WARNING"
- **THEN** 在光标处插入 `\n> [!WARNING]\n> 警告内容\n`

#### Scenario: 插入 CAUTION 引用块

- **WHEN** 用户点击"MD 语法 > 块元素 > 引用块: CAUTION"
- **THEN** 在光标处插入 `\n> [!CAUTION]\n> 小心操作\n`

#### Scenario: 选中文本时包裹

- **WHEN** 用户选中一段文本后点击某个 Alert 插入项
- **THEN** 将选中文本作为 Alert 的提示内容，插入 `\n> [!TYPE]\n> 选中文本\n`