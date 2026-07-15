# 召回笔记 UI 优化计划

## 摘要

重新设计 AI 消息中的召回笔记折叠面板，对齐搜索来源面板的交互模式（`<button>` 折叠头 + 动画过渡），替换 emoji 为 SVG 图标常量，提取共享渲染函数消除重复代码，移除 JS 硬截断改为 CSS line-clamp。

## 当前状态

### 后端数据结构
`RecallCard`（`internal/services/recall_service.go`）:
```go
type RecallCard struct {
    ID        uint   `json:"id"`
    Title     string `json:"title"`
    Content   string `json:"content"`
    FileExt   string `json:"file_ext"`
    Truncated bool   `json:"truncated"`
}
```
后端通过 `ai:recall-cards` 事件发射 JSON 字符串，前端 `JSON.parse` 后暂存到 `recallCards` 变量中。

### 前端渲染（两份重复代码）

| 位置 | 行号 | 触发时机 |
|------|------|---------|
| stream-done 回调 | 2319-2363 | AI 流式完成后 |
| addMessage 函数 | 2648-2686 | 切换会话加载历史消息 |

两份代码完全相同（都是 `document.createElement('details')` + `<summary>` + emoji + 内联 SVG + JS `slice(0,100)`）。

### 当前 CSS
`ai-chat.css#L840-927`：`.recall-cards` / `.recall-cards summary` / `.recall-cards-content` / `.recall-cards-item` / `.recall-cards-snippet`

### 存在问题

| 问题 | 影响 |
|------|------|
| `📄` emoji 作图标 | emoji 字形比文字高 2-4px，导致面板高度与搜索来源不一致，不跟随主题色 |
| `<details>` 原生折叠 | 无展开/收起动画，`::marker` 默认三角箭头不可控 |
| 两份重复 ~40 行 JS | 改一处需要同步改另一处，易遗漏 |
| JS 硬截断 `slice(0,100)` | 不如 CSS `-webkit-line-clamp` 灵活，截断位置生硬 |
| 内联 SVG 硬编码 | layers 图标 SVG 字符串写在 DOM 构建代码中，不可复用 |

### 搜索来源面板已有模式（参考，非复制）

搜索来源刚完成了类似优化，采用：
- `<button>` 折叠头（而非 `<details>`）→ 完全可控
- `max-height` + `opacity` 过渡动画 → 平滑展开/收起
- `panel.classList.toggle('open')` → 统一的折叠状态管理
- SVG 常量 + `CHEVRON_RIGHT_ICON` 箭头旋转 → 无 emoji
- `renderSearchSources(el, sources)` 共享函数 → 无重复代码

召回笔记面板只需要采用**同一套结构模式**（相同的交互设计语言），保持项目内视觉统一。

## 设计方向

```
┌─ 召回笔记 · 3 篇 ──────── [▶] ─┐  ← <button> 折叠头，带图标 + 箭头
│                                  │
│ ▦ 笔记标题之一                   │  ← 可点击，打开编辑器
│   摘要摘要摘要摘要摘要摘要摘要…    │  ← CSS 3 行截断
│                                   │
│ ▦ 另一篇笔记标题                 │
│   摘要摘要摘要摘要摘要摘要摘要…    │
│                                   │
└──────────────────────────────────┘
```

- 折叠头：NOTE_ICON（layers SVG，`stroke-width="1.5"`）+ "召回笔记 (N 篇)" + 箭头
- 展开时箭头旋转 90°（与搜索来源一致）
- `max-height` 300ms + `opacity` 200ms 过渡
- 摘要去掉 JS slice，改用 CSS `-webkit-line-clamp: 3`
- 点击卡片条目调用 `window.openEditor(card.id, true, false, true)` （行为不变）
- 侧边标注 `file_ext` 徽章（`.md`/`.txt`），强化"笔记"身份

## 变更清单

### 文件 1：`frontend/src/js/ai-chat.js`

#### 1.1 新增 `NOTE_ICON` 常量（`~L3118`，跟在搜索来源图标常量之后）

