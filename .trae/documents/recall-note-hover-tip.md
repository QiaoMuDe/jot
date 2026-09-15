# 召回笔记条目悬停提示（同款自定义样式）

## 摘要

将 AI 聊天「召回笔记」折叠面板中每个笔记条目的原生 `title` 提示（完整标题 / 完整摘要），替换为与消息统计卡、工具失败原因卡同款的 `.ai-mode-tip` portal 悬停卡：事件委托 + 单实例动态填充 + 300ms 延迟 + `isConnected` 守卫，复用现有定位与隐藏机制。

## 现状分析

- **渲染**：[renderRecallCards()](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5158-L5228) 生成 `.recall-cards-panel` > `.recall-cards-header`（折叠）+ `.recall-cards-body` > `.recall-cards-item`（整行可点击 → `window.openEditor(card.id, ...)` 打开笔记）。数据模型 `card = {id, title, content, file_ext}`。
- **原生样式来源（仅两处）**：
  - [L5191](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5191) `textSpan.title = card.title` —— 标题 CSS 单行省略截断（[ai-chat.css L1080-1085](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1080-L1085)）时悬停看全文；
  - [L5206](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5206) `snippet.title = card.content` —— 摘要 3 行 clamp（[ai-chat.css L1099-1109](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1099-L1109)）时悬停看全文。
- **调用点**：历史回放 [addMessage L4349](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4349)、流结束 `ai:agent-result` [L3858](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3858)。**每个面板只渲染一次、条目从不整表重建**（区别于工具行），故无需重建收起钩子；仅需 `isConnected` 守卫防消息删除后以失效坐标弹卡。
- **可复用的同款体系**：`initModeTips()` 委托块（[L692-735](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L692-L735)）已服务 AI 统计卡 / 用户统计卡 / 工具原因卡三类，共用 `HOVER_DELAY=300ms`、`position()` 视口防溢出、scroll/resize 隐藏；工具原因卡的 `.ai-tool-tip-reason`（pre-wrap + 8 行 clamp，[ai-chat.css L1537-1545](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1537-L1545)）与召回内容展示需求完全一致，可通用化共用。

## 拟议变更

### 1. [frontend/index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L2781-L2786)

**a) 工具卡正文行类名通用化**（供两张卡共用，消除 tool 专名误导）：

```html
<span class="ai-mode-tip-line ai-tip-fulltext" id="aiToolTipReason">—</span>
```

**b) portal 末尾（tool-record 卡之后）新增召回笔记卡：**

```html
<!-- 召回笔记悬停卡（笔记条目悬停触发，initModeTips 委托填充） -->
<div class="ai-mode-tip compact" data-tip="recall-note">
    <span class="ai-mode-tip-title" id="aiRecallTipTitle">—</span>
    <span class="ai-mode-tip-line ai-tip-fulltext" id="aiRecallTipContent">—</span>
</div>
```

标题用默认 accent 色（召回无失败/成功状态，不着色）；file_ext 徽标条目行内已展示，卡中不重复。

### 2. [frontend/src/js/ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js)

**a) `renderRecallCards()`（L5174-5211）— 数据改挂 dataset，删除原生 title**
- `item` 创建后挂载：

```js
// 完整标题/内容挂行 dataset 供同款悬停卡读取（替代原生 title）
item.dataset.recallTitle = card.title || '';
if (card.content) item.dataset.recallContent = card.content;
```

- 删除 `textSpan.title = card.title;` 及注释「标题 CSS 截断时悬停查看完整内容」
- 删除 `snippet.title = card.content;` 及注释「摘要 3 行 clamp 截断时悬停查看完整内容」
- 条目不重建，无需重建收起钩子。

**b) `initModeTips()` 委托块扩展（与 tool-record 卡同模式）**
- 取卡引用与填充元素：

