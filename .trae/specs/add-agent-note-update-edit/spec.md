# manage_note 新增 update / edit 编辑能力 Spec

## Why

manage_note 当前 7 个动作（create/list/view/pin/move/add_tag/remove_tag）**没有任何编辑能力**：模型不能改标题、改扩展名、改正文。用户要求模型可操作编写笔记，需补充两个动作——`update` 专属元数据编辑（标题/扩展名），`edit` 专属正文编辑（支持整篇替换与片段替换双模式，解决长笔记无法回传全文的问题）。

## What Changes

- 修改 `internal/agent/tools/manage_note.go`（唯一代码改动文件）：
  - 新增 `update` 动作：参数 `id`（必填）+ `title` / `file_ext`（至少一个，非空才更新），底层复用 `NoteService.Update`（content 传空不碰正文）
  - 新增 `edit` 动作：参数 `id`（必填）+ 双模式（互斥）——`content` 整篇替换；`find`+`replace`+`count` 片段替换（`replace` 缺省空=删除片段，`count` 缺省 1=第 1 次出现）
  - `ActionText` 增加两个动作文案；`Info()` 参数 schema 与 `Desc` 更新；头注释 7→9 动作
- 同步 `internal/agent/tools/doc.go` 与 `internal/agent/doc.go` 清单。
- 服务层零改动（`NoteService.Update`/`GetNoteContent` 现成）；无前端改动（工具名不变、action_text 走既有机制）；MCP 无需同步（mcpserver 未暴露笔记操作）。

## Impact

- Affected code: `internal/agent/tools/manage_note.go`、`internal/agent/tools/doc.go`、`internal/agent/doc.go`
- 更新原头注释"本工具不包含 update/删除类动作（spec 明确不暴露）"为"本工具不包含删除类/批量动作"

## ADDED Requirements

### Requirement: update 动作（元数据编辑）

系统 SHALL 在 manage_note 提供 `update` 动作，编辑笔记标题与文件扩展名，不修改正文。

#### Scenario: 更新标题/扩展名
- **WHEN** 模型调用 `update`，`id` 为正整数，且 `title` / `file_ext` 至少一个非空
- **THEN** 非空字段被更新、空字段保持不变，返回"笔记 #id：{标题}（扩展名 {ext}）已更新"

#### Scenario: 参数非法
- **WHEN** `id` 非正整数，或 `title` 与 `file_ext` 均空
- **THEN** 返回描述性 error（"缺少有效的 id" / "无可更新内容"），经 WrapWithError 回填模型

### Requirement: edit 动作（正文编辑，双模式互斥）

系统 SHALL 在 manage_note 提供 `edit` 动作，支持全量替换与片段替换两种模式，两种模式不可混用。

#### Scenario: 全量替换
- **WHEN** 模型调用 `edit`，`content` 非空且未提供 `find`
- **THEN** 整篇替换正文（`NoteService.Update` content 非空分支），返回"笔记 #id 正文已整篇更新"

#### Scenario: 片段替换
- **WHEN** 模型调用 `edit`，`content` 为空、`find` 非空
- **THEN** 读取当前正文，定位第 `count` 次（缺省 1）出现的 `find` 片段并替换为 `replace`（缺省空字符串即删除片段），返回"笔记 #id 正文片段已替换（第 n 处）"

#### Scenario: 片段未找到
- **WHEN** `find` 片段在正文中不存在（或第 `count` 次出现不存在）
- **THEN** 返回 error"未在笔记 #id 中找到（第 n 次出现的）该片段，请重新调用 view 获取精确原文后重试"，回填模型可重新 view 复制精确原文

#### Scenario: 模式冲突或无可更新
- **WHEN** `content` 与 `find` 同时非空；或两者均空
- **THEN** 返回 error（"两种模式不可混用" / "无可更新内容"），回填模型调整参数

### Requirement: 动作文案与工具描述

#### Scenario: tool_start 动作文案
- **WHEN** 模型发起 `update` / `edit` 调用
- **THEN** `action_text` 分别为"更新笔记标题/扩展名" / "编辑笔记正文"

#### Scenario: Desc 引导编辑边界
- **WHEN** 模型读取 manage_note 的 `Desc`
- **THEN** 描述包含：update 只改标题/扩展名不碰正文；edit 只改正文（整篇用 content、改删某段用 find+replace 且需精确复制原文、长笔记建议片段替换避免回传全文）

## REMOVED Requirements

无
