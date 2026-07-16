# 编辑器/预览区滚动条 hover 显隐方案

## 概述

仿照 `.toc-body` 已有的 hover 显隐模式，使 CM6 编辑器和预览区的水平和垂直滚动条在鼠标不悬停时隐藏，悬停时显现。

## 当前状态

| 区域 | 滚动条 | 当前行为 | 目标行为 |
|------|--------|----------|----------|
| CM6 编辑器（`.cm-scroller`） | 垂直+水平（6px） | 始终可见 | hover 时可见 |
| 预览区（`.md-rendered`） | 垂直（6px） | 始终可见 | hover 时可见 |
| 代码块内滚动条（`.md-rendered pre`） | 水平（6px） | 始终可见 | 保持不变（代码块内需要常显） |
| TOC 侧栏（`.toc-body`） | 垂直（3px） | hover 时可见 | 不变（已有模式） |

## 修改方案

### 修改 1：CM6 编辑器滚动条 hover 显隐

**文件：** `frontend/src/css/components/editor.css`（第 757-791 行）

**改动：** 将 `.cm-scroller` 及其 `::-webkit-scrollbar-*` 规则改为默认透明，hover 编辑器区域时显示。

```css
/* 默认状态 — 滚动条透明隐藏 */
.cm-scroller {
  padding: 0 1px 1px 0;
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
  transition: scrollbar-color 0.3s ease;
}
.cm-scroller::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: 3px;
  transition: background 0.3s ease;
}

/* hover 编辑器文本区时 — 显示滚动条 */
.editor-textarea:hover .cm-scroller {
  scrollbar-color: var(--scrollbar-thumb) transparent;
}
.editor-textarea:hover .cm-scroller::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
}
.editor-textarea:hover .cm-scroller::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-thumb-hover);
}

/* 保留滚动条轨道尺寸（::-webkit-scrollbar 的 width/height 不变） */
.cm-scroller::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
```

**注意：** `.cm-scroller::-webkit-scrollbar` 的 `width: 6px; height: 6px` 保持不变（隐藏时只是透明，不改变尺寸，防止布局抖动）。`scrollbar-gutter: stable` 确保滚动条空间始终预留。

### 修改 2：预览区滚动条 hover 显隐

**文件：** `frontend/src/css/components/editor.css`（第 1003-1045 行）

**改动：** 将 `.md-rendered` 的滚动条改为默认透明，hover 预览区时显示。

```css
/* 默认状态 — 滚动条透明隐藏 */
.md-rendered {
  scrollbar-color: transparent transparent;
  transition: scrollbar-color 0.3s ease;
}
.md-rendered::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: 3px;
  transition: background 0.3s ease;
}

/* hover 预览区时 — 显示滚动条 */
.md-rendered:hover {
  scrollbar-color: var(--scrollbar-thumb) transparent;
}
.md-rendered:hover::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
}
.md-rendered:hover::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-thumb-hover);
}
```

**注意：** 代码块（`pre`) 内的滚动条不变 — 保持原样式以便代码块内左右滚动时始终可见。

### 不动的内容

- `.md-rendered pre::-webkit-scrollbar-*` - 代码块内滚动条不隐藏
- 全局 `scrollbar.css` 的样式不变
- `.toc-body` 的 hover 模式不变（已经是 hover 显隐）
- 滚动条尺寸（6px）、圆角、轨道背景色不变
- `.editor-textarea` 的 `scrollbar-width: thin` 保留

## 验证步骤

1. 构建前端：`cd frontend && npm run build`
2. 运行应用，打开一篇长笔记
3. **CM6 编辑器**：鼠标移入编辑器区域 → 滚动条出现；移出 → 滚动条消失
4. **预览区**：切换到预览模式，鼠标移入预览区 → 滚动条出现；移出 → 消失
5. **代码块**：预览区内的代码块水平滚动条始终可见