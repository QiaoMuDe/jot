# 拆分 manage_note：将 list / view 独立为新工具 browse_notes

## Summary

从 `manage_note` 中把只读操作 **list / view** 拆出，形成独立新工具 **`browse_notes`**（浏览/阅读笔记库）。`manage_note` 保持原名、收窄为纯写/管理工具（create / update / edit / pin / move / add_tag / remove_tag）。工具总数 14 → 15。

仅新增读工具 + 收窄管理工具，不改动 `manage_note` 名字 → **无需迁移 `ai_agent_tools_disabled`**（该名单存的是工具名，`manage_note` 未变、`browse_notes` 为全新名）。

## Current State Analysis

- [manage_note.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/manage_note.go)`manageNoteTool` 结构体依赖 `note/tag/setting/ctx`（L123-128）。
- `ActionText`（L135-164）按 action 返回动作文案，当前含 list/view。
- `Info()`（L166-298）：扁平 24 参数；action 枚举含 list/view。
- `InvokableRun`（L301+）按 action 分发；L345 写操作确认 `isManageNoteWriteAction` 不含 list/view。
- `listNotes`（L446-534）：仅用 `m.note.SearchByNotebook/Search`，**不触 tag**；`viewNote`（L536-621）：用 `m.note.GetNoteContent` + `m.setting`(notePreviewThreshold) + `maxSectionLen/numberLines/splitNoteLines/resolveNoteIDs`。
- 共享包级助手函数都定义在 manage_note.go：`resolveNoteIDs`(L106)、`maxSectionLen`(L604)、`notePreviewThreshold`(L608)、`numberLines`、`splitNoteLines`——**browse_notes 可原样复用，无需移动**。
- 构造器 `NewManageNote(note, tag, setting, ctx)`（L1233）。
- 注册/展示：
  - [registry.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go) buildTools：manage_note 经 `tools.WrapWithError("manage_note", tools.NewManageNote(...), p.ctx)` 注册。
  - [meta.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/meta.go) `BuiltinTools()` 供前端工具开关列表。
  - app.go `GetAgentTools`（L2704-2744）自动由 BuiltinTools 派生 → 新增 browse_notes 后前端自动多一个开关，**无需改前端 JS**。
- 文档清单：tools/doc.go、internal/agent/doc.go、AGENTS.md 均有工具清单与计数。

## Design Decisions

- 读工具名：**`browse_notes`**（避免 search/recall/query 与向量召回 `recall_notes` 的语义重叠；动词 browse 覆盖「列表 + 读全文」）。
- 写工具名：**`manage_note` 保持不变**（manage_* 家族一致，且避免禁用名单迁移成本）。
- 读操作（list/view）一律走 browse_notes；写工具彻底移除 list/view 及其参数。
- 行号引用（edit 行级替换需 view 的 line_numbers）改为指向 `browse_notes` 的 view。
- 禁用名单：无需迁移（无工具名变更，仅新增工具）。

## Proposed Changes

### 1. 新建 internal/agent/tools/browse_notes.go（读工具）

- 结构体 `browseNotesTool{ note *services.NoteService; setting *services.SettingService; ctx *Context }`（list/view 不触 tag）。
- 构造器 `NewBrowseNotes(note *services.NoteService, setting *services.SettingService, ctx *Context) tool.InvokableTool`。
- 编译期断言 `var _ tool.InvokableTool = (*browseNotesTool)(nil)`。
- `ActionText`：`list` → "列出笔记"、`view` → "查看笔记全文"、默认 "执行"。
- `Info()`：
  - Name: `"browse_notes"`，Label 语义见 meta.go。
  - Desc（读定位，关键）：与前 `recall_notes` 边界 + 与 `manage_note` 边界——**本工具只读**：list=按条件/关键字/标签/日期/分页列出笔记；view=按 id 读全文（超阈值可 offset/length 分段续读，line_numbers=true 输出全局行号供 edit 寻址）。"如需创建/修改/置顶/移动/打标签笔记，请使用 manage_note"。
  - `action` 枚举：`["list", "view"]`（必填）。
  - 参数（仅读相关）：
    - list：`keyword`、`tag_ids`、`notebook_id`、`start_date`、`end_date`、`sort_by`（Enum updated_at/created_at/title）、`page`、`pageSize`
    - view：`ids`、`line_numbers`、`offset`、`length`
    - 以上全部 `Required:false`，`action` 除外。
- `InvokableRun`：解析 `action` → `list`→`listNotes(...)`、`view`→`viewNote(...)`。
- 方法 `listNotes`、`viewNote`：**从 manage_note.go 平移**，仅把 receiver 由 `*manageNoteTool` 改为 `*browseNotesTool`，逻辑不变。复用包级助手 `resolveNoteIDs / notePreviewThreshold / maxSectionLen / numberLines / splitNoteLines`。

### 2. 收窄 manage_note.go（写/管理）

