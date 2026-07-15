# 搜索来源 UI 优化计划

## 摘要

优化 AI 回复消息底部搜索来源的展示 UI：单一来源展示为内联引用卡片，多个来源展示为自定义折叠面板（带动画），替换 emoji 为 SVG 图标，提取共享渲染函数消除重复代码。

## 当前状态分析

### 数据流

* 后端 `SearchSource` 结构体（`internal/services/search_service.go#L14-L19`）：`title` / `url` / `content` / `source_label`（`tavily` / `zhihu_search` / `zhihu_global`）

* 前端通过 `ai:stream-done` 事件的 `sourcesJSON` 参数拿到 `streamSearchSources` 数组

* 数据库 `ai_messages.search_sources` 字段持久化，切换会话时通过 `addMessage()` 的 `searchSources` 参数恢复

### 当前渲染逻辑（两份重复代码）

1. **stream-done 回调**（`ai-chat.js#L2307-L2367`）：流式完成后渲染
2. **addMessage 函数**（`ai-chat.js#L2693-L2747`）：历史消息加载时渲染

两者都是使用原生 `<details>` + `<summary>` + 手动分组渲染，约 40 行完全相同的代码。

### 当前 CSS

* `ai-chat.css#L677-L740`：`search-sources` 相关样式（基础 border/background/links/snippet）

### 用户选择的设计方向

* 单一来源：内联引用卡片（紧凑链接预览）

* 多来源：自定义折叠面板，默认收起，点击展开

* 视觉风格：编辑器风格（VS Code 风格），扁平紧凑

## 变更清单

### 文件 1：`frontend/src/js/ai-chat.js`

#### 1.1 新增 SVG 图标常量（`~L3210`，在现有图标常量之后）

新增 6 个 SVG 图标常量：

* `SEARCH_SOURCE_ICON` — 地球网格（替代 `🌐`）

* `SATELLITE_ICON` — 卫星天线（替代 `📡` Tavily）

* `BOOK_ICON` — 书本（替代 `📖` 知乎搜索）

* `GLOBE_ICON` — 全球网络（替代 `🌍` 全网搜索）

* `EXTERNAL_ICON` — ↗ 外链图标（条目右上角）

* `CHEVRON_RIGHT_ICON` — ▶ 展开箭头（折叠头）

所有 SVG 使用 `currentColor` + `stroke-width="1.5"` + 14×14 viewBox，与现有图标体系一致。

#### 1.2 新增 `renderSearchSources(el, sources)` 共享函数（`~L3215`）

提取两份重复代码为统一函数：

```javascript
function renderSearchSources(el, sources) {
    if (!sources || sources.length === 0) return;

    if (sources.length === 1) {
        // 单一来源 → 内联引用卡片
        renderSingleSourceCard(el, sources[0]);
    } else {
        // 多来源 → 自定义折叠面板
        renderMultiSourcesPanel(el, sources);
    }
}
```

#### 1.3 实现 `renderSingleSourceCard(el, source)`（`~L3230`）

渲染结构：

```html
<div class="search-source-card">
    <div class="search-source-card-main">
        <span class="search-source-card-icon"><!-- SVG 方块图标 --></span>
        <span class="search-source-card-title">标题文本</span>
        <span class="search-source-card-link-icon"><!-- ↗ SVG --></span>
    </div>
    <div class="search-source-card-domain">example.com/path</div>
    <div class="search-source-card-snippet">摘要内容…</div>
</div>
```

* 整张卡片可点击（`click` → `window.open(url)`）

* 域名从 URL 解析（`new URL(url).hostname + pathname`），等宽字体

* 摘要 2 行渐隐截断（CSS mask-image 渐变）

* 无折叠/展开交互

#### 1.4 实现 `renderMultiSourcesPanel(el, sources)`（`~L3270`）

渲染结构：

```html
<div class="search-sources-panel">
    <!-- 折叠头 -->
    <button class="search-sources-header">
        <span class="search-sources-header-icon"><!-- 地球 SVG --></span>
        <span class="search-sources-header-text">来自 3 个来源 · 8 条结果</span>
        <span class="search-sources-header-arrow"><!-- ▶ SVG，展开时 rotate(90deg) --></span>
    </button>
    <!-- 内容区（展开/收起） -->
    <div class="search-sources-body">
        <!-- 每组的容器，由 renderSourcesGroup() 填充 -->
    </div>
</div>
```

