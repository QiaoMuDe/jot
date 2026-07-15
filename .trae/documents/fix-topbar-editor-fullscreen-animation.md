# 修复 topbar 品牌标识动画卡顿

## 摘要

修复 `openEditor` 时顶部品牌标识和更多菜单按钮向左滑动的动画卡顿问题（"先往左走一点卡一下再往左移动"）。

## 当前状态分析

### 触发条件

`openEditor` 第 3263 行：`document.getElementById('topbar').classList.add('editor-fullscreen')`

### 动画涉及的 CSS 属性变化

| 元素                 | 属性             | 变化                    | 过渡时间                            |
| ------------------ | -------------- | --------------------- | ------------------------------- |
| `#topbar`          | `padding-left` | 24px → 4px            | 0.35s cubic-bezier(0.4,0,0.2,1) |
| `.topbar-dropdown` | `opacity`      | 1 → 0                 | 0.2s cubic-bezier(0.4,0,0.2,1)  |
| `.topbar-dropdown` | `transform`    | scale(1) → scale(0.8) | 0.2s cubic-bezier(0.4,0,0.2,1)  |
| `.topbar-dropdown` | `width`        | auto → 0              | 0.35s cubic-bezier(0.4,0,0.2,1) |
| `.topbar-dropdown` | **`margin`**   | **-24px → 0**         | **无过渡（instant）**                |

### 根因

`margin-left: -24px` → `margin: 0` 的变化**没有过渡**。当 `.editor-fullscreen` 类被添加时，`margin` 从 `-24px` 瞬间变为 `0`：

1. 下拉按钮向右瞬跳 24px
2. 紧随其后的 `.topbar-brand`（品牌标识）被向右挤 24px
3. 同时 `#topbar` 的 `padding-left` 正在从 24px 过渡到 4px（向左移 20px）
4. 同时 `.topbar-dropdown` 的 `width` 正在过渡到 0（品牌标识向左填补空隙）

**净效果**：品牌标识 → 右移 24px（margin 瞬跳）→ 左移 20px（padding 过渡）+ 左移（width 收缩）→ 整体向左移动。中间出现"卡一下"的视觉停顿。

## 变更清单

### 文件 1：`frontend/src/css/components/editor.css`（第 79-86 行）

在 `#topbar.editor-fullscreen .topbar-dropdown` 中添加 `transition` 覆盖，将 `margin` 加入过渡：

```css
#topbar.editor-fullscreen .topbar-dropdown {
  opacity: 0;
  transform: scale(0.8);
  width: 0;
  margin: 0;
  overflow: hidden;
  pointer-events: none;
  transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1),
              transform 0.2s cubic-bezier(0.4, 0, 0.2, 1),
              width 0.35s cubic-bezier(0.4, 0, 0.2, 1),
              margin 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
```

### 文件 2：`frontend/src/css/components/topbar.css`（第 129-134 行）

在 `.topbar-dropdown` 基础样式中也补上 `margin` 过渡，确保移除 `editor-fullscreen` 时（`closeEditor`）动画同样平滑：

```css
.topbar-dropdown {
  position: relative;
  transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1),
              transform 0.2s cubic-bezier(0.4, 0, 0.2, 1),
              width 0.35s cubic-bezier(0.4, 0, 0.2, 1),
              margin 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}
```

## 验证步骤

1. 点击笔记卡片（查看/编辑模式）→ 品牌标识和更多菜单按钮向左平滑滑动，无卡顿
2. 点击新建笔记按钮 → 动画同样平滑
3. 关闭编辑器 → 品牌标识和更多菜单按钮向右平滑还原，无卡顿
4. `wails build` 成功

## 不变的部分

* `openEditor` / `closeEditor` 的 JS 逻辑不变

* 其他 CSS 属性不变

* `#topbar` 的 `padding-left` 过渡不变

