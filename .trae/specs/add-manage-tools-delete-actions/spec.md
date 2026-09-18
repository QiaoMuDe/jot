# 管理工具删除动作 Spec

## Why

manage_todo / manage_notebook / manage_tag / manage_note 四个管理工具均无删除类 action(唯一有 delete 的是 manage_memory),后端删除能力全部现成却对 Agent 不可达,「删掉这几篇笔记」「清掉已完成待办」等高频自然语言操作无法完成,管理能力名不副实。用户已拍板:删除动作全部暴露,且删除属于高危操作,审批分级一律 critical=true(而非常规审批)。

## What Changes

- manage_note 新增 `delete` action(软删进回收站,支持批量);恢复不提供,用户自行到回收站页面操作
- manage_notebook 新增 `delete` action,带可选参数 `with_notes`(默认 false:笔记迁入默认笔记本;true:连同笔记移入回收站)
- manage_tag 新增 `delete` action(仅删标签,笔记不受影响)
- manage_todo 新增 `delete`(单条,硬删)与 `clear`(清空已完成待办)两个 action,审批摘要注明「不可恢复」
- **所有删除类动作(delete/clear)审批分级一律 critical=true**
- 不暴露后端全量清空能力:TodoService.DeleteUnfinished / DeleteAll / NotebookService.ResetAll 不接入(属数据管理页职责)
- 四个工具 Info() 参数 schema 与描述文案、ActionText 动作文案同步扩展;meta.go 的 Label 微调补「删除」字样
- TOOLS.md / EVENTS.md(各工具审批分级段)同步更新
- manage_approval_test.go 扩展删除动作测试(审批断言 + 真实落库验证)

**BREAKING**: 无(纯增量 action,现有 action 行为不变)

## Impact

- Affected specs: add-agent-manage-note-tool、add-agent-manage-todo-tool、add-agent-manage-notebook-tool、add-agent-manage-memory-tool(先例)、migrate-note-write-approval(审批分级公式)、add-agent-workspace-fs-shell(审批三模式机制)
- Affected code:
  - `internal/agent/tools/manage_note.go` / `manage_notebook.go` / `manage_tag.go` / `manage_todo.go`(action 分发 + Info schema + ActionText + 审批摘要)
  - `internal/agent/tools/meta.go`(Label)
  - `internal/agent/TOOLS.md` / `EVENTS.md`(文档)
  - `internal/agent/tools/manage_approval_test.go`(测试)
  - 前端零改动(ActionTextProvider 机制已支持动作文案)

## ADDED Requirements

### Requirement: 删除动作统一高危审批

四个管理工具新增的全部删除类 action(`manage_note.delete` / `manage_notebook.delete` / `manage_tag.delete` / `manage_todo.delete` / `manage_todo.clear`)在请求审批时 SHALL 一律传 `critical=true`,使 review 模式下必确认、confirm_every 下全确认、auto 模式下放行并 `tool_auto_approval` 留痕。

#### Scenario: review 模式下删除必须确认

- **WHEN** 审批模式为 review,模型调用 manage_tag 的 delete action
- **THEN** 触发审批弹窗(critical=true),用户允许后才执行删除;拒绝则返回拒绝错误文本,标签不落库

#### Scenario: auto 模式下删除放行并留痕

- **WHEN** 审批模式为 auto,模型调用 manage_todo 的 clear action
- **THEN** 不弹窗直接执行,审批摘要记录 `tool_auto_approval` 审计留痕(既有机制,前端渲染警示样式)

### Requirement: manage_note 删除

manage_note SHALL 新增 `delete` action:参数 `ids`(复用现有 resolveNoteIDs,支持单条/批量),调 NoteService.Delete 软删进回收站,critical=true。恢复 SHALL NOT 由 Agent 提供,用户自行到回收站页面操作。

#### Scenario: 批量删除笔记进回收站

