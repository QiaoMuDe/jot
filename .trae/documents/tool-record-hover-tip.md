# 工具调用记录悬停提示（同款自定义样式）

## 摘要

将 AI 聊天中「工具调用记录」明细行的失败/部分失败原因提示，从浏览器原生 `title` 换成与模式提示/历史摘要进度/消息统计卡同款的 `.ai-mode-tip` 悬停卡（portal + 300ms 延迟 + 视口防溢出定位）。

## 现状分析

* 工具调用记录 = `.ai-tool-summary` 折叠摘要条（实时流与历史回放共用），明细行 `.ai-tool-status-item`。

* **原生样式的唯一来源**：失败（is-error）/ 部分失败（is-warning）行的 `.ai-tool-status-text` 截断 40 字符后，完整原因通过原生 `textEl.title = title` 显示（[ai-chat.js L5663-5688](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5663-L5688)），title 内容带 `：失败：` / `：部分来源失败：` 前缀。

* **同款悬停提示体系**（可直接复用）：

  * portal `#aiModeTipPortal`（[index.html L2736-2781](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L2736-L2781)），fixed + z-index 9999 + pointer-events: none。

  * `.ai-mode-tip` 样式（[ai-chat.css L1433-1535](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1433-L1535)）：卡片、箭头、`.ai-mode-tip-title`、`.ai-mode-tip-line`、`compact` 变体（260px）、`.tip-val`。

  * `initModeTips()`（[ai-chat.js L482-717](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L482-L717)）：消息统计卡走「事件委托绑 messagesEl + 单实例动态填充」模式，含 300ms HOVER\_DELAY、`position()` 中心对齐 + 视口 clamp + 上下翻转、scroll/resize 隐藏。

  * 模块级注入 `_hideMsgStatsTip`（L25 声明 / L704 赋值 / L4506 在 resetAIChatState 调用）。

* 工具行每次工具事件都会 `buildToolStatusRows` → `listEl.innerHTML = ''` 整表重建，悬停中残留的 tip 需在重建时收起。

## 拟议变更

### 1. [frontend/index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L2774-L2781)

在 portal 内（`user-msg-stats` 卡之后）新增单实例工具原因卡：

```html
<!-- 工具调用失败原因卡（工具明细行悬停触发，initModeTips 委托填充） -->
<div class="ai-mode-tip compact" data-tip="tool-record">
    <span class="ai-mode-tip-title" id="aiToolTipTitle">调用失败</span>
    <span class="ai-mode-tip-line"><em>工具：</em><span class="tip-val" id="aiToolTipName">—</span></span>
    <span class="ai-mode-tip-line ai-tool-tip-reason" id="aiToolTipReason">—</span>
</div>
```

### 2. [frontend/src/js/ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js)

**a)** **`buildToolStatusRows()`** **的** **`itemEl()`（L5663-5697）— 数据改挂 dataset，删除原生 title**

* `title` 变量不再存带前缀文案，只做「是否有原因」标记：error/partial 且有 `rec.result` 时在 `item` 上挂：

  * `item.dataset.tipText = rec.result`（完整原因，纯文本）

  * `item.dataset.tipStatus = rec.status`（`error` / `partial`）

  * `item.dataset.tipTool = getToolLabel(rec.name)`

* 删除 `if (title) textEl.title = title;`（textEl 只保留 40 字符截断文本）。

* 聚合成功行 `aggItemEl()`、running/ok 行不变（无 dataset，不弹卡）。

**b)** **`buildToolStatusRows()`** **开头（`listEl.innerHTML = ''`** **前）收起残留卡**

```js
_hideMsgHoverTip?.(); // 行即将整表重建：收起悬停中失败原因卡，避免残留悬浮
```

**c)** **`initModeTips()`** **消息统计委托块（L560-707）扩展**

* 取新卡引用与填充元素：

```js
const toolTip = portal.querySelector('.ai-mode-tip[data-tip="tool-record"]');
const toolEls = {
    title: document.getElementById('aiToolTipTitle'),
    name: document.getElementById('aiToolTipName'),
    reason: document.getElementById('aiToolTipReason'),
};
```

* 新增填充函数：

```js
const fillToolTip = (trigger) => {
    const isError = trigger.dataset.tipStatus === 'error';
    toolEls.title.textContent = isError ? '调用失败' : '部分来源失败';
    toolEls.title.className = 'ai-mode-tip-title ' + (isError ? 'is-error' : 'is-warning');
    toolEls.name.textContent = trigger.dataset.tipTool || '—';
    toolEls.reason.textContent = trigger.dataset.tipText;
};
```

