# 查找替换功能 Spec

## Why
笔记编辑器缺少文本查找和替换功能，用户在编辑长内容时需要手动定位关键字，效率低下。

## What Changes
- 在编辑器底部（footer 上方）新增查找条和替换条 UI
- 新建笔记、编辑笔记页面支持查找和替换；查看（只读）页面仅支持查找
- Ctrl+F 唤醒查找条，Ctrl+H 唤醒查找+替换条
- 替换功能仅纯文本模式下可用，Markdown 预览模式下提示切换到纯文本
- 查找后自动高亮所有匹配，用 `[` 和 `]` 键导航匹配项
- 查找/替换条的输入焦点保持稳定，不跳回内容区

## Impact
- Affected specs: add-editor-and-undo-features, add-md-editor
- Affected code: `frontend/index.html`, `frontend/src/style.css`, `frontend/src/app.css`, `frontend/src/main.js`

## Requirements

### Requirement: 查找替换 UI 组件
编辑器底部新增查找条和替换条，位于 `.editor-footer` 上方。

#### Scenario: 打开查找条
- **WHEN** 编辑器打开时用户按下 Ctrl+F
- **THEN** 在编辑器底部 footer 上方显示查找条
- **AND** 查找条包含：搜索输入框、匹配计数、上一个/下一个导航按钮、关闭按钮
- **AND** 输入框自动获得焦点

#### Scenario: 打开查找+替换条
- **WHEN** 编辑器打开时用户按下 Ctrl+H
- **THEN** 在编辑器底部 footer 上方同时显示查找条和替换条
- **AND** 查找条输入框自动获得焦点
- **AND** 若查找条已显示（通过 Ctrl+F），则 Ctrl+H 追加显示替换条

#### Scenario: 替换功能限制
- **WHEN** 用户在 Markdown 预览模式（`data-mode="preview"`）下尝试打开替换条
- **THEN** 显示短提示"请先切换到纯文本模式再进行替换操作"
- **AND** 不显示替换输入框

### Requirement: 查找行为
查找条输入后实时高亮内容区所有匹配项。

#### Scenario: 实时高亮
- **WHEN** 用户在查找输入框中输入内容
- **THEN** 内容区（textarea 或渲染区）中所有匹配文本被高亮标记
- **AND** 匹配计数实时更新（"3/12" 格式，当前/总数）

#### Scenario: 导航匹配项
- **WHEN** 用户按下 `[` 键
- **THEN** 跳转到上一个匹配项并滚动到可视区域
- **WHEN** 用户按下 `]` 键
- **THEN** 跳转到下一个匹配项并滚动到可视区域
- **AND** 当前激活的匹配项使用不同颜色高亮以区分

#### Scenario: 输入焦点稳定
- **WHEN** 用户在查找/替换输入框中输入
- **THEN** 焦点始终保持在查找/替换输入框中
- **AND** 输入的光标不跳回编辑器内容区

### Requirement: 替换行为
替换仅在新建和编辑页面的纯文本模式下可用。

#### Scenario: 替换单个
- **WHEN** 用户点击替换按钮
- **THEN** 替换当前激活的匹配项

#### Scenario: 替换全部
- **WHEN** 用户点击"全部替换"按钮
- **THEN** 替换内容中所有匹配项
- **AND** 更新匹配计数和导航状态

#### Scenario: 纯文本模式限制
- **WHEN** 编辑器处于 Markdown 预览模式（`data-mode="preview"`）
- **THEN** 替换按钮禁用
- **AND** 鼠标悬停时显示提示"请切换到纯文本模式"
- **AND** 若用户尝试使用替换快捷键或按钮，显示提示消息

### Requirement: 关闭查找/替换条
用户可关闭查找和替换条。

#### Scenario: 关闭
- **WHEN** 用户点击查找条或替换条的关闭按钮
- **THEN** 对应的条隐藏
- **WHEN** 用户关闭编辑器
- **THEN** 查找条和替换条自动关闭并重置状态

### Requirement: 快捷键冲突处理
当前 Ctrl+F 已绑定到顶部搜索栏聚焦，需要根据上下文区分。

#### Scenario: 编辑器内外 Ctrl+F
- **WHEN** 编辑器打开时（`.viewEditor.active`）按下 Ctrl+F
- **THEN** 触发编辑器内查找条
- **WHEN** 编辑器关闭时按下 Ctrl+F
- **THEN** 保持现有行为：聚焦顶部搜索栏

## 非功能需求
- 查找区分大小写：默认不区分（可选 toggle）
- 性能：内容较长时查找应无显著卡顿
- 样式：查找/替换条使用与编辑器风格一致的 CSS 变量
