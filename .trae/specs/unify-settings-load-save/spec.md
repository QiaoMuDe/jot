# 设置页数据加载保存统一化 Spec

## Why

当前设置页存在 9 个独立的 `loadXxxSetting()` 函数和 15+ 个分散的 `SetSetting()` 保存点。新增一个设置项需要：添加默认值、添加后端方法（如果有类型校验）、添加前端 load 函数、注册到 `reloadSettings`、注册到 `window` 导出、挂到 init 流程。同样，存时路径也不一致（有的走 `SaveAIConfig`、有的走 `SetSetting`、有的走专用方法）。

这导致：
- 加一个新设置至少改 5-6 处代码
- 漏掉任何一处就会导致重置/导入/还原后设置不生效
- 前后端设置字段不对齐容易出错

## What Changes

1. 后端定义统一的 `SettingsConfig` 结构体，包含全部 20 项设置（含正确类型：bool/int/string）
2. 后端新增 `GetAllSettings()` 和 `SaveAllSettings(cfg)` 方法，一次调用读写全部设置
3. 后端保留 `GetSetting/SetSetting` 供其他模块使用，但废弃 `GetAIConfig/SaveAIConfig`（前端不再调用）
4. 前端创建单一的 `loadSettings()` 和 `saveSettings()` 函数
5. 所有保存触发器（主题切换、字体选择、开关点击等）统一调用 `saveSettings()`
6. 所有加载入口（init、进入设置页、reloadSettings）统一调用 `loadSettings()`
7. 删除 9 个独立的 `loadXxxSetting()` 函数及其 `window.*` 导出
8. 清理 `ai-chat.js` 中 3 处 AI toggle 的 `localStorage` fallback，改用 `loadSettings()`
9. 清理 `ai-chat.js` 工具栏中 3 处独立的 `SetSetting()` 保存，统一走 `saveSettings()`

## Impact

- Affected specs: 无（不影响已有功能行为）
- Affected code: `app.go`, `types.go`, `main.js`, `data-management.js`, `ai-chat.js`
- 非影响：`setting_service.go`（保持泛用 Get/Set），`db.go`（InitDefaultSettings 不动）
- 兼容性：前端不再调用 `getAIConfig/SaveAIConfig`，但后端保留这两个方法不删（其他模块可能仍在用）

## ADDED Requirements

### Requirement: SettingsConfig 结构体
The system SHALL define a `SettingsConfig` struct in `internal/services/types.go` with all 20 settings fields.

### Requirement: GetAllSettings
The system SHALL provide a `GetAllSettings() SettingsConfig` method on `App` that reads all 20 settings from DB and returns the typed struct.

### Requirement: SaveAllSettings
The system SHALL provide a `SaveAllSettings(cfg SettingsConfig) error` method on `App` that writes all 20 fields to DB with proper type validation and range checks.

### Requirement: Single loadSettings on frontend
The system SHALL provide a single `loadSettings()` function on the frontend that:
- Calls `GetAllSettings()` once
- Fills all settings page DOM elements
- Applies visual settings (theme, font, syntax highlight)
- Syncs AI toolbar toggle states in ai-chat.js
- Replaces all 9 existing `loadXxxSetting()` functions

### Requirement: Single saveSettings on frontend
The system SHALL provide a single `saveSettings()` function on the frontend that:
- Gathers all settings values from DOM
- Calls `SaveAllSettings(cfg)` once
- Replaces all existing individual `SetSetting()` and `SaveAIConfig()` calls in settings page

### Requirement: reloadSettings simplification
`reloadSettings()` in `data-management.js` SHALL call only `window.loadSettings()` instead of 9 individual load calls.

### Requirement: AI chat toolbar integration
AI chat toolbar (`ai-chat.js`) SHALL use `loadSettings()` data instead of individual `GetSetting()` calls with localStorage fallback. Toolbar toggle saves SHALL call `saveSettings()`.

## MODIFIED Requirements

### Requirement: Init sequence
The app init sequence SHALL call `loadSettings()` instead of 9 individual `loadXxxSetting()` calls.

### Requirement: Settings page entry
`showTargetView('settings')` SHALL call `loadSettings()` instead of 8 individual settings load calls.

### Requirement: window exports
Replace 9 individual `window.loadXxxSetting = loadXxxSetting` exports with single `window.loadSettings = loadSettings`.

## REMOVED Requirements

### Requirement: Individual load functions
**Reason**: Redundant with unified `loadSettings()`.
**Migration**: Remove `loadThemeSetting`, `loadFontSettings`, `loadSortSettings`, `loadPageSizeSetting`, `loadQuickNoteSetting`, `loadSyntaxHighlightSetting`, `loadCodeHighlightThemeSetting`, `loadNoteOpenFullscreenSetting`, `loadAISettings`.

### Requirement: Individual save points
**Reason**: Replaced by unified `saveSettings()` with field-level idempotency.
**Migration**: All `onchange`/`onclick` save handlers call `saveSettings()`.
