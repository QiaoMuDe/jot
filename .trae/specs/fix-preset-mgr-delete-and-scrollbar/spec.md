# Fix Preset Manager Delete and Scrollbar Issues Spec

## Why

预设管理界面有两个问题导致功能不可用和体验不佳：
1. 确认删除后不执行删除动作和动画
2. 预设过多时滚动条出现后自动隐藏

## Root Cause Analysis

### Issue 1: Delete animation never plays (critical)

**Root cause**: CSS 级联冲突. `.preset-row-enter`（[line 915](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L915)）和 `.preset-delete-out`（[line 886](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L886)）具有相同的 CSS 特异性（一个类选择器），因此**后者胜出**。由于 `.preset-row-enter` 在 CSS 文件中定义得更晚（915 > 886），当两者同时应用于同一元素时，`preset-row-enter` 的 `animation` 简写覆盖了 `preset-delete-out` 的 `animation` 简写。

**关键代码**（[deleteProfile](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L2618-L2639)）中执行：

```js
rowEl.classList.add('preset-delete-out');
await new Promise(resolve => {
    rowEl.addEventListener('animationend', resolve, { once: true });
});
```

- 实际播放的是 `presetRowEnter` 而非 `presetDeleteOut`
- `animationend` 事件从未触发（`presetRowEnter` 已播放完毕或仍在播放）
- Promise 永不 resolve → 函数挂起 → 后续 API 调用不执行

### Issue 2: Scrollbar auto-hides

**Root cause**: `.preset-mgr-list`（[line 733](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L733)）的 `max-height: 500px` 完全来自 `mgrSlideDown` 动画的 `to` 关键帧，通过 `animation-fill-mode: both` 持久化。这不是一个独立的 CSS 声明，而是动画 fill 状态，在某些浏览器/WebView2 场景下可能被清除。一旦 `max-height` 丢失，容器扩展至内容高度，滚动条不再需要。

## What Changes

1. **Fix Issue 1**：在 `deleteProfile` 中添加 `preset-delete-out` 类时，同时移除 `preset-row-enter` 类，消除级联冲突
2. **Fix Issue 2**：在 `.preset-mgr-list` CSS 中添加独立的 `max-height` 声明，不依赖动画 fill 状态

## Impact

- Affected code: `frontend/src/css/components/settings-panel.css`, `frontend/src/main.js`

## ADDED Requirements

### Requirement: Preset delete animation plays when user confirms deletion

#### Scenario: Delete preset with exit animation
- **WHEN** user clicks "删除" on a preset row and confirms the dialog
- **THEN** the row plays a slide-out animation (`.preset-delete-out`)
- **AND** after the animation completes, the preset is deleted via API
- **AND** the list re-renders showing remaining presets

### Requirement: Preset manager scrollbar persists

#### Scenario: Many presets trigger scrollbar
- **WHEN** there are more preset rows than fit within the container's 500px max-height
- **THEN** the container shows a vertical scrollbar
- **AND** the scrollbar remains visible during scrolling and does not auto-hide after animations complete
