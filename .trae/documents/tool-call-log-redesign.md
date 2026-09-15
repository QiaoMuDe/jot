# 工具调用记录显示重设计（成功聚合 / 失败逐条）

## 摘要

修复 Agent 工具调用链展示中「多次调用时 ≥2 个报错看不到」的缺陷。根因是前端把一次工具调用错误地折叠到「按工具名唯一的一行」上：同一工具再次 `tool_start` 会重置该行，随后的成功 `tool_result` 会用 ✓ 覆盖掉先前的失败行，而失败计数 `_liveToolStats.fail` 只增不减，导致 header 显示「K 失败」但展开后没有对应 ❌ 行。

新设计统一「实时」与「历史回放」为同一条**逐调用记录**（每条调用一个独立记录）渲染：**执行成功的工具按名聚合（`✓ 名 ×N`），执行失败/部分失败的工具每条独立记录（`✗ 名 ×N · 该次原因`）**。所有计数（失败/部分/总数）由记录重算，保证 header 徽标数恒等于可见行数，彻底消除覆盖与计数不一致。

## 当前状态分析

| 位置                                        | 现状                                                                                                      | 问题              |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------- | --------------- |
| `ai-chat.js` L3431                        | `toolStatusItems` 以**工具名**为键，同名只用一行                                                                     | 多次调用互相覆盖，成功覆盖失败 |
| `ai-chat.js` L3629                        | `_liveToolStats.fail++` 只增，成功不回落                                                                        | 徽标数与可见行不一致      |
| `ai-chat.js` L5720-5879 `renderToolCalls` | 历史回放按名聚合，只保留**第一条**失败原因；「×N」混叠成功/失败                                                                     | 回放同样吞掉多次失败细节    |
| `ai-chat.js` L3685 / L3839                | `streamToolRecords`（原始事件数组）即持久化 `tool_calls`，`agent-result` 用后端权威值覆盖                                    | 可作统一数据源         |
| `ai-chat.css` L608-805                    | `.ai-tool-summary` / `.ai-tool-status-*` 已是独立折叠组件，14 主题走 `--danger/--error-bg/--warning/--accent` token | 可复用，仅需补充分节与失败强调 |

## 设计决策

1. **数据模型**：以 `streamToolRecords`（原始事件）为唯一数据源，派生 `toolRecords`（含 `status: running|ok|error|partial`、每名序号、`action_text`/`result`、实时 `startTime`）。
2. **渲染**：新增公共 `bisect/渲染` 函数，实时回调与历史回放共用：

   * **失败段**（置顶）：每条失败独立一行 `✗ 名 ×N  elapsed 失败：原因`，原因截断 40 + `title` 悬停看全（每行保留自己完整原因）。

   * **部分失败段**：同理 `⚠`（web\_search partial）。

   * **成功段**：按名聚合 `✓ 名 ×N`。
3. **计数自洽**：`fail = #{status==error}`、`partial = #{status==partial}`、`total = 记录数`，每次渲染重算，彻底取代 `_liveToolStats`。
4. **实时计时**：用一个 summary 级 `setInterval` 统一刷新所有 `running` 行耗时，取代 per-row timer（简化 + 防泄漏）。
5. **布局**：维持折叠摘要条（默认收起）、展开自然撑开、无内部滚动（沿用现状）；不新增成功段折叠开关（成功已按名聚合，行数有限）。

## 变更清单

### 1. `frontend/src/js/ai-chat.js`

**a. 新增公共函数（模块级，实时/回放共用）**

* `buildToolRecords(raw)`：输入原始事件数组，按顺序做 start↔终态（result/error/partial）配对（近期未配对同名 start 为基准；无 start 的事件防御性补记录），输出 `[{seqName, name, status, action_text, result}]`，`seqName` 为该工具第几次调用序号。

