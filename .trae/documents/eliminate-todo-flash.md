# 消除待办列表刷新闪跳 — 零重渲染方案

## 问题分析

当前三个操作都存在 `loadTodos()` 导致的列表整体重渲染：

| 操作 | 现有流程 | 问题 |
|------|---------|------|
| **toggle** (已完成) | exit anim → API → **`loadTodos()`** | 全部 `innerHTML` 替换，所有条目重播 `todoEnter` |
| **delete** (已完成) | exit anim → API → **`loadTodos()`** | 同上 |
| **add** (上一轮已修复) | 直接 DOM 插入 | 无问题 |

`loadTodos()` → `renderTodos()` → `listEl.innerHTML = ...` 的后果：
- 条目消失瞬间（innerHTML 清空），然后所有条目重新出现并播放 `todoEnter`
- 视觉上就是"闪跳"

## 方案

**核心思路：三个操作全部零重渲染。操作完直接操作 DOM，不调用 `loadTodos()`。**

### toggleTodo — 新流程

根据当前筛选模式走不同路径：

```
toggleTodo(id)
├─ _todoFilter === 'all' ─── 直接切换类 (无 exit anim) + DOM 移动
│    ├─ 标记完成: 追加到列表末尾
│    └─ 取消完成: 插入到列表顶部
│    └─ API 调用 + 统计更新
│
└─ _todoFilter !== 'all' ─── 播放 exit anim → 移除 DOM → API → 统计
     ├─ 标记完成(active 筛选): todo-completing → remove
     └─ 取消完成(done 筛选): todo-activating → remove
```

### deleteTodo — 新流程

```
deleteTodo(id)
└─ todo-deleting anim → 移除 DOM → API → 统计 + 空状态
```

### addTodo — 已修复，不变

直接 DOM 插入（已有 `todo-new` 动画），无 `loadTodos()`。

### 统计更新

不再通过 `loadTodos()` 连带更新统计。改用独立函数：

```js
async function refreshTodoStats() {
    try {
        const todos = await window.go.main.App.ListTodos();
        updateTodoStats(todos);
        // 空状态检查
        const filtered = todos.filter(t => {
            if (_todoFilter === 'active') return !t.done;
            if (_todoFilter === 'done') return t.done;
            return true;
        });
        if (els.todoEmpty) {
            els.todoEmpty.style.display = filtered.length === 0 ? 'flex' : 'none';
        }
    } catch (err) {
        console.error('刷新统计失败:', err);
    }
}
```

这样所有操作后都不再重渲染列表，只更新统计数字。

## 涉及的文件

| 文件 | 改动 |
|------|------|
| `frontend/src/main.js` | 重写 `toggleTodo()`、`deleteTodo()`，新增 `refreshTodoStats()` |

**CSS 无需改动。**

## 验证方式

1. 在"全部"筛选下标记完 → 条目原地切换样式 + 移动到列表底部，无闪烁
2. 在"全部"筛选下取消完成 → 条目原地切换样式 + 移动到列表顶部，无闪烁
3. 在"待办"筛选下标记完 → exit anim → 条目消失，无闪烁
4. 在"已完成"筛选下取消完成 → exit anim → 条目消失，无闪烁
5. 删除条目 → exit anim → 条目消失，无闪烁
6. 统计数字在所有操作后正确更新
7. 空状态在所有条目被移除后正确显示
