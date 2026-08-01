# 修复编辑器操作按钮在查看模式下 CM6 编辑器中的可见性

## 摘要

编辑器操作按钮（`#editorActionsBtn`）在"查看"（view）模式的 CM6 纯文本编辑器下本应隐藏，但当前实际显示。通过添加一条 CSS 规则即可修复。

## 当前状态分析

### 编辑器模式体系

存在两套独立的模式机制：

**1. 显示模式（editor-overlay mode）** — 由 `switchEditorMode()` 控制
- `data-mode="edit"`：显示 CM6 纯文本编辑器，隐藏预览
- `data-mode="preview"`：显示渲染预览，隐藏 CM6 编辑器

**2. 编辑模式（editor-view-mode）** — 由 `toggleEditorReadOnly()` 等函数控制
- `editor-view-mode` 类存在 → 只读/查看模式
- `editor-view-mode` 类不存在 → 新建/编辑模式

### 现有可见性控制

**CSS 规则** [editor.css:283-285](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css#L283-L285)：
```css
.editor-overlay[data-mode="preview"] #editorActionsBtn {
  display: none !important;
}
```
仅在预览模式下隐藏按钮，不处理查看模式。

**JS 逻辑** [main.js:224-226](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L224-L226)：
```javascript
if (els.editorActionsBtn) {
    els.editorActionsBtn.style.display = readOnly ? 'none' : '';
}
```
在 `toggleEditorReadOnly()` 中初始化时正确隐藏。

**JS 逻辑** [main.js:4644-4645](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L4644-L4645)：
```javascript
if (els.editorActionsBtn) {
    els.editorActionsBtn.style.display = mode === 'preview' ? 'none' : '';
}
```
在 `switchEditorMode()` 中**只**检查 `preview`，对于 `edit` 模式（包括查看模式下的 CM6 编辑器）会设置 `display: ''`，覆盖了之前设置的 `display: 'none'`。

### 根因

`switchEditorMode('edit')` 将按钮设为可见（`display: ''`），但未检查当前是否处于查看模式（`editor-view-mode` 类是否存在）。当用户在查看模式下从预览切回 CM6 纯文本编辑器时，按钮就会错误地显示出来。

## 变更方案

### 方案：添加 CSS 规则（推荐）

在 [editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css) 中，在现有预览模式隐藏规则（第 283 行）之后，添加一条查看模式隐藏规则：

```css
/* 查看模式下隐藏编辑器操作按钮 */
.editor-view-mode #editorActionsBtn {
  display: none !important;
}
```

**原理**：`editor-view-mode` 类在进入查看模式时添加到 [editorPanel](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L220)，在退出查看模式时移除。CSS 的 `!important` 优先级高于 `switchEditorMode` 中设置的 `display: ''` 内联样式，因此无论显示模式如何切换，按钮始终隐藏。

**为什么不需要改 JS**：`switchEditorMode` 中的 `display: ''` 会被 CSS `!important` 覆盖，无需额外修改。JS 逻辑保持现状，不影响其他场景。

### 场景覆盖验证

| 场景 | `editor-view-mode` | `data-mode` | 现有 CSS | 新增 CSS | 结果 |
|------|:---:|:---:|:---:|:---:|:---:|
| 新建笔记，CM6 编辑器 | 无 | `edit` | 不生效 | 不生效 | 显示 ✅ |
| 编辑笔记，CM6 编辑器 | 无 | `edit` | 不生效 | 不生效 | 显示 ✅ |
| 编辑笔记，预览 | 无 | `preview` | 隐藏 ✅ | 不生效 | 隐藏 ✅ |
| 查看笔记，预览 | 有 | `preview` | 隐藏 ✅ | 隐藏 ✅ | 隐藏 ✅ |
| 查看笔记，CM6 编辑器 | 有 | `edit` | 不生效 | 隐藏 ✅ | 隐藏 ✅（修复点） |

## 验证步骤

1. 构建项目，确认无编译错误
2. 测试"新建"模式：按钮应显示
3. 测试"编辑"模式：按钮应显示
4. 测试"编辑"模式预览：按钮应隐藏
5. 测试"查看"模式预览：按钮应隐藏
6. 测试"查看"模式 CM6 编辑器：按钮应隐藏（修复验证）