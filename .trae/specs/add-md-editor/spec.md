# 三种模式 MD 编辑器 Spec

## Why

编辑笔记时只有纯文本框，无法看到 Markdown 渲染效果。用户需在保存后回到查看模式才能确认格式是否正确。增加编辑时的模式切换，让用户在**纯文本编写**、**边写边看**和**预览效果**三种状态下自由切换，提升写作体验。

## What Changes

- **编辑器弹窗标题栏**：新增 3 个模式切换按钮（纯文本/分栏/预览），活跃状态高亮
- **纯文本模式**：textarea 占满编辑区，预览区隐藏（与当前编辑模式一致）
- **分栏模式**：textarea 居左 50%、预览区居右 50%，各滚各的，不支持同步滚动
- **预览模式**：textarea 隐藏，预览区占满编辑区（与当前查看模式一致）
- **预览自动更新**：分栏和预览模式下，输入内容后 300ms 防抖自动渲染 Markdown
- **不增加新依赖**：Markdown 渲染复用已有的 `marked` + `highlight.js`
- **模式状态**：切换模式不丢失已输入内容，关闭弹窗不保存模式偏好

## Impact

- Affected specs: 现有 `add-md-rendering`（复用其 marked 渲染逻辑）
- Affected code: `frontend/index.html`、`frontend/src/style.css`、`frontend/src/main.js`
- 后端：无变更

## ADDED Requirements

### Requirement: 模式切换按钮组

The system SHALL provide 3 个模式切换按钮在编辑器弹窗标题栏右侧，关闭按钮左侧。

#### Scenario: 正常显示
- **WHEN** 打开编辑器弹窗
- **THEN** 标题栏显示「纯文本」「分栏」「预览」三个按钮，当前模式按钮高亮（accent 色下划线或填充）

#### Scenario: 模式切换
- **WHEN** 用户点击某个模式按钮
- **THEN** 切换编辑区布局，对应按钮高亮，其他按钮取消高亮

### Requirement: 纯文本模式

The system SHALL 在纯文本模式下只显示 textarea，隐藏预览区。

#### Scenario: 编辑行为
- **WHEN** 用户切换到纯文本模式
- **THEN** textarea 宽度 100%，高度占满编辑区，预览区 `display: none`

### Requirement: 分栏模式

The system SHALL 在分栏模式下并排显示 textarea 和预览区。

#### Scenario: 分栏布局
- **WHEN** 用户切换到分栏模式
- **THEN** textarea 和预览区各占 50% 宽度，中间有分隔线，各自独立滚动

#### Scenario: 自动渲染
- **WHEN** 用户在分栏模式下输入内容
- **THEN** textarea 触发 input 事件，300ms 防抖后调用 `marked.parse()` 更新预览区内容
- **AND** 代码块自动调用 `highlight.js` 着色

### Requirement: 预览模式

The system SHALL 在预览模式下只显示渲染后的 HTML，隐藏 textarea。

#### Scenario: 预览行为
- **WHEN** 用户切换到预览模式
- **THEN** 预览区宽度 100%，textarea `display: none`
- **AND** 预览区内容立即渲染（若已有内容则重新渲染）

### Requirement: 模式切换不丢失内容

The system SHALL 确保切换模式时 textarea 中的内容不丢失。

#### Scenario: 内容保留
- **WHEN** 用户在 textarea 中输入内容后切换模式
- **THEN** 再切回纯文本或分栏模式时，textarea 内容保持不变

## MODIFIED Requirements

无

## REMOVED Requirements

无
