# 修复 Mock 模式下笔记修改不持久化的问题

## 问题
`loadNotes()` 在 Mock 模式下每次都执行 `state.notes = getMockNotes()`，生成新的原始数据。编辑器修改标签（或标题/内容）后调用 `loadNotes()`，所有改动被覆盖。

## 修改方案

### `frontend/src/main.js`

**1. 新增可变 `mockNotes` 变量**（在 `getMockNotes()` 附近）：

```js
// Mock 数据的可变副本，确保修改可持久化
let mockNotes = null;
```

**2. 修改 `loadNotes()`** — 第 172-173 行：

```js
} else {
    console.warn('GetNotes 未绑定，使用模拟数据');
    if (!mockNotes) {
        mockNotes = getMockNotes();
    }
    state.notes = mockNotes;
}
```

（catch 分支同理）

**3. 修改 `updateNote()` 的 Mock 路径** — 确保更新的是 `mockNotes` 而非 `state.notes`：

```js
} else {
    console.warn('UpdateNote 未绑定，模拟更新');
    const note = mockNotes.find((n) => n.id === id);
    if (note) {
        note.title = title;
        note.content = content;
        note.updated_at = new Date().toISOString();
        note.tags = state.tags.filter((t) => state.selectedTags.includes(t.id));
    }
}
```

## 验证
1. 编辑笔记 → 添加/移除标签 → 保存
2. 重新打开该笔记 → 标签变更已持久化
3. 编辑器标签可正常切换选中状态
4. 新建笔记也正常工作
