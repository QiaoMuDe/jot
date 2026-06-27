# 移除 note_type、纯靠 file_ext 推导行为 Spec

## Why

`note_type`（text/markdown）和 `file_ext`（.txt/.md）存储了本质相同的语义信息，却需要双向同步 —— 增加了复杂度。移除 `note_type`，纯靠 `file_ext` 一个字段决定笔记行为，为后续语法高亮等扩展功能提供统一的扩展名入口。

后端 API 同步彻底整改，为后续语法高亮对接做好铺垫：高亮引擎只需依赖 `file_ext` 即可确定语言。

## What Changes

- **后端模型**：从 `Note` struct 中移除 `NoteType` 字段（DB 列保留不动）
- **后端 API**：`CreateNote` / `UpdateNote` 的 `noteType` 参数替换为 `fileExt`
- **后端服务层**：`Create()` / `Update()` / `CreateWithNotebook()` 的 `noteType` 参数替换为 `fileExt`；删除 `fileExtFromNoteType()` 辅助函数
- **后端导入**：`ImportFiles()` 移除 `noteType` 推导，直接设置 `fileExt`
- **前端状态**：移除 `state.noteType`；新增 `noteTypeFromFileExt(ext)` 推导函数；移除 `fileExtFromNoteType()` 和 `switchNoteType()`
- **前端 UI**：移除顶部 `T`/`M` 切换按钮（`#editorTypeToggle`）
- **前端保存**：`createNote()` / `updateNote()` 传 `fileExt` 而非 `noteType`
- **种子数据**：移除 `NoteType`，改为 `FileExt` 赋值
- **AGENTS.md**：追加本次重构记录

## Impact

- Affected specs: `add-note-file-ext`（本文案是前者的演进）、`add-note-type`（将被取代）
- Affected code:
  - `internal/models/note.go` — 删除 NoteType 字段
  - `internal/services/note_service.go` — API 签名变更，删除辅助函数
  - `internal/database/db.go` — 仅移除 `note_type` 回填迁移（3 行），不加 DropColumn 代码
  - `app.go` — Wails 绑定签名变更
  - `frontend/index.html` — 删除类型切换按钮
  - `frontend/src/main.js` — 简化类型相关逻辑（25+处 → ~10 处）
  - `frontend/wailsjs/go/main/App.d.ts` — 类型声明同步
  - `frontend/wailsjs/go/main/App.js` — JS bridge 同步
  - `frontend/wailsjs/go/models.ts` — 移除 note_type
  - `tools/seed/main.go` — 移除 NoteType 赋值

## ADDED Requirements

### Requirement: file_ext 单一数据源
The system SHALL use `file_ext` as the single source of truth for determining note behavior.

#### Scenario: 以 .md 扩展名为准
- **WHEN** 笔记的 `file_ext` 为 `.md`
- **THEN** 系统将其视为 Markdown 笔记（支持预览、语法高亮、编辑/预览切换）
- **AND** 其他任何扩展名（`.txt`, `.py`, `.go` 等）视为纯文本笔记

#### Scenario: 扩展名变更即时生效
- **WHEN** 用户通过状态栏修改扩展名为 `.md`
- **THEN** Markdown 功能（预览按钮、语法高亮等）立即生效
- **WHEN** 用户将 `.md` 改为其他扩展名
- **THEN** Markdown 功能立即隐藏，按纯文本处理

### Requirement: 后端 API 签名变更
The system SHALL expose clean, extension-oriented APIs.

#### Scenario: CreateNote 传 fileExt
- **WHEN** 前端调用 `App.CreateNote(title, content, fileExt, notebookID)`
- **THEN** 后端直接存储 `file_ext`，不内部推导
- **AND** 返回包含 `file_ext` 的 Note 对象

#### Scenario: UpdateNote 传 fileExt
- **WHEN** 前端调用 `App.UpdateNote(id, title, content, fileExt)`
- **THEN** 后端更新 `file_ext` 字段，不再覆盖
- **AND** 不再接受 `noteType` 参数

### Requirement: 前端的向后兼容填充
The system SHALL provide a lightweight internal derivation for any remaining type-based logic.

#### Scenario: noteTypeFromFileExt 推导
- **WHEN** 需要判断当前是否为 Markdown 模式
- **THEN** 调用 `noteTypeFromFileExt(els.editorFileExt.textContent)` 返回 `'markdown'` 或 `'text'`
- **AND** 不存储 `state.noteType` 状态

## MODIFIED Requirements

### Requirement: 笔记导入流程
**更改前**：`ImportFiles()` 根据 `.md` 扩展名设置 `noteType`，再调用 `CreateWithNotebook(title, content, noteType, notebookID)`

**更改后**：`ImportFiles()` 直接根据扩展名设置 `fileExt`，调用 `CreateWithNotebook(title, content, fileExt, notebookID)`

### Requirement: 种子数据
**更改前**：seed 结构体包含 `NoteType string`，data 数组每项赋值 `NoteType: "markdown"` 或 `NoteType: "text"`

**更改后**：seed 结构体包含 `FileExt string`，data 数组每项赋值 `FileExt: ".md"` 或 `FileExt: ".txt"`

## REMOVED Requirements

### Requirement: 顶部 T/M 切换按钮
**Reason**：笔记类型完全由扩展名决定，无需手动切换
**Migration**：删除 `#editorTypeToggle` 按钮及其绑定的 click 处理函数

### Requirement: switchNoteType() 函数
**Reason**：不再需要显式类型切换，扩展名变更自动驱动行为变化
**Migration**：用 `noteTypeFromFileExt()` 推导函数替代

### Requirement: fileExtFromNoteType() 函数（前后端）
**Reason**：反向映射不再需要
**Migration**：直接读取/写入 `fileExt`
