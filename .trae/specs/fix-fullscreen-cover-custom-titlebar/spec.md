# 修复全屏模式遮挡自定义标题栏 Spec

## Why

当前使用 Wails Frameless 模式，`#topbar` 既是导航栏也是自定义窗口标题栏（含最小化/最大化/关闭按钮）。当用户在编辑器（查看/新建笔记）中点击全屏按钮时，`.editor-panel.fullscreen` 设置 `width: 100vw; height: 100vh` 覆盖整个视口，把 `#topbar` 完全挡住，用户无法在编辑全屏时操作窗口控制按钮（最小化/最大化/关闭）。

## What Changes

- **CSS**：修改 `.editor-panel.fullscreen` 和 `.editor-overlay.fullscreening`，使其全屏时避开 `#topbar` 区域（56px 高度），保证自定义标题栏始终可见
- **CSS**：确保全屏模式下编辑器面板顶部与 `#topbar` 底部对齐
- **CSS**：过渡动画适配新的全屏尺寸变化

## Impact

- Affected specs: `add-frameless-window`（自定义标题栏交互）
- Affected code:
  - `frontend/src/style.css` — 修改 `.editor-panel.fullscreen` 和 `.editor-overlay` 的全屏样式

## ADDED Requirements

### Requirement: 全屏模式保留自定义标题栏可见

The system SHALL 在编辑器进入全屏模式时，保留 `#topbar`（自定义窗口标题栏）始终可见，不被编辑器面板或遮罩层遮挡。

#### Scenario: 进入全屏编辑器
- **WHEN** 用户在编辑器中点击全屏按钮
- **THEN** 编辑器面板展开至 `#topbar` 下方区域，而非覆盖整个视口
- **AND** `#topbar` 保持可见，窗口控制按钮（最小化/最大化/关闭）可正常交互

#### Scenario: 退出全屏编辑器
- **WHEN** 用户点击退出全屏按钮
- **THEN** 编辑器面板恢复至原始位置和尺寸
- **AND** 过渡动画平滑

## MODIFIED Requirements

### Requirement: 编辑器全屏样式
原 `.editor-panel.fullscreen` 使用 `100vw`/`100vh` 覆盖全视口。修改后：
- 不覆盖 `#topbar` 区域（顶部 56px 保留给标题栏）
- 遮罩层 `.editor-overlay.fullscreening` 的 `position: fixed` 改为 `top: 56px` 或使用 `padding-top` 避免遮挡

## REMOVED Requirements

无
