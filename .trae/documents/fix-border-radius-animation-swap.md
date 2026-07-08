# 修复条目圆角丢失 — 最终方案

## 根因：动画冲突

`.todo-item` 基类始终有一个活跃的动画：

```css
.todo-item {
    animation: todoEnter 0.2s ease-out forwards;  /* ← 基类自带动画 */
}
```

当动态添加 `.todo-deleting` 类时：

```css
.todo-item.todo-deleting {
    animation: todoDelete 0.3s ease-out both;
}
```

浏览器必须**换掉**正在运行的 `todoEnter` 动画，启动 `todoDelete`。在动画置换的临界帧：
1. `todoEnter` 的 `forwards` 填充值被释放
2. `todoDelete` 的新关键帧尚未完全接管渲染
3. 浏览器在这一帧无法正确维持 `border-radius` 的裁剪渲染 → 出现直角闪烁

这不是 `transition` 或 `border-radius` 声明的问题，而是**两个动画在同一元素上置换**时浏览器渲染引擎的固有问题。

## 修复方案

**将基类的 `todoEnter` 动画移出，改为通过 JS 显式添加。**

### 改动 1：CSS — 从基类移除默认动画

去掉 `.todo-item` 的 `animation: todoEnter`，改为通过独立 class 控制

```css
/* 删掉基类的这一行 */
.todo-item {
    ...
    animation: todoEnter 0.2s ease-out forwards;  /* 删除 */
}

/* 新增独立 class，仅在 renderTodos 时使用 */
.todo-item.todo-enter {
    animation: todoEnter 0.2s ease-out forwards;
}
```

### 改动 2：JS — renderTodos 中显式添加 todo-enter class

```js
// renderTodos 中
<div class="todo-item todo-enter${todo.done ? ' completed' : ''}" data-id="${todo.id}">
```

这样：
- 基类 `.todo-item` → 无动画 → 添加 `.todo-deleting` 时没有冲突
- 初始渲染 → 通过 `todo-enter` class 播放入场动画
- 动态操作（add/toggle/delete）→ 各自独立动画，无冲突

### 改动 3：CSS — 给动画类加 border-radius 硬保障

即使动画冲突已消除，再加一层保险：

```css
.todo-item.todo-completing,
.todo-item.todo-activating,
.todo-item.todo-deleting,
.todo-item.todo-new {
    border-radius: var(--radius-lg);
}
```

## 涉及的文件

| 文件 | 改动 |
|------|------|
| `frontend/src/css/components/todo.css` | 删除基类 `animation`，新增 `.todo-enter` class，新增 4 个动画类的 `border-radius` 硬声明 |
| `frontend/src/main.js` | `renderTodos()` 中渲染模板添加 `todo-enter` class |

## 验证方式

1. 点击删除 → 圆角始终保持，没有任何帧变成直角
2. 标记完成 → 同左
3. 取消完成 → 同左
4. 新增条目 → 同左
5. 初始加载 / 筛选切换 → 条目仍以 `todoEnter` 动画入场（`todo-enter` class 保障）
