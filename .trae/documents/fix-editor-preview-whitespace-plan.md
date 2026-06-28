# 修复：笔记预览模式下标签与正文之间留白过大

## Summary

查看模式（Markdown 预览）和编辑模式下的预览模式中，标签区域与渲染后的正文内容之间存在过大的垂直留白。用户描述为"标签比较小，内容又比较靠下"。

## Current State Analysis

### 布局结构（`index.html`）

```
.editor-body (flex column)
  ├── input#editorNoteTitle  (margin-bottom: 16px)
  ├── .editor-section        (margin: 2px 0 12px)
  │   └── .tag-selector       (标签，flex-wrap)
  └── .editor-panes (flex: 1, flex column)
      ├── .editor-textarea    (CM6，预览模式 display:none)
      └── .md-rendered        (预览模式 display:block, padding-top: 0.5em)
```

### 留白来源

预览模式（`data-mode="preview"`）下，从标签区域底部到渲染内容起始的垂直间隙 = `.editor-section` 的 **`margin-bottom: 12px`** + `.md-rendered` 的 **`padding-top: 0.5em`**（约 8px）= **合计约 20px**。

在编辑模式（CM6 编辑区填满）下这 20px 不明显；但在预览模式下，标签区域很小（甚至无标签），这 20px 加上标题底部 16px margin，导致标签和正文之间产生明显的视觉断层。

### 触发场景

1. 查看模式下打开 `.md` 笔记 → `data-mode="preview"`，标签上方为正文章
2. 编辑模式下点击"预览"按钮 → `switchEditorMode('preview')`，同样 `data-mode="preview"`
3. 新建笔记后切到预览 → 同上

## Proposed Changes

仅修改 `frontend/src/css/components/editor.css`，增加两条 CSS 规则：

### 1. 预览模式下减少 `.editor-section` 底部间距

```css
.editor-overlay[data-mode="preview"] .editor-section {
  margin-bottom: 4px;
}
```

**原因**: 将 12px 缩减为 4px，消除标签与下方预览区之间多余的空白。

### 2. 预览模式下减少 `.md-rendered` 顶部内边距

```css
.editor-overlay[data-mode="preview"] .md-rendered {
  padding-top: 0.25em;
}
```

**原因**: 将 0.5em 缩减为 0.25em，让正文内容自然上移。

### 效果对比

| 元素 | 当前值 | 修改后 |
|------|--------|--------|
| `.editor-section` 底边距 | 12px | 4px |
| `.md-rendered` 上内边距 | 0.5em | 0.25em |
| **合计间隙** | **~20px** | **~8px** |

## Assumptions & Decisions

- 仅在预览模式下生效（`:data-mode="preview"`），不影响编辑模式下的布局
- 编辑模式的 `.editor-textarea`（CM6）区域内已有自身排版，无需额外调整
- 纯 CSS 改动，不涉及 JS
- `.editor-overlay[data-mode="preview"]` 选择器已存在于 CSS 中（用于控制 preview/edit 时 CM6 和 md-rendered 的显隐），直接复用即可

## Verification

1. 打开一个包含少量标签的 `.md` 笔记（查看模式）
2. 观察标签区域与渲染正文之间的间距，应与修改前有明显缩小
3. 编辑模式下切换到"预览"标签，间距同样缩小
4. 编辑模式下切回"纯文本"，布局不受影响
5. 验证构建通过