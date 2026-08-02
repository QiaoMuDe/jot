# 笔记卡片 UI 重构 — 方案 G（内容优先）Spec

## Why

当前笔记卡片 UI 样式陈旧，视觉层次扁平，标题和正文区分度不够，标签样式过于饱和突兀。方案 G 以"内容优先"为核心理念，通过大标题粗体、极浅阴影、紧凑圆角、幽灵风格标签等设计，让笔记内容本身成为视觉焦点，提升浏览体验。

## What Changes

- **CSS**: 重写 `frontend/src/css/components/main-content.css` 中 `.note-card` 及其子元素的样式规则
- **HTML 模板**: 修改 `frontend/src/main.js` 中卡片渲染模板，调整置顶标记方式
- **JS 逻辑**: 保留所有现有功能（置顶、批量选择、右键菜单、标签点击搜索等），仅调整 UI 表现

## Design Vision（方案 G 核心）

```
┌─────────────────────────────────┐  ← 圆角 10px, 纯色背景, 极浅阴影
│                                 │
│  ★ 设计系统规范文档              │  ← 1.1rem / 700w / 最大最粗标题
│                                 │
│  统一的组件库和设计语言是保证     │  ← 0.85rem / 灰色, 行高 1.6, 3 行截断
│  产品一致性的基础。本文档定义了   │
│  颜色、排版、间距、图标等...     │
│                                 │
│  ┌───────────────┬───────────┐  │
│  │ [设计] [规范]  │  🕐 昨天  │  │  ← 标签 & 时间 两端对齐
│  └───────────────┴───────────┘  │
│                                 │
│  📌 已置顶 (右上角, 置顶时显示)  │
│                             ← 3px 彩色边框 (置顶时显示)  │
└─────────────────────────────────┘
```

## Impact

- Affected specs: Card Grid, Note Card
- Affected code:
  - `frontend/src/css/components/main-content.css` — 卡片样式重写
  - `frontend/src/main.js` — 卡片模板调整（置顶标记方式）
  - CSS 变量 `variables.css` — 无需修改，复用现有令牌

## ADDED Requirements

### Requirement: 卡片容器样式重写
The system SHALL apply new card container styles:

- `border-radius: var(--radius-lg)` (10px) — 更紧凑
- `box-shadow: 0 1px 2px rgba(...)` — 极浅阴影，聚焦内容
- `border: 1px solid var(--border)` — 始终显示精致边框
- Hover: `translateY(-2px)`, `box-shadow` 升级到 `--shadow-md`, 边框变主题色
- Active: `scale(0.985)`

#### Scenario: 卡片 hover 交互
- **WHEN** 鼠标悬停在卡片上
- **THEN** 卡片上移 2px，阴影加深，边框变为主题色半透明

#### Scenario: 卡片点击交互
- **WHEN** 鼠标点击卡片
- **THEN** 卡片缩小到 0.985，松手恢复

### Requirement: 标题样式增强
The system SHALL display card titles with:

- `font-size: 1.1rem` — 当前最大字号
- `font-weight: 700` — 粗体
- `letter-spacing: -0.01em` — 字距略微收紧
- `line-height: 1.4`
- `display: -webkit-box`, `-webkit-line-clamp: 2` — 最多 2 行截断

### Requirement: 标签样式改为幽灵风格
The system SHALL render tags with ghost style:

- `font-size: 0.75rem`, `font-weight: 600` — 大号醒目
- `padding: 4px 12px`, `border-radius: var(--radius-sm)` (6px)
- 背景: `color-mix(in srgb, var(--tag-color) 12%, transparent)`
- 文字色: `var(--tag-color)`
- Hover: 背景加深到 22%，上移 1px
- 保留现有 `onclick` 事件和 `style="background-color: ..."` 内联样式，通过 CSS 变量 `--tag-color` 转换

### Requirement: 标签和时间两端对齐
The system SHALL position tags and time in the footer with `justify-content: space-between`:

- 标签在左侧，时间在右侧
- 无标签时时间单独在右侧
- 时间带 `🕐` SVG 图标

### Requirement: 置顶状态改为左侧边框
The system SHALL indicate pinned state with a left border:

- `.note-card.pinned`: `border-left: 3px solid var(--accent)`
- 置顶按钮改为 `.card-pin-badge` 在右上角，仅置顶时显示
- 保留 `note.pinned` 数据驱动逻辑

## MODIFIED Requirements

### Requirement: 卡片元素样式
The system SHALL update card element styles:

- `.card-body`: `padding: 18px 18px 0`
- `.card-content`: `font-size: 0.85rem`, `line-height: 1.6`, `margin-bottom: 14px`
- `.card-footer`: `padding: 0 18px 16px`, `display: flex`, `align-items: center`, `justify-content: space-between`, `gap: 12px`
- `.card-time`: `font-size: 0.75rem`, `display: flex`, `align-items: center`, `gap: 4px`

### Requirement: 卡片动作按钮位置调整
The system SHALL reposition the pin button:

- 由 `.card-actions`（右上角可交互按钮）改为 `.card-pin-badge`（只读标签）
- 置顶时显示 `📌 已置顶` badge，非置顶时隐藏
- 移除批量模式下的 `pin-btn` 显示逻辑

## REMOVED Requirements

### Requirement: 旧版卡片阴影
**Reason**: 方案 G 使用更浅的阴影和精致边框
**Migration**: 替换为 `box-shadow: 0 1px 2px` + `border: 1px solid var(--border)`

### Requirement: 旧版标签纯色背景
**Reason**: 方案 G 使用幽灵风格标签，更柔和精致
**Migration**: 标签使用 `color-mix` 生成半透明背景 + 彩色文字 + 细边框