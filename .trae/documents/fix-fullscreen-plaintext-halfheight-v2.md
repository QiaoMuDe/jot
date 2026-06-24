# Bug修复 v2：纯文本模式全屏后输入框变半高

前一次修复（删除 CSS opacity:0 + 保留 body inline opacity）不够，因为根本问题不是 opacity 而是 flex 高度传播。

## 根因分析

当 `toggleEditorFullscreen()` 的阶段 3 执行 `panel.style.animation = ''` 清除动画时：
1. Panel 的 `opacity`/`transform` 瞬间回退到 CSS 默认值
2. `.fullscreen` class 切换使 panel 尺寸瞬间变化
3. `void panel.offsetHeight` 强制重排
4. `panel.style.animation = 'modalEnter...'` 重新设置动画

在这个时序中，浏览器的 flex 子节点高度传播可能会出错，尤其是 `.cm-editor` 使用了 `height: 100%`（而非 `flex: 1`），当祖先的动画状态被突然切换时，`height: 100%` 无法可靠解析。

**核心问题**：`panel.style.animation` 的清除→重设操作与 flex 高度传播不兼容，导致 `.cm-editor` 在 `height: 100%` 模式下解析为错误的高度。

## 修复方案

### 方案：回到 CSS transition，简化 toggleEditorFullscreen

之前的尝试中，`width`/`height` 的 CSS transition 会导致鼠标卡顿，但那是因为当时**内容 DOM 未隐藏**。现在 `toggleEditorFullscreen` 已经会在切换前隐藏 `mdRendered`（`display: none`），内容 DOM 不在 layout/paint 树中，鼠标不会卡。

因此：

1. **CSS** — 给 `.editor-panel` 恢复 `width/height/max-width/max-height` 的 CSS transition
2. **JS** — `toggleEditorFullscreen()` 简化为：淡出内容 → 隐藏内容 DOM → 切换 class（CSS transition 处理 350ms 过渡）→ 恢复内容 → 淡入

### 改动 1：CSS — 添加回 width/height transition

**文件**: `frontend/src/style.css` L942-955

```css
.editor-panel {
  transition: width 0.35s cubic-bezier(0.16, 1, 0.3, 1),
              height 0.35s cubic-bezier(0.16, 1, 0.3, 1),
              max-width 0.35s cubic-bezier(0.16, 1, 0.3, 1),
              max-height 0.35s cubic-bezier(0.16, 1, 0.3, 1),
              background-color 0.3s ease-out,
              border-color 0.3s ease-out,
              border-radius 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}
```

### 改动 2：JS — 重写 toggleEditorFullscreen()

**文件**: `frontend/src/main.js` L2857-2914

```javascript
function toggleEditorFullscreen() {
    const panel = els.editorPanel;
    const btn = els.editorFullscreenBtn;
    const overlay = els.editorOverlay;
    const goFullscreen = !state._isFullscreen;
    const mdRendered = els.mdRendered;
    const body = panel.querySelector('.editor-body');

    if (panel._fsTimer) { clearTimeout(panel._fsTimer); panel._fsTimer = null; }

    /* 阶段1（0→50ms）：内容快速淡出 */
    if (body) body.style.transition = 'opacity 0.05s ease-out';
    if (body) body.style.opacity = '0';

    panel._fsTimer = setTimeout(() => {
        /* 阶段2（50ms）：隐藏内容 DOM，切换 class */
        if (mdRendered) mdRendered.style.display = 'none';
        if (body) body.style.transition = '';

        state._isFullscreen = goFullscreen;
        panel.classList.toggle('fullscreen', goFullscreen);
        overlay.classList.toggle('fullscreening', goFullscreen);
        btn.innerHTML = goFullscreen ? SVGS.editorExitFullscreen : SVGS.editorFullscreen;
        btn.title = goFullscreen ? '退出全屏' : '全屏编辑';
        btn.classList.toggle('fullscreen', goFullscreen);

        if (goFullscreen && els.notebookSidebar && !els.notebookSidebar.classList.contains('collapsed')) {
            els.notebookSidebar.classList.add('collapsed');
        }

        /* 阶段3（50ms + 350ms）：等 CSS transition 完成 */
        panel._fsTimer = setTimeout(() => {
            /* 恢复内容，淡入 */
            if (mdRendered) mdRendered.style.display = '';
            if (body) {
                body.style.transition = 'opacity 0.12s ease-out';
                body.style.opacity = '1';
                setTimeout(() => {
                    body.style.transition = '';
                }, 130);
            }
            panel._fsTimer = null;
        }, 350);
    }, 50);
}
```
