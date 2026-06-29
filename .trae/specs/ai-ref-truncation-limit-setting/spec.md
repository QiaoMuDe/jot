# AI 引用笔记截断字数设置 Spec

## Why

AI 助手引用笔记时，单条笔记的最大截断字数（目前硬编码为 4000）定义在后端，用户无法按需调整。将此值迁移到设置页，存入数据库，使前端可在设置页中修改，后端在构建引用上下文时动态读取。

## What Changes

- **修改**：`NoteService` 新增 `settingService` 字段，`BuildNoteRefContext` 从数据库读取截断配置替代硬编码常量
- **新增**：后端绑定 `GetAIRefMaxChars` / `SetAIRefMaxChars`，读写 `ai_ref_max_chars` 设置项
- **新增**：设置页 AI 助手区域增加"引用截断字数"输入项，默认 1000，用户可自定义
- **新增**：前端 `loadAISettings` 中加载该配置并填入输入框，用户修改后自动保存
- **影响**：设置页默认值 1000 写入数据库后，所有包含引用的 AI 对话均按新值截断

## Impact

- Affected specs: `add-ai-assistant`
- Affected code: `app.go`, `internal/services/note_service.go`, `frontend/index.html`, `frontend/src/main.js`

---

## ADDED Requirements

### Requirement: 后端可配置截断字数

The system SHALL replace the hardcoded `maxPerNote` constant in `BuildNoteRefContext` with a value read from the `ai_ref_max_chars` setting.

#### Scenario: 读取截断配置
- **WHEN** `BuildNoteRefContext` is called
- **THEN** the system reads setting key `ai_ref_max_chars` from the database
- **AND** if the value is empty or not a valid positive integer, uses 1000 as default
- **AND** uses this value as the per-note truncation character limit

#### Scenario: 设置截断配置
- **WHEN** frontend calls `SetAIRefMaxChars(chars)`
- **THEN** the system stores the value as setting key `ai_ref_max_chars`
- **AND** if chars is ≤ 0, returns an error "截断字数必须大于 0"
- **AND** if chars is > 50000, returns an error "截断字数不能超过 50000"

### Requirement: 设置页 UI

The system SHALL display a "引用截断字数" input field in the AI 助手 section of the settings page.

#### Scenario: 加载设置
- **WHEN** `loadAISettings()` runs
- **THEN** call `GetAIRefMaxChars()` and display the value in the input field

#### Scenario: 修改设置
- **WHEN** user changes the input value
- **THEN** call `SetAIRefMaxChars(newValue)` to persist
- **AND** show a small "已保存" indication or toast

---

## MODIFIED Requirements

### Requirement: `NoteService` 结构体（已存在）

Add a `settingService` field to `NoteService` so that `BuildNoteRefContext` can read settings from the database.

### Requirement: `BuildNoteRefContext`（已存在）

Replace `const maxPerNote = 4000` with a dynamic read from the `ai_ref_max_chars` setting.

### Requirement: `app.go` 绑定（已存在）

Add `GetAIRefMaxChars()` (returns int) and `SetAIRefMaxChars(chars int)` bindings.

---

## REMOVED Requirements

None.
