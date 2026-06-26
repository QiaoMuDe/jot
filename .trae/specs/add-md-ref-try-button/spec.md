# MD 语法页面「打开编辑器试试」按钮 Spec

## Why
学习 MD 语法的用户看完语法示例后，无法一键将示例内容放入编辑器练习。需要手动复制源码 → 回到首页 → 新建笔记 → 粘贴，路径太长。

## What Changes
- 每张 MD 语法卡片的脚注下方新增一个「打开编辑器试试」按钮
- 点击后切换到首页并打开编辑器，预填标题和源码内容，用户可直接编辑保存

## Impact
- Affected specs: MD 语法视图交互、笔记创建流程
- Affected code: `index.html`、`main.js`、`style.css`

## ADDED Requirements
### Requirement: 「打开编辑器试试」按钮
The system SHALL provide a button on each MD 语法 card that creates a new note with pre-filled content.

#### Scenario: 点击按钮 → 编辑器打开
- **WHEN** 用户在 MD 语法页面点击卡片的「打开编辑器试试」按钮
- **THEN** 页面切换到首页/笔记网格视图
- **AND** 编辑器为新笔记模式打开
- **AND** 标题格式设为 `[MD 语法] {badge文字}`（如 `[MD 语法] 标题 (Headings)`）
- **AND** 内容预填为该卡片 `<script type="text/plain" class="md-ref-source">` 中的源码文本
- **AND** 笔记类型设为 `markdown`

#### Scenario: 内容说明
- **WHEN** 源码内容中包含 `&gt;` 等 HTML 实体
- **THEN** 内容应解码为原始字符后再填入编辑器

## MODIFIED Requirements
无

## REMOVED Requirements
无
