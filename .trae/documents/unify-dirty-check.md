# 统一三处散落的笔记脏比较逻辑

## 摘要

上一功能「编辑器标题未保存改动星号」已提炼出统一函数 `isEditorDirty()`（`frontend/src/main.js` L381）。但代码中仍有三处**手工重复**的脏判定（对比 `_editSnapshot`），各自的 title/content 比较、tags `sort+JSON.stringify`、fileExt 比较公式均与 `isEditorDirty()` 完全一致。本次将这些散落判定收敛为复用 `isEditorDirty()`，消除重复、降低未来改动漂移风险。纯前端重构，不改业务行为。

## 当前状态分析（基于检索）

三处重复的脏判定，且公式与 `isEditorDirty()` 一致：

1. `updateNote`（L1133）保存前"无变更则跳过保存直接关闭"，见 L1143-L1153。
2. `closeEditorSafe`（L5295）编辑模式分支"无变更则直接关闭"，见 L5311-L5328。
3. `editorViewBtn` 点击处理器（L6382）"无变更则静默切回查看模式"，见 L6386-L6402。

关键一致性确认：
- 三处均用 `els.editorNoteTitle.value.trim()`、`getEditorContent().trim()`、`[...state.selectedTags].sort()`、`JSON.stringify(...) === JSON.stringify(snapshot.tags)`、`els.editorFileExt.textContent !== snapshot.fileExt` —— 与 `isEditorDirty()` 逐项相同。
- `isEditorDirty()` 在 `_editSnapshot === null` 时返回 `false`。上述三处保存入口仅在**编辑已有笔记**状态触发（`updateNote` 仅在 `editingNoteId` 非空时被调用；`viewBtn` 仅在从查看进入编辑后显示，此时快照必已建立；`closeEditorSafe` 查看模式已于上方提前返回、新建模式走独立分支），故 `null` 分支在此三处不可达：
  - `updateNote` 原逻辑 `if (snapshot) {...}`，null 时不拦截（继续保存）→ 替换为 `if (!isEditorDirty()) { closeEditor(); return; }` 在 null 时会跳过保存，仅当快照缺失才不同，而该状态不可达，等价。
  - `viewBtn` 原逻辑 `hasChanged = !snapshot || ...` → 统一后 `!isEditorDirty()`，null 时行为差异同上、不可达，等价。
  - `closeEditorSafe` 编辑分支原 `if (snapshot) {...} else { closeEditor(); return; }` → `if (!isEditorDirty()) { closeEditor(); return; }`，null 时原走 `closeEditor()`（不弹确认），统一后同样 `closeEditor()`，完全等价。
- `isEditorDirty()`（function 声明，L381）定义早于全部三处（L1133/L5295/L6382 + 处理器），存在函数提升，运行时无作用域问题。

## 变更方案

### 文件：`frontend/src/main.js`（唯一改动文件）

**a) `updateNote`（现 L1143-L1153）**：将手工判定块替换为对 `isEditorDirty()` 的调用。`title`/`content`（L1134-1135）保留（其后 L1157 仍用于 `UpdateNote`）；删除 `snapshot`/`currentTags`/`tagsChanged`/`extChanged` 变量。
```js
    const title = els.editorNoteTitle.value.trim();
    const content = getEditorContent().trim();
    if (!title) {
        nm.show('标题不能为空，请输入标题后再保存', 'warning');
        return;
    }
    // 保存前捕获当前编辑的笔记：保存完成时若用户已切换到其他笔记则不关闭编辑器
    const editingIdAtStart = state.editingNoteId;

    // 脏检测：无未保存改动 → 跳过保存直接关闭
    if (!isEditorDirty()) {
        closeEditor();
        return;
    }

    try {
```

**b) `closeEditorSafe` 编辑模式分支（现 L5311-L5328）**：删除整个 `snapshot` 判定块，改为 `isEditorDirty()` 取反，移除 `currentTitle`/`currentContent`/`currentTags`/`tagsChanged`/`extChanged` 变量。
```js
    } else {
        // 编辑模式：无未保存改动 → 直接关闭
        if (!isEditorDirty()) {
            closeEditor();
            return;
        }
    }
```
（保留外层 `if (!state.editingNoteId) { ... } else { ... }` 结构；新建分支 L5299-5310 不动。）

**c) `editorViewBtn` 处理器（现 L6386-L6402）**：删除 `snapshot`/`currentTags`/`tagsChanged`/`extChanged`/`hasChanged`，直接用 `isEditorDirty()` 判断；保留 `title`/`content`（其后 L6405 保存与 L6424 附近更新缓存时使用）。
```js
        const noteId = state.editingNoteId;
        if (!noteId) return;

        const title = els.editorNoteTitle.value.trim();
        const content = getEditorContent().trim();

        state.enteredFromViewMode = false;

        // 无未保存改动 → 静默切回查看模式
        if (!isEditorDirty()) {
            switchEditorReadOnly(true);
            return;
        }

        // 有变更：保存 + 通知 + 切回查看模式
        if (title && window.go?.main?.App?.UpdateNote) {
```

不动内容：
- `isEditorDirty()` / `refreshDirtyStar()`（L381-394）及全部星号刷新点保持原样。
- `createNote`、`handleAppExit`、`saveEditorContent` 不涉及此次统一。

## 假设与决策

- 三处判定公式与 `isEditorDirty()` 一致，替换后行为唯一来源为该函数，无行为改变（快照缺失分支在各入口均不可达，等价）。
- 统一后的语义口径：`isEditorDirty()` 返回 `true` = 存在未保存改动（需保存/拦截），返回 `false` = 无改动（可关闭/切回查看）。
- 仅合并现有散落逻辑，不扩展现行判定维度（不加内容长度、不改 tags 阈值等）。

## 验证步骤

1. 前端构建：`npm run build`（`frontend` 目录），随后 `wails build` 重新编译（JS 改动需重建嵌入资源）。
2. 回归路径：
   - 编辑已有笔记 → 改动标题/正文/标签/扩展名 → 点「查看」按钮：正常保存并切回查看，星号消失；无改动时点「查看」静默切回、不保存不通知。
   - 编辑有改动后点「关闭」（`editorCancelBtn`/蒙层/`editorCloseBtn`）：弹出保存确认，保存/放弃均正确关闭。
   - 保存按钮（`editorSaveBtn`）无改动时点击：直接关闭、不重复保存。
   - 新建笔记输入内容后关闭：走新建分支，正常弹保存确认（确认 `closeEditorSafe` 新建分支未受影响）。
3. 主题切换（深浅色）复查星号显示不受影响。