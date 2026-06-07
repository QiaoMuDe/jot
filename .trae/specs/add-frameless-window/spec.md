# 无边框窗口与自定义标题栏 Spec

## Why

当前 Jot 使用 Windows 原生标题栏，存在两个视觉问题：
1. **启动闪烁**：WebView 加载前窗口显示深蓝色背景，与页面实际背景色反差大
2. **失去焦点变黑**：深色主题下窗口失去焦点时，DWM 将标题栏渲染为系统默认黑色，与主题色反差极大

通过启用 Wails Frameless 模式，用前端 HTML/CSS 完全自定义标题栏，可彻底解决以上问题，同时让窗口装饰 100% 融入应用主题。

## What Changes

- **Go 后端**：启用 `Frameless: true`，移除 `CustomTheme` 和 `BackgroundColour` 相关逻辑
- **前端 HTML**：在现有 `#topbar` 上方新增自定义窗口标题栏 `#windowTitleBar`
- **前端 CSS**：新增标题栏样式，适配全部 6 套主题
- **前端 JS**：绑定窗口控制按钮（最小化/最大化/关闭），监听窗口最大化状态
- **移除**：`getWindowsOptions()`、`ApplyWindowTheme()` 中的 DWM API 调用、`findMainWindow()`、`setWindowAttribute()`

## Impact

- Affected specs: `add-theme-system`（主题颜色复用）
- Affected code:
  - `main.go` — Frameless 配置
  - `app.go` — 移除 DWM API 相关代码
  - `frontend/index.html` — 新增自定义标题栏
  - `frontend/src/style.css` — 新增标题栏样式
  - `frontend/src/main.js` — 窗口控制按钮事件绑定

## ADDED Requirements

### Requirement: 自定义窗口标题栏

The system SHALL 在页面顶部提供一个自定义标题栏，替代 Windows 原生标题栏。

#### Scenario: 标题栏显示
- **WHEN** 应用启动
- **THEN** 窗口顶部显示自定义标题栏，高度 32px，背景色跟随当前主题 `--topbar-bg`
- **AND** 左侧显示应用名称 "Jot"（可拖拽区域）
- **AND** 右侧显示最小化、最大化、关闭三个按钮

#### Scenario: 窗口拖拽
- **GIVEN** 用户在标题栏空白区域按住鼠标左键
- **WHEN** 拖动鼠标
- **THEN** 整个窗口跟随鼠标移动
- **AND** 三个控制按钮区域不可触发拖拽

#### Scenario: 双击最大化/还原
- **GIVEN** 用户在标题栏空白区域双击
- **WHEN** 窗口当前未最大化
- **THEN** 窗口最大化
- **WHEN** 窗口当前已最大化
- **THEN** 窗口还原为之前的大小和位置

### Requirement: 窗口控制按钮

The system SHALL 提供最小化、最大化/还原、关闭三个按钮，调用 Wails Runtime API。

#### Scenario: 最小化
- **WHEN** 用户点击最小化按钮
- **THEN** 调用 `runtime.WindowMinimise()` 最小化窗口

#### Scenario: 最大化/还原
- **WHEN** 用户点击最大化按钮且窗口未最大化
- **THEN** 调用 `runtime.WindowToggleMaximise()` 最大化窗口
- **AND** 按钮图标变为"还原"样式
- **WHEN** 用户点击还原按钮且窗口已最大化
- **THEN** 调用 `runtime.WindowToggleMaximise()` 还原窗口
- **AND** 按钮图标变为"最大化"样式

#### Scenario: 关闭
- **WHEN** 用户点击关闭按钮
- **THEN** 调用 `runtime.Quit()` 关闭应用

### Requirement: 主题适配（含失去焦点状态）

The system SHALL 自定义标题栏的颜色跟随当前主题自动变化，且**失去焦点时保持与激活状态完全相同的颜色**。

#### Scenario: 浅色主题
- **GIVEN** 当前主题为 default / light / nord
- **THEN** 标题栏背景色为浅色，按钮图标颜色为深色
- **AND** 窗口失去焦点后，标题栏背景色和按钮颜色保持不变

#### Scenario: 深色主题
- **GIVEN** 当前主题为 dark / monokai-pro / tokyo-night
- **THEN** 标题栏背景色为深色，按钮图标颜色为浅色
- **AND** 窗口失去焦点后，标题栏背景色和按钮颜色保持不变

#### Scenario: 失去焦点无视觉差异
- **GIVEN** 窗口当前处于任意主题
- **WHEN** 用户点击其他窗口使 Jot 失去焦点
- **THEN** 标题栏颜色、按钮颜色、文字颜色与激活状态完全一致
- **AND** 视觉上无法区分窗口是否处于激活状态

## MODIFIED Requirements

### Requirement: 启动背景色
原 `getThemeBackgroundColour()` 和 `BackgroundColour` 选项不再需要，因为 Frameless 模式下窗口边框已消失，不存在启动闪烁问题。

## REMOVED Requirements

### Requirement: DWM API 窗口主题同步
**Reason**: Frameless 模式下无原生标题栏，DWM API 不再适用。
**Migration**: 删除 `ApplyWindowTheme()`、`findMainWindow()`、`setWindowAttribute()` 及相关 Windows API 常量。
