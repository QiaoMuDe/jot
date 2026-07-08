# 修复 todo-completing / todo-deleting 圆角丢失 — 消除 0% transform

## 问题

`.todo-completing` 和 `.todo-deleting` 的 `@keyframes` 在 0% 就声明了 `transform`。当元素处于 hover 状态时，`transform` 从 `translateY(-1px)`（hover）变成 `scale(1) translateX(0)`（动画 0%）→ `transition: transform` 检测到变化 → 介入冲突 → 圆角丢失。

## 修复

**两个 `@keyframes` 的 0% 关键帧去掉 `transform`**，只保留 `opacity`。`transform` 仅在 100% 设置，由动画从元素的当前值自然过渡到终态。

```css
/* 之前 */
@keyframes todoComplete {
    0%   { opacity: 1; transform: scale(1) translateX(0); ... }
    20%  { opacity: 1; transform: scale(1.04) translateX(0); ... }
    100% { opacity: 0; transform: scale(0.92) translateX(-14px); ... }
}

/* 之后 */
@keyframes todoComplete {
    0%   { opacity: 1; border-radius: var(--radius-lg); }
    100% { opacity: 0; transform: scale(0.92) translateX(-14px); border-radius: var(--radius-lg); }
}
```

同样处理 `todoDelete`，去掉中间的 25% 弹跳帧以进一步简化。

同步移除 `transition: none`（不再需要）。

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/src/css/components/todo.css` | 重写 `todoComplete` 和 `todoDelete` keyframes（消除 0% transform），移除 `transition: none` |

## 验证

1. 点击待办条目标记完成 → 淡出 + 向左滑动，圆角全程稳定
2. 点击删除（待办/已完成条目）→ 淡出 + 向右滑动，圆角全程稳定
3. 点击已完成条目取消完成 → 不受影响（回归）
