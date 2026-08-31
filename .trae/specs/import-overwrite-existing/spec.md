# 导入文件智能覆盖 Spec

## Why

当前每次导入文件都会创建新笔记，导致重复导入同一文件产生大量重复笔记。用户期望：导入同一文件时自动识别已有笔记并覆盖，而非新建；同时在笔记比文件更新时给予用户选择权，避免误覆盖。

## What Changes

- **后端 `NoteService`** — 新增按标题+后缀+笔记本查询已有笔记的方法
- **后端 `processImportFile`** — 导入前先查找匹配笔记，根据时间对比决定新建/覆盖/提示
- **后端 `App`** — 新增 `ResolveImportConflict` 方法处理用户冲突选择
- **前端 `FileImportResult`** — 扩展状态字段（`status`/`file_time`/`note_time`）
- **前端新增 `showImportConflictDialog`** — 冲突解决弹窗，列表式展示所有冲突文件，支持逐个操作和批量操作
- **前端 `showImportResults`** — 识别冲突状态，收集冲突项后调用弹窗
- **前端 `handleFileDropPaths`** — 冲突弹窗关闭后再统一刷新 UI

## Impact

- Affected specs: `add-drag-drop-import`
- Affected code:
  - `internal/services/note_service.go` — 新增查询方法
  - `app.go` — 改造 `processImportFile`，新增 `ResolveImportConflict`
  - `frontend/src/main.js` — 改造导入结果处理和冲突弹窗

## ADDED Requirements

### Requirement: 按标题匹配已有笔记

The system SHALL 在导入文件时，按标题（文件名去后缀）+ 文件后缀 + 笔记本 ID 在数据库中查找已有笔记。

#### Scenario: 找到匹配笔记
- **WHEN** 导入文件 `readme.md`，当前笔记本中已存在标题为 `readme` 且后缀为 `.md` 的笔记
- **THEN** 系统识别为"已有笔记"，进入时间对比逻辑

#### Scenario: 未找到匹配笔记
- **WHEN** 导入文件 `readme.md`，当前笔记本中不存在标题为 `readme` 且后缀为 `.md` 的笔记
- **THEN** 系统创建新笔记（现有行为不变）

### Requirement: 时间对比决定覆盖策略

The system SHALL 对比文件修改时间与笔记更新时间，自动决定覆盖或提示用户。

#### Scenario: 文件比笔记新 → 直接覆盖
- **WHEN** 导入文件的修改时间 > 已有笔记的 `UpdatedAt`
- **THEN** 系统直接用新文件内容覆盖笔记（标题、内容、后缀）
- **AND** 返回 `status: "updated"`

#### Scenario: 笔记比文件新 → 提示用户选择
- **WHEN** 已有笔记的 `UpdatedAt` > 导入文件的修改时间
- **THEN** 系统返回 `status: "conflict"`，附带文件时间和笔记时间
- **AND** 前端弹窗显示："笔记「xxx」最后编辑于 {笔记时间}，导入文件修改于 {文件时间}。是否用文件内容覆盖笔记？"
- **AND** 用户选择"覆盖" → 调用 `ResolveImportConflict(noteID, true)` 执行覆盖
- **AND** 用户选择"保留" → 跳过该文件，不创建也不覆盖

#### Scenario: 时间相同 → 跳过
- **WHEN** 文件修改时间 == 笔记更新时间（精确到秒）
- **THEN** 系统跳过该文件，返回 `status: "skipped"`

### Requirement: 冲突选择 API

The system SHALL 提供 `ResolveImportConflict(noteID uint, overwrite bool)` 方法。

#### Scenario: 用户确认覆盖
- **WHEN** 前端调用 `ResolveImportConflict(noteID, true)`
- **THEN** 后端用传入的内容更新笔记的标题、内容、后缀

#### Scenario: 用户选择保留
- **WHEN** 前端调用 `ResolveImportConflict(noteID, false)`
- **THEN** 后端不做任何操作，返回成功

### Requirement: 批量导入中的冲突处理

The system SHALL 在批量导入中将所有冲突文件汇总到一个弹窗中，用户在同一个界面内逐个选择覆盖或保留。

#### Scenario: 多文件导入含冲突
- **WHEN** 用户同时拖入 5 个文件，其中 3 个与已有笔记冲突
- **THEN** 先处理 2 个无冲突文件（自动新建或覆盖）
- **AND** 弹出一个冲突解决弹窗，列出 3 个冲突文件的列表
- **AND** 每项显示：文件名、笔记标题、笔记时间、文件时间
- **AND** 每项提供"覆盖笔记"和"保留笔记"两个按钮
- **AND** 用户逐个操作，全部处理完后关闭弹窗，统一显示结果通知

#### Scenario: 批量冲突弹窗交互
- **WHEN** 冲突弹窗展示时
- **THEN** 弹窗顶部显示"发现 N 个冲突文件"
- **AND** 顶部提供"全部覆盖"和"全部保留"快捷按钮，一键处理所有冲突项
- **AND** 用户选择后对应项从列表中移除（已处理的不再显示）
- **AND** 全部项处理完后弹窗自动关闭

#### Scenario: 仅单个冲突
- **WHEN** 导入多个文件仅有 1 个冲突
- **THEN** 弹窗仍以列表形式展示（仅 1 项），交互方式与多冲突一致

## MODIFIED Requirements

### Requirement: 拖拽导入文件（来自 add-drag-drop-import）

原需求"拖拽文件创建笔记"不变，新增匹配逻辑作为前置判断。

#### Scenario: 成功导入单个文件（无匹配）
- **WHEN** 导入文件在当前笔记本中无匹配笔记
- **THEN** 行为与原需求一致：创建新笔记

#### Scenario: 成功导入单个文件（有匹配，文件更新）
- **WHEN** 导入文件匹配到已有笔记且文件更新
- **THEN** 覆盖已有笔记内容，不创建新笔记

## REMOVED Requirements

N/A
