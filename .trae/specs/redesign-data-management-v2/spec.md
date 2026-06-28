# 数据管理页面 UI 重新设计 Spec

## Why

当前数据管理页面的操作按钮采用全宽卡片式设计（`padding: 18px 20px`、`40×40px` 大图标盒、边框+阴影），视觉上笨重臃肿，与 jot 整体简约精致的调性不匹配。用户明确反馈"大按钮看起来怪怪的"。需要从根本上改变交互元素的视觉语言。

## What Changes

- **操作按钮**：从全宽卡片按钮改为**设置列表行样式**（类似 iOS Settings），紧凑行布局：小图标 + 标签 + 描述 + 右箭头 `>`，点击整行触发
- **统计卡片**：保持 5 列网格，但卡片内边距缩小、数值字号微调，整体更紧凑轻量
- **分层结构**：移除外层 `.data-section-card` 边框卡片容器，直接使用内容区域+分隔线区分层次，减少视觉嵌套
- **快速备份**：整合到操作列表中作为一个分组，不再独立为单独的外框卡片
- **图标**：使用更小的 24×24px 或 20×20px 图标，不再用 40×40px 大图标背景盒

## Impact

- Affected code:
  - `frontend/index.html` — 数据管理区域 HTML 结构完全重写
  - `frontend/src/css/components/data-view.css` — 全部样式重写
  - `frontend/src/main.js` — 无需改动（函数名/ID 不变，仅 DOM 结构变化）
- No breaking changes。所有后端 API 和 JS 函数不变。

## ADDED Requirements

### Requirement: 统计卡片区

- 保持 5 列网格，5 张卡片
- 卡片更紧凑：`padding: 16px 12px`，去掉 hover 位移效果，仅保留颜色变化
- 数值 `1.25rem 700w`，标签 `0.688rem`
- 卡片背景用 `var(--card-bg)`，无边框或 `1px solid var(--border)` 极淡边框
- 入场交错动画保留

### Requirement: 操作列表区

- 取代现有的全宽卡片按钮，采用**设置列表行**样式
- 每行结构：左侧小图标（20×20px、无背景盒） + 中间标签/描述 + 右侧 `>` 箭头
- 行高约 48px，`padding: 0 16px`，`gap: 12px`
- hover 时整行背景变色（`var(--hover-bg)`）
- 行与行之间用 `1px solid var(--border)` 分隔线隔开（左侧缩进）
- 危险操作（恢复出厂设置）在视觉上独立分隔，行文本红色

### Requirement: 功能分组

- 操作列表分为三个功能组，组间用间距分隔：
  1. **数据迁移**：导出数据、导入数据
  2. **数据库维护**：数据库瘦身、打开数据目录
  3. **快速备份**：备份信息（文字标签）、一键备份、一键还原
  4. **危险操作**（底部）：恢复出厂设置

### Requirement: 危险操作独立区域

- 在底部独立显示，与上方操作列表有明显间距
- 使用红色文本 + 红色分隔线
- 点击前需二次确认（已有逻辑不变）

## MODIFIED Requirements

无

## REMOVED Requirements

### Requirement: 卡片式操作按钮
**Reason**: 全宽卡片按钮视觉笨重，不符合简约设计方向
**Migration**: 替换为设置列表行样式