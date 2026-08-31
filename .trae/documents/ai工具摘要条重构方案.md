# AI 工具调用摘要条重构方案

## 一、概述

将当前"实时全展开工具链 + 淡出消失"与"历史回放折叠摘要"两套并存的展示体系，统一为**单一的折叠摘要条组件**：固定于气泡正文上方，流式时实时更新（执行中自动展开明细），正文开始后收起保留不消失，历史回放复用同一组件同一位置。同步删除 max-height 内部滚动、虚线分隔线闪烁与整套淡出/取消/重建状态机。

## 二、现状分析

### 两套并存体系（同一份 `streamToolRecords`/落库 `tool_calls` 数据）

| <br /> | 实时工具链                                                                                                                   | 历史回放折叠摘要                                                                                                       |
| ------ | ----------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| DOM    | `.ai-tool-status-list-live`（[ai-chat.js#L2588-2595](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2588-L2595)） | `.ai-tool-summary`（[ai-chat.js#L4380-4513](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4380-L4513)） |
| 位置     | 正文**上方**（insertBefore contentDiv）                                                                                       | 正文**下方**（appendChild 消息末尾）                                                                                     |
| 形态     | 全量展开，每工具一行                                                                                                              | 折叠卡片，点击展开明细                                                                                                    |
| 生命周期   | 正文开始 → 350ms 延迟淡出 → 移除                                                                                                  | 永久保留                                                                                                           |
| 触发点    | tool\_start/result/error/partial 事件                                                                                     | done 时 renderToolCalls 重建 / 会话回放                                                                               |

### 四个核心痛点

1. **内部滚动截断**：`.ai-tool-status-list-live` 上限 `max-height: 220px` + `overflow-y: auto`（[ai-chat.css#L529-533](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L529-L533)）。深研究/多工具场景完整调用链被压进小窗口，嵌套滚动与主滚动冲突。
2. **虚线分隔线随淡出闪烁**：`border-bottom: 1px dashed`（[ai-chat.css#L524](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L524)）挂在容器上，容器淡出/重建时跟着消失出现。
3. **显示/隐藏状态机复杂**：`fadeOutToolStatusList`/`cancelToolBarFade`/`_toolBarFadeTimer`/`_toolBarFadeToken`（[ai-chat.js#L2543-2586](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2543-L2586)）+ done 兜底移除，已历两轮竞态修复。本质问题是"工具条会消失"本身引入了复杂时序。
4. **前后端观感不一致**：实时链（上方、临时、全展开）→ 切会话后变折叠摘要（下方、永久）→ 用户感知"所见非所存"。

## 三、目标设计（用户已确认三项决策）

### 结构（单一组件，复用现有 `.ai-tool-summary` 折叠卡片）

```
[⚙️] 正在调用工具…                ▾    ← 执行中：is-working + 自动展开
     web_search  搜索中  1.2s
     read_url    执行中  0.5s
            ↓ 正文开始（收起保留）
[⚙️] 已调用 3 次 · 2 个工具  1失败  ▾    ← 点击展开/收起
     ✓ web_search ×2  已完成  1.2s
     ✕ read_url ×1    失败：404  0.8s
```

### 状态演进

| 事件                             | 行为                                                                       |
| ------------------------------ | ------------------------------------------------------------------------ |
| 首个 tool\_start                 | 创建摘要条（insertBefore contentDiv），自动展开（.open），新增明细行（is-active + 200ms 耗时计时） |
| 后续 tool\_start                 | 新增/激活明细行，更新统计                                                            |
| tool\_result / error / partial | 更新该行状态（✓/✕/⚠ + 耗时），更新统计与 header 徽标                                       |
| 正文首个 chunk                     | **收起**摘要条（移除 .open），保留不消失                                                |
| stream-done                    | 无需移除（摘要条天然保留），仅在兜底路径渲染                                                   |
| 切换会话回放                         | 同一组件、同一位置（正文上方），所见即所存                                                    |

### 决策记录

* **位置**：正文上方（thinking 之下，与实时链现状一致）；回放摘要从正文下方统一上移

* **正文开始后**：收起保留，不淡出不消失

* **明细**：自然撑开（展开态 `max-height: none`，无内部滚动、无截断）

## 四、改动清单

### 1. `frontend/src/js/ai-chat.js`

#### 1.1 删除淡出状态机（当前 L2543-2586）

* 删除 `_toolBarFadeTimer`、`_toolBarFadeToken`、`fadeOutToolStatusList`、`cancelToolBarFade` 定义

* `handleStreamChunk`（当前 L2508-2516）中删除"延迟淡出"块

#### 1.2 handleStreamChunk 首 chunk 块内新增收起逻辑（当前 L2501-2507 区域）

在 `if (!hasReceivedChunk)` 块内追加：

```js
// 正文开始 → 收起工具折叠摘要（保留不消失）
if (toolSummaryEl) toolSummaryEl.classList.remove('open');
```

#### 1.3 新增流式折叠摘要（替代实时链）

在工具状态条变量区（当前 L2541-2586）改为：

* 闭包变量：`let toolSummaryEl = null`（.ai-tool-summary 容器）、`let _liveToolStats = { total: 0, names: {}, fail: 0, partial: 0 }`

* `ensureToolSummary()`：懒创建 `.ai-tool-summary`（header + body，结构与 `renderToolCalls` 一致），`streamingEl.insertBefore(summaryEl, contentDiv)`；header 点击切换 `.open` + aria-expanded，展开时 `body.style.maxHeight = 'none'`（自然撑开）、折叠时 `'0'`

* `updateToolSummaryHeader()`：按 `_liveToolStats` 更新 header 文本与徽标；执行中任一工具 is-active 时显示"正在调用工具…"+ `is-working`，全部结束显示"已调用 N 次 · X 个工具"+ 失败/部分徽标

* 修改 `showToolStatusStart`（当前 L2599-2647）：行渲染进 summary body 的 list（复用现有 `.ai-tool-status-item` 结构、toolStatusItems 缓存、200ms 计时、is-active 脉冲、`list.scrollTop = list.scrollHeight` 删除——无内部滚动）；累计 `_liveToolStats.total/names`；调 `updateToolSummaryHeader`；确保 summary 展开（.open）

* 修改 `showToolStatusDone/Error/Partial`（当前 L2655-2729）：更新行状态后累计 `_liveToolStats.fail/partial`、调 `updateToolSummaryHeader`

* 保留 `setToolElapsed`（行内耗时）与 `clearStreamedText`（中间文本清除，与本次无关）

#### 1.4 tool\_status 事件处理器（当前 L2748-2768）

* tool\_start 分支：删除 `cancelToolBarFade()`，保留 `clearStreamedText()` + `showToolStatusStart()`

#### 1.5 stream-done 处理（当前 L2883-2902、L3002-3011）

* 删除 `_toolBarFadeTimer` 清理 + 工具条兜底移除块（L2887-2902）

* 工具链渲染兜底（L3004-3011）改为：`if (streamToolRecords.length > 0 && !toolSummaryEl) { renderToolCalls(streamingEl, streamToolRecords); ... }`（流式已有摘要则跳过重建，防止重复卡片）

#### 1.6 error 处理（当前 L3035 起）

* 气泡整体移除（streamingEl.remove()），摘要随之销毁，无需新增逻辑；确认无残留 `toolStatusListEl` 引用

#### 1.7 历史回放 `renderToolCalls`（当前 L4380-4513）

* summary 插入位置：`el.appendChild(summary)` → `el.insertBefore(summary, el.querySelector('.msg-content'))`（正文上方、thinking 之下，与流式一致）

* 展开逻辑同步改为 JS 动态 max-height（`.open` 时 `body.style.maxHeight = 'none'`）

* 明细行 nameEl 文本统一：`getToolLabel(name) + ' ×' + count`（与流式明细一致）

### 2. `frontend/src/css/components/ai-chat.css`

#### 2.1 删除（当前 L529-537）

* `.ai-tool-status-list-live`（max-height 220px / overflow-y / transition）

* `.ai-tool-status-list-live.exiting`

#### 2.2 修改/新增

* `.ai-tool-summary`（L627 起）：新增执行中态 `.ai-tool-summary.is-working .ai-tool-summary-header-icon { animation: ai-icon-pulse 1.2s ... infinite }`（复用现有 keyframes，L614-617）；header 文本在执行中可加轻微强调色

* `.ai-tool-summary-body`（L695-704）：展开高度改由 JS 控制（`max-height: none`），CSS 保留 `transition: max-height .3s ease, opacity .2s ease`（折叠时 max-height→0 仍有 opacity 过渡）

* 保留 `.ai-tool-summary-body > .ai-tool-status-list`（L706-710）明细样式

* 流式摘要位于正文上方：确认 `.ai-tool-summary` 的 `margin`（现 6px 0 4px）在流式气泡内观感合适，必要时微调

### 3. 后端

* **不动**（`tool_calls` 落库与 `ai:tool-status`/`ai:agent-result` 事件协议均不变）

## 五、假设与边界

1. **流式与回放 DOM 顺序统一**：thinking（details/thinking-content）→ 工具摘要（.ai-tool-summary）→ 正文（.msg-content）。流式侧两者都 `insertBefore(contentDiv)`，先插入 thinking 后插入 summary 即可保证顺序（当前 thinking 插入于 L2479，summary 在其后）。
2. **agent-result 兜底路径**：若流式 tool\_status 事件异常缺失导致 `toolSummaryEl` 未创建，done 兜底 `renderToolCalls` 仍能渲染摘要，功能不受损。
3. **极端超长明细**（深研究 200+ 轮）：展开态 `max-height: none` 完整显示，随气泡主滚动滚动，无截断无嵌套滚动。
4. **`prefers-reduced-motion`**：现有 `@media (prefers-reduced-motion: reduce)`（L619-624、L712-717）自动生效，无需新增。
5. **plan / ask\_user 场景**：create\_plan、ask\_user 均为普通工具，走同一摘要行；ask\_user 问句文本仍由 `clearStreamedText` 清除、反问面板独立显示，不受影响。

## 六、验证步骤

1. **静态检查**：`frontend` 目录下运行 `.\node_modules\.bin\eslint.cmd src/js/ai-chat.js`，0 error（仅保留既有 2 个 warning）
2. **wails dev 实测**：

   * 多工具串行（工具间有过渡文本）：摘要条实时更新、正文开始收起、无闪烁

   * 并行工具：每行独立状态与耗时

   * ask\_user 反问：问句不出现在正文、摘要行保留、回答后正文正常

   * 工具失败（如 read\_url 404）：失败徽标 + 行内红色

   * 深度研究长链（30+ 工具）：明细自然撑开无截断、无内部滚动条

   * 切换会话回放：摘要条同位置同组件，与当时所见一致
3. **reduced-motion** 开启时过渡动画关闭、功能不受影响

