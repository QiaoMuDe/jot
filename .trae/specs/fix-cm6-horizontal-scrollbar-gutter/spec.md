# 修复 CM6 水平滚动条遮挡行号 Spec

## Why
在编辑页和查看页的纯文本编辑器（CodeMirror 6）中，当某行内容过长触发水平滚动条时，滚动条会覆盖左侧行号（gutter）区域，影响阅读与交互。

## What Changes
- 修改 `frontend/src/main.js` 中 CM6 主题的 `.cm-gutters` 样式，将背景从透明改为与编辑器卡片背景一致
- 在 `frontend/src/style.css` 中补充/覆盖 `.cm-gutters` 背景色与 `z-index`，确保行号区域位于水平滚动条之上
- 保留 gutter 的 sticky/fixed 行为与垂直滚动体验

## Impact
- Affected code: `frontend/src/main.js`, `frontend/src/style.css`
- Affected UI: 编辑页纯文本模式、查看页纯文本模式、深浅色主题

## ADDED Requirements
### Requirement: 行号区域背景遮罩
The system SHALL 为 CodeMirror 6 行号 gutter 设置与编辑器卡片背景一致的不透明背景，并使其层级高于水平滚动条。

#### Scenario: 纯文本编辑器出现水平滚动条
- **WHEN** 笔记内容包含超长行并触发水平滚动条
- **THEN** 水平滚动条仅出现在内容区域下方，不覆盖左侧行号区域

## MODIFIED Requirements
### Requirement: 纯文本编辑器主题样式
现有的 CM6 主题与全局 CSS SHALL 更新，使 `.cm-gutters` 背景不再透明，同时不影响垂直滚动与行号的 sticky 行为。

## REMOVED Requirements
无。
