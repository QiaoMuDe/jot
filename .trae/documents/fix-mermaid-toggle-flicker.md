# 修复 Mermaid 渲染→源码切换闪烁问题

## 问题总结

从渲染模式切换回源码显示时，代码块会闪烁两下。根因是 `pre` 元素上持久的 `transition: opacity 0.2s ease` 在反向切换时产生副作用。

## 当前状态分析

### 相关文件

| 文件                                       | 行号          | 内容                                                                |
| ---------------------------------------- | ----------- | ----------------------------------------------------------------- |
| `frontend/src/css/components/editor.css` | L1340-L1341 | `.pre-wrapper.has-mermaid pre { transition: opacity 0.2s ease; }` |
| `frontend/src/main.js`                   | L3902-L3949 | `toggleMermaidView()` 函数                                          |

### 问题链路

1. **正向（源码→渲染）**：`pre` 加 `pre-hiding` 类 → `opacity: 0` 淡出 → 200ms 后定时器设 `display: none`，移除 `pre-hiding`，显示渲染结果
2. **定时器内**：`pre.style.display = 'none'` 后移除 `pre-hiding`，此时 `transition: opacity 0.2s ease` 触发了 `opacity: 0 → 1` 的过渡，但由于 `display: none` 不可见，浏览器可能内部追踪了此过渡状态
3. **反向（渲染→源码）**：`pre.style.display = ''` 恢复显示时，浏览器将之前残留的 `opacity` 过渡应用到渲染上，导致 `pre` 从 `opacity: 0` 淡入到 `opacity: 1`，表现为闪烁

## 修改方案

### 方案：将 `transition` 从 `pre` 基础选择器移到 `pre.pre-hiding` 上

**改动 1：CSS —** **`editor.css`**

将：

```css
.pre-wrapper.has-mermaid pre {
    transition: opacity 0.2s ease;
}
```

改为：

```css
.pre-wrapper.has-mermaid pre.pre-hiding {
    transition: opacity 0.2s ease;
}
```

**原理**：

* `transition` 只在 `pre-hiding` 类存在时生效，即只在淡出动画期间有过渡

* 正向切换（源码→渲染）：`pre-hiding` 被添加 → `opacity: 1 → 0` 平滑淡出 ✅

* 定时器回调：移除 `pre-hiding` 时 `transition` 也随之移除，`pre` 恢复 `opacity: 1` 无过渡（此时 `display: none` 已设，无视觉影响） ✅

* 反向切换（渲染→源码）：`pre.style.display = ''` 恢复显示时，`pre` 上无 `transition`，直接以 `opacity: 1` 出现，无闪烁 ✅

**改动 2：JS —** **`main.js`**

无需修改 JS 代码。CSS 改动后，`toggleMermaidView()` 中的逻辑完全兼容，无需调整。

## 验证步骤

1. 在笔记预览中测试 Mermaid 代码块：源码→渲染→源码，确认无闪烁
2. 在 AI 消息中测试 Mermaid 代码块：源码→渲染→源码，确认无闪烁
3. 确认正向切换（源码→渲染）的淡出动画仍然正常
4. 确认 Mermaid 切换按钮的显示/隐藏行为正常
5. 确认复制按钮与 Mermaid 切换按钮的交互（`copying` 状态隐藏）正常

