# 全屏切换时搜索框和更多菜单的平滑过渡动画

## 总结

当前全屏模式下搜索框和更多菜单通过 `display: none` 瞬间消失/出现，显得生硬。改为用 CSS transition 同时实现**淡出 + 尺寸收缩**的效果，进入和退出全屏都丝滑自然。

## 当前状态分析

已有规则（[style.css#L878-L881](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/style.css#L878-L881)）：
```css
#topbar.editor-fullscreen .topbar-search,
#topbar.editor-fullscreen .topbar-dropdown {
  display: none;   /* ← 瞬间消失，无过渡 */
}
```

需要动画的元素：
- `.topbar-search` — 搜索输入框容器：`flex: 1; max-width: 320px; padding: 0 4px 0 12px; transition: var(--transition)(150ms ease-out)`
- `.topbar-dropdown` — ☰ 更多菜单容器：`position: relative;`（无线宽）
- `.topbar-actions` — 右侧操作栏：`margin-left: auto;` 会在搜索框收缩后自然靠左移动

当搜索框收缩时，`.topbar-actions`（含窗口控制按钮）会在 flex 布局中随 `.topbar-search` 宽度减小而自动向左滑动。

## 提议的修改

### 修改文件：`frontend/src/style.css`

#### 改动 1：替换 `.topbar-search` 的 transition

在现有 `.topbar-search` 选择器（[L38-L47](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/style.css#L38-L47)）中，将 `transition: var(--transition)` 替换为多属性过渡，并添加 `overflow: hidden` 以在收缩时裁剪内容：

```css
.topbar-search {
  /* 保留原有 flex, align-items, flex:1, max-width, background, border-radius, padding */
  overflow: hidden;
  transition: opacity 0.15s ease,
              max-width 0.25s ease,
              padding 0.25s ease,
              margin 0.25s ease;
}
```

**效果**：进入全屏时，搜索框先淡出（150ms），紧接着宽度收缩至 0（250ms），右侧窗口控件随之平滑左移。退出全屏时反向（宽度先展开再淡入）。

#### 改动 2：给 `.topbar-dropdown` 添加 transition

在现有 `.topbar-dropdown` 选择器（[L130-L132](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/style.css#L130-L132)）中添加过渡属性：

```css
.topbar-dropdown {
  position: relative;
  transition: opacity 0.15s ease, transform 0.15s ease;
}
```

**效果**：淡出同时略微缩小（`scale(0.8)`），表现为"缩小消失"。

#### 改动 3：替换 `display: none` 为平滑过渡状态

在现有全屏状态规则（[L878-L881](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/style.css#L878-L881)）中，用过渡属性替换 `display: none`：

```css
#topbar.editor-fullscreen .topbar-search {
  opacity: 0;
  max-width: 0;
  padding-left: 0;
  padding-right: 0;
  margin: 0;
  pointer-events: none;
}

#topbar.editor-fullscreen .topbar-dropdown {
  opacity: 0;
  transform: scale(0.8);
  pointer-events: none;
}
```

**关于 `.topbar-actions` 自动左移**：`.topbar-actions` 使用 `margin-left: auto`，当 `.topbar-search` 的 `max-width` 过渡到 0 时，flex 布局自然重新分配空间，`.topbar-actions` 会平滑向左滑动，无需额外的过渡代码。

### 动画时序

| 方向 | 阶段 1 | 阶段 2 | 总时长 |
|------|--------|--------|--------|
| **进入全屏**（隐藏） | 搜索框/菜单淡出 150ms | 搜索框宽度收缩 250ms | ~250ms |
| **退出全屏**（显示） | 搜索框宽度展开 250ms | 搜索框/菜单淡入 150ms | ~250ms |

两者过渡同时触发，但宽度变化在时间上略长，形成自然的先后层次感。

## 假设与决策

- 不添加 JS 动画控制，纯 CSS 实现 — 简单可靠，性能更好
- `.topbar-search` 使用 `overflow: hidden` — 在正常模式下 `.topbar-search-input` 有 `flex: 1` 不会溢出其父容器，所以 `overflow: hidden` 不会产生副作用
- 不修改 JS 代码（`toggleEditorFullscreen` / `closeEditor`）— 原有的 class 切换逻辑保持不动，只改 CSS
- `pointer-events: none` 替代 `display: none` 防止隐藏元素被点击/聚焦，同时在 `max-width: 0` 时元素实际上不可见且不可交互

## 验证步骤

1. 打开编辑器 → 点击全屏 → 观察搜索框淡出收缩，右侧窗口控件平滑左移，☰ 按钮缩小消失
2. 退出全屏 → 观察搜索框宽度先展开再淡入，☰ 按钮放大恢复
3. 多次快速切换全屏/退出全屏 → 过渡不卡顿、不闪烁
4. 窗口控制按钮（最小化/最大化/关闭）在全屏模式下始终可点击
