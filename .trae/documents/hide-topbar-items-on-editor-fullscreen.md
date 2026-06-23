# 全屏编辑时隐藏搜索框和更多菜单按钮

## 总结

编辑器全屏模式下，`#topbar` 中的搜索框（`.topbar-search`）和更多菜单（`.topbar-dropdown`）占用空间且无实际用途，应予以隐藏，让标题栏更简洁。

## 当前状态分析

`#topbar`（[index.html#L33-L63](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L33-L63)）包含三个主要区域：

1. `.topbar-brand` — "Jot" 文本（左侧，保留）
2. `.topbar-search` — 搜索输入框（中间，全屏时隐藏）
3. `.topbar-actions` — 右侧操作区：
   - `.topbar-dropdown` — ☰ 更多菜单按钮（全屏时隐藏）
   - 窗口控制按钮（最小化/最大化/关闭，保留）

全屏切换函数 `toggleEditorFullscreen()`（[main.js#L2655-L2678](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2655-L2678)）目前已控制 `.editor-panel` 和 `.editor-overlay` 的样式类。关闭函数 `closeEditor()`（[main.js#L2680-L2726](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2680-L2726)）在清理阶段移除全屏相关类。

## 提议的修改

### 文件 1: `frontend/src/main.js` — JS 事件绑定

**`toggleEditorFullscreen()` 中（L2661 附近）**：在现有全屏切换逻辑基础上，给 `#topbar` 添加/移除 `editor-fullscreen` 类：

```js
document.getElementById('topbar').classList.toggle('editor-fullscreen', goFullscreen);
```

**`closeEditor()` 中（L2700 附近）**：在退出全屏清理时，同步移除 `#topbar` 上的 `editor-fullscreen` 类：

```js
document.getElementById('topbar').classList.remove('editor-fullscreen');
```

**原因**：JS 控制比纯 CSS 方案更可靠，因为 `#topbar` 不是 `.editor-panel` 的 DOM 祖先/子孙关系，通过 JS 在正确的时机添加/移除类是干净的做法。

### 文件 2: `frontend/src/style.css` — CSS 样式

在现有的全屏样式区域（`.editor-panel.fullscreen` 附近）追加：

```css
#topbar.editor-fullscreen .topbar-search,
#topbar.editor-fullscreen .topbar-dropdown {
  display: none;
}
```

**原因**：通过 CSS 控制显示/隐藏，与现有样式定义方式一致，便于维护。

## 假设与决策

- 全屏时只隐藏 `.topbar-search` 和 `.topbar-dropdown`，不隐藏 `.topbar-brand` 和窗口控制按钮 — 保持品牌可见且用户仍可拖拽窗口/点击控制按钮
- 用 `display: none` 而非 `visibility: hidden`，让隐藏后空间被其余元素自然填充
- 不添加过渡动画（隐藏搜索框和菜单不需要动画），保持简单

## 验证步骤

1. 打开编辑器 → 点击全屏按钮 → 确认 `#topbar` 中搜索框和 ☰ 按钮消失，窗口控制按钮保留
2. 退出全屏 → 确认搜索框和 ☰ 按钮恢复显示
3. 关闭编辑器 → 确认 `#topbar` 恢复正常状态（快速笔记自动全屏场景也覆盖）
