# 修复窗口控制按钮未贴右的问题

## 摘要

右上角窗口控制按钮（最小化/最大化/关闭）因 `#topbar` 的 24px 右侧内边距而无法紧贴窗口右边缘，需要移除右侧 padding。

## 当前状态分析

- `#topbar`（同时作为 Frameless 窗口标题栏）设置 `padding: 0 24px`，左右各有 24px 内边距
- `.topbar-actions`（窗口控制按钮容器）已通过 `margin-left: auto` 推到 flex 末端，且 `padding-right: 0px`
- 控制按钮距离窗口右边缘仍有 24px 空白间距，不符合 Frameless 窗口的常规设计

## 修改方案

只修改一个 CSS 属性，不改动 HTML/JS：

### 文件：`frontend/src/css/components/topbar.css`

| 位置 | 当前值 | 修改为 |
|------|--------|--------|
| 第 15 行，`#topbar` | `padding: 0 24px;` | `padding: 0 0 0 24px;` |

即将双向 padding 改为仅左侧 padding，右侧归零。

### 兼容性检查

- `#topbar.editor-fullscreen`（第 22-24 行）仅覆盖 `padding-left`，不受影响
- `.topbar-actions` 已有 `padding-right: 0px`，无需额外调整
- `.topbar-left` 及其他子元素不受影响

## 可视化效果

```
修改前：
  ┌─ 24px ─┬─ [菜单][Jot] ────┬─ [—][□][×] ─┬─ 24px ─┐
  
修改后：
  ┌─ 24px ─┬─ [菜单][Jot] ────┬─ [—][□][×] ┤ 0px
```

## 验证方式

1. 确认窗口控制按钮紧贴右边缘，无多余间距
2. 检查所有 12 套主题下按钮位置均正确
3. 验证编辑器全屏模式（`editor-fullscreen`）下布局仍正常
4. 验证窗口拖拽功能不受影响（`--wails-draggable` 属性未改动）
