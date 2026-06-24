# Tasks

- [x] 修改 `openEditor()` 动画逻辑：在 lines 2633-2641 的动画块中添加条件判断，`startFullscreen` 为 true 时跳过 `modalEnter/viewEnter/overlayFadeIn` 动画，直接设置 `overlay.style.opacity = '1'`
