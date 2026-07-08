# 待办条目编辑高度与文本截断优化计划

## 概述

修改待办清单模块，实现：
1. **日常显示**：每个条目只显示一行，超出内容截断并显示 `...`
2. **双击编辑**：条目高度自动拉高，输入框变高，提供更多编辑空间

---

## 当前状态分析

- **`.todo-text`（显示 span）**：位于 `frontend/src/css/components/todo.css` 第 192-216 行，当前 `word-break: break-word` 允许文本换行，无截断限制
- **`.todo-item`（条目容器）**：`display: flex; align-items: center`，高度由内容撑开
- **`editTodo()`（编辑函数）**：位于 `frontend/src/main.js` 第 7789-7845 行，将 `<span>` 替换为 `<input type="text">`，当前输入框 padding 为 `2px 10px`，line-height 为 `2`
- **`renderTodos()`（批量渲染）**：第 7516-7529 行，动态生成 `.todo-item` HTML
- **`buildTodoItemHTML()`（单条目构建）**：第 7643-7659 行

---

## 修改方案

### 1. `frontend/src/css/components/todo.css`

#### 1a. `.todo-text` — 添加单行截断

将当前 `word-break: break-word` 替换为单行截断：

```css
.todo-text {
    /* 保留所有现有样式，修改/新增以下属性 */
    white-space: nowrap;        /* 禁止换行 */
    overflow: hidden;           /* 隐藏溢出 */
    text-overflow: ellipsis;    /* 溢出显示... */
    /* 移除 word-break: break-word — 与 nowrap 冲突 */
}
```

`max-width: 100%` 不需要加，因为 `flex: 1` 已限制宽度。

#### 1b. 新增 `.todo-item.editing` — 编辑态样式

```css
.todo-item.editing {
    align-items: stretch;  /* 让输入框填充条目高度 */
}
```

编辑时 `align-items: center` → `stretch`，使输入框垂直填满条目容器。

### 2. `frontend/src/main.js`

#### 2a. `editTodo()` 函数 — 编辑开始时

- 在函数开头，找到 `.todo-item` 后添加 `editing` 类
- 将输入框 padding 从 `2px 10px` 改为更大值，提供更多编辑空间
- 调整 line-height 使文本展示更舒适

关键改动位置（第 7789-7812 行）：

```javascript
// 1. 添加 editing 类
item.classList.add('editing');

// 2. 输入框使用更大的 padding
input.style.padding = '12px 12px';   // 从 2px 10px → 12px 12px
input.style.lineHeight = '1.6';       // 从 2 → 1.6（更自然的行高）
```

#### 2b. `finishEdit()` — 编辑结束时

在恢复为 `<span>` 之前，移除 `.todo-item` 上的 `editing` 类：

```javascript
item.classList.remove('editing');
```

---

## 影响范围

| 文件 | 改动类型 | 说明 |
|---|---|---|
| `frontend/src/css/components/todo.css` | 修改 + 新增 | 修改 `.todo-text`，新增 `.todo-item.editing` |
| `frontend/src/main.js` | 修改 | 修改 `editTodo()` 函数 |

- 不涉及后端代码
- 不涉及 HTML 模板
- 不涉及 `renderTodos()` 和 `buildTodoItemHTML()` 的 HTML 结构

---

## 验证

1. 启动应用后，查看待办列表：长文本条目应只显示一行，结尾显示 `...`
2. 双击条目进入编辑模式：条目高度应自动拉高，输入框变高且可舒适编辑
3. 按 Enter 保存或 Escape 取消后：条目恢复为单行截断显示
