# 修复 todo-completing 和 todo-deleting 圆角丢失

## 根因

`.transition` 属性中包含 `transform`，当动画启动时：

| 动画类               | 当前 `transform`             | 动画 0% `transform`                 | 变化 | 冲突 |
| ----------------- | -------------------------- | --------------------------------- | -- | -- |
| `todo-activating` | `none`（`.completed:hover`） | `scale(1) translateX(0)` = `none` | 无  | 无  |
| `todo-completing` | `translateY(-1px)`（hover）  | `scale(1) translateX(0)`          | 有  | 有  |
| `todo-deleting`   | `translateY(-1px)`（hover）  | `scale(1) translateX(0)`          | 有  | 有  |

`todo-activating` 稳定是因为 `.completed:hover` 设置了 `transform: none`，动画 0% 等效于 `none`，`transition: transform` 没有检测到变化 → 不介入 → 不冲突。

## 修复

在 `.todo-completing` 和 `.todo-deleting` 上添加 `transition: none`，禁用过渡系统在动画期间的干涉：

```css
.todo-item.todo-completing,
.todo-item.todo-deleting {
    transition: none;
}
```

这与项目已有模式一致：动画类接管时禁用过渡，避免两个系统同时控制同一属性。

## 涉及文件

| 文件                                     | 改动                                    |
| -------------------------------------- | ------------------------------------- |
| `frontend/src/css/components/todo.css` | 第 322-324 行的两个规则添加 `transition: none` |

## 验证

1. 点击待办条目的复选框标记完成 → 圆角全程稳定
2. 点击已完成条目的复选框取消完成 → 仍稳定（回归）
3. 点击删除按钮（待办/已完成）→ 圆角全程稳定
4. 鼠标悬停/移出 → 过渡不受影响（动画类移除后才恢复 transition）

