# 修复：切换后缀返回查看模式后显示错误

## Summary

切换笔记后缀类型（如 `.md` → `.txt`）并返回查看模式时，视图仍显示旧后缀对应的渲染方式（Markdown 预览），需要关闭编辑器重新打开才能正确显示。

## Current State Analysis

### 问题触发路径

1. 用户在编辑模式下通过后缀编辑对话框或快速切换按钮（`M`/`T`）修改笔记后缀
2. 修改后缀→单击"返回查看模式"按钮
3. 回查看模式按钮处理器检测到变更（含后缀变更）→ 保存到后端
4. 保存后调 `openEditor(noteId, true)` 以只读模式重新打开

### 根因

回查看模式保存处理器中，保存后端后更新了本地缓存，但**遗漏了 `file_ext`**：

```js
// frontend/src/main.js ~L3896-L3901
const cached = state.notes.find(n => n.id === noteId);
if (cached) {
    cached.title = title;
    cached.content = content;
    cached.updated_at = new Date().toISOString();
    // ❌ 缺少 cached.file_ext
}
```

而 `openEditor` 在 view mode 中（L2537）从 `state.notes` 本地缓存读取后缀：

```js
const ext = (noteData && noteData.file_ext) || '.txt';
```

由于本地缓存的 `file_ext` 未被更新，`openEditor` 拿到旧后缀（`.md`），于是走 Markdown 预览分支（L2561-2584），而按预期应该走纯文本分支（L2558-2559）。

`closeEditorSafe` 走的是 `updateNote()` 路径，该函数先 `closeEditor()` 再 `loadNotes()` 全量刷新，不存在此问题。

## Proposed Changes

### 1. `frontend/src/main.js` — 回查看模式本地缓存同步

**位置**: L3897-L3901（回查看模式按钮事件处理）

**改动**: 在 `if (cached) { ... }` 块中增加 `cached.file_ext` 同步

```js
if (cached) {
    cached.title = title;
    cached.content = content;
    cached.file_ext = els.editorFileExt.textContent;  // ← 新增
    cached.updated_at = new Date().toISOString();
}
```

**原因**: 使 `openEditor(noteId, true)` 从本地缓存读到正确的后缀，从而正确选择纯文本/ Markdown 预览分支。

## Assumptions & Decisions

- `updateNote()` 路径（保存按钮/closeEditorSafe）无此问题，因为它会 `closeEditor()` + `loadNotes()` 全量刷新
- `handleAppExit` → `saveEditorContent` 路径也不经过 `openEditor`，无此问题
- 不需要额外更改后端逻辑
- 仅需同步本地缓存，无需重新从后端获取

## Verification

1. 打开一个 `.md` 笔记进入编辑模式
2. 通过 M/T 按钮切换为 `.txt`，或通过后缀编辑对话框改为 `.txt`
3. 单击"返回查看模式"
4. 预期：显示纯文本查看界面（无 Markdown 预览），后缀显示为 `.txt`
5. 重新打开该笔记确认后缀已持久化