交互：

* 点击 header 切换 `open` class

* 内容区 `max-height` + `opacity` 过渡动画（300ms ease）

* 箭头 SVG 旋转 90° 动画

#### 1.5 实现 `renderSourcesGroup(sources, groupOrder, groupLabels, startIndex)`（`~L3310`）

渲染单个来源组（复用现有分组逻辑，改样式输出）：

```html
<div class="search-sources-group">
    <div class="search-sources-group-header">
        <span class="search-sources-group-icon"><!-- 来源类型 SVG --></span>
        <span class="search-sources-group-label">Tavily</span>
        <span class="search-sources-group-count">4 条</span>
    </div>
    <!-- 条目列表 -->
    <div class="search-sources-group-items">
        <!-- 每个条目 -->
        <div class="search-sources-group-item">
            <span class="search-source-item-num">①</span>
            <div class="search-source-item-body">
                <div class="search-source-item-top">
                    <span class="search-source-item-title">标题</span>
                    <span class="search-source-item-domain">example.com</span>
                    <span class="search-source-item-link"><!-- ↗ SVG --></span>
                </div>
                <div class="search-source-item-snippet">摘要…</div>
            </div>
        </div>
    </div>
</div>
```

* 编号跨组连续（从 `startIndex + 1` 开始）

* 域名使用等宽字体

* 条目 hover：背景 `var(--hover-bg)`

* 摘要 2 行渐隐截断

#### 1.6 替换两处调用点

**stream-done 回调（`L2307-L2367`** **原代码）**：

```javascript
// 替换前
if (streamSearchSources && streamSearchSources.length > 0) {
    const details = document.createElement('details');
    // ... ~60 行 ...
}

// 替换后
renderSearchSources(streamingEl, streamSearchSources);
```

**addMessage 函数（`L2693-L2747`** **原代码）**：

```javascript
// 替换前
if (role === 'assistant' && searchSources && searchSources.length > 0) {
    const details = document.createElement('details');
    // ... ~55 行 ...
}

// 替换后
renderSearchSources(el, searchSources);
```

### 文件 2：`frontend/src/css/components/ai-chat.css`

#### 2.1 移除旧样式（`L677-L740`）

删除 `.search-sources`、`.search-sources summary`、`.search-sources[open] summary`、`.search-sources-content`、`.search-sources-group-header`、`.search-sources-item`、`.search-sources-item a`、`.search-sources-snippet` 全部旧规则。

#### 2.2 新增搜索来源卡片样式（`~L677`）

```css
/* ── 搜索来源内联卡片（单一来源） ── */
.search-source-card {
    margin: 12px 0 4px 0;
    padding: 10px 12px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-secondary);
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s;
}
.search-source-card:hover {
    border-color: var(--accent);
    background: var(--hover-bg);
}

.search-source-card-main {
    display: flex;
    align-items: baseline;
    gap: 6px;
}

.search-source-card-icon {
    flex-shrink: 0;
    color: var(--accent);
    width: 14px;
    height: 14px;
}

.search-source-card-title {
    flex: 1;
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--accent);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.search-source-card-link-icon {
    flex-shrink: 0;
    color: var(--text-muted);
    opacity: 0;
    transition: opacity 0.15s;
}
.search-source-card:hover .search-source-card-link-icon {
    opacity: 1;
}

.search-source-card-domain {
    font-size: 0.75rem;
    font-family: var(--font-mono);
    color: var(--text-muted);
    margin-top: 2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.search-source-card-snippet {
    font-size: 0.78rem;
    color: var(--text-muted);
    line-height: 1.5;
    margin-top: 4px;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
}
```

#### 2.3 新增折叠面板样式（在卡片样式后）

