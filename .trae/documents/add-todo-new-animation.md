# 新增待办入场动画方案

## 问题分析

当前 `addTodo()` 流程：

```
回车 → CreateTodo API → loadTodos() → 全部 innerHTML 替换
```

问题在于 `loadTodos()` 会**重新渲染所有条目**，导致：

* 已有条目重新跑 `todoEnter` 动画，所有条目"闪跳"一次

* 新条目和旧条目没有视觉区分，缺少"新增"的丝滑感

## 设计方案

**核心思路**: 不再整体重新渲染，而是将新条目直接插入 DOM。已有条目完全不动，只有新条目播放入场动画。

**新流程**:

```
回车 → CreateTodo API (返回新条目数据) → 清空输入 → 
直接构建新条目 DOM → prepend 到列表 → 更新统计
```

这样做的好处：

* 已有条目**完全不动**，零闪烁

* 新条目有专属入场动画

* 性能更好（不重新渲染全部）

## 具体改动

### 1. 新 CSS 动画 (`frontend/src/css/components/todo.css`)

```css
/* 新增条目入场：优雅滑入 + 淡入 */
@keyframes todoNew {
    0%   { opacity: 0; transform: translateY(-18px) scale(0.97); }
    100% { opacity: 1; transform: translateY(0) scale(1); }
}

.todo-item.todo-new { animation: todoNew 0.35s ease-out forwards; }
```

比 `todoEnter` 更夸张一些的入场：下滑距离更长 + 带轻微缩放，让新增操作有明显的"送入"感。

### 2. 修改 `addTodo()` (`frontend/src/main.js`)

```js
async function addTodo() {
    const input = els.todoInput;
    if (!input) return;
    const text = input.value.trim();
    if (!text) return;

    try {
        if (!window.go?.main?.App?.CreateTodo) return;
        const newTodo = await window.go.main.App.CreateTodo(text);
        input.value = '';

        // 直接构建新条目 DOM 插入到列表顶部
        const wrapper = document.createElement('div');
        wrapper.innerHTML = `
            <div class="todo-item todo-new" data-id="${newTodo.id}">
                <button class="todo-checkbox" onclick="toggleTodo(${newTodo.id})">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                        <polyline points="20 6 9 17 4 12"/>
                    </svg>
                </button>
                <span class="todo-text" ondblclick="editTodo(${newTodo.id})">${escapeHtml(newTodo.text)}</span>
                <button class="todo-delete-btn" onclick="deleteTodo(${newTodo.id})" title="删除">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                </button>
            </div>
        `;
        const itemEl = wrapper.firstElementChild;
        els.todoList.prepend(itemEl);

        // 隐藏空状态
        if (els.todoEmpty) els.todoEmpty.style.display = 'none';

        // 更新统计
        const allTodos = await window.go.main.App.ListTodos();
        updateTodoStats(allTodos);
    } catch (err) {
        console.error('添加待办失败:', err);
    }
}
```

### 3. 关于 `@keyframes todoNew` 的动画设计

| 阶段   | 效果                                         | 说明        |
| ---- | ------------------------------------------ | --------- |
| 0%   | `translateY(-18px) scale(0.97)`, opacity 0 | 从上方略小状态入场 |
| 100% | `translateY(0) scale(1)`, opacity 1        | 落到正常位置    |

使用 `ease-out` 缓动曲线，结尾自然减速，配合 0.35s 时长，感觉轻盈不拖沓。

## 涉及的文件

| 文件                                     | 改动                                                 |
| -------------------------------------- | -------------------------------------------------- |
| `frontend/src/css/components/todo.css` | 新增 `@keyframes todoNew` + `.todo-item.todo-new` 样式 |
| `frontend/src/main.js`                 | 重写 `addTodo()` — 不再 reload，直接 DOM 插入               |

## 验证方式

1. 在输入框中输入文字，回车 → 新条目从上方优雅滑入
2. 已有条目完全不动，不闪烁不重排
3. 空状态在有条目时正确隐藏
4. 统计栏数字正确更新
5. 添加后输入框清空

