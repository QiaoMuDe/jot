# 修复快速笔记启动时先闪现首页再切全屏的问题

## 问题 Summary

启用快速笔记后，程序启动时会先短暂显示笔记首页（网格视图），然后才闪到全屏的新建笔记页面。视觉上有一个明显的"闪烁"。

## 根因分析

问题出在 `openEditor()` 函数中 `startFullscreen=true` 分支的执行时序：

```
第 3247 行: els.viewEditor.classList.add('active')
  → viewEditor 从 display:none → display:flex
  → 内部的 .editor-overlay（position:fixed）覆盖页面
  → 但 overlay 的 CSS 默认 opacity: 0（透明）
  → 用户能看到背后的 viewGrid              ← 问题！

第 3265-3278 行: 添加 fullscreening/fullscreen 类、强制回流等

第 3279 行: overlay.style.opacity = '1'    ← 此时才变不透明
第 3280 行: panel.style.opacity = '1'
```

第 3269 行 `void panel.offsetHeight` 强制触发回流，浏览器可能在此时渲染一帧。在这一帧中 overlay 是透明的，用户看到了背后的 `viewGrid`。

## 修改方案

**核心思路**：在全屏模式下，将 `overlay` 和 `panel` 的 `opacity` 设置提前到 `viewEditor.classList.add('active')` 之前，确保 viewEditor 一显示 overlay 就已经是不透明的。

**非全屏模式不受任何影响**，因为改动只在 `startFullscreen=true` 分支中。

### 改动：`frontend/src/main.js`

**位置**：[openEditor()](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3245-L3281)

将全屏模式的 opacity/transform 设置从第 3279-3281 行**提前**到第 3247 行（`viewEditor.classList.add('active')`）之前。

#### 修改前代码

```javascript
// ── 立即显示面板 + 骨架屏（不等数据加载） ──
els.mainContent.style.overflow = 'hidden';
els.viewEditor.classList.add('active');

// 在 CM6 挂载点显示骨架屏 shimmer
const contentArea = document.getElementById('editorNoteContent');
contentArea.innerHTML = ''
    + '<div class="editor-skeleton">'
    + '<div class="editor-skeleton-line"></div>'
    + '<div class="editor-skeleton-line"></div>'
    + '<div class="editor-skeleton-line"></div>'
    + '<div class="editor-skeleton-line"></div>'
    + '</div>';

// 启动入场动画
const overlay = els.editorOverlay;
const panel = els.editorPanel;
const body = panel.querySelector('.editor-body');
document.getElementById('topbar').classList.add('editor-fullscreen');

if (startFullscreen) {
    panel.style.transition = 'none';
    overlay.classList.add('fullscreening');
    panel.classList.add('fullscreen');
    void panel.offsetHeight;
    panel.style.transition = '';
    state._isFullscreen = true;
    if (els.editorFullscreenBtn) {
        els.editorFullscreenBtn.innerHTML = SVGS.editorExitFullscreen;
        els.editorFullscreenBtn.title = '退出全屏';
        els.editorFullscreenBtn.classList.add('fullscreen');
    }
    if (els.notebookSidebar && !els.notebookSidebar.classList.contains('collapsed'))
        els.notebookSidebar.classList.add('collapsed');
    overlay.style.opacity = '1';
    panel.style.opacity = '1';
    panel.style.transform = 'scale(1)';
} else {
    // ...非全屏动画...
}
```

#### 修改后代码

```javascript
// ── 立即显示面板 + 骨架屏（不等数据加载） ──
els.mainContent.style.overflow = 'hidden';

// 提前获取 overlay/panel 引用（全屏模式下需要在显示 viewEditor 前设置 opacity）
const overlay = els.editorOverlay;
const panel = els.editorPanel;
const body = panel.querySelector('.editor-body');

// 全屏模式：先让 overlay 不透明，再显示 viewEditor，避免透出背后的网格
if (startFullscreen) {
    overlay.style.opacity = '1';
    panel.style.opacity = '1';
    panel.style.transform = 'scale(1)';
}

els.viewEditor.classList.add('active');

// 在 CM6 挂载点显示骨架屏 shimmer
const contentArea = document.getElementById('editorNoteContent');
contentArea.innerHTML = ''
    + '<div class="editor-skeleton">'
    + '<div class="editor-skeleton-line"></div>'
    + '<div class="editor-skeleton-line"></div>'
    + '<div class="editor-skeleton-line"></div>'
    + '<div class="editor-skeleton-line"></div>'
    + '</div>';

// 启动入场动画（topbar 相关设置）
document.getElementById('topbar').classList.add('editor-fullscreen');

if (startFullscreen) {
    panel.style.transition = 'none';
    overlay.classList.add('fullscreening');
    panel.classList.add('fullscreen');
    void panel.offsetHeight;
    panel.style.transition = '';
    state._isFullscreen = true;
    if (els.editorFullscreenBtn) {
        els.editorFullscreenBtn.innerHTML = SVGS.editorExitFullscreen;
        els.editorFullscreenBtn.title = '退出全屏';
        els.editorFullscreenBtn.classList.add('fullscreen');
    }
    if (els.notebookSidebar && !els.notebookSidebar.classList.contains('collapsed'))
        els.notebookSidebar.classList.add('collapsed');
    // opacity/transform 已在上方提前设置，此处不再重复
} else {
    // ...非全屏动画（完全不变）...
}
```

### 不改动的文件

* 后端 Go 代码：不需要改动

* `index.html`：不需要改动

* 所有 CSS 文件：不需要改动

* 其他 JS 文件：不需要改动

## 验证步骤

1. 启用快速笔记 → 重启程序 → 编辑器以全屏模式直接打开，不再闪现首页网格
2. 禁用快速笔记 → 重启程序 → 正常显示首页网格
3. 正常点击新建笔记（非全屏）→ 仍然以悬浮卡片动画打开，不受影响
4. 全屏切换按钮 → 正常工作
5. 关闭编辑器后 → 回到正常网格视图

