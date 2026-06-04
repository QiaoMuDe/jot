# 验证标签行为：查看/编辑模式

## 结论
当前代码已完全满足需求，**无需任何修改**。

## 验证结果

### 查看模式（`readOnly=true`）
- **文件**：`frontend/src/main.js:renderTagSelector(true)` → 第 562-574 行
- **行为**：标签渲染为 `<span class="card-tag">`，无 `onclick`，`cursor: default`
- **符合要求**：✅ 不能点击跳转，只能查看

### 编辑模式（`readOnly=false`）
- **文件**：`frontend/src/main.js:renderTagSelector(false)` → 第 577-588 行
- **行为**：标签渲染为 `<div class="tag-chip">`，`onclick="window.toggleEditorTag(id)"` 用于切换选中
- **符合要求**：✅ 不能点击跳转搜索，可以添加/删除标签（选中/取消选中）

## 决策
无需改动，当前实现已匹配需求。
