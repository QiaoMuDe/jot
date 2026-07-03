# 引用笔记选择器键盘支持 + 笔记页面屏蔽 Ctrl+数字快捷键 Spec

## Why
- 引用笔记选择器目前只能通过鼠标操作（点选/点击按钮），缺少 ESC 返回和 Enter 确认的键盘快捷操作，影响操作效率。
- 查看/编辑/新建笔记时，Ctrl+数字组合键会误触全局导航快捷键（如 Ctrl+6 切换到设置页），应在此类页面中屏蔽。

## What Changes
- **新增**: 引用笔记选择器浮层 ESC 键→关闭浮层并停留在 AI 聊天界面（不触发全局 ESC 导航到首页）
- **新增**: 引用笔记选择器浮层 Enter 键→触发确认引用按钮
- **修改**: 全局 `handleKeyboardNavigation` 中 ESC 处理逻辑：浮层打开时跳过全局 ESC 导航
- **修改**: 全局 `handleKeyboardNavigation` 中 Ctrl+数字快捷键的触发条件：笔记查看/编辑/新建页面打开时跳过

## Impact
- Affected specs: 无
- Affected code:
  - `frontend/src/js/ai-chat.js`：引用笔记选择器的 ESC/Enter 键盘支持
  - `frontend/src/js/main.js`：全局 ESC 和 Ctrl+数字快捷键的触发条件修改

## ADDED Requirements / MODIFIED Requirements

### Requirement: 引用笔记选择器 ESC 支持
系统 SHALL 在引用笔记选择器浮层打开时，按 ESC 键关闭浮层并停留在 AI 聊天界面，不触发视图切换。

#### Scenario: 引用笔记选择器打开时按 ESC
- **WHEN** 引用笔记选择器浮层（`#aiNoteRefModal`）显示中
- **AND** 用户按下 ESC 键
- **THEN** 浮层关闭（`closeNoteRefModal()`）
- **AND** 视图保持当前状态（AI 聊天界面），**不**触发全局 ESC 导航（不切换到笔记首页/网格视图）

#### 现有代码分析
- `ai-chat.js` 第 1186-1195 行已有全局 ESC 监听，会关闭浮层
- `main.js` 第 5136-5171 行全局 ESC 处理中，在浮层关闭后仍继续执行并切换到 grid 视图
- 需在 `main.js` 的全局 ESC 处理器中，提前检测浮层是否打开，若是则 `return` 跳过后续导航

#### 实现方案
- `main.js` `handleKeyboardNavigation` ESC 分支开头添加：如果 `#aiNoteRefModal` 的 `display !== 'none'`，则 `return`（由 ai-chat.js 的 ESC 监听器处理关闭）

### Requirement: 引用笔记选择器 Enter 支持
系统 SHALL 在引用笔记选择器浮层打开时，按 Enter 键触发"确认引用"按钮的点击事件。

#### Scenario: 引用笔记选择器打开时按 Enter
- **WHEN** 引用笔记选择器浮层（`#aiNoteRefModal`）显示中
- **AND** 用户已选中至少一篇笔记
- **AND** 用户按下 Enter 键（且焦点不在搜索输入框内）
- **THEN** 等同于点击"确认引用"按钮
- **AND** 浮层关闭，选中的笔记被引用到 AI 输入上下文

#### 实现方案
- 在 `ai-chat.js` 引用笔记选择器浮层初始化完成后，添加 `keydown` 事件监听
- 使用 `e.stopPropagation()` 防止事件冒泡干扰其他组件
- 排除搜索输入框中的 Enter 键（已有搜索功能）

### Requirement: 笔记查看/编辑/新建页面屏蔽 Ctrl+数字快捷键
系统 SHALL 在笔记查看、编辑、新建页面中，屏蔽 Ctrl+1~9 的全局导航快捷键，防止误触发。

#### Scenario: 在笔记编辑器中按 Ctrl+6
- **WHEN** 用户正在编辑笔记（`#viewEditor.active`）
- **AND** 焦点在编辑器中
- **AND** 用户按下 Ctrl+6
- **THEN** **不**触发 `switchView('settings')`
- **AND** 事件被忽略，编辑器正常工作

#### Scenario: 在笔记查看页面中按 Ctrl+6
- **WHEN** 用户正在查看笔记（`#viewPreview.active`）
- **AND** 用户按下 Ctrl+6
- **THEN** **不**触发 `switchView('settings')`
- **AND** 事件被忽略

#### Scenario: 在新建笔记页面中按 Ctrl+数字
- **WHEN** 新建笔记面板显示中
- **AND** 用户按下 Ctrl+1~9 任意键
- **THEN** 所有 Ctrl+数字快捷键被忽略

#### 现有代码分析
- `main.js` 第 5194-5239 行 Ctrl+数字快捷键目前仅排除焦点在 `input`/`textarea`/`[contenteditable]` 的场景
- CodeMirror 6 编辑器的主内容区域可能不匹配上述选择器，导致 Ctrl+数字仍能触发
- 需要额外检查笔记视图是否激活

#### 实现方案
- 在 `main.js` `handleKeyboardNavigation` 的 Ctrl+数字分支入口处，添加条件：
  - 如果 `#viewEditor.active` 或 `#viewPreview.active` 存在，则 `return`（跳过 Ctrl+数字处理）

### REMOVED Requirements
无
