# 修复代码演示容器滚动条不显示的问题

## 问题

代码高亮演示容器（`.code-preview`）的滚动条完全不可见，无法通过鼠标发现可滚动区域。

## 根因

`scrollbar.css` 第 79 行的 Firefox 兼容规则：

```css
#mainContent,
.search-results,
.ai-chat-messages,
.preset-mgr-list,
.code-preview .cm-editor .cm-scroller {
    scrollbar-width: thin;
    scrollbar-color: transparent transparent;
}
```

`.code-preview .cm-editor .cm-scroller` 被包含在此组中，设置了 `scrollbar-color: transparent transparent`。这条规则的意图是仅为 Firefox 设置透明滚动条（配合 `.scrolling` 类显隐），但 **Chromium 89+（包括 WebView2）也支持 `scrollbar-color` 属性**，因此在 WebView2 环境中，该规则将滚动块和轨道都设为透明，覆盖了全局 `::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); }` 样式，导致滚动块完全不可见。

## 修复方案

从 Firefox 规则组中移除 `.code-preview .cm-editor .cm-scroller`，让该容器使用全局默认的 `::-webkit-scrollbar-thumb` 样式（始终可见的 `var(--scrollbar-thumb)`）。

**文件：** `frontend/src/css/scrollbar.css` 第 75-82 行

**修改：**
```diff
 #mainContent,
 .search-results,
 .ai-chat-messages,
-.preset-mgr-list,
-.code-preview .cm-editor .cm-scroller {
+.preset-mgr-list {
     scrollbar-width: thin;
     scrollbar-color: transparent transparent;
 }
```

保留的其他 `.code-preview` 滚动条样式不变：
- `.code-preview .cm-editor .cm-scroller::-webkit-scrollbar-button` → 隐藏按钮 ✓
- `.code-preview .cm-editor .cm-scroller::-webkit-scrollbar-track` → 透明轨道 ✓
- `.code-preview .cm-editor .cm-scroller` 上的 `scrollbar-width: thin` 被移除（受全局影响仍为 thin？不，全局 `::-webkit-scrollbar { width: 6px }` 已足够）

实际上 `scrollbar-width: thin` 在 Chromium 中也受支持。移除 `scrollbar-color: transparent transparent` 后，`scrollbar-width: thin` 仍然保持细滚动条外观。但为了更干净，我们整体移除该容器。

## 影响

| 文件 | 变更 |
|------|------|
| `frontend/src/css/scrollbar.css` | 从 Firefox 规则组中删除 `.code-preview .cm-editor .cm-scroller` |

## 验证

1. 打开设置 → 高亮主题
2. 观察代码演示容器右侧是否显示 6px 宽的细滚动条
3. 滚动验证滚动块可见
4. 悬停容器，滚动块应变为 `var(--scrollbar-thumb-hover)` 颜色
