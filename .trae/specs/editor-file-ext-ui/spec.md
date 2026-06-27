# 编辑器文件后缀显示与编辑 Spec

## Why

笔记数据库已有 `file_ext` 字段（后端自动根据 note_type 推导 `.md`/`.txt`），但前端编辑器未展示该信息。用户希望在编辑器状态栏看到当前笔记的文件后缀，并支持点击修改，以便在导出等场景中使用自定义后缀。

## What Changes

- **前端**：编辑器状态栏（左下角）字数统计右侧增加文件后缀显示 `| .ext`，点击弹出修改对话框
- **前端**：新建笔记时根据 note_type 显示默认后缀（`.md`/`.txt`），与后端一致
- **后端**：新增 `UpdateNoteFileExt(id, fileExt)` API，供前端单独修改后缀
- **后端**：`UpdateNote` 仍保持自动从 note_type 推导 FileExt 的逻辑不变

## Impact

- Affected specs: `add-note-file-ext`（前端展示层扩展）
- Affected code:
  - `app.go` — 新增 `UpdateNoteFileExt` 绑定方法
  - `internal/services/note_service.go` — 新增 `UpdateFileExt` 方法
  - `frontend/index.html` — 状态栏增加 `.file-ext` 显示元素 + 后缀编辑对话框 HTML
  - `frontend/src/css/components/editor.css` — 后缀显示/对话框样式
  - `frontend/src/main.js` — `openEditor()`/`updateWordCount()` 中显示后缀；点击事件绑定后缀编辑逻辑

## ADDED Requirements

### Requirement: 状态栏显示文件后缀
The system SHALL display the current note's file extension in the editor status bar.

#### Scenario: 查看笔记时显示后缀
- **WHEN** 用户打开现有笔记进入编辑模式
- **THEN** 状态栏左下角显示 `字数：N ｜ 字符：N | .md`（或 `.txt`/自定义后缀）
- **AND** 后缀文字与字数统计同行，竖线 `|` 分隔
- **AND** 后缀文字可点击（cursor: pointer），点击触发编辑对话框

#### Scenario: 新建笔记时显示默认后缀
- **WHEN** 用户点击新建按钮打开空白编辑器
- **THEN** 状态栏显示 `| .txt`（默认纯文本模式）或 `| .md`（若切换为 Markdown 模式）
- **AND** 不调用后端 API，纯前端根据 note_type 推导显示

### Requirement: 点击修改后缀
The system SHALL allow the user to change the file extension via a dialog.

#### Scenario: 点击后缀弹出编辑对话框
- **WHEN** 用户点击状态栏中的后缀文字
- **THEN** 弹出模态对话框，包含：
  - 标题：「修改文件后缀」
  - 输入框：预填当前后缀（如 `.md`）
  - 提示文字：「以 . 开头，如 .md、.txt、.py」
  - 确认按钮「保存」、取消按钮「取消」
- **AND** 对话框居中显示，背景半透明遮罩

#### Scenario: 验证后缀格式
- **WHEN** 用户点击「保存」
- **THEN** 校验输入：
  - 不能为空
  - 必须以 `.` 开头
  - 长度 2-10 个字符（含 `.`）
  - 只能包含字母、数字、下划线
- **AND** 校验不通过时显示错误提示（不关闭对话框）

#### Scenario: 保存后缀
- **WHEN** 用户输入合法后缀并点击「保存」
- **THEN** 前端调用 `App.UpdateNoteFileExt(noteId, fileExt)` 更新后端
- **AND** 状态栏后缀立即更新为新值
- **AND** 对话框关闭
- **AND** 若更新失败，显示错误提示并保持对话框打开

### Requirement: 后端更新后缀 API
The system SHALL provide a dedicated API to update only the file extension.

#### Scenario: 调用 UpdateNoteFileExt
- **WHEN** 前端调用 `App.UpdateNoteFileExt(id, ".py")`
- **THEN** 后端更新对应笔记的 `file_ext` 字段为 `".py"`
- **AND** 不修改笔记的 title/content/note_type 等其他字段
- **AND** 返回更新后的 Note 对象

## MODIFIED Requirements

### Requirement: CreateNote / UpdateNote 后端逻辑
后端 `CreateNote` 和 `UpdateNote` 仍按现有逻辑自动根据 note_type 推导 FileExt。用户通过 `UpdateNoteFileExt` 单独修改后缀后，后续调用 `UpdateNote`（修改标题/内容/类型）时会**覆盖**为 note_type 对应的默认后缀。这是预期行为——类型变更时后缀应同步。

## REMOVED Requirements

无删除。
