# 修复删除动画时条目圆角丢失问题

## 问题

点击删除按钮时，待办条目的圆角在动画开始的瞬间短暂变为直角，然后才播删除动画。

## 根因

`.todo-item` 基类有 `transition: all 0.2s ease`（过渡全部属性），同时动画 `@keyframes` 中没有显式声明 `border-radius`。当动画类（`.todo-deleting` 等）被添加时：

1. `transition: all` 尝试介入过渡
2. 动画系统也试图控制属性
3. 两者冲突导致 `border-radius` 在一帧内丢失 → 变直角
4. 动画 `forwards` 填充模式接管后才恢复

## 修复方案

在所有动画 `@keyframes` 的每个关键帧中显式声明 `border-radius: var(--radius-lg)`，让动画直接控制保持圆角，不受 transition 干扰。

涉及的动画：

* `todoComplete` — 标记完成

* `todoActivate` — 取消完成

* `todoDelete` — 删除

* `todoNew` — 新增入场

## 涉及的文件

| 文件                                     | 改动                                                         |
| -------------------------------------- | ---------------------------------------------------------- |
| `frontend/src/css/components/todo.css` | 4 组 `@keyframes` 每个关键帧添加 `border-radius: var(--radius-lg)` |

## 验证方式

1. 点击删除 → 条目从动画开始到结束始终保持 `--radius-lg` 圆角
2. 标记完成 → 动画全程圆角
3. 取消完成 → 动画全程圆角
4. 新增条目入场 → 动画全程圆角

