# 预览图片灯箱 + 图片尺寸调整 Spec

## Why

笔记预览中的图片目前只能跟随内容流展示，大图会撑满预览区域宽度（`max-width: 100%`），用户无法查看原图全尺寸细节。需要灯箱功能让用户点击图片后全屏查看原图，同时适当调小日常显示尺寸让排版更舒适。

## What Changes

- **灯箱 CSS** — 新增 `.image-lightbox` 全屏遮罩样式（深色背景、flex 居中、大图等）
- **灯箱 JS** — 在 `_applyPreviewDOMHelpers` 中为每张 `<img>` 添加点击事件，点击时创建遮罩展示原图，点击遮罩关闭
- **图片尺寸调小** — 修改 `.md-rendered img`：`max-width: 85%` + `display: block` + `margin: 0.5em auto`（居中显示）
- **光标反馈** — 图片 `cursor: zoom-in` 提示可点击

## Impact

- Affected code: [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js)（_applyPreviewDOMHelpers 新增图片点击处理）、[editor.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css)（新增灯箱样式 + 调整图片尺寸）

## ADDED Requirements

### Requirement: 图片灯箱

系统 SHALL 在 .md 笔记的预览模式下，点击图片时打开全屏灯箱展示原图。

#### Scenario: 查看/编辑/新建模式预览 - 点击图片
- **WHEN** 用户在 .md 笔记的预览区域中点击一张图片
- **THEN** 创建一个固定定位的全屏遮罩层（`rgba(0,0,0,.85)`）
- **THEN** 遮罩中央显示原始图片，`max-width: 90vw; max-height: 90vh; object-fit: contain`
- **THEN** 点击遮罩任意位置关闭灯箱

### Requirement: 图片尺寸调小

系统 SHALL 将预览中图片的日常显示宽度从 `max-width: 100%` 调整为 `max-width: 85%`，并居中显示。

#### Scenario: 预览图片日常显示
- **WHEN** 用户查看 .md 笔记的预览区域
- **THEN** 图片宽度不超过预览区域的 85%
- **THEN** 图片水平居中显示

### Requirement: 光标反馈

系统 SHALL 让预览中的图片鼠标指针显示为 `zoom-in`，提示用户可点击。

## REMOVED Requirements

无。