* `upsertToolSummary(anchorEl, records, {runningStartTime})`：创建/复用折叠摘要条容器，`update(records)` 重算计数并重建列表（失败/部分逐条、成功聚合），返回句柄含单一定时器；`anchorEl` 在实时 = 正文上方，在回放 = `.msg-content` 前（保持现有位置一致）。

**b. 实时路径重构**（在 `ai:tool-status` 闭包 L3677-3697）

* `ai:tool_start` 仍 `clearStreamedText()`；改：`streamToolRecords.push(payload)` → `toolRecords = buildToolRecords(streamToolRecords)`，把新 running 记录写入 `upsertToolSummary(...).update(...)` 并启动计时器。

* `tool_result/error/partial`：同样 push 进 `streamToolRecords` → 重建 `toolRecords` → `update(...)`。不再操作 `toolStatusItems`/`toolNameSeq`。

* 删除 `toolStatusItems`、`toolNameSeq`、`_liveToolStats`、`showToolStatusStart/Done/Error/Partial`、`setToolElapsed` 的相关维护逻辑（若仍供别处复用则收口到 `upsertToolSummary`）。

* `ensureToolSummary`/`updateToolSummaryHeader` 并入 `upsertToolSummary`（header 徽标由重算得到）。

**c. 回放路径重构**（`renderToolCalls` L5726）

* 解析 `tool_calls`（string|array）→ `buildToolRecords(raw)` → `upsertToolSummary(...).update(records)`。

* 移除按名聚合的 `byName` 逻辑与 `detail()` 首条保留逻辑。

**d. 保持持久化格式不变**

* `streamToolRecords` 仍以原始事件数组落库（L3959 `JSON.stringify` 不动）；`toolRecords` 仅作派生视图，不落库。

### 2. `frontend/src/css/components/ai-chat.css`

* 新增 `.ai-tool-status-section`：分节小标签（上游 `--text-muted`、小号、`margin-top` 分隔），用于「失败(K)/部分失败(P)/成功」三段。

* 强化 `.ai-tool-status-item.is-error`：行背景 `--error-bg`（柔性）、`.ai-tool-status-name` 加粗，图标+文字沿用 `--danger`——失败行在长链中突出（保持 kit token，适配 14 主题）。

* 复用既有 `.ai-tool-summary*`、`.ai-tool-status-*` 其余样式，不加全局新风。

## 假设与决策

* 失败原因**每条保留自己完整原因**（悬停查看），不合并（用户已确认）。

* 成功**按名聚合**、失败**逐条**（用户已确认核心取舍）。

* 展开**无内部滚动**（沿用现状），成功段不做额外折叠开关。

* 历史回放的耗时无持久化数据时不显示时间（优雅降级）；如需持久化耗时属后端改动，本期不做。

* 不改变持久化 `tool_calls` 格式，`streamToolRecords` 仍为落库数据源。

## 验证步骤

1. `npm run build`（前端资源）通过；`wails build`（如需整包运行）。
2. 手动（AI 助手对话框）：

   * 触发同一工具多次调用，其中 ≥1 次失败后成功 → **失败仍以 ❌ 独立行可见**，header「K 失败」= 可见 ❌ 行数（复现原 bug 的根治验证）。

   * 触发两个不同工具都失败 → 两行 ❌ 各带各自原因可见。

   * 成功多次同一工具 → 显示 `✓ 名 ×N` 单行聚合。
3. 历史回放（切会话/刷新重载旧消息）与实时行为一致。
4. 4-5 个主题切换，失败强调色、分节标签无障碍对比（`--danger/--error-bg` 等 token）。
5. `prefers-reduced-motion` 下计时/动画不闪烁。
6. 回归：单工具、零失败、纯成功链路的 header 文案与折叠交互不变。

## 参考

* 相关源码：`frontend/src/js/ai-chat.js`（L3430-3697 实时、L3949-3980 落库、L5720-5879 回放）、`frontend/src/css/components/ai-chat.css`（L608-805）。

* 设计来源：会话内根因分析 + user 决策（成功聚合/失败逐条）。