- **WHEN** 模型调用 manage_note action=delete ids=[3,5],用户批准
- **THEN** 两篇笔记 deleted_at 置位(回收站可见、用户可在页面自行恢复),返回删除条数

### Requirement: manage_notebook 删除

manage_notebook SHALL 新增 `delete` action:参数 `id` 必填、`with_notes` 可选布尔(默认 false)。false 调 NotebookService.Delete(其下笔记迁入默认笔记本 id=1);true 调 NotebookService.DeleteWithNotes(笔记移入回收站)。critical=true 恒定。默认笔记本(id=1)删除 SHALL 返回既有「默认笔记本不可删除」错误。

#### Scenario: 温和删除笔记本

- **WHEN** 模型调用 manage_notebook action=delete id=2(默认 with_notes=false),用户批准
- **THEN** 笔记本 2 软删,其下全部笔记 notebook_id 迁为 1,数据零丢失

#### Scenario: 连带删除进回收站

- **WHEN** 模型调用 manage_notebook action=delete id=2 with_notes=true,用户批准
- **THEN** 笔记本 2 软删,其下全部笔记移入回收站(可恢复)

#### Scenario: 默认笔记本保护

- **WHEN** 模型调用 manage_notebook action=delete id=1
- **THEN** 返回「默认笔记本不可删除」错误,不触发审批弹窗

### Requirement: manage_tag 删除

manage_tag SHALL 新增 `delete` action:参数 `id` 必填,调 TagService.Delete,critical=true。标签删除不影响笔记内容与关联数据。

#### Scenario: 删除标签

- **WHEN** 模型调用 manage_tag action=delete id=4,用户批准
- **THEN** 标签 4 删除,笔记本身内容不变

### Requirement: manage_todo 删除与清空

manage_todo SHALL 新增:
- `delete`:参数 `id` 单条,调 TodoService.Delete(硬删),critical=true,审批摘要注明「不可恢复」
- `clear`:无 id 参数,调 TodoService.DeleteCompleted 仅清空已完成待办(硬删),critical=true,审批摘要注明「不可恢复」并返回清理条数

`TodoService.DeleteUnfinished` / `DeleteAll` SHALL NOT 接入(防模型全量清空用户未完成待办)。

#### Scenario: 删除单条待办

- **WHEN** 模型调用 manage_todo action=delete id=7,用户批准
- **THEN** 待办 7 硬删除,摘要含「不可恢复」警示

#### Scenario: 清空已完成待办

- **WHEN** 模型调用 manage_todo action=clear,用户批准,当前已完成 3 条
- **THEN** 3 条已完成待办删除,未完成待办不受影响,返回清理条数

#### Scenario: 拒绝后不落库

- **WHEN** 用户在审批面板拒绝任一删除动作
- **THEN** 返回拒绝错误文本,数据库无任何变化(查询验证)

### Requirement: 文案与文档同步

- 四个工具 Info() 的参数 JSON Schema 与描述 SHALL 覆盖新 action(模型可见)
- ActionText SHALL 为新 action 提供中文动作文案(「删除笔记(移入回收站)」/「删除笔记本」/「删除标签」/「删除待办」/「清空已完成待办」),缺失回退「执行」
- meta.go BuiltinTools 的 Label SHALL 补「删除」字样(顺序不变)
- TOOLS.md 各工具 action 清单与审批分级说明、EVENTS.md 各工具审批分级表 SHALL 同步(删除类恒 critical)

### Requirement: 测试覆盖

manage_approval_test.go SHALL 扩展(沿用现有 rejectApprover/真实 Service/内存 SQLite 模式):

- 每个 delete/clear 动作:审批参数断言(critical=true)、拒绝不落库、批准后落库生效
- manage_note delete 批量软删(deleted_at 置位)
- manage_notebook 默认迁默认本 / with_notes=true 进回收站 / id=1 保护
- manage_todo clear 仅清已完成并返回条数
