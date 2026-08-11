# manage_note 笔记管理工具 Spec

## Why

Agent 目前只能通过 `recall_notes`"召回"笔记片段作为回答上下文，无法结构化操作笔记库（创建、查看全文、置顶、移动笔记本、打/移除标签），与"智能笔记助手"定位不匹配。`NoteService`（37 个方法）与 `TagService` 的笔记标签关联能力（`AddTagToNote` / `RemoveTagFromNote`）均未被任何工具暴露。本 spec 新增 `manage_note` 工具补齐笔记库操作能力。

## What Changes

- 新增 `manage_note` 工具（[internal/agent/tools/manage_note.go](internal/agent/tools/manage_note.go)，新建），通过 `action` 参数区分 7 个动作：
  - `create`：创建笔记（`Create` / `CreateWithNotebook` + 逐个 `AddTagToNote`）
  - `list`：列出笔记（`Search`，关键词 / 笔记本 / 标签 AND / 日期范围 / 排序 / 分页）
  - `view`：查看笔记全文（`GetNoteContent`，按大文件预览阈值截断）
  - `pin`：置顶 / 取消置顶（`TogglePin`）
  - `move`：移动笔记本（`MoveToNotebook`）
  - `add_tag`：给笔记打标签（`TagService.AddTagToNote`）
  - `remove_tag`：移除笔记标签（`TagService.RemoveTagFromNote`）
- `Deps` 新增 `Note *services.NoteService` 字段（[agent.go](internal/agent/agent.go#L44-L54)）；[registry.go](internal/agent/registry.go#L19-L28) 追加注册；[app.go](app.go) 构造 `NewAgentService` 与 `rebuildServices` 末尾重建 AgentSvc 处传参。
- `view` 全文截断复用设置项 `ai_large_file_preview_threshold`（大文件预览阈值，默认 10000 字符，范围 1-100000），防模型上下文爆炸；超过阈值时返回截断内容并提示。
- **明确不暴露**：`update`（编辑笔记，用户暂缓）；`Delete`/`Restore`/`PermanentDelete`/`EmptyTrash`（删除类，Agent 无法弹确认框，属不可逆/风险操作）；批量操作（BatchDelete/BatchPin/BatchMove）。
- 文档同步：[TOOLS.md](internal/agent/TOOLS.md) 工具清单、[tools/doc.go](internal/agent/tools/doc.go)、[doc.go](internal/agent/doc.go)。

## Impact

- Affected specs: Agent 工具集（现有 7 个工具不变，`manage_note` 与 `recall_notes` 边界在 `Info().Desc` 中说明）
- Affected code:
  - [internal/agent/tools/manage_note.go](internal/agent/tools/manage_note.go)（新建）
  - [internal/agent/registry.go](internal/agent/registry.go#L19-L28)（注册）
  - [internal/agent/agent.go](internal/agent/agent.go#L44-L54)（Deps 新增 Note 字段）
  - [app.go](app.go)（`NewAgentService` 构造传参 + `rebuildServices` 末尾重建 AgentSvc）
  - [internal/agent/TOOLS.md](internal/agent/TOOLS.md)、[internal/agent/tools/doc.go](internal/agent/tools/doc.go)、[internal/agent/doc.go](internal/agent/doc.go)（文档清单）

## ADDED Requirements

### Requirement: manage_note 工具

系统 SHALL 提供 `manage_note` 工具，通过 `action` 参数区分 create / list / view / pin / move / add_tag / remove_tag 七个动作。工具注入 `NoteService`、`TagService`、`SettingService`（读大文件预览阈值）与共享 `Context`，返回纯文本结果；列表返回 `[数字]` 编号供后续动作按 id 定位（与 manage_todo / manage_notebook / manage_tag 一致）。工具用 `WrapWithError` 包装（失败回填模型不中断 ReAct 循环），动作分发前执行 `ctx.Err()` 取消检查。

#### Scenario: 创建笔记（create）

- **WHEN** 用户要求"帮我记一条笔记：<内容>"
- **THEN** 工具以 `title`、`content` 创建笔记，`file_ext` 缺省 `.md`（可指定）；可选 `notebook_id` 指定笔记本（`CreateWithNotebook`）、`tag_ids` 数组给新笔记打标签（逐个 `AddTagToNote`）；返回新笔记编号与标题。参数缺失（title/content 为空）返回错误回填模型。

#### Scenario: 列出笔记（list）

- **WHEN** 用户要求"列出/找一下我的笔记"
- **THEN** 调用 `Search` 返回分页列表：`[数字]` 编号 + 标题 + 内容预览（前 200 字符）+ 标签 + 置顶标记；返回"共 n 条、第 x/y 页"，当页未展示完时提示可翻页。可选过滤：`keyword`（标题/内容模糊）、`notebook_id`、`tag_ids`（多标签 AND）、`start_date`/`end_date`（YYYY-MM-DD，updated_at 范围）、`sort_by`（updated_at 缺省 / created_at / title）、`page`（从 1 起）、`page_size`（缺省 10、上限 50）。

#### Scenario: 查看笔记全文（view）

- **WHEN** 用户要求"看看某篇笔记的完整内容"
- **THEN** 按 `id` 读取全文（`GetNoteContent`）。全文超过设置项 `ai_large_file_preview_threshold`（大文件预览阈值，默认 10000）字符时按该值截断并在结果末尾提示"内容过长已截断，如需继续阅读可要求分段查看"。

#### Scenario: 置顶 / 移动 / 打标签 / 移除标签（pin / move / add_tag / remove_tag）

- **WHEN** 用户要求"把这篇置顶 / 移到某个笔记本 / 打上某标签 / 去掉某标签"
- **THEN** 按 `id` 定位笔记执行动作：`pin` 切换置顶状态；`move` 需 `notebook_id`（目标笔记本，`manage_notebook` list 的编号）；`add_tag` / `remove_tag` 需 `tag_id`（`manage_tag` list 的编号）。`id`/`notebook_id`/`tag_id` 均为正整数，非正整数或缺失返回错误回填模型。

### Requirement: 工具边界

`manage_note` 的 `Info().Desc` SHALL 说明与 `recall_notes` 的边界：`recall_notes` 用于语义召回片段回答知识类问题；`manage_note` 用于结构化操作笔记库（创建 / 列表 / 读全文 / 置顶 / 移动 / 打标签）。

#### Scenario: 模型正确选择工具

- **WHEN** 用户问"我笔记里有没有 XX 的知识"
- **THEN** 模型调用 `recall_notes`（知识召回），而非 `manage_note`
- **WHEN** 用户要求"帮我新建一条笔记 / 把昨天那篇置顶 / 列出我的笔记"
- **THEN** 模型调用 `manage_note`

### Requirement: 依赖装配与重建

系统 SHALL 在 `Deps` 中新增 `Note *services.NoteService`；`NewAgentService` 构造与 `rebuildServices` 末尾重建 `AgentSvc` 时注入同一 `NoteService` 实例，避免数据库重置/切换后工具持有旧服务指针。

#### Scenario: 数据库重置后工具仍可用

- **WHEN** 用户执行恢复出厂 / 还原备份（触发 `rebuildServices`）
- **THEN** `AgentSvc` 以新 `NoteService` 实例重建，`manage_note` 操作不因旧连接失效。
