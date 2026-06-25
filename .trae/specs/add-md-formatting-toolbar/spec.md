# Markdown 格式化工具栏 Spec

## Why

Markdown 笔记编辑时，用户需要手动输入 `**粗体**`、`*斜体*`、`## 标题` 等语法符号，效率低且对不熟悉 MD 语法的用户不友好。在 CM6 编辑区顶部增加一个格式化工具栏，点击即可应用 MD 语法，同时配套 Ctrl+B/I/U 等快捷键。

## What Changes

- 在 `editor-panes` 顶部新增 `.editor-toolbar` 元素，只有笔记类型为 `markdown` 且处于编辑模式时显示
- 工具栏包含 10 个按钮 + 1 个标题下拉选择器（H1-H6）：加粗 / 斜体 / 删除线 / 行内代码 / 标题 H1-H6 / 链接 / 图片 / 无序列表 / 引用 / 分割线
- 配? 3 个快捷键：`Ctrl+B` 加粗、`Ctrl+I` 斜体、`Ctrl+U` 删除线
- 点击工具栏按钮不丢失 CM6 焦点

## Impact

- **Affected specs**: 编辑器系统、CM6 集成
- **Affected code**: `frontend/index.html`（HTML 结构）、`frontend/src/style.css`（工具栏样式）、`frontend/src/main.js`（格式化函数 + 事件绑定 + 快捷键 + 显隐控制）

## ADDED Requirements

### Requirement: 工具栏 HTML 结构

The system SHALL 在 `editor-panes` 内第一个子元素位置插入 `.editor-toolbar` 容器。

#### Scenario: 默认结构
- **WHEN** markdown 笔记编辑器打开
- **THEN** `editor-panes` 内存在 `<div class="editor-toolbar" id="editorToolbar">` 作为第一个子元素

#### Scenario: 工具栏按钮列表
- **WHEN** 工具栏渲染
- **THEN** 包含以下元素，按顺序排列：
  - 加粗（B 图标，快捷键 Ctrl+B）
  - 斜体（I 图标，快捷键 Ctrl+I）
  - 删除线（S 图标，快捷键 Ctrl+U）
  - 行内代码（`</>` 图标）
  - 分隔符（视觉竖线）
  - 标题选择器（下拉菜单，显示 H 标签，点击展开 H1-H6 选项）
  - 分隔符
  - 链接（链环图标）
  - 图片（图片图标）
  - 无序列表（列表图标）
  - 引用（引号图标）
  - 分割线（横线图标）

#### Scenario: 标题下拉选择器设计
- **WHEN** 用户点击标题按钮
- **THEN** 展开一个浮动下拉面板，包含 6 个选项：标题 1、标题 2、标题 3、标题 4、标题 5、标题 6
- **WHEN** 用户点击某个标题选项
- **THEN** 应用对应级别的标题格式，下拉面板关闭
- **WHEN** 用户点击下拉面板外部
- **THEN** 下拉面板关闭

### Requirement: 格式化功能

The system SHALL 实现以下 11 个格式化操作，每个操作适配"有选中文本"和"无选中文本"两种场景。

| 操作 | 有选中文本 | 无选中文本 |
|------|-----------|-----------|
| 加粗 | `**选中文本**`，选中不变 | 插入 `****`，光标居中 |
| 斜体 | `*选中文本*`，选中不变 | 插入 `**`，光标居中 |
| 删除线 | `~~选中文本~~`，选中不变 | 插入 `~~~~`，光标居中 |
| 行内代码 | `` `选中文本` ``，选中不变 | 插入 ````，光标居中 |
| 标题 H1-H6 | 行首 `#~###### ` + 选中文本 | 行首插入 `#~###### ` |
| 链接 | `[选中文本](url)`，弹窗输入 URL | 插入 `[链接文字](url)`，弹窗输入 |
| 图片 | — | 插入 `![](url)`，弹窗输入 URL |
| 无序列表 | 行首 `- ` + 选中文本 | 行首插入 `- ` |
| 引用 | 行首 `> ` + 选中文本 | 行首插入 `> ` |
| 分割线 | — | 插入 `\n---\n` |

#### Scenario: 加粗操作
- **WHEN** 用户点击加粗按钮，编辑器选中了"你好"
- **THEN** 文本变为 `**你好**`，选中区域仍为"你好"
- **WHEN** 用户点击加粗按钮，编辑器无选中文本
- **THEN** 插入 `****`，光标定位在 4 个星号中间

