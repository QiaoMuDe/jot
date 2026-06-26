# 工具栏开关设置 Spec

## Why

Markdown 格式化工具栏对不熟悉 MD 语法的用户友好，但熟悉 MD 的用户会觉得它占用了编辑区垂直空间（36px）。之前已为了最大化内容区域压缩了 topbar、padding、scrollbar gap 等，工具栏的固定存在与此方向矛盾。需要一个设置开关让用户自行决定是否显示工具栏。

## What Changes

- 在设置页「编辑器选项」区域新增一个 toggle：「Markdown 笔记显示格式化工具栏」
- 存储键 `md_toolbar_enabled`，默认 `true`（工具栏默认显示，不影响现有用户）
- 打开笔记编辑器时，工具栏的显隐逻辑增加对设置的判断：仅当 `noteType === 'markdown'` AND 编辑模式 AND 设置启用时显示
- 纯文本笔记始终不显示工具栏（不受此设置影响）

## Impact

- **Affected specs**: 编辑器系统、工具栏系统、设置系统
- **Affected code**: `frontend/index.html`（设置页 HTML）、`frontend/src/main.js`（显隐逻辑 + 读取设置）

## ADDED Requirements

### Requirement: 设置项 UI

The system SHALL 在设置页「编辑器选项」区域新增一个 toggle 开关。

#### Scenario: 默认位置
- **WHEN** 用户打开设置页
- **THEN**「编辑器选项」区域中，「纯文本编辑器启用 Markdown 语法高亮」下方出现一行新设置
- **THEN** 设置项标签文字为「Markdown 笔记显示格式化工具栏」
- **THEN** 说明文字为「开启后，Markdown 笔记编辑时在顶部显示格式化工具栏（加粗、标题、链接等）」
- **THEN** toggle 的 id 为 `mdToolbarToggle`

#### Scenario: 默认值
- **WHEN** 用户首次使用（localStorage 中无 `md_toolbar_enabled` 记录）
- **THEN** toggle 默认为开启状态
- **THEN** 实际存储值为 `true`

### Requirement: 存储与读取

The system SHALL 使用 localStorage 存储和读取工具栏开关状态。

#### Scenario: 切换设置
- **WHEN** 用户切换 `mdToolbarToggle` 的开关状态
- **THEN** 立即将当前状态（`true`/`false`）写入 localStorage 键 `md_toolbar_enabled`

#### Scenario: 读取设置
- **WHEN** 其他逻辑需要读取工具栏开关状态
- **THEN** 调用 `localStorage.getItem('md_toolbar_enabled')`，值为 `null`（未设置）时视为 `true`

### Requirement: 工具栏显隐逻辑修改

The system SHALL 修改工具栏显隐逻辑，增加设置项判断。

#### Scenario: 打开编辑器时
- **WHEN** `populateEditor()` 执行，且 `state.noteType === 'markdown'`，且为非只读模式
- **THEN** 额外检查 `md_toolbar_enabled` 设置：启用则显示工具栏，禁用则隐藏
- **WHEN** `state.noteType === 'text'`
- **THEN** 始终隐藏工具栏（与现有行为一致，不受此设置影响）

#### Scenario: 类型切换时
- **WHEN** 用户点击类型切换按钮从纯文本切换为 Markdown
- **THEN** 同样检查 `md_toolbar_enabled` 设置决定工具栏显隐
- **WHEN** 从 Markdown 切换为纯文本
- **THEN** 始终隐藏工具栏

### Requirement: 设置变更即时生效

The system SHALL 在用户切换设置时，如果编辑器当前处于打开状态，即时更新工具栏显隐。

#### Scenario: 编辑中切换设置
- **WHEN** 编辑器处于打开状态（编辑 Markdown 笔记），用户进入设置页关闭该开关
- **THEN** 工具栏立即隐藏
- **WHEN** 用户重新开启开关
- **THEN** 工具栏立即显示

## MODIFIED Requirements

### Requirement: 显隐控制（原 spec 修改）

原逻辑：
```
els.editorToolbar.style.display = (state.noteType === 'markdown' && !isReadOnly) ? 'flex' : 'none';
```

修改为：
```
const toolbarEnabled = localStorage.getItem('md_toolbar_enabled') !== 'false';
els.editorToolbar.style.display = (state.noteType === 'markdown' && !isReadOnly && toolbarEnabled) ? 'flex' : 'none';
```

类型切换处的逻辑同理修改。