从现有内联 SVG 提取为常量，`stroke-width` 从 `2` → `1.5` 对齐项目图标规范：

```javascript
const NOTE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg>';
```

#### 1.2 新增 `renderRecallCards(el, cards)` 函数（`~L3340`，跟在搜索来源函数之后）

```javascript
function renderRecallCards(el, cards) {
    if (!cards || cards.length === 0) return;

    var panel = document.createElement('div');
    panel.className = 'recall-cards-panel';

    var header = document.createElement('button');
    header.className = 'recall-cards-header';
    header.setAttribute('aria-expanded', 'false');
    header.innerHTML = '<span class="recall-cards-header-icon">' + NOTE_ICON + '</span>'
        + '<span class="recall-cards-header-text">召回笔记 (' + cards.length + ' 篇)</span>'
        + '<span class="recall-cards-header-arrow">' + CHEVRON_RIGHT_ICON + '</span>';

    var body = document.createElement('div');
    body.className = 'recall-cards-body';

    cards.forEach(function(card) {
        var item = document.createElement('div');
        item.className = 'recall-cards-item';
        item.addEventListener('click', function() {
            window.openEditor(card.id, true, false, true);
        });

        var titleRow = document.createElement('div');
        titleRow.className = 'recall-cards-item-title';
        titleRow.innerHTML = '<span class="recall-cards-item-icon">'
            + NOTE_ICON.replace('width="14"', 'width="12"').replace('height="14"', 'height="12"')
            + '</span>'
            + '<span class="recall-cards-item-text">' + escapeHtml(card.title) + '</span>'
            + (card.file_ext ? '<span class="recall-cards-item-ext">' + escapeHtml(card.file_ext) + '</span>' : '');
        item.appendChild(titleRow);

        if (card.content) {
            var snippet = document.createElement('div');
            snippet.className = 'recall-cards-snippet';
            snippet.textContent = card.content;
            item.appendChild(snippet);
        }

        body.appendChild(item);
    });

    panel.appendChild(header);
    panel.appendChild(body);
    el.appendChild(panel);

    header.addEventListener('click', function() {
        var isOpen = panel.classList.toggle('open');
        header.setAttribute('aria-expanded', isOpen);
    });
}
```

关键设计：
- `escapeHtml(card.title)` / `escapeHtml(card.file_ext)` 防止 XSS
- `NOTE_ICON` 条目图标缩为 12×12，配合标题字号
- 新增 `file_ext` 徽章（如 `.md`），强化笔记类型感知
- 移除 JS `slice(0, 100)` 截断，内容全文输出，CSS 控制截断

#### 1.3 替换 stream-done 回调中的代码（`L2319-L2363`）

删除原 `<details>` 构建代码，替换为：

```javascript
renderRecallCards(streamingEl, recallCards);
```

插入逻辑保持原样（在 `<details>` 构建代码原来的位置）。

#### 1.4 替换 addMessage 函数中的代码（`L2648-2686`）

删除原 `<details>` 构建代码，替换为：

```javascript
renderRecallCards(el, recallCards);
```

#### 1.5 确认 addMessage 第 9 参传递正确

当前调用点均已传 `msg.recall_cards`（`L1594` / `L1645`），无需改动。

### 文件 2：`frontend/src/css/components/ai-chat.css`

#### 2.1 移除旧样式（`L840-927`）

删除 `.recall-cards`、`.recall-cards summary`、`.recall-cards[open] summary`、`.recall-cards-content`、`.recall-cards-item`、`.recall-cards-item-title`、`.recall-cards-item-title-icon`、`.recall-cards-snippet` 等全部旧规则。

#### 2.2 新增样式（替换原位置 `L840` 处）

