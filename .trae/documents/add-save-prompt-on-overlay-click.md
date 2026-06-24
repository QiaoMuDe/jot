# 点击蒙层关闭编辑器时弹出保存确认

## 总结

修改编辑器蒙层（overlay）点击事件。区分两种场景：
- **编辑已有笔记**：用前端快照对比判断内容是否实际改动
- **新建笔记**：内容非空即提示

## 当前状态分析

蒙层点击处理（[main.js:3879](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3879)）直接 `closeEditor()`，无保存提示。

## 方案选择：前端快照对比

放弃 CM6 原生 API（不存在 dirty 相关 API）和数据库 hash（需后端查询，成本高），采用**前端快照对比**：

进入编辑模式时，将初始状态存为快照，点击蒙层时与当前值做字符串对比：

```javascript
// 快照结构
state._editSnapshot = {
    title: string,
    content: string, 
    tags: string[]   // 已排序的标签名列表
}

// 对比逻辑
currentTitle !== snapshot.title || 
currentContent !== snapshot.content ||
JSON.stringify(sortedTags) !== JSON.stringify(snapshot.tags)
```

## 修改方案

### 1. 编辑模式下记录快照（[main.js openEditor()](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js)）

在 `openEditor()` 中，当 `isReadOnly = false`（编辑/新建模式）时：
- 如果 `state.editingNoteId` 不为 null（编辑模式）→ 记录 `state._editSnapshot = { title, content, tags }`
- 如果 `state.editingNoteId` 为 null（新建模式）→ 不记录快照，直接用内容非空判断

### 2. 清理快照

| 时机 | 操作 |
|------|------|
| `closeEditor()` cleanup | `state._editSnapshot = null` |
| 手动保存成功后 | `state._editSnapshot = null` |

### 3. 修改蒙层点击处理（[main.js:3879](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3879)）

```
点击蒙层 → 
  !editor.active → 忽略
  查看模式（保存按钮不可见） → 直接 closeEditor()
  编辑/新建模式（保存按钮可见） →
    ─ 编辑模式（editingNoteId 非空） →
      有快照且与当前一致（无改动） → 直接 closeEditor()
      有快照且与当前不一致（有改动） → 弹出保存确认
    ─ 新建模式（editingNoteId 为空） →
      标题和内容都为空 → 直接 closeEditor()
      标题或内容非空 → 弹出保存确认
    
    弹出确认后：
    ┣━ 保存 → saveEditorContent() → 清除草稿(新建) → closeEditor()
    ┣━ 不保存 → 清除草稿(新建) → closeEditor()
    ┗━ 取消 → 不操作
```

### 4. 手动保存成功后清理

- `createNote()` 成功后 → `state._editSnapshot = null`
- `updateNote()` 成功后 → `state._editSnapshot = null`
- 查看模式保存成功后 → `state._editSnapshot = null`

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/src/main.js` | 快照记录/对比/清理 + 蒙层点击处理 |

## 假设与决策

- **ESC 键**: 保持现状（直接 `closeEditor()`），仅处理蒙层点击
- **对比范围**: 标题、内容、标签三项全部对比
- **草稿清除**: 新建笔记 discard 时调用 `ClearDraft()`，与 Ctrl+Q 一致
- **快照替代 dirty 标志位**: 快照方案避免了手动管理脏标志位的麻烦（不用考虑何时重置、何时设置），且不会遗漏任何改动场景

## 验证步骤

1. 编辑已有笔记 → 不修改 → 点击蒙层 → 直接关闭
2. 编辑已有笔记 → 修改标题 → 点击蒙层 → 弹出保存确认
3. 编辑已有笔记 → 修改内容 → 点击蒙层 → 弹出保存确认
4. 编辑已有笔记 → 修改标签 → 点击蒙层 → 弹出保存确认
5. 编辑已有笔记 → 修改后手动保存 → 不继续改 → 点击蒙层 → 直接关闭
6. 编辑已有笔记 → 修改后手动保存 → 继续改 → 点击蒙层 → 弹出保存确认
7. 新建笔记 → 输入内容 → 点击蒙层 → 弹出保存确认
8. 新建笔记 → 不输入 → 点击蒙层 → 直接关闭
9. 新建笔记 → 选"保存" → 保存后关闭
10. 新建笔记 → 选"不保存" → 关闭，草稿被清除
11. 新建笔记 → 选"取消" → 不关闭
12. 查看模式 → 点击蒙层 → 直接关闭
