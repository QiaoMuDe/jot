# 日历 UI 美化计划

## 概述

对日历视图进行视觉打磨，包含 8 项具体改动。97% 为 CSS 改动，1 项 HTML 文案调整，2 处 HTML SVG 尺寸修改。

## 改动清单

### 1. 面板间距 & 取消硬分割线

**文件**: `frontend/src/css/components/calendar.css`

| 选择器 | 修改 | 原因 |
|--------|------|------|
| `.calendar-layout` | `gap: 0` → `gap: 12px` | 左右面板之间留出间距 |
| `.calendar-sidebar` | 删除 `border-right: 1px solid var(--border)`，增加 `background: var(--card-bg)` | 用背景色深浅分区代替生硬分隔线 |
| `.calendar-notes-panel` | `padding: 20px 0 20px 24px` → `padding: 20px 0 20px 0` | 间距由 gap 提供，无需额外左内边距 |

### 2. 导航箭头放大

**文件**: `frontend/index.html`（2 个 SVG）

| 元素 | 修改 |
|------|------|
| `#calendarPrevBtn svg` | `width="14" height="14"` → `width="16" height="16"` |
| `#calendarNextBtn svg` | `width="14" height="14"` → `width="16" height="16"` |

### 3. 选中日期外圈高亮

**文件**: `frontend/src/css/components/calendar.css`

```css
.calendar-day.selected {
    background: var(--accent);
    color: #fff;
    font-weight: 600;
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 30%, transparent);  /* 新增 */
}
```

### 4. 日期格子加大

**文件**: `frontend/src/css/components/calendar.css`

| 选择器 | 修改 |
|--------|------|
| `.calendar-day` | `width: 36px; height: 36px;` → `width: 40px; height: 40px;` |

### 5. 统计卡片：数字放大、间距加大

**文件**: `frontend/src/css/components/calendar.css`

| 选择器 | 属性 | 修改 |
|--------|------|------|
| `.calendar-stat-value` | `font-size` | `1.125rem` → `1.375rem` |
| `.calendar-stats-grid` | `gap` | `12px` → `16px` |
| `.calendar-stats-bar` | `margin-top` | `6px` → `12px` |
| `.calendar-stats-bar` | `margin-bottom` | 新增 `4px` |

### 6. 日期标题放大、增加底边距

**文件**: `frontend/src/css/components/calendar.css`

| 选择器 | 属性 | 修改 |
|--------|------|------|
| `.calendar-date-label` | `font-size` | `1rem` → `1.25rem` |
| `.calendar-notes-header` | `margin-bottom` | `12px` → `16px` |

### 7. 笔记条目去掉方框，改底部分割线

**文件**: `frontend/src/css/components/calendar.css`

| 选择器 | 修改前 | 修改后 |
|--------|--------|--------|
| `.calendar-note-item` | `background: var(--card-bg)` | 删除 |
| `.calendar-note-item` | `border: 1px solid var(--border)` | 删除 |
| `.calendar-note-item` | `border-radius: 8px` | 删除 |
| `.calendar-note-item` | `padding: 12px 16px` | `padding: 14px 4px` |
| `.calendar-note-item` | — | 新增 `border-bottom: 1px solid var(--divider)` |
| `.calendar-note-item:last-child` | — | 新增 `border-bottom: none` |
| `.calendar-note-item:hover` | `transform: translateX(4px)` | 删除；hover 效果改为 `background: var(--hover-bg)` |
| `.calendar-note-item:active` | `transform: translateX(4px) scale(0.99)` | 改为 `transform: scale(0.99)` |

### 8. 笔记内部层级优化

**文件**: `frontend/src/css/components/calendar.css`

- `.calendar-note-notebook`: 增加标签样式
  - `background: var(--accent-lighter)`
  - `color: var(--accent-dark)`
  - `padding: 1px 6px`
  - `border-radius: 3px`
  - `font-size: 0.688rem`
- `.calendar-note-time`: 弱化显示
  - `opacity: 0.6`
  - `font-size: 0.688rem`

### 9. 空状态文案优化

**文件**: `frontend/index.html`

- `这天还没有笔记` → `该日期暂无笔记`

## 验证步骤

1. `npm run build` 构建无错误
2. 检查左右面板间距正常（无硬分割线，背景色区分）
3. 检查导航箭头尺寸放大，点击热区正常
4. 检查选中日期有外圈高亮 + 底色强调
5. 检查日期格子 40×40，点击区域变大
6. 检查统计卡片数字放大、间距增加
7. 检查日期标题字号变大，与笔记列表间距拉开
8. 检查笔记条目无外边框，只有底部分割线，hover 效果平滑
9. 检查笔记本标签有彩色标签样式，时间文字弱化
10. 检查空状态显示"该日期暂无笔记"
