# Agent 前置意图感知（阶段 1 规则层）实施计划

## Summary

在 Agent ReAct 循环前新增一层轻量规则意图分类器（L1），对当前用户输入做意图感知，产出两类消费：

1. **针对性强化提示**：按意图向 Instruction 注入一段"本请求意图识别"提示（写操作→强调 confirm 确认；模糊→强调 ask\_user 强制反问；统计→指向 get\_stats；搜索→指向 web\_search）。
2. **高置信工具裁剪**：仅对无歧义的纯闲聊/纯时间查询做内置工具裁剪（复用现有 disabled 黑名单机制），其余意图不裁剪、只注入提示，避免误伤。

定位为"感知与信号强化"层：不改动 agent.go 循环逻辑、不改工具 schema、无额外 API 调用（纯本地规则，零成本零延迟），最终决策权仍在循环内模型。

## Current State Analysis

* 用户输入处理现状（[app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2602-L2620)）：拼装 Instruction → 转换历史消息 → 读取 disabledTools → 直接调用 `AgentSvc.Run`，**无任何前置意图分类**，意图判断完全由模型在 ReAct 循环内隐式完成。

* 系统提示词含三类全局规范（本地知识优先 / ask\_user 强制调用 / 写操作强制确认），但均为"模型自觉"型软约束，无感知层强化。

