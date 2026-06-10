# 预览模式滚动条修复 Spec

## Why
查看/新建/编辑页面的 Markdown 预览区域（`.md-rendered`）隐藏了滚动条，用户无法通过拖动滚动条浏览长内容，只能使用鼠标滚轮或键盘，交互方式受限。

## What Changes
- 移除 `.md-rendered` 上隐藏滚动条的 CSS 规则，恢复滚动条显示
- 为预览区域滚动条应用与应用主内容区一致的精致微调样式（6px 半透明+主题色适配）

## Impact
- Affected code: `frontend/src/style.css` — `.md-rendered` 滚动条相关 CSS

## MODIFIED Requirements

### Requirement: 预览区域滚动条恢复

#### Scenario: 查看模式切换到 Markdown 笔记
- **WHEN** 用户打开 markdown 类型笔记，自动切换到预览模式
- **THEN** 预览内容可垂直滚动，滚动条可见、可拖动

#### Scenario: 编辑模式切换到预览模式
- **WHEN** 用户在编辑器内切换到预览模式
- **THEN** 预览内容可垂直滚动，滚动条可见、可拖动

#### Scenario: 新建笔记切换预览
- **WHEN** 用户新建 markdown 笔记，切换到预览模式
- **THEN** 预览内容可垂直滚动，滚动条可见、可拖动