#### Scenario: 标题操作（H1-H6）
- **WHEN** 用户从标题下拉中选择"标题 2"，当前光标在第 3 行行中
- **THEN** 在第 3 行行首插入 `## `

#### Scenario: 链接操作（需弹窗）
- **WHEN** 用户点击链接按钮，编辑器选中了"示例"
- **THEN** 弹出输入框，提示"输入链接 URL"，用户输入 `https://example.com`
- **THEN** 文本变为 `[示例](https://example.com)`，选中区域仍为"示例"

### Requirement: 快捷键

The system SHALL 注册以下 CM6 keymap 快捷键：

- `Ctrl+B`：加粗操作（同加粗按钮）
- `Ctrl+I`：斜体操作（同斜体按钮）
- `Ctrl+U`：删除线操作（同删除线按钮）

#### Scenario: 快捷键加粗
- **WHEN** 编辑器聚焦，用户按下 Ctrl+B
- **THEN** 执行加粗操作，行为与点击加粗按钮一致

### Requirement: 显隐控制

The system SHALL 根据笔记类型和编辑模式控制工具栏显隐。

#### Scenario: 纯文本笔记隐藏工具栏
- **WHEN** `state.noteType === 'text'`
- **THEN** `.editor-toolbar` 显示为 `display: none`
- **WHEN** 用户通过类型切换按钮将笔记切换为 `markdown`
- **THEN** `.editor-toolbar` 显示为 `display: flex`

#### Scenario: 预览模式隐藏工具栏
- **WHEN** 编辑模式切换到 `data-mode="preview"`
- **THEN** `.editor-toolbar` 显示为 `display: none`
- **WHEN** 切回 `data-mode="edit"`
- **THEN** `.editor-toolbar` 恢复显示

### Requirement: 焦点管理

The system SHALL 在点击工具栏按钮后不丢失 CM6 焦点。

#### Scenario: 点击按钮保持焦点
- **WHEN** 用户点击工具栏任意按钮
- **THEN** 格式化操作执行完毕后自动调用 `cmEditor.focus()`

#### Scenario: 标题下拉不丢失焦点
- **WHEN** 用户点击标题下拉选择器中的选项
- **THEN** 标题格式化执行完毕后自动调用 `cmEditor.focus()`

### Requirement: 工具栏样式与设计

The system SHALL 提供与当前主题系统一致的工具栏样式。

#### Scenario: 视觉设计
- **WHEN** 工具栏渲染
- **THEN** 满足以下设计要求：
  - 高度 36px，背景 `var(--bg-secondary)`，下边框 1px `var(--border-color)`
  - 按钮尺寸 28×28px，圆角 6px，图标/文字居中
  - 按钮默认状态：颜色 `var(--text-secondary)`，背景透明
  - 按钮 hover 态：颜色 `var(--text-primary)`，背景 `var(--hover-bg)`
  - 按钮 active 态：背景 `var(--accent-light)`，颜色 `var(--accent)`
  - 分隔符：1px 竖线，颜色 `var(--border-color)`，高度 16px，margin 0 2px
  - 所有按钮使用现有 CSS 变量，6 主题自动适配
  - 按钮之间无间隙（gap: 0），整体居左对齐 padding-left: 4px
  - 标题选择器按钮显示 H 标签，点击展开下拉面板

#### Scenario: 标题下拉面板样式
- **WHEN** 标题下拉面板展开
- **THEN** 满足以下设计要求：
  - 浮动定位在工具栏下方，与标题按钮左对齐
  - 宽度 140px，背景 `var(--bg-primary)`，圆角 8px，阴影 `var(--shadow-lg)`
  - 边框 1px `var(--border-color)`
  - 每个选项高度 32px，padding 0 12px，hover 背景 `var(--hover-bg)`
  - 选项文字显示格式：`H1  标题 1`（H 标签 + 中文描述）
  - 选项 hover 时左侧显示 3px accent 竖条指示
  - 点击选项后面板立即关闭

#### Scenario: 全屏模式
- **WHEN** 编辑器进入全屏模式
- **THEN** 工具栏位置不变，样式一致

#### Scenario: prefers-reduced-motion
- **WHEN** 用户开启 `prefers-reduced-motion`
- **THEN** 工具栏按钮和下拉面板无过渡动画（`transition: none`）
