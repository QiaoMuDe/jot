# 笔记类型功能 Spec

## Why
用户需要书写纯文本笔记（如待办列表、快速记录等），无需 Markdown 语法解析，保留原始文本格式。当前所有笔记都按 Markdown 处理，导致纯文本内容中的特殊符号（如 `#`、`*`）被错误渲染。

## What Changes
- **后端**: Note 模型新增 `NoteType string` 字段，新建/更新由前端传入 `"text"`/`"markdown"`，旧笔记该字段为空字符串时默认按 `"text"` 处理（兼容现有纯文本体验）
- **前端编辑器**: 底部状态栏左侧新增笔记类型切换器（两按钮：Markdown / 纯文本），**新建笔记默认选中「纯文本」（text）**，底部隐藏编辑/预览切换按钮
- **编辑器模式控制**: 纯文本模式隐藏底部「编辑/预览」模式切换按钮，仅显示纯文本编辑区；切换 Markdown 模式立即显示切换按钮
- **查看渲染**: 纯文本笔记直接显示原始文本（`<pre>` 保留换行/空格），跳过 marked 解析；Markdown 笔记走现有渲染流程
- **交互**: 类型切换即时生效，无需额外确认，创建/更新笔记时 `note_type` 字段随请求写入

## Impact
- Affected specs: add-md-rendering（Markdown 渲染逻辑受影响）
- Affected code:
  - `internal/models/note.go` — 新增 `NoteType string` 字段
  - `internal/database/db.go` — AutoMigrate 自动新增列
  - `frontend/src/main.js` — 编辑器类型切换器、查看渲染分支、save/create 携带类型
  - `frontend/src/style.css` — 新增类型切换器按钮样式

## ADDED Requirements

### Requirement: 笔记类型管理
The system SHALL provide note type selection in the note editor.

#### Scenario: 新建笔记默认类型
- **WHEN** 用户打开新建笔记编辑器
- **THEN** 类型默认选中「纯文本」（text）
- **AND** 底部状态栏不显示「编辑/预览」切换按钮

#### Scenario: 切换笔记类型
- **WHEN** 用户在编辑器中点击「Markdown」按钮
- **THEN** 底部状态栏立即显示「编辑/预览」切换按钮
- **AND** 用户可切换预览模式查看 Markdown 渲染效果
- **WHEN** 用户再切回「纯文本」
- **THEN** 底部「编辑/预览」切换按钮立即隐藏

#### Scenario: 编辑已有笔记加载类型
- **WHEN** 用户编辑已存在的笔记
- **THEN** 编辑器读取 `note_type` 字段值
- **AND** 该值为 `"text"` 或 `""` 时按纯文本模式显示
- **AND** 该值为 `"markdown"` 时按 Markdown 模式显示

### Requirement: 纯文本内容显示
The system SHALL display plain text notes without Markdown parsing.

#### Scenario: 查看纯文本笔记
- **WHEN** 用户查看 `note_type="text"` 的笔记
- **THEN** 内容以 `<pre>` 标签原始格式显示
- **AND** 跳过 marked 解析和 highlight.js 代码着色

#### Scenario: 查看 Markdown 笔记
- **WHEN** 用户查看 `note_type="markdown"` 的笔记
- **THEN** 内容按现有 Markdown 渲染流程处理（marked + highlight.js）

#### Scenario: 查看无类型设置的旧笔记
- **WHEN** 用户查看 `note_type=""` 的旧笔记
- **THEN** 默认按 `"text"` 处理，以 `<pre>` 原始格式显示

### Requirement: 创建/更新携带类型
The system SHALL save the `note_type` field when creating or updating notes.

#### Scenario: 创建笔记
- **WHEN** 用户保存新建笔记
- **THEN** `CreateNote` 接口传入 `noteType` 参数
- **AND** 后端存入 `note_type` 字段

#### Scenario: 更新笔记
- **WHEN** 用户保存编辑已有笔记
- **THEN** `UpdateNote` 接口传入 `noteType` 参数
- **AND** 后端更新 `note_type` 字段

## MODIFIED Requirements
无修改，均为新增。

## REMOVED Requirements
无删除。
