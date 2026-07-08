# 待办条目切换/删除动画方案

## 问题分析

当前 `toggleTodo()` 和 `deleteTodo()` 的执行流程：

1. 调用后端 API
2. 立即 `loadTodos()` 完全刷新 innerHTML
3. 条目瞬间出现/消失，零过渡

## 动画设计

为三个操作设计不同的退出/入场表情：

| 操作                       | 动画感觉     | 关键词                    |
| ------------------------ | -------- | ---------------------- |
| **标记完成** (active → done) | 满足感、轻轻滑走 | 轻微弹跳 → 向左滑出 + 缩小       |
| **取消完成** (done → active) | 重新激活、回来  | 向右滑出 (离开已完成列表)         |
| **删除**                   | 丢弃、消失    | 畏缩(flinch) → 放大 → 向右飞出 |

## 具体改动

### 1. 新增 3 个 CSS 动画 (`frontend/src/css/components/todo.css`)

```css
/* 标记完成：弹跳 → 向左淡出 */
@keyframes todoComplete {
    0%   { opacity: 1; transform: scale(1) translateX(0); }
    20%  { opacity: 1; transform: scale(1.04) translateX(0); }   /* 弹跳 */
    100% { opacity: 0; transform: scale(0.92) translateX(-14px); }
}

/* 取消完成：向右淡出（离开已完成视图） */
@keyframes todoActivate {
    0%   { opacity: 1; transform: scale(1) translateX(0); }
    100% { opacity: 0; transform: scale(0.92) translateX(14px); }
}

/* 删除：畏缩 → 放大 → 向右飞出 */
@keyframes todoDelete {
    0%   { opacity: 1; transform: scale(1) translateX(0); }
    25%  { opacity: 1; transform: scale(1.06) translateX(-4px); }  /* 畏缩 */
    100% { opacity: 0; transform: scale(0.88) translateX(20px); }
}
```

对应的 class：

```css
.todo-item.todo-completing { animation: todoComplete 0.3s ease-out forwards; }
.todo-item.todo-activating { animation: todoActivate 0.25s ease-out forwards; }
.todo-item.todo-deleting   { animation: todoDelete 0.3s ease-out forwards; }
```

### 2. 修改 JS (`frontend/src/main.js`)

#### `toggleTodo(id)` — 切换完成状态

新流程：

1. 通过 `data-id` 找到当前 DOM 中的 `.todo-item`
2. 判断当前状态：`.todo-checkbox` 是否有 `.checked` class

   * 无 checked → 即将变成完成 → 加 `todo-completing` class

   * 有 checked → 即将变成激活 → 加 `todo-activating` class
3. 用 `setTimeout` 等待动画时长 (300ms)
4. 调用 `ToggleTodo` API → `loadTodos()`

```js
async function toggleTodo(id) {
    const item = els.todoList?.querySelector(`.todo-item[data-id="${id}"]`);
    if (item) {
        const isDone = item.querySelector('.todo-checkbox')?.classList.contains('checked');
        item.classList.add(isDone ? 'todo-activating' : 'todo-completing');
        await new Promise(r => setTimeout(r, 300));
    }
    try {
        if (!window.go?.main?.App?.ToggleTodo) return;
        await window.go.main.App.ToggleTodo(id);
        await loadTodos();
    } catch (err) {
        console.error('切换待办状态失败:', err);
    }
}
```

#### `deleteTodo(id)` — 删除

新流程：

1. 通过 `data-id` 找到 `.todo-item`
2. 加 `todo-deleting` class
3. 等待 300ms
4. 调用 `DeleteTodo` API → `loadTodos()`

```js
async function deleteTodo(id) {
    const item = els.todoList?.querySelector(`.todo-item[data-id="${id}"]`);
    if (item) {
        item.classList.add('todo-deleting');
        await new Promise(r => setTimeout(r, 300));
    }
    try {
        if (!window.go?.main?.App?.DeleteTodo) return;
        await window.go.main.App.DeleteTodo(id);
        await loadTodos();
    } catch (err) {
        console.error('删除待办失败:', err);
    }
}
```

### 3. 关于 `addTodo()` 的入场

`addTodo()` 直接调用 `loadTodos()` 重新渲染，而每个 `.todo-item` 已有 `todoEnter` 动画，所以新添加的条目会自动有入场动画。不需要额外修改。

## 涉及的文件

| 文件                                     | 改动                                        |
| -------------------------------------- | ----------------------------------------- |
| `frontend/src/css/components/todo.css` | 新增 3 个 `@keyframes` + 3 个 animation class |
| `frontend/src/main.js`                 | 重写 `toggleTodo()` 和 `deleteTodo()`        |

## 验证方式

1. 点击待办复选框 → 条目先弹跳再向左滑出 → 刷新列表
2. 在"已完成"视图下点击完成条目 → 条目向右滑出 → 刷新列表
3. 点击删除按钮 → 条目先畏缩再向右飞出 → 刷新列表
4. 切换/删除过程中，其他条目不受影响

