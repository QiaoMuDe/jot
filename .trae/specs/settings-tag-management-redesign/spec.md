# 设置页标签管理卡片重构 Spec

## Why
当前标签管理面板布局存在问题：
- 添加表单在标签列表底部，标签多时需要滚动到底部才能操作
- 原生 `<input type="color">` 外观不一致且点击区域小
- 标签项缺乏互动反馈和视觉层次
- 空状态简陋
- 整体缺少设计感

## What Changes

### 1. HTML 结构调整
- **添加表单上移**：`.tag-add-form` 从 `.tag-list` 下方移到上方
- **预设色块选择器**：在原生 color input 后新增 `.color-presets`，包含 8 种半透明、带白色色块标记的颜色色块 + 1 个自定义入口（调色板图标）
- **标签计数**：每个标签项模板中新增 `<span class="tag-count">` 用于显示使用次数

### 2. CSS 样式重写
- **标签列表**：从 `flex-wrap: wrap` 改为 `display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr))` 网格布局
- **标签项卡片化**：`.tag-item` 使用 `card-bg` 背景、圆角、左侧 3px 颜色竖条 + 颜色圆点 + 名称 + 计数 + 删除按钮
- **添加区域视觉分离**：分隔线区分添加区和标签列表
- **空状态**：带 SVG 图标的完整空状态
- **动画**：添加淡入（`@keyframes tagFadeIn`）、删除淡出（`@keyframes tagFadeOut`）、hover 交互效果

### 3. JS 逻辑更新
- **`renderTagList()`**：使用新模板结构渲染，从后端 Tag 模型获取使用计数
- **预设色块交互**：点击选中色块更新 `selectedColor` 状态，自定义入口弹出原生 color picker
- **`createTag()`**：读取 `selectedColor` 变量而非直接读 input value
- **删除动画**：先添加 `.tag-deleting` 类，监听 `animationend` 后调后端删除

## Impact
- Affected specs: 无
- Affected code:
  - `frontend/index.html` — 标签管理面板 HTML 结构调整
  - `frontend/src/css/components/settings-panel.css` — 新增/修改标签管理相关样式
  - `frontend/src/main.js` — `renderTagList()`、`createTag()`、删除逻辑修改

## ADDED Requirements

### Requirement: 预设色块选择器
The system SHALL provide 8 preset color swatches + 1 custom color picker entry.

#### Scenario: 点击预设色块
- **WHEN** 用户点击预设色块
- **THEN** 该色块显示选中态（勾选标记），`selectedColor` 更新为该色块颜色

#### Scenario: 点击自定义入口
- **WHEN** 用户点击自定义入口（+ 图标）
- **THEN** 弹出原生 `<input type="color">`，选择后更新 `selectedColor` 并标记选中态

### Requirement: 标签项卡片化
The system SHALL render each tag as a card with left color bar, color dot, name, usage count, and delete button.

#### Scenario: 标签显示
- **GIVEN** 有标签数据
- **WHEN** `renderTagList()` 执行
- **THEN** 每个标签以卡片网格布局显示，包含颜色标识、名称、使用计数

### Requirement: 删除动画
The system SHALL animate tag deletion with a fade-out scale-down effect.

#### Scenario: 删除标签
- **WHEN** 用户点击删除按钮
- **THEN** 标签添加 `.tag-deleting` 类触发淡出动画
- **AND** `animationend` 事件后调用后端 API 删除

## MODIFIED Requirements

### Requirement: createTag 颜色来源
**修改前**：`createTag()` 直接读取 `els.newTagColor.value`
**修改后**：`createTag()` 读取 `selectedColor` 变量

## REMOVED Requirements
无
