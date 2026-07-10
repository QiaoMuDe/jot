# 代码预览滚动条不可见 - 修复方案

## 问题描述

设置页的代码高亮预览容器（`.code-preview`）中，`.cm-scroller` 的内容超出 `max-height: 200px` 时，滚动条完全不可见。

## 当前状态分析

### 文件结构

```
.code-preview                         ← settings-panel.css: overflow: hidden
  └── .cm-editor                      ← settings-panel.css: height: auto; max-height: 200px
       └── .cm-scroller               ← settings-panel.css + CM6 主题: overflow: auto
            └── .cm-content (19行代码)
```

### 已有的滚动条样式（正确部分）

**`scrollbar.css`** 中：
- 全局 `::-webkit-scrollbar-thumb` 设置 `background: var(--scrollbar-thumb)` — 对所有 WebKit 滚动条生效
- `.code-preview` 已从所有 auto-hide 组中移除
- `.code-preview` 已从 Firefox `scrollbar-color: transparent transparent` 组中移除
- 仅保留了 hover 加亮（`var(--scrollbar-thumb-hover)`）

### 已消除的潜在原因

| 可能原因 | 状态 |
|----------|------|
| 在 Firefox `scrollbar-color: transparent` 组中 | ✅ 已修复 |
| 在 auto-hide 组中（`.scrolling` 类控制） | ✅ 已修复 |
| 全局 `::-webkit-scrollbar-thumb` 被覆盖 | ✅ 无覆盖规则 |
| CSS 变量 `--scrollbar-thumb` 为透明 | ✅ 非透明（各主题均为 rgba 半透明值） |

## 根因分析

经过排查，仍有 **两个问题** 共存导致滚动条不可见：

### 问题 1：父容器 `overflow: hidden` 裁剪滚动条

**位置**：[settings-panel.css 第 486 行](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L486)

```css
.code-preview {
    overflow: hidden;  /* ← 问题所在 */
    border-radius: var(--radius-md);
}
```

`.code-preview` 设置了 `border-radius` + `overflow: hidden`。当子元素 `.cm-scroller` 的滚动条渲染在容器边缘时，`overflow: hidden` 会将其裁剪。尤其在 Chromium/WebView2 中，自定义滚动条的 6px 宽度紧贴容器右边界，圆角区域 + overflow hidden 的组合会将滚动条完全裁剪掉。

### 问题 2：`max-height` 作用在 `.cm-editor` 而非 `.cm-scroller` 上

**位置**：[settings-panel.css 第 490-492 行](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L490-L492)

```css
.code-preview .cm-editor {
    height: auto;
    max-height: 200px;  /* ← 约束在编辑器容器上 */
}
```

CM6 的 `.cm-editor` 默认使用 flex 布局。当 `.cm-editor` 的 `height: auto` 且 `max-height: 200px` 时，flex 子项 `.cm-scroller` 的高度计算可能出现问题——某些情况下 flex 容器按内容撑开（无视 `max-height` 的约束），导致 `.cm-scroller` 的 `overflow: auto` 不会触发滚动条。

**更可靠的方式**：将 `max-height` 直接放到 `.cm-scroller` 上。

## 修改方案

### 修改 1：[settings-panel.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css)

**1a. 移除 `.code-preview` 的 `overflow: hidden`**（第 486 行）

`overflow: hidden` 原本用于配合 `border-radius` 裁剪圆角溢出。但它的副作用是裁剪了子元素的滚动条。对于代码预览容器，不需要 `overflow: hidden`，因为子元素 `max-height` 已经约束了高度，不会有内容溢出到容器之外需要裁剪。

```diff
 .code-preview {
     margin-top: 12px;
     border: 1px solid var(--border);
     border-radius: var(--radius-md);
-    overflow: hidden;
     transition: border-color 0.15s;
 }
```

**1b. 将 `max-height` 从 `.cm-editor` 移动到 `.cm-scroller`**（第 490-501 行）

这样滚动容器直接获得高度约束，确保 `overflow: auto` 能正确触发滚动条。`.cm-editor` 保留 `background` 即可。

```diff
 .code-preview .cm-editor {
-    height: auto;
-    max-height: 200px;
     background: var(--input-bg);
 }
 
 .code-preview .cm-editor .cm-scroller {
     font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
     font-size: 12px;
     padding: 10px;
     overflow: auto;
+    max-height: 200px;
 }
```

同时在 CM6 主题中同步修改（`buildCodePreview` 函数中），把 `maxHeight` 从 `'&'` 移到 `'.cm-scroller'`：

**位置**：[main.js 第 7494-7495 行](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L7494-L7495)

```diff
 EditorView.theme({
-    '&': { height: 'auto', maxHeight: '200px', backgroundColor: 'transparent' },
-    '.cm-scroller': { overflow: 'auto', fontFamily: 'Consolas, Monaco, monospace', fontSize: '12px' },
+    '&': { backgroundColor: 'transparent' },
+    '.cm-scroller': { overflow: 'auto', maxHeight: '200px', fontFamily: 'Consolas, Monaco, monospace', fontSize: '12px' },
```

### 修改 2：[scrollbar.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/scrollbar.css)

添加显式的滚动条 thumb 样式，确保不论全局规则的级联顺序如何，代码预览的滚动条 thumb 始终可见：

```css
/* 代码预览：始终显示滚动条滑块 */
.code-preview .cm-editor .cm-scroller::-webkit-scrollbar-thumb {
    background: var(--scrollbar-thumb);
}
```

## 影响范围

- **仅影响** `.code-preview` 容器（设置页的代码高亮展示区域）
- `.code-preview` 的 `border-radius` 圆角视觉效果不变——子元素 `.cm-editor` 和 `.cm-scroller` 默认不会溢出圆角，去掉 `overflow: hidden` 不会导致圆角失效
- 其他组件的滚动条行为不受影响

## 验证方式

1. 在设置页中切换代码高亮主题，确认代码预览内容超过 200px 高度时，右侧出现 6px 宽的滚动条
2. 滚动条 thumb 颜色为 `var(--scrollbar-thumb)`，hover 时为 `var(--scrollbar-thumb-hover)`
3. 滚动功能正常
4. 代码预览容器圆角正常，无内容溢出
