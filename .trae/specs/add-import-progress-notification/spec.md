# 导入文件进度通知 Spec

## Why
拖入大文件（特别是办公文档）时 markitdown 转换耗时较长，当前无任何视觉反馈，用户不知道系统正在工作。需要实时进度通知提升体验。

## What Changes
- 后端 `ImportFiles` 通过 Wails Events 发射进度事件（"正在导入..."）
- 后端 `readAIChatFiles` 通过 Wails Events 发射进度事件（"正在上传..."）
- 前端 `NotificationManager` 新增 `showProgress` + `updateProgress` 方法
- 前端 `handleFileDropPaths` 监听进度事件实时更新通知
- 前端新增 `handleAIChatFileUploadProgress` 监听 AI 上传进度事件
- 快速批量场景使用 `requestAnimationFrame` 防抖更新 DOM
- 前端在导入/上传全部完成后清理事件监听器

**NOT breaking** — `ImportFiles`/`ReadAIChatFiles` 签名不变，前端旧逻辑可作为 fallback。

## Impact
- Affected specs: file import, AI chat file upload
- Affected code:
  - `app.go` — `ImportFiles` 函数增加事件发射
  - `app.go` — `readAIChatFiles` 函数增加事件发射
  - `frontend/src/js/notification.js` — `NotificationManager` 新增方法
  - `frontend/src/main.js` — `handleFileDropPaths` 改造 + AI 上传进度监听
  - `css/notification.css` — 新增 `.notification.progress` 样式

## ADDED Requirements

### Requirement: 后端 ImportFiles 进度事件
The system SHALL emit progress events during note import processing.

#### Scenario: 导入多个文件
- **WHEN** `ImportFiles` 开始处理
- **THEN** 发射 `"import:progress"` 事件，type=`"start"`，payload=`{total}`

- **WHEN** 每个文件处理完成
- **THEN** 发射 `"import:progress"` 事件，type=`"update"`，payload=`{total, current, title}`

- **WHEN** 全部文件处理完成
- **THEN** 发射 `"import:progress"` 事件，type=`"complete"`，payload=`{total, successCount, failCount}`

### Requirement: 后端 readAIChatFiles 进度事件
The system SHALL emit progress events during AI file upload processing.

#### Scenario: 上传多个文件到 AI 聊天
- **WHEN** `readAIChatFiles` 开始处理
- **THEN** 发射 `"import:ai-progress"` 事件，type=`"start"`，payload=`{total}`

- **WHEN** 每个文件处理完成
- **THEN** 发射 `"import:ai-progress"` 事件，type=`"update"`，payload=`{total, current, title}`

- **WHEN** 全部文件处理完成
- **THEN** 发射 `"import:ai-progress"` 事件，type=`"complete"`，payload=`{total, successCount, failCount}`

### Requirement: NotificationManager 进度通知
The system SHALL provide a persistent progress notification UI.

#### Scenario: 显示进度通知
- **WHEN** `showProgress(total)` 被调用
- **THEN** 在右上角创建一条 `.notification.progress` 类型通知
- **THEN** 内容包含旋转动画图标 + `"正在导入 0/{total}"` 文字
- **THEN** 通知不会自动关闭，无关闭按钮

#### Scenario: 更新进度通知
- **WHEN** `updateProgress(ctrl, current, total, title)` 被调用
- **THEN** 更新通知文字为 `"正在导入 {current}/{total} {title}"`

- **WHEN** 场景是 AI 上传
- **THEN** 前缀文本为 `"正在上传"` 而非 `"正在导入"`

### Requirement: 快速批量场景防抖
The system SHALL debounce rapid progress DOM updates.

#### Scenario: 短时间内大量 update 事件
- **WHEN** 100 个文件在 50ms 内完成处理
- **THEN** 使用 `requestAnimationFrame` 累积状态，每帧只渲染一次
- **THEN** DOM 不产生布局抖动

### Requirement: 前后端事件生命周期管理
The system SHALL properly manage event listener lifecycle.

#### Scenario: 事件监听器清理
- **WHEN** 导入/上传完成
- **THEN** 前端调用 `EventsOff` 移除对应事件监听器
- **THEN** 防止后续导入残留监听器

## MODIFIED Requirements

### Requirement: handleFileDropPaths 增加进度监听
[Modified — 重构 `handleFileDropPaths` 逻辑]

- **GIVEN** 用户拖拽文件到应用
- **WHEN** 调用 `ImportFiles` 前
- **THEN** 先注册 `EventsOn("import:progress", callback)`
- **WHEN** 收到 `type="start"` 事件
- **THEN** 调用 `nm.showProgress(total)` 显示进度通知
- **WHEN** 收到 `type="update"` 事件
- **THEN** 调用防抖函数更新进度通知内容
- **WHEN** 收到 `type="complete"` 事件
- **THEN** 关闭进度通知，显示成功/失败汇总通知
- **THEN** 执行 `loadNotes()` / `loadNotebooks()` / `flashNoteCards()`
- **THEN** 调用 `EventsOff("import:progress")` 清理监听器

## REMOVED Requirements
（无）
