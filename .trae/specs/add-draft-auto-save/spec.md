# 草稿自动保存与恢复 Spec

## Why
新建笔记在未保存时如果意外关闭（误点 ×、崩溃等），已输入的内容会丢失。需要一个临时存储机制，让用户在重新进入应用时有机会恢复未完成的笔记。

## What Changes

### 后端
- 新增 `drafts` 表（仅 1 行记录），存储未保存的标题和内容
- 新增 `DraftService`（3 个方法：SaveDraft/GetDraft/ClearDraft）
- `app.go` 绑定 3 个新方法暴露给前端
- `db.go` AutoMigrate 追加 Draft 模型

### 前端
- 修改 `startAutoSave()` 使其支持新建笔记（无 editingNoteId）时写入草稿表
- 在 `loadNotes()` 成功后延迟 1s 检测草稿是否存在
- 有草稿时弹出恢复确认弹窗
- 恢复、保存、取消、关闭编辑器时清除草稿

## Impact
- Affected specs: 编辑器功能、自动保存行为
- Affected code:
  - 后端：`internal/models/draft.go`（新增）、`internal/services/draft_service.go`（新增）、`app.go`（绑定）、`database/db.go`（AutoMigrate）
  - 前端：`frontend/src/main.js`（`startAutoSave`/`openEditor`/`closeEditor`/`createNote`/`loadNotes`）

## ADDED Requirements

### Requirement: Draft 数据模型
系统 SHALL 提供一个 `drafts` 表，仅允许存储 1 行记录：
- `ID`（uint，固定为 1）
- `Title`（string，长度 500）
- `Content`（string，长度不限）
- `CreatedAt` / `UpdatedAt`

#### Scenario: 写入草稿
- **WHEN** 用户新建笔记时有 ID 为 null，且自动保存触发
- **THEN** 系统调用 `SaveDraft(title, content)` 对 ID=1 执行 upsert

#### Scenario: 读取草稿
- **WHEN** 前端调用 `GetDraft()`
- **THEN** 系统返回 drafts 表中 ID=1 的记录（不存在则返回 null/nil）

#### Scenario: 清除草稿
- **WHEN** 前端调用 `ClearDraft()`
- **THEN** 系统删除 drafts 表中 ID=1 的记录

### Requirement: 新建笔记自动保存草稿
系统 SHALL 在新建笔记时自动保存到 `drafts` 表。

#### Scenario: 新建笔记自动保存
- **WHEN** 用户打开新建编辑器（`openEditor(null)`）
- **AND** 用户在标题/内容输入框中打字
- **AND** 距离上次输入已过 3 秒（与现有自动保存一致的防抖策略）
- **AND** 标题或内容非空
- **THEN** 系统调用 `App.SaveDraft(title, content)` 保存到草稿表
- **AND** 底部栏显示"草稿已保存 ✓"指示（复用现有 autoSaveIndicator）

### Requirement: 草稿检测与恢复弹窗
系统 SHALL 在用户进入首页（所有笔记视图）时检测是否存在草稿并提供恢复选项。

#### Scenario: 检测草稿并弹窗
- **WHEN** `loadNotes()` 完成加载（首次启动 / 从其他视图切回首页 / 窗口焦点切回）
- **AND** 编辑器未打开
- **AND** 延迟 1 秒后调用 `App.GetDraft()`
- **AND** 返回的草稿标题或内容非空
- **THEN** 弹出确认弹窗：「发现未保存的草稿，是否恢复？」
- **AND** 弹窗包含两个按钮：「恢复」(primary) 和「放弃」(次要)

#### Scenario: 恢复草稿
- **WHEN** 用户在恢复弹窗中点击「恢复」
- **THEN** 调用 `App.ClearDraft()` 清除草稿记录
- **AND** 调用 `openEditor(null)` 打开新建编辑器
- **AND** 将草稿的标题和内容填入编辑器相应输入框
- **AND** 自动保存机制重新激活，后续输入继续写入草稿表

#### Scenario: 放弃草稿
- **WHEN** 用户在恢复弹窗中点击「放弃」
- **THEN** 调用 `App.ClearDraft()` 清除草稿记录
- **AND** 弹窗关闭，不做其他操作

### Requirement: 草稿清除时机
系统 SHALL 在以下时机清除草稿：
1. 用户在恢复弹窗中点击「恢复」（清除后立即填入编辑器）
2. 用户在恢复弹窗中点击「放弃」
3. 用户在编辑器中点击「保存」按钮（`createNote()` 成功后）

关闭编辑器（取消按钮 / × 按钮 / ESC 键）**不**清除草稿，以便用户下次进入首页时恢复。

#### Scenario: 保存后清除
- **WHEN** `createNote()` 执行成功（笔记已创建到 notes 表）
- **THEN** 调用 `App.ClearDraft()` 清除草稿

### Requirement: 草稿为空时不保存
系统 SHALL NOT 在标题和内容均为空时保存草稿。

#### Scenario: 空内容不保存
- **WHEN** 自动保存触发
- **AND** 标题和内容均为空或仅含空白字符
- **THEN** 不做任何保存操作
- **AND** 不显示「草稿已保存」指示

## MODIFIED Requirements

### Requirement: 自动保存逻辑扩展（复用现有机制）
现有 `startAutoSave()` 当前只对 `editingNoteId` 有值时生效（编辑已有笔记）。现修改为：
- **IF** `editingNoteId` 有值 → 调用 `UpdateNote()`（现有行为，不变）
- **ELSE**（新建笔记）→ 调用 `App.SaveDraft(title, content)` 保存到草稿表
- 3 秒防抖策略、标题非空判断等规则保持不变

## REMOVED Requirements
无。
