# UI 视觉品质升级 Spec

## Why

当前 Jot 已有一套完整的 6 主题设计系统和组件样式，但在视觉质感上仍有提升空间。主要问题包括：图标使用 Unicode 字符而非 SVG 图标、组件层级感和精致度不够、色彩对比度和语义可进一步优化、部分交互缺少微反馈、空状态缺少视觉引导。目标是让应用视觉质感从"可用"提升到"精致"。

## What Changes

- **图标系统**：将 Unicode 字符按钮（`─ □ ✕ ⛶ ☰ ✎`）替换为 SVG 图标（统一线条风格）
- **色彩系统**：扩展语义化颜色（info/warning/success 等）、阴影分层更细致
- **组件精修**：卡片悬浮动效升级、按钮交互反馈层次化、编辑器面板视觉润色、侧栏书签隐喻强化
- **空状态/加载态**：为空视图和加载过程添加视觉化提示（非纯文字）
- **全局一致性**：统一圆角/间距 Token 使用、对齐 4px 网格

## Impact

- Affected specs: `redesign-ui`（原始视觉重设计）、`add-theme-system`（主题变量扩展）
- Affected code:
  - `frontend/src/app.css` — 主题变量扩展（新语义色、阴影层级）
  - `frontend/src/style.css` — 组件样式重写（图标、按钮、卡片、侧栏等）
  - `frontend/index.html` — 图标替换（SVG 替换 Unicode）
  - `frontend/src/main.js` — 新的微交互处理（如需）

## ADDED Requirements

### Requirement: SVG 图标系统

The system SHALL 使用矢量 SVG 图标替换所有 Unicode 字符图标。

#### Scenario: 窗口控制按钮
- **WHEN** 渲染最小化/最大化/关闭按钮
- **THEN** 使用 SVG 图标替换 `─ □ ✕` 字符
- **AND** SVG 图标颜色跟随 `currentColor`，适配所有主题

#### Scenario: 功能按钮
- **WHEN** 渲染编辑器工具栏（全屏/编辑/类型切换/关闭）
- **THEN** 使用 SVG 图标替换 `⛶ ✎ T ✕` 等字符
- **AND** 所有 SVG 图标使用统一的描边宽度（1.5px/2px）

#### Scenario: 统一图标集
- **WHEN** 渲染任何图标或符号
- **THEN** 所有图标来自同一个矢量图标集（如 Lucide/Heroicons），保持风格一致
- **AND** 图标尺寸使用设计 Token 统一管理

### Requirement: 色彩系统精化

The system SHALL 优化现有主题的色彩表现。

#### Scenario: 语义色扩展
- **WHEN** 显示状态反馈
- **THEN** 使用统一的语义色 Token（成功绿/警告黄/错误红/信息蓝），所有主题中保持可分辨
- **AND** 语义色文本在对应主题中满足 WCAG AA 4.5:1 对比度

#### Scenario: 阴影层次
- **WHEN** 渲染不同层级的组件（卡片/下拉菜单/模态框/Toast）
- **THEN** 使用分层的阴影系统（4+ 层级），每一层有独立的 spread/blur/opacity
- **AND** 阴影颜色融入主题色调（暖色主题偏暖，冷色主题偏冷）

#### Scenario: 卡片与背景层级
- **WHEN** 渲染卡片、侧栏、面板
- **THEN** 通过微妙的面板背景递进（bg → card-bg → elevated-card）建立视觉层次
- **AND** 不使用纯白/纯黑背景，使用含细微色调的"准白色"/"准黑色"

### Requirement: 组件交互升级

The system SHALL 为关键组件添加更丰富的交互反馈。

#### Scenario: 按钮按压
- **WHEN** 用户按下按钮
- **THEN** 按钮提供分层反馈：hover 轻微上浮 → active 按压缩放(0.97) → 释放恢复
- **AND** 主操作按钮（保存/确认）添加微妙的涟漪或发光反馈

#### Scenario: 卡片悬浮
- **WHEN** 鼠标悬浮在卡片上
- **THEN** 卡片上浮并放大阴影层级，边框微亮
- **AND** 过渡平滑（0.25s spring）

#### Scenario: 主题切换
- **WHEN** 用户切换主题
- **THEN** CSS 变量过渡平滑（300ms ease-out），无闪烁
- **AND** 切换时卡片/面板有统一的延迟过渡

### Requirement: 空状态与加载态优化

The system SHALL 为无数据状态提供更具引导性的视觉提示。

#### Scenario: 空笔记列表
- **WHEN** 当前笔记本没有笔记
- **THEN** 显示带有图标的空状态卡片，包括引导文案和操作提示（"点击 + 按钮创建第一条笔记"）

#### Scenario: 空搜索
- **WHEN** 搜索无结果
- **THEN** 显示搜索无结果的 SVG 插图 + 提示文案

#### Scenario: 加载态
- **WHEN** 笔记列表加载中
- **THEN** 显示骨架屏（skeleton cards）而非纯文本 loading 指示器

## MODIFIED Requirements

### Requirement: 字体一致性
现有字体保持为系统字体栈（`system-ui, -apple-system, sans-serif`），但确保所有组件统一使用字体 Token（`var(--font-family)`），消除硬编码字体声明。

### Requirement: 侧栏视觉
现有 `.notebook-sidebar` 的宽度从 `172px`（非 4px 网格对齐）改为 `176px`（44×4），与间距系统对齐。

### Requirement: 编辑器面板
现有 `.editor-panel` 的顶部 3px accent 色条改为更微妙的装饰处理，如渐变色或内嵌阴影线。

## REMOVED Requirements

无
