# MD 语法手册添加锚点目录

## Summary

在 MD 语法页面的标题下方，添加一个可点击的锚点目录（TOC），列出全部 10 个语法模块名称，点击后平滑滚动到对应卡片。

---

## Current State Analysis

- `#viewMdRef` 视图标准结构：`.view-header`（返回按钮 + MD 语法标题）→ `.md-ref-content`（卡片网格）
- 10 张卡片均**无 `id` 属性**，无法通过 `#hash` 或 `scrollIntoView` 定位
- 代码库已有多处使用 `scrollIntoView({ behavior: 'smooth' })` 的模式

---

## Proposed Changes

### Change 1: 给每张卡片添加 `id` 属性

**文件：** `d:\峡谷\Dev\本地项目\jot\frontend\index.html`
**范围：** 10 个 `.md-ref-card` 元素

添加语义化的 `id` 属性，格式为 `md-ref-card--{英文标识}`：

| 行号 | 卡片 | id |
|------|------|----|
| ~464 | ⌗ 标题 (Headings) | `md-ref-card--headings` |
| ~503 | 𝐁 文本样式 | `md-ref-card--text` |
| ~540 | 🔗 链接与图片 | `md-ref-card--links` |
| ~573 | 📋 列表 | `md-ref-card--lists` |
| ~616 | 💻 代码块 | `md-ref-card--code` |
| ~667 | 💬 引用 | `md-ref-card--blockquote` |
| ~710 | 📊 表格 | `md-ref-card--table` |
| ~747 | ✅ 任务列表 | `md-ref-card--tasks` |
| ~782 | ➖ 分割线 | `md-ref-card--hr` |
| ~819 | \ 转义字符 | `md-ref-card--escape` |

### Change 2: 添加 TOC HTML 结构

**文件：** `d:\峡谷\Dev\本地项目\jot\frontend\index.html`
**位置：** 在 `.view-header` 结束后、`.md-ref-content` 开始前插入

```html
<div class="md-ref-toc">
    <a class="md-ref-toc-item" href="#md-ref-card--headings">⌗ 标题</a>
    <a class="md-ref-toc-item" href="#md-ref-card--text">𝐁 文本样式</a>
    <a class="md-ref-toc-item" href="#md-ref-card--links">🔗 链接与图片</a>
    <a class="md-ref-toc-item" href="#md-ref-card--lists">📋 列表</a>
    <a class="md-ref-toc-item" href="#md-ref-card--code">💻 代码块</a>
    <a class="md-ref-toc-item" href="#md-ref-card--blockquote">💬 引用</a>
    <a class="md-ref-toc-item" href="#md-ref-card--table">📊 表格</a>
    <a class="md-ref-toc-item" href="#md-ref-card--tasks">✅ 任务列表</a>
    <a class="md-ref-toc-item" href="#md-ref-card--hr">➖ 分割线</a>
    <a class="md-ref-toc-item" href="#md-ref-card--escape">\ 转义</a>
</div>
```

设计要点：
- 使用 `<a href="#id">` 原生跳转，不依赖 JS（渐进增强）
- 图标 + 短名称（省略括号说明文字，保持紧凑）
- 一行展示不下时自然换行（flex-wrap）

需要额外处理 sticky header 偏移：`.view-header` 是 sticky 定位（~52px 高），浏览器原生 `#hash` 跳转会挡住卡片顶部。需要 JS 做 `scroll-margin-top` 或 scroll 偏移补偿。

### Change 3: CSS 样式

**文件：** `d:\峡谷\Dev\本地项目\jot\frontend\src\css\components\md-reference.css`

```css
/* ===== 锚点目录 ===== */
.md-ref-toc {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding: 0 32px 16px;
    border-bottom: 1px solid var(--border);
    margin-bottom: 8px;
}

.md-ref-toc-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    font-size: 12px;
    color: var(--text-secondary);
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 6px;
    text-decoration: none;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: inherit;
}

.md-ref-toc-item:hover {
    background: rgba(var(--accent-rgb), 0.1);
    border-color: var(--accent-light);
    color: var(--accent);
}
```

设计要点：
- flex-wrap 布局，自动换行
- 标签风格（与卡片内标签/筛选器 chip 风格一致）
- 响应式：窄屏时 padding 随 `.md-ref-content` 缩减
- 每个 `.md-ref-card` 添加 `scroll-margin-top: 60px`（sticky header 高度 + 间距补偿，确保跳转时卡片不被标题挡住）

响应式补充：
```css
@media (max-width: 768px) {
    .md-ref-toc {
        padding: 0 16px 12px;
    }
}
```

### Change 4: 卡片添加 scroll-margin-top

在 `md-reference.css` 中为 `.md-ref-card` 添加：
```css
.md-ref-card {
    scroll-margin-top: 60px;
}
```

CSS `scroll-margin-top` 解决 sticky header 遮挡问题，与 JS 方案配合使用。

### Change 5: JS 添加平滑滚动 + 活跃态跟踪

**文件：** `d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`
**位置：** 在 `renderMdRefCards()` 末尾新增

替换原生 `#hash` 锚点跳转为 JS `scrollIntoView({ behavior: 'smooth', block: 'start' })`，提供更可控的缓动曲线。

```javascript
// 绑定 TOC 平滑滚动
document.querySelectorAll('.md-ref-toc-item').forEach(item => {
    item.addEventListener('click', (e) => {
        e.preventDefault();
        const targetId = item.getAttribute('href').slice(1);
        const target = document.getElementById(targetId);
        if (target) {
            target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            // 更新活跃态
            document.querySelectorAll('.md-ref-toc-item').forEach(el => el.classList.remove('active'));
            item.classList.add('active');
        }
    });
});
```

设计要点：
- `e.preventDefault()` 阻止 URL hash 变化，避免潜在的路由冲突
- `behavior: 'smooth'` 提供 OS 原生的缓动曲线（macOS 上是 cubic-bezier(0.25, 0.1, 0.25, 1)）
- 点击后高亮对应 TOC 项，配合 CSS `.md-ref-toc-item.active` 样式

CSS 补充活跃态样式：
```css
.md-ref-toc-item.active {
    background: var(--accent-light);
    border-color: var(--accent);
    color: var(--accent);
    font-weight: 500;
}
```

---

## Assumptions & Decisions

- 使用 CSS `scroll-margin-top` 而非 JS 处理偏移（更简单、无 CLS 风险）
- 使用原生 `<a href="#id">` 而非 JS `scrollIntoView`（渐进增强，即使 JS 加载失败也能跳转）
- TOC 硬编码在 HTML 中而非 JS 动态生成（卡片也是硬编码的，保持一致性；减少 JS 运行时开销）
- 目录项使用短名称 + 图标（省略括号说明），保持紧凑的同时保留视觉辨识度

---

## Verification

1. `npm run build` 通过
2. 切换到 MD 语法页面，标题下方可见目录行
3. 10 个目录项均显示正确名称 + 图标
4. 点击任一目录项，页面平滑滚动到对应卡片
5. 卡片顶部不被 sticky header 遮挡
6. 窗口缩窄时目录自动换行
7. 点击「返回」再进入，目录正常渲染
