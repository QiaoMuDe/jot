# 编辑器功能操作菜单 Spec

## Why
笔记编辑器的头部操作栏目前只有目录、类型切换、编辑/查看、全屏、关闭等基础按钮，缺少对编辑内容的加工能力。用户希望新增一个「功能操作菜单」按钮，点击后弹出下拉菜单，可对**选中内容**（未选中则对**全部内容**）执行格式化、编码解码等操作。该菜单是长期规划的功能入口：将来会承载**格式化**（JSON/XML/CSS…）、**编码解码**（Base64/URL…）、**AI 操作**等分组，首期只实现 **JSON 格式化** 与 **JSON 压缩** 两个操作，但菜单结构必须为后续分组扩展预留好空间。

> 注意：该按钮不命名为「更多菜单」，它是面向编辑器内容的**操作菜单**，与顶栏「更多」菜单（全局功能入口）定位不同。

## What Changes
- 在 `frontend/index.html` 的编辑器头部操作栏（`.editor-header-actions`）中新增一个操作菜单按钮 `editorActionsBtn`，点击弹出下拉菜单
- 菜单采用**分组结构**：分组标签（如「格式化」）+ 分隔线 + 操作项，首期包含两组：
  - 格式化：JSON 格式化、JSON 压缩
  - （预留分组标签占位：编码解码、AI，首期不渲染或渲染为空分组——决定：首期**仅渲染「格式化」分组**，预留结构不显示空分组）
- 新增 `frontend/src/js/editor-actions.js`（或等价模块），采用**配置驱动的操作注册表** `EDITOR_ACTIONS`，每个操作定义为 `{ id, group, label, icon, handler }`，后续新增操作只需追加条目
- 操作作用范围规则：编辑器存在非空选中文本 → 处理选中文本；否则处理全部内容；处理结果通过 CodeMirror 6 `dispatch` 写回，**支持 Ctrl+Z 撤销**
- 查看（只读）模式下按钮隐藏（复用 `editorTypeToggle` 的显隐逻辑）；预览模式下点击操作自动切回编辑模式
- JSON 解析失败时通过现有通知系统 `nm.show()` 提示，不修改内容

## Impact
- Affected specs: 无既有 spec 与本功能重叠
- Affected code:
  - `frontend/index.html` — 新增按钮 + 菜单 DOM（`editor-header-actions` 内）
  - `frontend/src/css/components/editor.css` — 菜单定位样式（右对齐弹出）、分组标签样式
  - `frontend/src/main.js` — 初始化函数调用、只读/预览模式联动
  - `frontend/src/js/editor-actions.js` — 新文件：操作注册表 + 操作执行引擎（JSON 格式化/压缩为纯函数，无后端依赖）

## ADDED Requirements

### Requirement: 编辑器操作菜单入口按钮
系统 SHALL 在编辑器头部操作栏提供「操作菜单」按钮，点击后弹出下拉菜单；查看（只读）模式下按钮隐藏，编辑与新建模式下可见。

#### Scenario: 打开操作菜单
- **WHEN** 用户在编辑或新建模式点击操作菜单按钮
- **THEN** 弹出下拉菜单，菜单在按钮下方右对齐展开
- **AND** 菜单包含「格式化」分组下的「JSON 格式化」「JSON 压缩」两个操作项
- **AND** 点击菜单外部区域或按 Esc 时菜单关闭

#### Scenario: 查看模式隐藏
- **WHEN** 编辑器处于查看（只读）模式
- **THEN** 操作菜单按钮不显示
- **AND** 切换到编辑模式后按钮重新显示

### Requirement: 操作作用范围（选中优先，否则全文）
系统 SHALL 在用户选择某个操作时，优先处理编辑器中的选中文本；若无选中内容则处理全文。

#### Scenario: 有选中内容
- **WHEN** 用户选中部分文本后选择「JSON 格式化」
- **THEN** 仅选中的文本被格式化替换，其余内容不变
- **AND** 操作可 Ctrl+Z 撤销，撤销后恢复原文本

#### Scenario: 无选中内容
- **WHEN** 用户未选中任何文本时选择「JSON 压缩」
- **THEN** 整个文档内容被压缩替换

### Requirement: JSON 格式化
系统 SHALL 将 JSON 文本格式化为带 2 空格缩进、换行的可读形式。

#### Scenario: 合法 JSON
- **WHEN** 用户对合法 JSON（对象或数组）执行「JSON 格式化」
- **THEN** 文本被替换为 `JSON.stringify(parsed, null, 2)` 的结果

#### Scenario: 非法 JSON
- **WHEN** 用户对无法解析的内容执行「JSON 格式化」
- **THEN** 编辑器内容不被修改
- **AND** 通过通知提示「不是合法的 JSON」

### Requirement: JSON 压缩
系统 SHALL 将 JSON 文本压缩为去除所有非必要空白的最小形式。

#### Scenario: 合法 JSON
- **WHEN** 用户对合法 JSON 执行「JSON 压缩」
- **THEN** 文本被替换为 `JSON.stringify(parsed)` 的结果（无缩进、无多余空白）

#### Scenario: 非法 JSON
- **WHEN** 用户对无法解析的内容执行「JSON 压缩」
- **THEN** 编辑器内容不被修改
- **AND** 通过通知提示「不是合法的 JSON」

### Requirement: 预览模式联动
系统 SHALL 在预览模式下点击操作菜单项时，自动先切回编辑模式再执行操作。

#### Scenario: 预览模式下操作
- **WHEN** 用户处于 Markdown 预览模式并选择任意操作
- **THEN** 编辑器自动切回纯文本编辑模式，操作作用于编辑内容

### Requirement: 操作注册表可扩展
系统 SHALL 以配置数组形式集中管理操作项，分组与操作条目与 UI 解耦，为后续「编码解码」「AI 操作」分组扩展预留结构。

#### Scenario: 新增操作
- **WHEN** 开发者向 `EDITOR_ACTIONS` 数组追加一个 `{ group, label, handler }` 条目
- **THEN** 无需修改菜单渲染与事件绑定代码，新操作自动出现在对应分组下
- **AND** 空分组（无任何条目）不渲染

## MODIFIED Requirements
无（首期不修改既有功能，仅新增按钮与菜单；`editor-header-actions` 的 flex 布局已支持追加按钮）。

## REMOVED Requirements
无。
