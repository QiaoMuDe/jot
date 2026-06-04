# 修复 Mock 模式下编辑器标签不可用的问题

## 问题
后端未绑定时，`loadTags()` 和 `loadTagsForEditor()` 将 `state.tags` 设为 `[]`。编辑器标签选择器显示"暂无可用标签"，无法添加或删除标签。

## 修改方案

### `frontend/src/main.js`

**1. 新增 `getMockTags()` 函数**（在 `getMockNotes()` 附近）：

```js
function getMockTags() {
    return [
        { id: 1, name: '入门', color: '#6366f1' },
        { id: 2, name: '设计', color: '#8b5cf6' },
        { id: 3, name: '待办', color: '#f59e0b' },
    ];
}
```

（与 `getMockNotes()` 中的标签数据一致）

**2. 修改 `loadTags()`** — 第 337-338 行：

```js
console.warn('GetAllTags 未绑定');
state.tags = getMockTags();
```

**3. 修改 `loadTagsForEditor()`** — 相应的 else 分支：

```js
console.warn('GetAllTags 未绑定');
state.tags = getMockTags();
```

## 验证
1. 打开编辑器（新建/编辑笔记）→ 标签选择器显示 3 个可切换标签
2. 点击标签 → 选中/取消选中（active 样式切换）
3. 保存笔记 → 标签关联正确
4. 设置页标签列表也显示 Mock 标签
