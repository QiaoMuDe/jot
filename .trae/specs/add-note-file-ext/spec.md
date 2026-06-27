# 笔记文件后缀字段 Spec

## Why
当前 `note_type` 字段存储逻辑类型（`"text"`/`"markdown"`），但缺少具体的文件后缀信息。需要一个独立字段记录后缀（`.txt`/`.md`），为后续的功能打下基础：
1. **类型高亮** — 编辑器/列表中根据后缀显示对应的文件类型标签
2. **导出拼接** — `ExportNoteAsMarkdown` 中不再硬编码 `.md`，直接用笔记自身后缀

## What Changes
- **后端**: Note 模型新增 `FileExt string` 字段（`.txt`/`.md`），AutoMigrate 自动增列
- **创建/更新**: `CreateNote`/`UpdateNote` 中根据 `noteType` 自动计算并写入 `FileExt`
- **旧数据迁移**: `note_type='markdown'` → `.md`，其他（含空值）→ `.txt`
- **导出方法**: `ExportNoteAsMarkdown` 改用 `note.FileExt` 拼接默认文件名，不再硬编码 `.md`
- **绑定方法**：前端无需感知 `FileExt` 参数，由后端自动推导，绑定方法签名不变

## Impact
- Affected specs: `add-note-type`（扩展了类型字段体系）
- Affected code:
  - `internal/models/note.go` — 新增 `FileExt string` 字段
  - `internal/database/db.go` — AutoMigrate（自动增列）
  - `app.go` — `CreateNote`/`UpdateNote` 中自动设置 `FileExt`；`ExportNoteAsMarkdown` 改用 `note.FileExt`
  - `frontend/...` — **无变更**（前端不直接接触此字段）

## ADDED Requirements

### Requirement: 文件后缀字段
The system SHALL automatically derive and store the file extension based on note type.

#### Scenario: 新建笔记时自动设置后缀
- **WHEN** 用户创建 `note_type="markdown"` 的笔记
- **THEN** 后端自动设置 `file_ext = ".md"`
- **WHEN** 用户创建 `note_type="text"` 的笔记
- **THEN** 后端自动设置 `file_ext = ".txt"`

#### Scenario: 更新笔记类型时更新后缀
- **WHEN** 用户将笔记从 `"text"` 切换为 `"markdown"`
- **THEN** 后端自动更新 `file_ext` 从 `".txt"` 改为 `".md"`
- **WHEN** 用户将笔记从 `"markdown"` 切换为 `"text"`
- **THEN** 后端自动更新 `file_ext` 从 `".md"` 改为 `".txt"`

#### Scenario: 启动时迁移存量数据
- **WHEN** 应用启动执行 AutoMigrate
- **THEN** 所有存量 `note_type="markdown"` 的笔记自动填入 `file_ext=".md"`
- **AND** 所有其他笔记（含 `note_type=""` 的旧笔记）自动填入 `file_ext=".txt"`

### Requirement: 导出方法使用 FileExt
The system SHALL use the stored file extension when exporting notes.

#### Scenario: 导出 Markdown 笔记
- **WHEN** 用户导出 `note_type="markdown"` 的笔记
- **THEN** 保存对话框默认文件名使用 `{title}.md`
- **AND** 不改变现有 `.md` 行为

#### Scenario: 导出纯文本笔记（当前无此功能，但为未来铺垫）
- **WHEN** 未来新增纯文本导出功能
- **THEN** 可直接使用 `note.FileExt` 确定文件名后缀

## MODIFIED Requirements
无修改。

## REMOVED Requirements
无删除。