```css
/* ── 召回笔记折叠面板 ── */
.recall-cards-panel {
    margin: 12px 0 4px 0;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-secondary);
    overflow: hidden;
}

.recall-cards-header {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 8px 12px;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-muted);
    font-size: 0.82rem;
    font-family: inherit;
    text-align: left;
    transition: color 0.15s, background 0.15s;
}
.recall-cards-header:hover {
    color: var(--accent);
    background: var(--hover-bg);
}

.recall-cards-header-icon {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
    color: var(--accent);
}

.recall-cards-header-text {
    flex: 1;
    font-weight: 500;
}

.recall-cards-header-arrow {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
    transition: transform 0.2s ease;
    color: var(--text-muted);
}
.recall-cards-panel.open .recall-cards-header-arrow {
    transform: rotate(90deg);
}

.recall-cards-body {
    max-height: 0;
    overflow: hidden;
    transition: max-height 0.3s ease, opacity 0.2s ease;
    opacity: 0;
}
.recall-cards-panel.open .recall-cards-body {
    max-height: 2000px;
    opacity: 1;
}

/* ── 条目样式 ── */
.recall-cards-item {
    padding: 10px 12px;
    cursor: pointer;
    transition: background 0.1s;
    border-top: 1px solid var(--border);
}
.recall-cards-item:first-child {
    border-top: none;
}
.recall-cards-item:hover {
    background: var(--hover-bg);
}

.recall-cards-item-title {
    display: flex;
    align-items: baseline;
    gap: 6px;
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--accent);
}

.recall-cards-item-icon {
    flex-shrink: 0;
    width: 12px;
    height: 12px;
    color: var(--text-muted);
    margin-top: 2px;
}

.recall-cards-item-text {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.recall-cards-item-ext {
    flex-shrink: 0;
    font-size: 0.7rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    opacity: 0.7;
    background: var(--border);
    padding: 0 5px;
    border-radius: 3px;
    line-height: 1.4;
}

.recall-cards-snippet {
    margin: 4px 0 0 0;
    font-size: 0.78rem;
    color: var(--text-muted);
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    padding-left: 18px;  /* 12px icon + 6px gap */
}
```

### 文件 3：`AGENTS.md`

新增第一百九十一节记录：
- 替换 emoji 为 NOTE_ICON SVG 常量
- 改为 `<button>` + `max-height` 动画折叠面板
- 共享 `renderRecallCards()` 函数消除重复
- CSS line-clamp 替代 JS hard truncation
- 新增 `file_ext` 徽章显示

## 不变的部分

| 方面 | 原因 |
|------|------|
| 后端数据结构 | 不改动 recall_service.go |
| 后端事件传输 | `ai:recall-cards` 事件名不变 |
| DB 持久化 | `ai_messages.recall_cards` 字段不变 |
| 打开笔记行为 | `window.openEditor(card.id, true, false, true)` 不变 |
| 插入位置 | 均在 `.ai-msg-actions` 之前 |
| 搜索来源面板 | 不改动 |

## 实施步骤

1. **CSS**：删除旧 `.recall-cards` 样式块 → 新增 `.recall-cards-panel` 样式
2. **JS 常量**：在搜索来源图标常量后新增 `NOTE_ICON`
3. **JS 函数**：在 `renderSearchSources` 函数后新增 `renderRecallCards`
4. **JS 替换**：替换 stream-done 回调中的 `<details>` 代码为函数调用
5. **JS 替换**：替换 addMessage 函数中的 `<details>` 代码为函数调用
6. **更新 AGENTS.md**
7. **Wails 构建验证**

## 验证步骤

1. **折叠功能**：点击折叠头，面板展开/收起动画流畅
2. **打开笔记**：点击卡片条目，编辑器正确打开该笔记
3. **高度一致**：折叠态面板高度与搜索来源面板一致（不因 emoji 偏高）
4. **截断正确**：摘要 CSS 3 行截断，长内容正确显示省略号
5. **切换会话**：加载历史消息，召回卡片正确渲染
6. **无卡片**：无召回卡片时，不产生任何 DOM 节点
7. **构建通过**：`wails build` 成功

## 假设与决策

- 不改动后端任何代码
- `stroke-width` 从 2 改为 1.5 以对齐项目现有图标规范
- 移除 JS `slice(0, 100)` 截断，改为 CSS 全文 + line-clamp
- 新增 `file_ext` 徽章（如 `.md`）作为视觉增强
