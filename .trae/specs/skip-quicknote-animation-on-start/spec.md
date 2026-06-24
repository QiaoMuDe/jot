# 快速笔记启动跳过动画 Spec

## Why

开启快速笔记后启动程序，`loadQuickNoteSetting()` 调用 `openEditor(null, false, true)` 打开全屏新建页面时会执行 `modalEnter`（scale+fade）和 `viewEnter`（translateY+fade）动画。由于此时程序刚从空白启动，没有前面的视图状态，动画导致界面闪烁——编辑器先不可见再渐入，视觉上不流畅。

## What Changes

- 在 `openEditor()` 中，当 `startFullscreen` 为 true 时跳过 `modalEnter` / `viewEnter` / `overlayFadeIn` 动画，改为直接设置 `opacity: 1`
- 仅影响快速笔记启动场景，不影响普通新建/查看笔记的打开动画

## Impact

- Affected code: `frontend/src/main.js` — `openEditor()` 函数动画部分
- No CSS changes needed

## ADDED Requirements

### Requirement: 快速笔记启动跳过动画

The system SHALL skip the editor opening animation when `startFullscreen` is true.

#### Scenario: 快速笔记启动直接显示

- **WHEN** 程序启动加载调用 `openEditor(null, false, true)`
- **THEN** 编辑器面板和内容立即显示，无 `modalEnter` / `viewEnter` 动画
- **AND** 全屏 class 照常添加，按钮图标照常更新

#### Scenario: 普通打开保留动画

- **WHEN** 用户点击新建笔记按钮调用 `openEditor(null)`（无 startFullscreen）
- **THEN** `modalEnter` / `viewEnter` 动画正常播放