```js
const recallTip = portal.querySelector('.ai-mode-tip[data-tip="recall-note"]');
const recallEls = {
    title: document.getElementById('aiRecallTipTitle'),
    content: document.getElementById('aiRecallTipContent'),
};
```

- 块入口条件追加 `|| recallTip`
- 新增填充函数：

```js
/** 召回笔记悬停卡：标题行完整标题，内容行完整摘要（无内容时隐藏该行） */
const fillRecallTip = (trigger) => {
    recallEls.title.textContent = trigger.dataset.recallTitle || '—';
    const content = trigger.dataset.recallContent || '';
    recallEls.content.textContent = content;
    recallEls.content.style.display = content ? '' : 'none';
};
```

- `mouseover` / `mouseout` 的 `closest()` 选择器追加 `.recall-cards-item`（含 `relatedTarget` 检查）
- `mouseover` 新增分支（置于工具行分支之前，同样先于 `findMsgEntry`）：

```js
// 召回笔记条目：行 dataset 携带完整标题/内容，条目不重建无需重建收起
if (target.classList.contains('recall-cards-item')) {
    if (!recallTip) return;
    hoverTimer = setTimeout(() => {
        if (!target.isConnected) return; // 倒计时期间条目随消息删除：放弃弹卡
        fillRecallTip(target); showMsgTip(target, recallTip);
    }, HOVER_DELAY);
    return;
}
```

- 复用现有 `position()`、`showMsgTip`、`hideMsgTip`、scroll/resize 隐藏、ResizeObserver 跟随，无需新写。

### 3. [frontend/src/css/components/ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1537-L1545)

类名通用化（选择器与注释更新，规则不变）：

```css
/* ── 悬停卡完整正文：保留换行，超长按 8 行截断（工具失败原因卡/召回笔记卡共用） ── */
.ai-tip-fulltext { ... }
```

## 假设与决策

| 决策点 | 结论 | 理由 |
|---|---|---|
| 覆盖范围 | 全部召回条目（标题 + 摘要合并为一张卡） | 每个条目都有原生 title；合并展示比原文两处分离的 title 信息更完整 |
| 触发区域 | 整个 `.recall-cards-item` 行 | 与工具行决策一致（用户确认过"同款逻辑"）；无 dataset 场景不存在（标题必有），无需空拦截 |
| 数据载体 | 行 `dataset.recallTitle / recallContent` | 条目渲染一次后静态不变，dataset 与行同生命周期 |
| 重建收起钩子 | 不接入 `_hideToolReasonTip` | 条目从不整表重建（仅消息删除时随 DOM 移除），`isConnected` 守卫已覆盖 |
| 卡片宽度/内容截断 | 复用 `compact`（260px）+ `.ai-tip-fulltext`（8 行 clamp） | 与工具原因卡完全同款；召回摘要通常数百字符，8 行足够 |
| file_ext | 卡中不展示 | 条目行内徽标已可见，避免冗余 |
| 点击行为 | 不变（openEditor） | portal 为 pointer-events: none，悬停卡不干扰点击 |
| 类名 | `.ai-tool-tip-reason` → `.ai-tip-fulltext` | 两卡共用同一截断规则，通用命名避免误导 |

## 验证步骤

1. `cd frontend && npm run lint && npm run validate:html && npm run build` 通过。
2. 启动应用，Agent 模式触发召回（如让 AI 查询笔记）：
   - 展开召回面板，悬停任一条目 → 300ms 后弹出同款卡片：完整标题 + 完整摘要（超 8 行截断）；无摘要条目仅显示标题行。
   - 点击条目仍正常打开笔记；快速划过多条目无残留；移开 / 滚动 / 缩放窗口 → 立即收起。
   - 悬停期间删除所在消息（或触发消息 DOM 重建）→ 不弹幽灵卡。
3. 回归确认：工具失败原因卡、AI 耗时统计卡、用户 token 卡、模式按钮提示均不受影响；含召回卡片的历史会话回放同样弹出。
