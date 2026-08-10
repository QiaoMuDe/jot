# 优化 Agent 工具调用状态展示（带工具名 + 完善失败/部分失败信息）

## Summary

优化前端 Agent 工具调用状态展示，解决三个问题：

1. **开始调用文案不明确**：改为"调用「工具名」：动作"格式，明确告知正在调用哪个工具、在做什么。
2. **完成后的折叠明细看不出调用了哪个工具**：历史回放 `renderToolCalls` 每条明细带上工具名，形成简洁记录。
3. **失败 / 部分失败信息不完善**：实时状态条与历史明细统一格式，并展示失败原因 / 失败来源说明。

改动集中在 `frontend/src/js/ai-chat.js`（文案生成）与 `frontend/src/css/components/ai-chat.css`（少量样式），不动后端。

## Current State Analysis

* 实时状态条（事件驱动 `ai:tool-status`）：[showToolStatusStart / Done / Error / Partial](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2534-L2649)。

  * start 文案：web\_search 显示"正在搜索：{query}"、recall\_notes 显示"正在检索本地笔记..."、refine\_search\_query 显示"正在精炼搜索词..."——**均未明确"调用某工具"**。

  * result 文案：仅"已完成搜索 / 已完成检索 / 已获取结果"，**无工具名**。

  * 工具名映射 `TOOL_NAMES` 只含 3 个旧工具，缺 `get_current_time`。

* 历史回放（完成后折叠组件，含实时完成态渲染）：[renderToolCalls](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4059-L4188)。

  * 只渲染 `tool_start` 记录；每条明细为"✓ 已完成搜索 / ❌ 联网搜索失败 / ⚠ 部分来源失败"，**除失败态外均无工具名**，且失败/部分失败**未展示原因文本**（数据里 `tool_error`/`tool_partial` 记录的 `result` 字段实际携带原因，但渲染时丢弃）。

  * 工具名映射 `TOOL_LABELS`（L4068）与实时 `TOOL_NAMES`（L2509）**重复定义**，需要合并统一。

* 样式：[.ai-tool-status-item](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L438-L478) 已有 is-active / is-done / is-error / is-warning 四态，折叠摘要样式已有，本次仅需微调。

## Proposed Changes

### 1. `frontend/src/js/ai-chat.js`：统一工具名映射

在事件处理函数外（或文件级）定义一份全局映射，删除两处局部定义，并新增 `get_current_time`：

```js
var TOOL_LABELS = {
    web_search: '联网搜索',
    recall_notes: '笔记检索',
    refine_search_query: '搜索词精炼',
    get_current_time: '获取当前时间'
};
var getToolLabel = function(name) { return TOOL_LABELS[name] || name || '工具'; };
```

* 删除 [L2509 的 TOOL\_NAMES](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2509) 与 [L4068 的 TOOL\_LABELS](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4068)（renderToolCalls 内），统一改用全局 `getToolLabel`。

* `showToolStatusStart` 内 `getToolName` 同步改用全局 `getToolLabel`。

### 2. `frontend/src/js/ai-chat.js`：实时状态条文案（统一"调用「X」工具：动作"格式）

`showToolStatusStart`（[L2535](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2535)）label 生成改为：

| 工具                    | 文案                                                     |
| --------------------- | ------------------------------------------------------ |
| web\_search           | 有 query：`调用「联网搜索」工具：搜索 {query}`；否则 `调用「联网搜索」工具：搜索互联网`  |
| recall\_notes         | `调用「笔记检索」工具：检索本地笔记`（有 notebook\_ids 数量时 `检索 {N} 个笔记本`） |
| refine\_search\_query | `调用「搜索词精炼」工具：精炼搜索关键词`                                  |
| get\_current\_time    | `调用「获取当前时间」工具：获取当前日期时间`                                |
| 其他                    | `调用「{name}」工具：执行`                                      |

`showToolStatusDone`（[L2576](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2576)）label 统一为 `「{label}」：已完成`（移除实际取不到的"N 条结果"逻辑，保持简洁）。

`showToolStatusError`（[L2601](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2601)）label 统一为 `「{label}」：失败` + 原因（保留现截断 40 字符逻辑）。

`showToolStatusPartial`（[L2635](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2635)）label 统一为 `「{label}」：部分来源失败` + 失败来源说明。

### 3. `frontend/src/js/ai-chat.js`：历史回放明细带工具名 + 完善原因

`renderToolCalls`（[L4065](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4065)）改造：

* 第一遍扫描时，把 `failedNames` / `partialNames` 从 `{name: true}` 改为**存原因文本**：`failedDetails[name] = rec.result`（截断 40 字符）、`partialDetails[name] = rec.result`（截断 40 字符）。

* 每条 `tool_start` 明细渲染为（工具名加粗，状态文本常规）：

  * 完成：`「{label}」：已完成`

  * 失败：`「{label}」：失败：{原因前40字}`（原因来自 tool\_error 记录 result；无原因则省略冒号后部分）

  * 部分失败：`「{label}」：部分来源失败：{说明前40字}`（同上去省略）

* 实现方式：明细内创建两个 span——`ai-tool-status-name`（工具名「」包裹，加粗）+ `ai-tool-status-text`（状态与原因），替代当前单一 textEl 的 `textContent` 拼接；仍使用 `item.textContent` 语义的段落不可直接加粗，故改为 append 子 span（不引入 innerHTML，保持安全）。

### 4. `frontend/src/css/components/ai-chat.css`：工具名加粗样式

在 [.ai-tool-status-item](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L438-L478) 区域新增：

```css
.ai-tool-status-name {
    font-weight: 600;
    color: var(--text-primary);
}
```

（实时状态条 start 文案走整体 label 时无需 name span；历史明细用 name span。实时各态可保持单文本拼接，不强制加粗——若实现中实时也用 name span 则同样生效，方案不区分。）

## Assumptions & Decisions

1. **统一格式约定**：开始调用 = `调用「X」工具：动作`；完成 = `「X」：已完成`；失败 = `「X」：失败：原因`；部分失败 = `「X」：部分来源失败：说明`。工具名始终带 `「」`，清晰可辨。
2. **移除"N 条结果"动态文案**：前端 `tool_result` 的 result 是纯文本，现有 `JSON.parse` 取条数的逻辑实际不生效，改为静态"已完成"更可靠简洁。
3. **历史明细补充原因**：利用落库 `tool_calls` 中 `tool_error` / `tool_partial` 记录的 `result` 字段展示失败原因 / 失败来源，截断 40 字符避免明细过长。
4. **历史回放仅渲染 tool\_start 记录**：保持现有"结果并入最终态"策略不变，仅增强每条记录的展示信息。
5. **不改后端 / 数据契约**：`tools.Record` 结构、事件流、落库格式均不动，纯前端展示层优化。

## Verification

1. `cd frontend && npm run build` 构建通过（Vite）。
2. 手动验证（Agent 模式，dev 运行）：

   * 模型调用工具瞬间，状态条显示"调用「联网搜索」工具：搜索 xxx"等明确文案；

   * 工具完成后，折叠摘要点击展开，每条明细显示工具名（如"「联网搜索」：已完成"）；

   * 制造一次搜索失败 / 部分来源失败，确认实时与历史明细均显示"「联网搜索」：失败：原因" / "「联网搜索」：部分来源失败：说明"；

   * `get_current_time` 工具显示"调用「获取当前时间」工具"中文文案。

