# Markdown 渲染查看 Spec

## Why
笔记查看页目前只显示纯文本（`<textarea readonly>`），无法展示标题层级、列表、代码块等格式。引入 Markdown 渲染后，用户在编辑时可以写 Markdown，查看时直接看到格式化后的 HTML，提升阅读体验。

## What Changes
- **查看模式**：将 readonly textarea 替换为 `<div class="md-rendered">`，用 marked 渲染 Markdown 内容为 HTML
- **编辑模式**：textarea 不变，用户照常写 Markdown 语法
- **代码高亮**：查看模式中的代码块用 highlight.js 自动着色
- **样式**：新增 `.md-rendered` 容器样式（标题/列表/代码块/引用/表格）
- **依赖**：通过 npm 安装 `marked` + `highlight.js`，Vite 打包时内联，不依赖 CDN

## Impact
- Affected specs: 查看笔记流程
- Affected code:
  - `frontend/package.json` — 新增 marked + highlight.js 依赖
  - `frontend/src/main.js` — `openEditor()` 查看分支渲染逻辑
  - `frontend/src/style.css` — 新增 `.md-rendered` 样式

## ADDED Requirements

### Requirement: 查看页 Markdown 渲染
The system SHALL render note content as formatted Markdown HTML in view mode.

#### Scenario: 查看含 Markdown 的笔记
- **WHEN** 用户打开查看模式（左击卡片或右键菜单「查看」）
- **THEN** 内容区显示格式化的 HTML（标题/加粗/列表/代码块等），而非纯文本
- **AND** 代码块使用 highlight.js 着色

#### Scenario: 编辑模式不变
- **WHEN** 用户编辑笔记
- **THEN** textarea 保持纯文本输入，用户写 Markdown 语法

#### Scenario: 空内容展示
- **WHEN** 笔记内容为空
- **THEN** 查看模式显示占位文字「暂无内容」

#### Scenario: 切换查看/编辑不丢失内容
- **WHEN** 用户从查看模式切换到编辑模式
- **THEN** textarea 内容与查看时渲染的内容一致（Markdown 源码）

## MODIFIED Requirements
暂无

## REMOVED Requirements
暂无