```css
/* ── 搜索来源折叠面板（多个来源） ── */
.search-sources-panel {
    margin: 12px 0 4px 0;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-secondary);
    overflow: hidden;
}

.search-sources-header {
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
    transition: color 0.15s, background 0.15s;
    text-align: left;
}
.search-sources-header:hover {
    color: var(--accent);
    background: var(--hover-bg);
}

.search-sources-header-icon {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
    color: var(--accent);
}

.search-sources-header-text {
    flex: 1;
    font-weight: 500;
}

.search-sources-header-arrow {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
    transition: transform 0.2s ease;
    color: var(--text-muted);
}
.search-sources-panel.open .search-sources-header-arrow {
    transform: rotate(90deg);
}

.search-sources-body {
    max-height: 0;
    overflow: hidden;
    transition: max-height 0.3s ease, opacity 0.2s ease;
    opacity: 0;
}
.search-sources-panel.open .search-sources-body {
    max-height: 2000px;
    opacity: 1;
}

/* ── 来源组 ├── */
.search-sources-group {
    padding: 0 12px;
}
.search-sources-group:not(:last-child) {
    border-bottom: 1px solid var(--border);
}

.search-sources-group-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 0 4px;
}

.search-sources-group-icon {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
    color: var(--accent);
}

.search-sources-group-label {
    font-size: 0.82rem;
    font-weight: 600;
    color: var(--text-primary);
}

.search-sources-group-count {
    font-size: 0.75rem;
    font-family: var(--font-mono);
    color: var(--text-muted);
    margin-left: auto;
}

/* ── 来源条目 ── */
.search-sources-group-items {
    padding: 4px 0 8px;
}

.search-source-group-item {
    display: flex;
    gap: 6px;
    padding: 6px 4px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.1s;
}
.search-source-group-item:hover {
    background: var(--hover-bg);
}

.search-source-item-num {
    flex-shrink: 0;
    font-size: 0.75rem;
    font-family: var(--font-mono);
    color: var(--text-muted);
    width: 16px;
    text-align: center;
    margin-top: 1px;
}

.search-source-item-body {
    flex: 1;
    min-width: 0;
}

.search-source-item-top {
    display: flex;
    align-items: baseline;
    gap: 6px;
}

.search-source-item-title {
    font-size: 0.85rem;
    color: var(--accent);
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.search-source-item-domain {
    font-size: 0.7rem;
    font-family: var(--font-mono);
    color: var(--text-muted);
    flex-shrink: 0;
}

.search-source-item-link {
    flex-shrink: 0;
    width: 12px;
    height: 12px;
    color: var(--text-muted);
    opacity: 0;
    transition: opacity 0.15s;
}
.search-source-group-item:hover .search-source-item-link {
    opacity: 1;
}

.search-source-item-snippet {
    font-size: 0.78rem;
    color: var(--text-muted);
    line-height: 1.5;
    margin-top: 2px;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 60%, transparent 100%);
}
```

### 文件 3：`AGENTS.md`

新增记忆点记录本次优化。

## 实施步骤

1. **CSS 替换**（ai-chat.css）：删除旧 `.search-sources` 样式 → 新增卡片样式 → 新增折叠面板样式
2. **新增 SVG 图标常量**（ai-chat.js）：在 `DELETE_ICON`（L3211）之后新增 6 个 SVG 常量
3. **新增共享渲染函数**（ai-chat.js）：`renderSearchSources()` + `renderSingleSourceCard()` + `renderMultiSourcesPanel()` + `renderSourcesGroup()`
4. **替换调用点**（ai-chat.js）：stream-done 回调中替换为 `renderSearchSources()` → addMessage 函数中替换为 `renderSearchSources()`
5. **更新 AGENTS.md**：新增第一百九十节记录

## 验证步骤

1. **单一来源测试**：只启用一个搜索源（如仅 Tavily），发送消息，检查 AI 回复底部是否展示内联引用卡片（非折叠面板）
2. **多来源测试**：启用多个搜索源，检查是否展示自定义折叠面板（默认收起），点击展开是否正确
3. **来源分组测试**：检查不同来源类型（Tavily/知乎/全网）是否分组展示，计数徽章是否正确
4. **交互测试**：展开/收起动画是否流畅，hover 态是否一致
5. **功能回归测试**：点击来源链接是否能正确打开浏览器，摘要截断是否正常
6. **切换会话测试**：切换会话后加载历史消息，搜索来源 UI 是否与刚完成时一致
7. **空状态测试**：无搜索来源时不显示任何来源 UI

## 假设与决策

* 不改动后端任何代码：`SearchSource` 结构体、`search_sources` 字段、数据传输链路均不变

* 不改动召回卡片 UI：本次只优化搜索来源，召回卡片保持不变

* 新建 CSS 类名不与旧类名冲突：全部使用 `search-source-*` / `search-sources-*` 新命名空间，删除旧类名

* SVG 图标复用现有项目的 stroke 风格（`stroke-width="1.5"`、`currentColor`）

