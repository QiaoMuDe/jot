# 编辑器写作体验优化计划

## 概要

对 CM6 编辑器进行 6 项写作体验优化：面板加宽、标题区呼吸感、当前行高亮、编辑/预览切换动画、预览区字体行高对齐。

***

## 改动清单

### 1. 面板默认宽度：560px → 680px

**文件**: `frontend/src/css/components/editor.css` 第 31 行

```css
/* 之前 */
width: 560px;
/* 之后 */
width: 680px;
```

**原因**: 560px 一行仅容 25-30 中文字，写作时频繁换行。680px 接近 A4 纸文字区宽度，也是主流写作工具的常用宽度。

***

### 2. 标题区域加大呼吸空间

**文件**: `frontend/src/css/components/editor.css` 第 292-305 行

```css
/* 之前 */
padding: 2px 20px;
margin-bottom: 16px;

/* 之后 */
padding: 8px 20px;
margin-bottom: 24px;
```

**原因**: 标题与正文之间太紧凑，加大 padding 和 margin 让标题更独立、更舒展。

***

### 3. 当前行高亮加强

**文件**: `frontend/src/js/cm6-syntax-highlight.js` 第 146-148 行

```js
// 之前
backgroundColor: 'rgba(var(--accent-rgb), 0.05)',
// 之后
backgroundColor: 'rgba(var(--accent-rgb), 0.1)',
```

**原因**: 0.05 几乎看不出，调到 0.1 能明显感知光标所在行，方便写作时快速定位。

***

### 4. 编辑 ↔ 预览切换增加淡入动画

**方案**: 由于当前使用 `display: none/block` 硬切换（无法做 CSS transition），改为在切换时对新显示的面板施加一个 CSS `fadeIn` 动画。

**文件 A**: `frontend/src/css/components/editor.css` — 在编辑/预览模式规则后追加：

```css
/* 切换模式时的淡入动画（仅对新显示的面板生效） */
.editor-overlay[data-mode="edit"] .editor-textarea,
.editor-overlay[data-mode="preview"] .md-rendered {
  animation: modeFadeIn 0.2s ease-out;
}

@keyframes modeFadeIn {
  from { opacity: 0; }
  to   { opacity: 1; }
}
```

**原因**: 简单有效，不改变现有 `display` 切换逻辑，仅在面板出现时添加一次淡入。每次切换 `data-mode` 时 CSS 选择器重新匹配，动画自动触发。

***

### 5. 预览区字体大小对齐编辑区

**文件**: `frontend/src/css/components/editor.css` 第 1140-1155 行

```css
/* .md-rendered 中 */
/* 之前 */
font-size: 0.938rem;
/* 之后 */
font-size: 1rem;
```

**原因**: 编辑区字体已改为 `1rem`，预览区仍为 `0.938rem`，两种模式之间字体大小不一致，切换时有"缩放"感。

***

### 6. 预览区行高对齐编辑区

**文件**: `frontend/src/css/components/editor.css` 第 1149 行

```css
/* .md-rendered 中 */
/* 之前 */
line-height: 1.7;
/* 之后 */
line-height: 1.85;
```

**原因**: 编辑区行高已改为 `1.85`，预览区仍为 `1.7`，两种模式的节奏感不一致。

***

## 不涉及的文件

* `main.js` — 无需修改 JS 逻辑

* `index.html` — 无需修改结构

* `cm6-syntax-highlight.js` — 仅修改 jotTheme 中的 activeLine（第 3 项）

## 验证方式

修改完成后逐项检查：

1. 新建笔记 → 面板宽度应为 680px（全屏前）
2. 标题输入框上下 padding 明显增大，标题与正文间距更宽松
3. 打字时当前行背景高亮可见（不再是几乎透明）
4. 编辑 ↔ 预览切换时新面板有淡入效果
5. 预览模式下正文字体大小与编辑模式一致
6. 预览模式下行距与编辑模式一致（1.85）