- `Info()`：
  - action Enum 移除 `"list"`、`"view"` → `["create","update","edit","pin","move","add_tag","remove_tag"]`。
  - Desc 精简：删除 list/view 长段说明，改为「本工具为写/管理：创建/更新/编辑/置顶/移动/打标签；**读取/浏览笔记请用 browse_notes**」。edit 中「行号来自 view 的 line_numbers=true 输出」→「行号来自 **browse_notes** 的 view(line_numbers=true) 输出」。
  - 删除参数定义（不再用于任何写 action）：`keyword`、`start_date`、`end_date`、`sort_by`、`page`、`pageSize`、`line_numbers`、`offset`、`length`。
  - 保留：`action`、`ids`、`confirm`、`title`、`content`、`file_ext`、`notebook_id`、`tag_ids`、`tag_id`、`find`、`replace`、`count`、`replace_all`、`line_start`、`line_end`（15 参数）。
- `ActionText`：删除 `list`/`view` 分支。
- `InvokableRun`：删除 `list`/`view` 分发分支；`args` 匿名结构体删除不再使用的读字段（`keyword/start_date/end_date/sort_by/page/pageSize/line_numbers/offset/length`；`tag_ids` 仍被 create 用，保留）。
- 删除已平移的方法：`listNotes`、`viewNote`（移到 browse_notes.go）。**保留**全部包级共享函数（resolveNoteIDs / splitNoteLines / numberLines / notePreviewThreshold / maxSectionLen），因为它们被 browse_notes.go 复用（numberLines 亦被 lineEditPreview 使用）。
- 头部/action 级注释：删除 list/view 描述；同步 edit/line_start 关于行号来源的措辞。

### 3. 注册与展示

- [registry.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go) buildTools 在 manage_note 前追加：
  `{"browse_notes", tools.WrapWithError("browse_notes", tools.NewBrowseNotes(p.deps.Note, p.deps.Setting, p.ctx), p.ctx)},`
  （确认 `p.deps` 暴露 Note/Setting；沿用现有 WrapWithError 风格。）
- [meta.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/meta.go) `BuiltinTools()` 在 manage_note 前加入 `{Name: "browse_notes", Label: "浏览/阅读笔记（列出或用行号/分段查看全文）"}`。

### 4. 文档同步

- tools/doc.go：构造器清单加 `NewBrowseNotes`；工具清单加 browse_notes；计数 14 → 15。
- internal/agent/doc.go：只读工具清单加 browse_notes（并归入"只读"）。
- [TOOLS.md](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/TOOLS.md)：
  - L4 适用对象清单：可加 `browse_notes`。
  - L18 目录树 `manage_note.go` 注释的「create/list/view/...」改为「create/update/edit/pin/move/tags（写/管理）」，并新增一行 `browse_notes.go`（list/view，读）。
  - L374/L386 的 ActionText 示例注释「create/list/view/...」改为「create/update/edit/...（manage_note）」，并注明 `browse_notes` 承担 list/view 的动作文案。
- AGENTS.md：工具目录树与计数描述 15 → 16（若含计数）；manage_note 描述改为写管理；browse_notes 说明读职责。所有「行号来自 view」措辞改 browse_notes。
- 注意：read_url.go 复用 `maxSectionLen` 不受影响（常量未动）。

## Assumptions & Decisions

- `browse_notes` 只读，不含 confirm、不写库。
- list/view 的返回格式（`[数字] id`）不变，`manage_note` 的写操作取 id 协作链照旧。
- 现有 manage_note 用户禁用名单不受影响（工具名未变）；browse_notes 默认启用。
- `ActionText` 行为保持：前端 tool_start 仍能显示「列出笔记/查看笔记全文」（browse_notes）与写动作文案（manage_note）。

## Verification

1. `go build ./...`
2. `go vet ./internal/agent/...`
3. `go test ./internal/agent/...`（tools 包内网络相关测试除外——沙箱无 tcp6；其余应全绿）
4. 断言检查：
   - `manage_note` 的 action Enum 不含 list/view；browse_notes 仅含 list/view。
   - 全仓搜索确认无 `listNotes`/`viewNote` 遗漏 receiver 导致编译错（由 build 兜底）。
5. 行为抽查：
   - browse_notes `list`：关键字/标签/日期/分页过滤正常，返回 `[数字] id`。
   - browse_notes `view`：offset=0 截断→按返回 offset 续读，行号连续；line_numbers 行号供 manage_note edit 行级替换。
   - manage_note `create/edit/pin/move/add_tag/remove_tag`：正常，confirm 机制保留；不再响应 list/view。
6. 前端：打开 Agent 工具开关列表应出现 browse_notes（BuildinTools 派生，重建前端资源后可见）。
7. 可选：`npm run build` + `wails build` 使工具开关面板可见（本次无前端逻辑改动，仅后端）。