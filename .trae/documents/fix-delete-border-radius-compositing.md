# 修复删除时条目圆角丢失 — GPU 合成层问题

## 根因

经过前 5 轮修复后（移除基类动画、`animationend` 清理、`border-radius` 硬声明等），删除操作仍然丢失圆角。之前的修复都是 CSS 属性层面的冲突，但删除的问题根源不同。

**真正原因：GPU 合成层创建时的 border-radius 裁剪丢失**

当 `.todo-deleting` 动画被添加时：

1. 动画的 `@keyframes` 包含 `transform` 变换（`scale()`/`translateX()`）
2. 浏览器检测到 `transform` 动画，将元素提升到独立的 GPU 合成层
3. 在新合成层创建的临界帧中，`border-radius` 的裁剪渲染可能不完整
4. 表现为一帧的"直角"闪烁

这解释了为什么之前的修复对删除无效 — 它不是一个 CSS 冲突问题，而是浏览器渲染引擎的合成层行为。

## 修复方案

在 `.todo-item` 基类添加 `backface-visibility: hidden`。

```css
backface-visibility: hidden;
-webkit-backface-visibility: hidden;
```

`backface-visibility: hidden` 的作用：

* 强制浏览器在 GPU 上预渲染元素作为一个合成层

* 合成层在动画开始前就已存在 → 不需要在动画启动时创建

* 避免了合成层创建时的 `border-radius` 裁剪丢失

这是一个在业界广泛使用的 CSS 动画防闪烁技巧。

## 涉及的文件

| 文件                                     | 改动                                                                                    |
| -------------------------------------- | ------------------------------------------------------------------------------------- |
| `frontend/src/css/components/todo.css` | `.todo-item` 添加 `backface-visibility: hidden` 和 `-webkit-backface-visibility: hidden` |

## 验证方式

1. 点击删除按钮 → 条目从开始到结束始终保持圆角，无直角帧
2. 点击完成/取消完成 → 保持不变（已有修复仍然有效）
3. 新增条目动画 → 不受影响
4. 初始加载入场动画 → 不受影响

