# 批量管理标签 Spec

## Why

批量模式下目前只有「批量删除」，无法对选中的多条笔记统一添加或移除标签。每次需要逐条编辑，效率低。

## What Changes

- 批量操作栏新增「批量添加标签」和「批量移除标签」两个按钮
- 点击后弹出标签选择弹窗，展示所有可用标签，选中后执行批量操作
- 后端新增 `BatchRemoveTagFromNotes` 方法（`BatchAddTagToNotes` 已有但前端未使用）
- 操作完成后刷新笔记列表并显示通知

## Impact

- Affected specs: batch management, tag management
- Affected code: `app.go`, `tag_service.go`, `index.html`, `main.js`, `style.css`

## ADDED Requirements

### Requirement: 批量移除标签后端方法
The system SHALL provide a backend method to batch remove a tag from multiple notes.

#### Scenario: 成功批量移除标签
- **WHEN** 用户选中多条笔记后执行批量移除标签
- **THEN** 后端遍历所有指定笔记，移除指定的标签
- **THEN** 不存在的笔记自动跳过，不报错

### Requirement: 批量标签选择弹窗
The system SHALL provide a modal dialog for selecting a tag during batch operations.

#### Scenario: 弹窗交互
- **WHEN** 用户点击「批量添加标签」或「批量移除标签」
- **THEN** 弹出居中弹窗，展示所有可用标签供点击选择
- **THEN** 选中一个标签后立即执行操作并关闭弹窗
- **THEN** 无可用标签时显示「暂无标签」提示
- **THEN** 点击弹窗外区域可关闭弹窗

## MODIFIED Requirements

### Requirement: 批量操作栏
- 批量操作栏新增两个按钮：「批量添加标签」和「批量移除标签」
- 按钮样式与现有「批量删除」一致，使用 `.batch-btn` 类
- 点击对应按钮触发标签选择弹窗
