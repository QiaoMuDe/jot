# 搜索弹窗日历日期选择器 Spec

## Why

当前搜索弹窗的时间筛选只有 4 个预设选项（不限/今天/最近7天/最近30天），且**实际不生效**（后端无日期参数，前端无过滤逻辑）。用户需要更灵活的任意日期范围筛选——通过日历界面选择开始和结束日期。

## What Changes

1. **后端**：`SearchNotes` 函数签名增加 `startDate`/`endDate` 可选字符串参数（ISO 日期格式 `"2006-01-02"`），SQL 追加 `updated_at BETWEEN` 条件
2. **前端状态**：替换 `state.searchModalDateRange: 'all'` 为 `state.searchModalDateStart: ''` + `state.searchModalDateEnd: ''`
3. **前端 UI**：将日期筛选从静态 HTML 下拉菜单改为日历弹窗（自定义纯 JS 实现，无外部依赖）
4. **前端过滤**：`searchModalLoadPage` 将日期范围参数传给后端，真正执行过滤
5. **Wails 绑定**：更新 Wails 自动生成的 JS/TS 绑定以匹配新的函数签名

## 日历选择器设计方案

### 设计方向

- **风格**：精致轻量，与搜索弹窗现有设计语言一致（暖色调、CSS 变量驱动、spring 动画）
- **定位**：点击"时间"筛选按钮后，在按钮下方弹出（替换旧下拉菜单）
- **布局**：单面板月视图，7 列 CSS Grid，底部快捷预设行
- **字体**：使用项目现有字体系统（`--font-family`），星期头用 `0.688rem` 字重 600
- **配色**：全部使用现有 CSS 变量（`--accent`、`--accent-light`、`--bg`、`--card-bg`、`--text-primary`、`--text-muted`、`--border`、`--hover-bg`），6 主题自动适配

### 日历布局结构

```
┌─────────────────────────────────┐
│  2026年6月                ←  →  │ ← header: 月年标签 + 前后翻月按钮
├─────────────────────────────────┤
│  日  一  二  三  四  五  六     │ ← weekday headers
├─────────────────────────────────┤
│         1   2   3   4   5   6   │
│   7   8   9  10  11  12  13    │
│  14  15  16  17  18  19  20    │ ← 6 行日期网格
│  21  22  23  24  25  26  27    │   今天: 深色圆形标记
│  28  29  30                    │   选中范围: 浅琥珀背景
│                                │   开始/结束: 实心圆角标记
├─────────────────────────────────┤
│  今天    最近7天    最近30天    │ ← 预设快捷按钮
│  不限                          │
└─────────────────────────────────┘
```

### 交互状态矩阵

| 元素 | 状态 | 样式 |
|------|------|------|
| 日格子 | 默认 | `--text-primary` 文字，36×36px 圆形区域，`font-size: 0.813rem` |
| 日格子 | hover | `--hover-bg` 圆形背景，`cursor: pointer`，150ms 过渡 |
| 日格子 | 今天 | 文字 `font-weight: 700` + `--accent` 色 + 底部 2px `--accent` 实心圆点标记 (6px 直径) |
| 日格子 | 已选开始/结束 | `--accent` 实心圆角背景 + `#fff` 白色文字，`border-radius: 50%` |
| 日格子 | 范围内（非端点） | `--accent-light` (alpha 0.25) 背景，两端无圆角 |
| 日格子 | 非本月日期 | `--text-muted` 灰色文字，opacity 0.35，无 hover 效果 |
| 日格子 | 禁用（过去/未来） | 无（所有日期可选，包括今天之前的日期） |
| 预设按钮 | 默认 | `--card-bg` 背景 + `--border` 边框 + `--text-secondary` 文字，`padding: 4px 12px`，`border-radius: 6px` |
| 预设按钮 | hover | `--hover-bg` 背景，150ms 过渡 |
| ← → 按钮 | 默认 | 32×32px，`--text-secondary` 颜色，`opacity: 0.6` |
| ← → 按钮 | hover | `--hover-bg` 圆形背景 + `opacity: 1`，150ms 过渡 |

### 尺寸约束

- 日历容器宽度：`280px`（与过滤器按钮对齐）
- 日格子：36×36px（交互区域），实际文字 24×24px
- Header 高度：40px（含上下 padding）
- 快捷预设行：36px
- 日历总高度：~320px（动态，随月份变化）

### 动画

- 弹出：`opacity 0→1` + `translateY(-6px→0)`，200ms `cubic-bezier(0.16, 1, 0.3, 1)`
- 关闭：`opacity 1→0` + `translateY(0→-4px)`，150ms `ease-in`
- 月切换：内容即时替换（无过渡动画，保持响应感）
- 格子 hover：`background-color` 150ms `ease`
- `prefers-reduced-motion`: 全部动画降级为 0ms

## Impact

- Affected specs: refine-search-modal-ui（任务全部完成，无冲突）
- Affected code:
  - `frontend/src/main.js` — 状态管理、date renderer、searchModalLoadPage 传参、日历事件绑定
  - `frontend/src/style.css` — 新增日历选择器 ~150 行样式
  - `frontend/index.html` — 替换日期下拉 HTML 结构
  - `internal/services/note_service.go` — `Search()`/`SearchByNotebook()` 追加日期 WHERE 条件
  - `internal/services/types.go` — 可能需新增日期参数结构
  - `app.go` — `SearchNotes` 函数签名变更，接收日期参数

## ADDED Requirements

