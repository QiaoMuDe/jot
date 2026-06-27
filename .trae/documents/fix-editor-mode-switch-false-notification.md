# 修复编辑模式切换时虚假「笔记已更新」提示

## Summary

当用户从查看模式进入编辑模式，又不做任何修改就点击"返回查看模式"时，系统仍会调用 `App.UpdateNote()` 后端写入并弹出"笔记已更新"提示。需要增加变更检测，无修改时静默切换。

---

## Current State Analysis

**入口函数：** `els.editorViewBtn` click handler (`main.js` 第 3832-3869 行)

**流程：**
```
用户点击「编辑」 → state.enteredFromViewMode = true → openEditor(noteId, false)
   → state._editSnapshot = { title, content, tags } (第 2603 行)
用户不做任何修改 → 点击「返回查看模式」
   → 读 title + content → 调用 App.UpdateNote() → nm.show('笔记已更新') → 切回查看模式
```

**问题：** `editorViewBtn` 的 click handler **没有使用** `state._editSnapshot` 做变更检测，每次点击都无条件触发后端写入 + 弹通知。

**已有的变更检测机制（可复用）：**
- `closeEditorSafe()` (第 3068-3083 行) 已有相同的 snapshot 比对逻辑：
  ```js
  const currentTitle = els.editorNoteTitle.value.trim();
  const currentContent = getEditorContent().trim();
  const currentTags = [...state.selectedTags].sort();
  const tagsChanged = JSON.stringify(currentTags) !== JSON.stringify(snapshot.tags);
  if (currentTitle === snapshot.title && currentContent === snapshot.content && !tagsChanged) {
      closeEditor();  // 无变更 → 直接关闭
  }
  ```

---

## Proposed Changes

### Change 1: `editorViewBtn` click handler 增加变更检测

**文件：** `d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`
**位置：** 第 3832-3869 行

**What：**
在读取 `title` 和 `content` 后、`App.UpdateNote()` 之前，对比 `state._editSnapshot`：

```
current title vs snapshot.title
current content vs snapshot.content
current tags vs snapshot.tags
```

- **有变更** → 走原有的保存+通知+缓存更新+模式切换流程
- **无变更** → 跳过 `App.UpdateNote()`、不弹通知、直接切回查看模式

**How（具体改动）：**

```javascript
els.editorViewBtn.addEventListener('click', async () => {
    const noteId = state.editingNoteId;
    if (!noteId) return;
    
    const snapshot = state._editSnapshot;
    const title = els.editorNoteTitle.value.trim();
    const content = getEditorContent().trim();
    const currentTags = [...state.selectedTags].sort();
    
    // 变更检测：无修改则静默切回查看模式
    const tagsChanged = snapshot ? JSON.stringify(currentTags) !== JSON.stringify(snapshot.tags) : true;
    const hasChanged = !snapshot || title !== snapshot.title || content !== snapshot.content || tagsChanged;
    
    state.enteredFromViewMode = false;
    
    if (!hasChanged) {
        // 无变更：直接切回查看模式，不弹通知
        openEditor(noteId, true);
        return;
    }
    
    // 有变更：保存 + 通知 + 切回查看模式
    if (title && window.go?.main?.App?.UpdateNote) {
        try {
            await window.go.main.App.UpdateNote(noteId, title, content, state.noteType);
            // 更新标签：先移除所有标签再重新添加选中的
            const note = await window.go.main.App.GetNote(noteId);
            if (note?.tags) {
                for (const t of note.tags) {
                    try { await window.go.main.App.RemoveTagFromNote(noteId, t.id); } catch (e) {}
                }
            }
            for (const tagId of state.selectedTags) {
                try { await window.go.main.App.AddTagToNote(noteId, tagId); } catch (e) {}
            }
        } catch (err) {
            console.error('保存失败:', err);
        }
    }
    nm.show('笔记已更新', 'success');
    // 同步更新 state.notes 中的本地缓存
    const cached = state.notes.find(n => n.id === noteId);
    if (cached) {
        cached.title = title;
        cached.content = content;
        cached.note_type = state.noteType;
        cached.updated_at = new Date().toISOString();
    }
    state._editSnapshot = null;
    openEditor(noteId, true);
    await loadNotes();
});
```

**变更要点：**
1. 提前声明 `title`/`content`/`currentTags`（在 `if (noteId)` 块顶部，不在 `if (title && ...)` 内部）
2. 新增 `snapshot` 与 `hasChanged` 判断
3. 无变更分支：只 `openEditor(noteId, true)`，跳过 save/tagsync/notify/cache update/loadNotes
4. `state.enteredFromViewMode = false` 提前到分支前，确保无论如何都清除标志
5. 有变更分支：完全保留原有保存+标签同步+通知+缓存+刷新逻辑

---

## Assumptions & Decisions

- `state._editSnapshot` 必然存在（`editorViewBtn` 只在从查看模式进入编辑后显示，此时 snapshot 已创建）
- 内容比较使用 `.trim()`（现有的 snapshot 创建时也用了 `.trim()`，保持一致）
- 标签比较用 `JSON.stringify` 序列化排序后的 ID 数组（与 `closeEditorSafe()` 一致）
- 无变更时不调用 `App.UpdateNote()` 也意味着不需要重新同步标签（标签没变）
- 无变更时不调用 `loadNotes()` 因为视图切换本身靠 `openEditor(noteId, true)` 完成，且缓存未变

---

## Verification

1. `npm run build` 通过
2. **场景 1：无任何操作** — 查看模式进入编辑 → 不做任何修改 → 点击返回查看模式 → 无通知、无后端写入、静默切换
3. **场景 2：修改标题** — 查看模式进入编辑 → 修改标题 → 点击返回查看模式 → 弹"笔记已更新"、内容已保存
4. **场景 3：修改内容** — 查看模式进入编辑 → 修改正文 → 点击返回查看模式 → 弹"笔记已更新"、内容已保存
5. **场景 4：修改标签** — 查看模式进入编辑 → 增减标签 → 点击返回查看模式 → 弹"笔记已更新"、标签已保存
6. **场景 5：输入后删回原样** — 查看模式进入编辑 → 输入文字再删掉 → 点击返回查看模式 → 无通知（内容回到 snapshot 状态）
