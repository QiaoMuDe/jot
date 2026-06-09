# Lazy Content Loading Spec

## Why

列表查询（`GetAllByNotebook`）和搜索查询（`Search`）使用 GORM `SELECT *`，每条笔记的完整 `Content`（`type:text`）被全量加载到内存中。当笔记数量多或单条内容大时，列表页、搜索页的内存占用和网络传输量急剧膨胀，导致 UI 卡顿。

## What Changes

### 后端

1. **`GetAllByNotebook` 与 `Search` 排除 Content 字段**
   - 使用 GORM `Select` 限制查询列，Content 替换为 `SUBSTR(content, 1, 200) AS content`（保留 200 字符用于卡片预览）
   - 不影响 `LIKE %keyword%` 的 WHERE 过滤（SQL 层面仍可扫描 content 列）
   - 不影响 Tags 的 `Preload`（GORM 会独立生成子查询）

2. **新增 `GetNoteContent(id) (string, error)`**
   - 独立查询仅返回笔记的完整 content 文本
   - 用于 `openEditor()` 打开笔记时按需加载

3. **`GetByID` 保持不变**（单条笔记始终返回完整 Content，用于编辑/查看页）

### 前端

1. **`loadNotes()` / `loadMoreNotes()`** — 使用 `GetNotes()` 得到截断后的 content，足够卡片预览
2. **`searchNotes()`** — 使用 `SearchNotes()` 得到截断后的 content
3. **`openEditor()`** — 打开笔记时若 content 为空或仅含截断文本（length ≤ 200），调用 `GetNoteContent(noteId)` 获取完整内容
4. **`loadAllRemainingNotes()`** — `GetNotes()` 已无全量 content，直接使用（加速批量加载）

## Impact

- Affected specs: note list, search, editor
- Affected code:
  - `internal/services/note_service.go` — 修改查询方法
  - `internal/services/types.go` — 无需修改
  - `app.go` — 新增 `GetNoteContent` 方法
  - `frontend/src/main.js` — `openEditor` 增加按需加载 content 逻辑
- Performance:
  - 列表查询：Content 从全量 → 200 字符截断，对大笔记可减少 99%+ 传输量
  - 搜索查询：同上
  - 编辑器打开：每次多一次 `GetNoteContent` 调用（单行查询，毫秒级）
- UX: 卡片预览和搜索结果摘要功能不受影响（截断的 200 字符足够）

## ADDED Requirements

### Requirement: Backend lazy content

The system SHALL provide a method to fetch full note content on demand.

#### Scenario: 列表查询不加载全量 Content
- **WHEN** 用户打开笔记列表首页
- **THEN** `GetNotes` 返回的笔记包含截断的 content（前 200 字符）
- **AND** 返回的 content 足以渲染卡片预览

#### Scenario: 搜索查询不加载全量 Content
- **WHEN** 用户搜索笔记
- **THEN** `SearchNotes` 返回的笔记包含截断的 content（前 200 字符）
- **AND** WHERE 子句仍能匹配完整 content

#### Scenario: 编辑器按需加载完整 Content
- **WHEN** 用户打开笔记进入编辑/查看页面
- **THEN** 前端调用 `GetNoteContent(id)` 获取完整 content
- **AND** 完整内容注入 CM6 编辑器

## MODIFIED Requirements

### Requirement: GetAllByNotebook
**变更**: 增加 GORM `Select` 限制，Content 替换为 `SUBSTR(content, 1, 200)`。

### Requirement: Search
**变更**: 同上，Content 替换为截断版本。
