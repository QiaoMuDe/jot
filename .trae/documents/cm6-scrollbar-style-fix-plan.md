# CM6 编辑器滚动条样式与底部留白修复方案

## 问题概述

1. **CM6 滚动条样式不一致**：CM6 编辑器区域的垂直/水平滚动条看起来比预览区（`.md-rendered`）的滚动条更粗、更宽，缺乏预览区的"细长"精致感。
2. **水平滚动条底部留白**：CM6 编辑区底部的水平滚动条与底部状态栏（`.editor-footer`）之间存在一段空白间隙。

---

## 当前状态分析

### 问题 1：滚动条样式差异

| 区域 | 选择器 | WebKit 滚动条宽度 | Firefox `scrollbar-width` |
|------|--------|-------------------|--------------------------|
| **预览区** | `.md-rendered` | 有显式 `::-webkit-scrollbar { width: 6px }` | `scrollbar-width: thin`（直接在自身） |
| **CM6 编辑器** | `.cm-scroller` | **无显式声明**，依赖全局 `::-webkit-scrollbar` 的 6px | 无 `scrollbar-width: thin`（该属性在父级 `.editor-textarea` 上） |

虽然全局 `::-webkit-scrollbar` 也设置了 `width: 6px`，但：
- `.cm-scroller` 缺少独立的 `::-webkit-scrollbar` 显式规则，在某些 WebView2/Chromium 版本中表现不一致
- `.cm-scroller` 缺少 Firefox 的 `scrollbar-width: thin` — 该属性设置在父级 `.editor-textarea` 上，对 `.cm-scroller` 本身不生效

### 问题 2：水平滚动条底部留白

布局结构：
```
.editor-body (padding: 0 0 10px)     ← 底部 10px 内边距
  └── .editor-panes (flex: 1)
       └── .editor-textarea
            └── .cm-editor
                 └── .cm-scroller (padding-bottom: 1px)  ← 水平滚动条在此
                       ↓  约 11px 间隙  ↓
.editor-find-bar (查找替换条，或隐藏)
.editor-footer (底部状态栏，border-top: 1px)
```

间隙来源：
- `.cm-scroller` `padding-bottom: 1px`
- `.editor-body` `padding-bottom: 10px`
- 合计约 **11px** 的留白

---

## 修复方案

仅修改一个文件：`frontend/src/css/components/editor.css`

### 修改 1：为 `.cm-scroller` 添加显式滚动条样式

在第 757-759 行 `.cm-scroller` 规则块之后，追加 `::-webkit-scrollbar` 规则和 Firefox `scrollbar-width` 属性。

**编辑器文本区滚动条：匹配预览区细长样式**
```css
.cm-scroller {
  padding: 0 1px 1px 20px;
  /* 匹配预览区的细滚动条 */
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar-thumb) transparent;
}
.cm-scroller::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.cm-scroller::-webkit-scrollbar-track {
  background: transparent;
}
.cm-scroller::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
  border-radius: 3px;
}
.cm-scroller::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-thumb-hover);
}
.cm-scroller::-webkit-scrollbar-corner {
  background: transparent;
}
.cm-scroller::-webkit-scrollbar-button {
  width: 0 !important;
  height: 0 !important;
  display: none !important;
}
```

**对比预览区现有样式（第 1001-1018 行）：**
```css
.md-rendered::-webkit-scrollbar {
  width: 6px;
}
.md-rendered::-webkit-scrollbar-track {
  background: transparent;
}
.md-rendered::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
  border-radius: 3px;
}
.md-rendered::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-thumb-hover);
}
.md-rendered::-webkit-scrollbar-button {
  width: 0 !important;
  height: 0 !important;
  display: none !important;
}
```

新增的 CM6 规则与预览区完全一致，额外增加：
- `::-webkit-scrollbar-corner`：透明背景，防止角落出现白点
- `height: 6px`：显式指定水平滚动条高度
- Firefox `scrollbar-width: thin`：直接设置在 `.cm-scroller` 自身

### 修改 2：移除 `.editor-body` 底部内边距

第 132 行：
```css
/* 修改前 */
.editor-body {
  padding: 0 0 10px;
  ...
}

/* 修改后 */
.editor-body {
  padding: 0;
  ...
}
```

**原因：** 移除底部 10px padding 后，`.editor-panes` 延伸到 `.editor-body` 底部边缘，`.cm-scroller` 的底部（含水平滚动条）与 `.editor-find-bar` / `.editor-footer` 之间的间隙从 ~11px 减为仅 `.cm-scroller` 自身的 `padding-bottom: 1px`，水平滚动条更贴近底部状态栏。

### 不受影响的区域

| 元素 | 说明 |
|------|------|
| `.editor-input` 、 `.editor-section` | 不依赖 `.editor-body` 的底部 padding |
| `.md-rendered`（预览区） | 预览区不受 CM6 滚动条样式变更影响 |
| `.editor-find-bar`、`.editor-footer` | 是 `.editor-body` 的兄弟元素 |
| 全屏模式 | 布局结构不变 |

---

## 验证步骤

1. 打开编辑器，切换至**纯文本编辑模式**，检查 CM6 垂直和水平滚动条是否变细（与预览区一致）
2. 检查 CM6 底部水平滚动条是否更贴近底部状态栏
3. 切换至**预览模式**，确认预览区滚动条样式未受影响
4. 打开**查找替换条**，确认水平滚动条与查找条之间间隙合理
5. 测试**全屏模式**下滚动条表现
6. 运行 `vite build` 确认 CSS 语法无错误
