# GitHub 风格 Alert 引用块 Spec

## Why

使笔记预览模式和 CM6 纯文本编辑器能够正确渲染和高亮显示 `> [!NOTE]`、`> [!TIP]`、`> [!IMPORTANT]`、`> [!WARNING]`、`> [!CAUTION]` 五种 GitHub 风格 Alert 引用块，提升 Markdown 扩展语法的支持度。

## What Changes

- **预览模式（marked）**：安装 `marked-alert` 包，在主线程和 Web Worker 中注册
- **CSS 样式**：新增 alert 类样式，利用已有的语义色 CSS 变量（`--info`、`--warning`、`--error`、`--success`）
- 覆盖 3 个渲染区域：笔记预览、AI 消息、笔记引用预览
- **CM6 编辑器**：不做特殊处理，Alert 语法以普通 blockquote 样式显示

## Impact

- Affected specs: markdown rendering, code editor
- Affected code:
  - `frontend/package.json` — 新增 `marked-alert` 依赖
  - `frontend/src/main.js` — 注册 marked 扩展
  - `frontend/src/js/preview-worker.js` — 注册 marked 扩展
  - `frontend/src/css/components/editor.css` — alert 样式
  - `frontend/src/css/components/ai-chat.css` — alert 样式
  - `frontend/src/css/components/md-reference.css` — alert 样式

## ADDED Requirements

### Requirement: 预览模式渲染 Alert

The system SHALL render `> [!NOTE]`、`> [!TIP]`、`> [!IMPORTANT]`、`> [!WARNING]`、`> [!CAUTION]` 五种语法为带图标和语义色边框的 Alert 块。

#### Scenario: 标准 Alert 渲染

- **WHEN** 用户编写 `> [!NOTE]\n> 普通提示信息`
- **THEN** 预览中显示带 `--info` 色左边框和背景的 Alert 块，包含 NOTE 标签

### Requirement: 多区域覆盖

The system SHALL 在笔记预览、AI 消息、笔记引用预览三个区域都支持 Alert 渲染。

#### Scenario: 多区域渲染

- **WHEN** 笔记内容、AI 回复、引用预览中包含 Alert 语法
- **THEN** 三个区域均正确渲染为 Alert 样式