* 已有可复用机制：

  * `disabledTools` 黑名单（[app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2585-L2600) 读取 → Request.DisabledTools → [agent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go#L117-L122) 转 map 过滤 buildTools），可复用为意图裁剪通道。

  * skill 提示词注入（[app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2537-L2543) `instruction.WriteString`）证明"循环前注入指令"通道现成。

  * `req.UserText` 在 agent 包可用（[agent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go#L444-L449)），但本次分类调用点放在 app.go（userText 是 CallAIAgentStream 入参，注入与裁剪点集中）。

  * `tools.BuiltinTools()`（[meta.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/meta.go#L13-L28)）返回全部内置工具名，供裁剪列表生成，避免硬编码漂移。

## Proposed Changes

### 1. 新增 `internal/agent/intent.go`（规则意图分类器）

纯本地规则实现，确定性、可单测。导出：

```go
// Intent 意图类型
type Intent int

const (
    IntentUnknown Intent = iota
    IntentChitchat  // 闲聊/问候（无任务，裁剪全部内置工具）
    IntentTime      // 纯时间/日期查询（裁剪其余内置工具）
    IntentSearch    // 联网搜索需求（注入提示）
    IntentStats     // 数据统计需求（注入提示）
    IntentNoteWrite // 笔记/待办/标签写操作（注入 confirm 强化提示）
    IntentVague     // 需求模糊（注入 ask_user 强制反问提示）
    IntentQuery     // 本地知识/笔记查询或未识别（默认）
)

// IntentResult 意图感知结果
type IntentResult struct {
    Intent  Intent   // 意图类型
    Disable []string // 高置信裁剪的工具名列表（并入禁用集合，可 nil）
    Prompt  string   // 注入 Instruction 的强化提示文本（可空串）
}

// ClassifyIntent 对用户输入做规则意图分类（阶段 1，零 API 成本）
func ClassifyIntent(userText string) IntentResult
```

**分类顺序与规则（决策完备）**：

1. **纯时间查询**（`isPureTimeQuery`）：命中时间词（现在几点/几点了/现在时间/当前时间/今天几号/今天是几号/今天星期几/今天是星期几/现在日期/今天日期）**且整句 ≤15 rune 且不含任务词** → `IntentTime`，裁剪除 `get_current_time` 外的全部内置工具，不注入提示。
2. **纯闲聊**（`isChitchat`）：整句 ≤20 rune、含问候/感谢/道别词（你好/您好/嗨/哈喽/hello/hi/早上好/中午好/下午好/晚上好/谢谢/感谢/辛苦了/再见/拜拜/晚安/好的/嗯/嗯嗯/ok/知道了/明白了）**且不含任务词** → `IntentChitchat`，裁剪全部内置工具，不注入提示。
3. **写操作**（`isWriteOp`）：动作词（创建/新建/新增/添加/修改/编辑/更新/改一下/改下/删除/删掉/移动/移到/置顶/取消置顶/打标签/加标签/移除标签/重命名/改名）× 对象词（笔记/待办/任务/todo/笔记本/标签）任一组合命中 → `IntentNoteWrite`，**不裁剪**，注入 confirm 强化提示。
4. **模糊需求**（`isVague`）：命中模糊词（随便/都行/都可以/你看着办/你看情况/你来定/你决定/整理一下/简单弄一下/大概/差不多/帮我看看）→ `IntentVague`，**不裁剪**，注入 ask\_user 强制反问提示。
5. **统计需求**（`isStatsQuery`）：命中统计词（统计/汇总/概览/一共/总共/多少篇/几条/几个/数量）→ `IntentStats`，**不裁剪**，注入 get\_stats 提示。
6. **联网搜索**（`isSearchQuery`）：**先排除**时间词与本地对象词（笔记/待办/标签/笔记本），命中搜索词（搜索/搜一下/查一下/查询/查查/搜索一下/联网/网上/最新/新闻/资讯/天气/实时）→ `IntentSearch`，**不裁剪**，注入 web\_search 提示。
7. 其余 → `IntentQuery` / `IntentUnknown`，无注入无裁剪。

**辅助函数**：

* `builtinToolNames()`：遍历 `tools.BuiltinTools()` 返回全部内置工具名。

* `allToolNames()` / `exceptToolNames(keep...)`：生成裁剪列表。

* `hasTaskWord(s)`：命中任务词（查/搜/创建/新建/修改/编辑/删除/移动/置顶/统计/整理/写/笔记/待办/标签/笔记本/帮等）——用于"纯"判断，防止"谢谢，帮我查天气"被误判为纯闲聊。

* `hasTimeWord(s)` / `hasLocalObj(s)`：时间词/本地对象词判断。

**注入提示文案（固定，含【本请求意图识别】前缀）**：

* write：`本条请求疑似笔记/待办/标签等写操作，执行前必须先向用户确认修改意图（确认流程见上文写操作规范），不得在未获用户同意时执行。`

* vague：`本条请求需求模糊、信息不完整，必须先调用 ask_user 工具向用户反问澄清（见上文强制调用规范），严禁猜测后直接执行。`

* stats：`本条请求为数据统计类需求，应优先调用 get_stats 工具获取统计数据，再基于数据回答。`

* search：`本条请求为联网搜索类需求，可直接调用 web_search（必要时先用 refine_search_query 精炼关键词）；是否先检索本地笔记由【本地知识优先】规范决定。`

### 2. 修改 `app.go`（CallAIAgentStream 接入点）

插入位置：[L2600](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2600)（disabledTools 读取完成）之后、L2602 日志之前，追加：

```go
// 前置意图感知（阶段 1 规则层）：对当前输入分类意图，
// 注入针对性强化提示 + 高置信工具裁剪（仅纯闲聊/纯时间查询裁剪，其余仅注入提示）
ir := agent.ClassifyIntent(userText)
if ir.Prompt != "" {
    instruction.WriteString("\n\n" + ir.Prompt)
}
if len(ir.Disable) > 0 {
    disabledTools = append(disabledTools, ir.Disable...)
    a.LogSvc.Logger.Debugw("Agent 意图感知：按意图裁剪工具",
        fastlog.String("intent", ir.Intent.String()),
        fastlog.Any("disabled", ir.Disable))
} else if a.LogSvc.Logger != nil {
    a.LogSvc.Logger.Debugw("Agent 意图感知",
        fastlog.String("intent", ir.Intent.String()))
}
```

* `disabledTools` 在 L2617 原样传入 `Request.DisabledTools`，合并后天然生效，**agent.go 零改动**。

* 需确认 `app.go` 已 import `jot/internal/agent`（已存在，引用 AgentService/Request）与 `gitee.com/MM-Q/fastlog`（已存在）。

## Assumptions & Decisions

* 裁剪是**硬约束**，因此仅对高置信无歧义意图（纯闲聊/纯时间查询，短句 + 无任务词双条件）启用；其余意图只注入提示，最终决策权保留给循环内模型。

* 闲聊裁剪只覆盖内置 12 工具；MCP 外部工具走独立装配路径（agent.go L196-L213），不在本次裁剪范围（数量少、风险低）。

* 时间查询裁剪保留 `get_current_time`，避免用户问时间时模型无工具可用。

* 分类仅基于当前轮 `userText`（不含历史），接受多轮意图漂移的局限（阶段 1 边界，后续可扩展）。

* 提示注入为追加式，不删除/覆盖既有全局规范，文本均含【本请求意图识别】前缀便于识别。

* 阶段 3（前端展示"我理解你是想……"）与 L2 模型层（低置信时按需调模型）不在本次范围。

## Verification

1. `go build ./...` 通过
2. `go vet ./internal/agent/...` 通过
3. `GetDiagnostics` 检查 app.go / intent.go 无错误
4. 手动路径（重启应用后，Agent 模式）：

   * 输入"现在几点" → 日志显示 intent=time，仅 get\_current\_time 可用

   * 输入"谢谢" → 日志显示 intent=chitchat，全部内置工具被裁剪

   * 输入"帮我把笔记 #3 改一下" → 注入写操作 confirm 强化提示（意图=write）

   * 输入"随便帮我整理一下" → 注入 ask\_user 强制反问提示（意图=vague）

   * 输入"统计一下我有多少篇笔记" → 注入 get\_stats 提示（意图=stats）

   * 输入"谢谢，顺便帮我查一下今天的新闻" → 因含任务词不判 chitchat，正常走 search/其他流程

