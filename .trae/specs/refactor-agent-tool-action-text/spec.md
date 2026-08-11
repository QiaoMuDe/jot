# Agent 工具动作文案下沉到工具实现 Spec

## Why

前端 [showToolStatusStart](frontend/src/js/ai-chat.js#L2531-L2597) 用约 40 行 switch 为每个工具维护"开始调用动作文案"（如"创建待办"）。每新增工具/动作都要改前端 JS 并重新构建前端资源（`npm run build` + `wails build`），新增的 `manage_note`（7 个动作）尚未补分支。将动作文案维护点下沉到每个工具实现内（工具最清楚自己在做什么动作），前端删除 switch、直接读取后端下发的 `action_text`，新增工具只需在对应工具文件内实现，前端零改动。

## What Changes

- [internal/agent/tools/context.go](internal/agent/tools/context.go)：
  - 新增可选接口 `ActionTextProvider`（`ActionText(argumentsInJSON string) string`）；
  - `Record` 新增字段 `ActionText string json:"action_text,omitempty"`；
  - `WrapWithError` 改为自定义 wrapper 结构体（**保持现有失败回填 / `tool_error` 事件 / 用户取消不误报行为完全不变**），wrapper 实现 `ActionTextProvider` 转发给被包装工具（未实现返回空串）。
- [internal/agent/agent.go](internal/agent/agent.go)：`Run()` 构建 name→tool 映射；`emitToolStart` 增加映射参数，按 `tc.Function.Name` 断言 `ActionTextProvider` 生成 `ActionText` 填入 `Record` 并随 `ai:tool-status` 下发。
- 8 个工具文件各自实现 `ActionText`（解析自己的 action/关键参数返回中文文案）。
- 前端 [ai-chat.js#L2531-L2571](frontend/src/js/ai-chat.js#L2531-L2571)：`showToolStatusStart` 中 action 改为 `payload.action_text || '执行'`，**删除整个工具名 switch**。
- [internal/agent/TOOLS.md](internal/agent/TOOLS.md)：工具开发维护规范更新为"动作文案在工具实现内维护（可选实现 `ActionTextProvider`），前端不再维护"。

## Impact

- Affected specs: Agent 工具集（8 个工具）的动作文案机制；前端工具状态条展示协议（`ai:tool-status` 的 `tool_start` 记录新增 `action_text` 字段）
- Affected code:
  - [internal/agent/tools/context.go](internal/agent/tools/context.go)（接口 + Record 字段 + WrapWithError 改造）
  - [internal/agent/agent.go](internal/agent/agent.go)（emitToolStart 填充 action_text）
  - [internal/agent/tools/](internal/agent/tools/) 8 个工具文件（各实现 ActionText）
  - [frontend/src/js/ai-chat.js](frontend/src/js/ai-chat.js)（删除 switch，读 action_text）
  - [internal/agent/TOOLS.md](internal/agent/TOOLS.md)（规范更新）

## ADDED Requirements

### Requirement: ActionTextProvider 可选接口

系统 SHALL 在 tools 包定义可选接口 `ActionTextProvider`，工具实现后由父包在 `tool_start` 时自动采用其动作文案；未实现（返回空串）时前端回退"执行"。

#### Scenario: 工具实现了 ActionTextProvider

- **WHEN** 模型调用某个实现了 `ActionTextProvider` 的工具
- **THEN** `tool_start` 记录携带 `action_text` 字段，前端状态条显示"调用「工具名」工具：{action_text}"

#### Scenario: 工具未实现 ActionTextProvider

- **WHEN** 模型调用未实现该接口的工具（或返回空串）
- **THEN** 前端回退显示"调用「工具名」工具：执行"

### Requirement: 8 个工具的动作文案

系统 SHALL 让各工具在自己文件内实现 `ActionText(argumentsInJSON string) string`，文案与前端现有一致：

| 工具 | 动作 | 文案 |
|---|---|---|
| `web_search` | query 非空 | `搜索 {query}` |
| `web_search` | query 为空 | `搜索互联网` |
| `recall_notes` | notebook_ids 数量 n>0 | `检索 n 个笔记本` |
| `recall_notes` | 无 notebook_ids | `检索本地笔记` |
| `refine_search_query` | 任意 | `精炼搜索关键词` |
| `get_current_time` | 任意 | `获取当前日期时间` |
| `manage_todo` | create / list / toggle / update | `创建待办` / `列出待办` / `更新待办状态` / `修改待办文本` |
| `manage_notebook` | create / rename / list | `创建笔记本` / `重命名笔记本` / `列出笔记本` |
| `manage_tag` | create / list / update | `创建标签` / `列出标签` / `更新标签` |
| `manage_note` | create / list / view / pin / move / add_tag / remove_tag | `创建笔记` / `列出笔记` / `查看笔记全文` / `置顶或取消置顶笔记` / `移动笔记` / `添加标签` / `移除标签` |

#### Scenario: manage_note 各动作文案

- **WHEN** 模型以 action=create 调用 manage_note
- **THEN** `action_text` 为"创建笔记"；action=add_tag 时为"添加标签"，依此类推

### Requirement: 前端展示

系统 SHALL 让 `showToolStatusStart` 直接读取 `payload.action_text`（缺失回退"执行"），删除全部工具名 switch 分支，前端不再维护任何工具动作文案。

#### Scenario: 实时状态条展示

- **WHEN** 任意工具开始调用
- **THEN** 状态条显示 `调用「{工具名}」工具：{action_text}`，文案来自后端下发

## MODIFIED Requirements

### Requirement: WrapWithError 包装器

`WrapWithError` SHALL 保持现有全部行为：失败错误文本回填模型继续推理、记 `tool_error` 记录并发射事件、用户取消（`ctx.Err()`）不误报失败。在此基础上 wrapper SHALL 实现 `ActionTextProvider` 并转发给被包装工具（未实现返回空串），使父包可对包装后的工具做统一断言。

#### Scenario: 工具执行失败

- **WHEN** 工具执行返回 error（非取消）
- **THEN** 行为与改造前完全一致：回填"工具执行失败：…"、发射 `tool_error`、记录失败态

## REMOVED Requirements

### Requirement: 前端工具动作文案 switch

**Reason**: 维护点下沉到工具实现，新增工具/动作无需改前端，且避免前端重建。
**Migration**: 由后端 `tool_start` 记录的 `action_text` 字段替代；旧数据（历史会话回放）仍只展示工具名与状态，不受影响。