* `mouseover`（L676）与 `mouseout`（L694）的 `closest()` 选择器追加 `.ai-tool-status-item`。

* `mouseover` 处理改为三分支：工具行分支**先于** `findMsgEntry`（工具行虽在 `.ai-msg` 内能查到 entry，但不需要）：

```js
if (target.classList.contains('ai-tool-status-item')) {
    if (!toolTip || !target.dataset.tipText) return; // 无失败原因：不弹卡
    hoverTimer = setTimeout(() => { fillToolTip(target); showMsgTip(target, toolTip); }, HOVER_DELAY);
    return;
}
const entry = findMsgEntry(target);
...
```

* 复用现有 `position()`、`showMsgTip`、`hideMsgTip`、messagesEl `scroll` / `resize` 隐藏、ResizeObserver 跟随——全部无需新写。

**d) 注入变量改名（语义通用化，3 处引用同步）**

* L25 `let _hideMsgStatsTip = null;` → `let _hideMsgHoverTip = null;`（注释改为「收起消息统计/工具原因悬停卡」）

* L704 赋值、L4506 `resetAIChatState` 调用同步改名。

### 3. [frontend/src/css/components/ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1525-L1535)

在消息统计悬停卡样式块之后追加：

```css
/* ── 工具调用失败原因卡：完整原因正文，超长按 8 行截断 ── */
.ai-tool-tip-reason {
    white-space: pre-wrap;
    word-break: break-word;
    display: -webkit-box;
    -webkit-line-clamp: 8;
    -webkit-box-orient: vertical;
    overflow: hidden;
}

/* 标题随状态着色（与明细行失败/部分失败色一致） */
#aiToolTipTitle.is-error { color: var(--danger, #e5484d); }
#aiToolTipTitle.is-warning { color: var(--warning, #d97706); }
```

## 假设与决策

| 决策点          | 结论                                      | 理由                                                                |
| ------------ | --------------------------------------- | ----------------------------------------------------------------- |
| 覆盖范围         | 仅失败/部分失败行（即现有原生 title 的位置）              | 用户诉求是「替换现有原生样式」，成功/聚合/running 行无额外信息可展示                           |
| 单实例 vs 静态多实例 | 单实例动态填充（`data-tip="tool-record"`）       | 与消息统计卡同模式；工具行数量不定且频繁整表重建                                          |
| 数据载体         | 行元素 `dataset.tipText/tipStatus/tipTool` | 实时流期间 tool\_calls 尚未落 chatHistory，无法反查；dataset 与行渲染同生命周期，流式/回放都成立 |
| 触发区域         | 整个 `.ai-tool-status-item` 行             | 仅文本 span 触发区域太窄；无 `tipText` 时静默不弹                                 |
| 原因展示         | 全文 + CSS line-clamp 8 行截断               | portal 为 pointer-events:none，tip 内滚动不可行；原生 title 实际也有长度截断         |
| 卡片宽度         | 复用 `compact`（260px）                     | 与消息统计卡一致，保持「同款」                                                   |
| 标题配色         | error→danger、warning→warning            | 与明细行状态色语义一致；卡片结构仍为同款                                              |
| 300ms 延迟     | 复用 HOVER\_DELAY                         | 用户既有偏好（防划过误弹）                                                     |

## 验证步骤

1. `cd frontend && npm run lint && npm run validate:html && npm run build` 通过。
2. 启动应用，Agent 模式让 AI 调用工具：

   * 展开工具折叠条，悬停失败/部分失败行 → 300ms 后弹出同款卡片，标题/工具名/完整原因正确，箭头指向行、视口边缘不溢出。

   * 悬停成功/聚合/执行中行 → 不弹卡。

   * 快速划过多行 → 无闪烁残留；移开即收起；滚动消息区 / 缩放窗口 → 卡片立即隐藏。

   * 流式执行中悬停失败行后等下一次工具事件 → 卡片随重建立即收起，无残留悬浮。

   * 悬停 AI 耗时 / 用户 token / 模式按钮 / 压缩圆环 → 原有卡片不受影响。
3. 历史回放含失败工具调用的会话 → 同样弹出；恢复出厂/还原备份（resetAIChatState）→ 残留卡片被收起。

