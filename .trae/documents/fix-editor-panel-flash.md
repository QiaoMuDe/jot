# 修复编辑器面板打开闪烁 & 清理残余工具栏 CSS

## Summary

两个独立问题：一是编辑器面板打开时能看到一次短暂的闪现；二是上次工具栏移除后在 `editor.css` 中残留了 2 条孤立的 CSS 规则。

---

## Current State Analysis

### 问题 1: 编辑器面板打开闪烁

**入口:** `main.js` `openEditor()` 函数 (~第 2593 行)

**顺序：**
1. `els.viewEditor.classList.add('active')` — 将视图从 `display:none` 切换到 `display:flex`（使子元素进入布局管线）
2. `initCodeMirror(...)` — 同步创建 CM6 编辑器实例（可能触发浏览器回流/重绘）
3. `overlay.style.animation = 'overlayFadeIn 0.2s...'` + `panel.style.animation = 'modalEnter 0.3s...'` — 应用渐入动画

**根因：** 步骤 1 和步骤 3 之间有一个**渲染间隙**。`.editor-panel::before` 伪元素有一条 CSS 中声明的动画 `editorBarEnter`，它在 `.editor-panel` 进入 DOM 布局树时立即触发。如果在 `initCodeMirror()` 期间浏览器执行了一次渲染帧，这个伪元素可能会在 `opacity: 0` 被覆盖之前短暂地绘制出来——形成"闪烁"。

- `.editor-overlay` 默认 CSS: `opacity: 0`
- `.editor-panel` 默认 CSS: `opacity: 0; transform: scale(0.85)`
- `.editor-panel::before` CSS: `animation: editorBarEnter 0.35s 0.05s both`（视图可见即开始）

### 问题 2: 残余工具栏 CSS

`editor.css` 第 149-157 行 `prefers-reduced-motion` 降级媒体查询中：
```css
@media (prefers-reduced-motion: reduce) {
  .editor-toolbar-btn { transition: none; }
  .heading-dropdown-panel { transition: none; }
}
```
这两个 class 在 HTML 中已删除、在 CSS 中的主样式也已删除，只剩下这里 2 条孤立规则。

---

## Proposed Changes

### Change 1: 修复编辑器面板闪烁 (`main.js`)

**文件:** `d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`
**位置:** `openEditor()` 函数内，在 `els.viewEditor.classList.add('active')` 之前

**What:**
在显示视图之前，用 `requestAnimationFrame()` 或一个微小的 `void element.offsetHeight` 强制浏览器先完成布局计算，然后再执行 `initCodeMirror()`。

**Why:**
确保 `.editor-overlay` 和 `.editor-panel` 的 `opacity: 0` 在视图可见前的**同一个渲染帧**内已生效，避免 `initCodeMirror()` 过程中的意外渲染触发伪元素动画。

**How (具体改动):**

在 `els.mainContent.style.overflow = 'hidden';` 之前，加上：

```javascript
// 先确保 overlay 和 panel 的 opacity:0 已应用
overlay.style.opacity = '0';
panel.style.opacity = '0';
// 强制回流以应用上述内联样式
void overlay.offsetHeight;
```

然后在应用动画的地方（`overlay.style.animation = 'overlayFadeIn...'` 之前），**不要**再通过 `animation` 属性控制透明度，而是：

```javascript
// 移除之前设置的内联透明度
overlay.style.opacity = '';
panel.style.opacity = '';
// 应用动画
overlay.style.animation = 'overlayFadeIn 0.2s ease-out forwards';
panel.style.animation = 'modalEnter 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards';
```

或者更简单的方案：直接在 `viewEditor.classList.add('active')` **之后、`initCodeMirror()` 之前**插入 `void overlay.offsetHeight` 强制回流，不修改现有动画逻辑：

```javascript
els.viewEditor.classList.add('active');
void overlay.offsetHeight; // <-- 强制浏览器在 initCodeMirror 之前完成 layout
initCodeMirror(...);
```

**推荐第一个方案**（显式控制 overlay/panel opacity），因为：
- 明确解决了竞态条件
- 不依赖神奇的 `offsetHeight` 技巧
- 更容易理解和维护

### Change 2: 清理残余工具栏 CSS

**文件:** `d:\峡谷\Dev\本地项目\jot\frontend\src\css\components\editor.css`
**位置:** 第 149-157 行 `prefers-reduced-motion` 媒体查询

**What:**
删除 `.editor-toolbar-btn` 和 `.heading-dropdown-panel` 两条规则。

**Why:**
两个 CSS class 对应的 HTML/JS 已全部删除，为孤立代码。

**How:**
```css
/* 原内容 */
/* prefers-reduced-motion 降级 */
@media (prefers-reduced-motion: reduce) {
  .editor-toolbar-btn {
    transition: none;
  }
  .heading-dropdown-panel {
    transition: none;
  }
}
```
替换为：
```css
/* 空 */
```
即直接删除整个 `@media` 块。

---

## Assumptions & Decisions

- 编辑器面板"闪烁"是肉眼可见的问题，不是微小的性能指标
- 使用内联 `opacity: 0` + 强制回流是最小侵入的修复方式
- 不需要修改 CSS 动画定义本身（`overlayFadeIn`、`modalEnter`、`editorBarEnter` 都没问题）
- 残余 CSS 清理不影响任何功能
- `toggleEditorFullscreen()` 中的淡出/淡入逻辑是独立功能，不在本次修复范围内

---

## Verification

1. `npm run build` 通过
2. grep 在 `main.js`、`index.html`、CSS 文件中无 `editor-toolbar-btn` 或 `heading-dropdown-panel` 残留
3. 手动验证：编辑器面板打开时无瞬间闪烁、动画流畅
