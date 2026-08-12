# Agent 新增 read_note_section 笔记分段读取工具 Spec

## Why

`manage_note` 的 view 动作读取笔记全文超过 `ai_large_file_preview_threshold`（默认 10000 字符）时会被截断，并提示"如需继续阅读可要求分段查看"——但当前工具集没有任何动作能真正分段续读，该提示是空头支票。需要一个独立的 `read_note_section` 工具，让模型按字符偏移读取笔记的后续分段。

## What Changes

- 新增 `internal/agent/tools/read_note_section.go`：实现 `readNoteSectionTool`（`tool.InvokableTool` + `ActionTextProvider`），参数 `id`（必填笔记编号）、`offset`（必填起始字符位置，rune 索引）、`length`（可选读取字符数，缺省取 `ai_large_file_preview_threshold`，上限 100000）。内部复用 `services.NoteService.GetNoteContent` 读取全文后按 rune 切片。
- 修改 `internal/agent/tools/manage_note.go`：把 viewNote 中读取 `ai_large_file_preview_threshold` 的逻辑提取为包级 helper `notePreviewThreshold(setting)`（viewNote 与 read_note_section 共用）；viewNote 截断提示改为结构化指引——"内容共 N 字符，已显示前 X，可调用 read_note_section 工具（id={id}, offset={X}）继续读取"。
- 注册链路：`internal/agent/registry.go` 追加注册、`internal/agent/tools/meta.go` 追加展示文案、`internal/agent/tools/doc.go` 与 `internal/agent/doc.go` 清单同步。
- 前端零改动：工具名直接展示英文，动作文案走既有 `ActionTextProvider` 机制。
- DB 层零改动：内容本就全量读取，rune 切片在工具层完成。

## Impact

- Affected specs: Agent 工具体系（TOOLS.md §2 新增工具流程）；manage_note 的 view 截断提示文案
- Affected code:
  - `internal/agent/tools/read_note_section.go`（新增）
  - `internal/agent/tools/manage_note.go`（提取 helper + 修改截断提示）
  - `internal/agent/registry.go`、`internal/agent/tools/meta.go`、`internal/agent/tools/doc.go`、`internal/agent/doc.go`（注册与清单）

## ADDED Requirements

### Requirement: read_note_section 工具

系统 SHALL 提供 `read_note_section` 工具，供 Agent 模式模型在 `manage_note` view 结果提示截断后，按字符偏移读取笔记的后续分段。

#### Scenario: 成功读取分段
- **WHEN** 模型调用 `read_note_section`，参数 `id` 为正整数、`offset` 为 ≥0 且小于内容总字符数的位置
- **THEN** 工具读取笔记全文，按 rune 从 offset 起读取 length 字符（缺省 `ai_large_file_preview_threshold`，上限 100000），返回 `笔记 #id 第 {start}-{end} 字符的内容（共 N 字符）：\n{内容}`，模型可据 N 判断是否还有后续

#### Scenario: 参数缺失或非法
- **WHEN** `id` 非正整数，或 `offset` 缺失/为负，或 `offset ≥ 内容总字符数`
- **THEN** 工具返回描述性 error（"缺少有效的 id"、"缺少有效的 offset"、"offset 超出内容范围（共 N 字符）"），经 `WrapWithError` 回填模型继续推理，不中断 ReAct 循环

#### Scenario: 笔记不存在
- **WHEN** `id` 对应笔记不存在
- **THEN** 工具返回 `GetNoteContent` 的错误，`WrapWithError` 捕获并发射 `tool_error` 事件

#### Scenario: 工具动作文案
- **WHEN** 模型发起 `read_note_section` 调用，`tool_start` 事件生成
- **THEN** `action_text` 为"读取笔记 #id 第 offset 字符起"，参数解析失败时回退"读取笔记分段"

### Requirement: view 截断提示联动

系统 SHALL 修改 `manage_note` view 的截断提示，为模型提供明确的续读指引。

#### Scenario: view 截断时给出下一步指令
- **WHEN** `view` 返回内容被 `ai_large_file_preview_threshold` 截断
- **THEN** 提示文案为"内容共 N 字符，已显示前 X。如需继续阅读，可调用 read_note_section 工具（id={id}, offset={X}）"，其中 N 为全文总字符数、X 为截断阈值、id 为当前笔记编号

## MODIFIED Requirements

### Requirement: 提取 notePreviewThreshold helper

原 `viewNote` 中读取 `ai_large_file_preview_threshold` 的逻辑（`m.setting.Get` + `strconv.Atoi`，失败或 ≤0 回退 10000）SHALL 提取为包级函数 `notePreviewThreshold(setting *services.SettingService) int`，`viewNote` 与 `read_note_section` 共同调用，行为不变。

## REMOVED Requirements

无