### Requirement: 后端支持日期范围查询

The system SHALL support filtering search results by a date range on `updated_at`.

#### Scenario: 搜索带日期范围
- **WHEN** frontend calls `SearchNotes(keyword, page, pageSize, notebookID, startDate, endDate)`
- **AND** `startDate` and `endDate` are non-empty ISO date strings (e.g. `"2026-06-01"`)
- **THEN** the SQL query SHALL include `WHERE updated_at BETWEEN '2026-06-01 00:00:00' AND '2026-06-25 23:59:59'`

#### Scenario: 日期参数为空
- **WHEN** `startDate` is empty or `endDate` is empty
- **THEN** the SQL query SHALL NOT include any date range condition (backwards compatible)

### Requirement: 日历日期选择器 UI

The system SHALL provide a calendar-based date range picker in the search modal.

#### Scenario: 首次打开日历
- **WHEN** user clicks the "时间" filter button
- **THEN** a calendar popup appears below the button
- **AND** the calendar shows the current month by default
- **AND** "today" cell is visually marked
- **AND** the header shows "选择开始日期" as prompt

#### Scenario: 选择开始日期
- **WHEN** user clicks a date cell
- **AND** no start date is currently selected
- **THEN** that date is marked as the start date with a rounded filled marker
- **AND** the header updates to "选择结束日期"
- **AND** the calendar stays open

#### Scenario: 选择结束日期（正常范围）
- **WHEN** user clicks a date cell
- **AND** a start date is already selected
- **AND** the clicked date is >= start date
- **THEN** the clicked date is marked as the end date with a rounded filled marker
- **AND** all dates between start and end are highlighted with a range background
- **AND** the calendar closes
- **AND** the filter button label updates to "2026-06-01 ~ 2026-06-25"
- **AND** `_triggerFilterSearch()` fires with the date range

#### Scenario: 选择结束日期（点击早于开始日期）
- **WHEN** user clicks a date cell
- **AND** a start date is already selected
- **AND** the clicked date is < start date
- **THEN** the clicked date replaces the start date (reset start)
- **AND** the header prompt stays "选择结束日期"
- **AND** the calendar stays open

#### Scenario: 点击已有范围
- **WHEN** both start and end dates are already displayed in the calendar
- **AND** user clicks a date within the selected range
- **THEN** the selection is preserved (no change)

#### Scenario: 使用快捷预设
- **WHEN** user clicks "今天" quick button
- **THEN** start = today, end = today, dropdown closes, filter fires
- **WHEN** user clicks "最近7天" quick button
- **THEN** start = today-6, end = today, dropdown closes, filter fires
- **WHEN** user clicks "最近30天" quick button
- **THEN** start = today-29, end = today, dropdown closes, filter fires
- **WHEN** user clicks "不限" quick button
- **THEN** start = '', end = '', dropdown closes, filter fires

#### Scenario: 清除日期范围
- **WHEN** a range is selected and user clicks "清除" button in the filter button area
- **THEN** start and end reset to empty string
- **AND** the filter button label returns to "不限"
- **AND** `_triggerFilterSearch()` fires (no date filter applied)

### Requirement: 日历导航与月切换

#### Scenario: 上月/下月切换
- **WHEN** user clicks the "←" (prev month) button
- **THEN** the calendar re-renders to show the previous month
- **AND** any previously selected start/end dates that fall within visible month are still highlighted
- **WHEN** user clicks the "→" (next month) button
- **THEN** the calendar re-renders to show the next month

### Requirement: 结果同步

#### Scenario: 日期筛选与实际搜索结果同步
- **WHEN** search returns results with a date range active
- **THEN** only notes whose `updated_at` falls within the range are displayed
- **AND** the total count in the footer reflects the filtered count

## MODIFIED Requirements

### Requirement: SearchNotes 后端函数

**Before (app.go):**
```go
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint) (*services.PaginatedResult, error)
```

**After:**
```go
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, startDate, endDate string) (*services.PaginatedResult, error)
```

**Before (note_service.go Search/SearchByNotebook):**
```go
func (s *NoteService) Search(keyword string, page, pageSize int, notebookID uint) (*PaginatedResult, error) {
    // WHERE title LIKE '%kw%' OR content LIKE '%kw%'
    // AND deleted_at IS NULL
}
```

**After:**
```go
func (s *NoteService) Search(keyword string, page, pageSize int, notebookID uint, startDate, endDate string) (*PaginatedResult, error) {
    // WHERE title LIKE '%kw%' OR content LIKE '%kw%'
    // AND deleted_at IS NULL
    // AND (startDate/endDate 非空时) updated_at BETWEEN 'startDate 00:00:00' AND 'endDate 23:59:59'
}
```

## REMOVED Requirements

### Requirement: 静态日期下拉菜单

**Reason**: 替换为日历弹窗选择器

**Migration**: 
- HTML 中 `#searchModalDateDropdown` 的静态选项（不限/今天/最近7天/最近30天）将被删除
- 快捷预设（今天/最近7天/最近30天/不限）保留，移到日历弹窗底部作为 quick buttons
- `state.searchModalDateRange: 'all'` 替换为 `searchModalDateStart: ''` + `searchModalDateEnd: ''`
- `renderDateFilterDropdownSelection()` 替换为 `renderDatePickerCalendar()`
- 删除旧 CSS 中日期下拉特有样式（通用 `.search-modal-filter-dropdown` 样式保留给笔记本/标签使用）
