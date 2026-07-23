# 日历视图 Spec

## Why

笔记按时间创建，但当前缺少一个按**创建日期**浏览笔记的入口。日历视图让用户以"某天我写了什么"的视角回溯笔记，与首页卡片网格（按更新时间排序）形成互补。

## What Changes

1. **后端新增两个接口**：`GetNotesByDate` 按创建日期获取笔记列表，`GetMonthNoteCounts` 获取某月每天的笔记数（日历圆点标记）
2. **前端新增日历视图**：独立页面视图 `viewCalendar`，包含日历组件 + 日期笔记列表
3. **更多菜单新增入口**：在"待办清单"上方的"更多菜单"中添加"日历"项
4. **点击笔记跳转**：从日历视图点击笔记 → 切回网格视图 → 筛选到该笔记所在笔记本 → 打开编辑器

## 设计方向

### 美学定位

- **风格**：精致文具/手账风格（Editorial / Journalistic），融合 Jot 现有暖色调主题体系
- **差异化记忆点**：日历日期格底部的"墨点"圆点标记（ink dot）—— 用深浅不一的圆点表示笔记数量，而非传统数字角标。创作密集的日子圆点大而实，稀疏的日子小而淡，形成独特的视觉节奏
- **字体**：日历 header 用稍加粗的 `--font-family`，日期数字保持简洁
- **配色**：全部使用现有 CSS 变量（`--accent`、`--accent-light`、`--bg`、`--card-bg`、`--text-primary`、`--text-muted`、`--border`、`--hover-bg`），14 套主题自动适配
- **动画**：日历切换月份时淡入 + 微缩放，笔记列表 item 入场 stagger 动画

### 布局

```
┌─────────────────────────────────────────────────────┐
│  ← 返回     日历                     2026年 7月    │ ← view-header
├──────────────────────┬──────────────────────────────┤
│                      │  7月23日 星期四              │
│   2026年 7月         │  ┌─────────────────────────┐ │
│  ┌───────────────┐   │  │ 📝 项目会议记录         │ │
│  │  日 一 二 三 四 五 六│  │    笔记本: 工作       │ │
│  │        1  2  3  4│  │    创建于 14:30        │ │
│  │  5  6  7  8  9 10 11│  └─────────────────────────┘ │
│  │ 12 13 14 15 16 ● 18│  │ 📝 读书笔记 - 设计模式  │ │
│  │ 19 20 21 22 23 24 25│  │    笔记本: 学习        │ │
│  │ 26 27 28 29 30 31   │  │    创建于 09:15        │ │
│  └───────────────┘   │  └─────────────────────────┘ │
│  ◀  ▶                │                              │
│                      │  没有更多笔记了               │
│  左侧日历            │  右侧日期笔记列表            │
│  ~280px              │  flex: 1                     │
└──────────────────────┴──────────────────────────────┘
```

### 交互流程

```
用户打开日历视图
  ↓
日历渲染当前月份，有笔记的日期显示 ○ 圆点标记（圆点大小/透明度表示笔记数量）
  ↓
用户点击某天 → 右侧笔记列表区域加载该日创建的所有笔记
  ↓
用户点击某条笔记 → 触发跳转：
  1. 关闭编辑器（如有打开）
  2. switchView('grid')
  3. 设置 state.activeNotebookId = 笔记的 notebook_id
  4. 切换到对应笔记本
  5. loadNotes() 重新加载
  6. 调用 openEditor(note) 打开该笔记
```

## Impact

- Affected specs: none (独立新功能)
- Affected code:
  - `app.go` — 新增 `GetNotesByDate`、`GetMonthNoteCounts` 两个绑定方法
  - `internal/services/note_service.go` — 新增 `GetByDate`、`GetMonthCounts` 两个方法
  - `frontend/index.html` — 新增 `viewCalendar` HTML 容器 + 更多菜单入口
  - `frontend/src/main.js` — 新增 viewMap 项、els 引用、switchView handler、视图激活回调
  - `frontend/src/js/calendar.js` — **新文件**：日历视图的完整 JS 逻辑（日历渲染、笔记列表、跳转）
  - `frontend/src/css/components/calendar.css` — **新文件**：日历视图全部样式

## ADDED Requirements

### Requirement: 后端按创建日期查询

The system SHALL support querying notes by their creation date.

#### Scenario: 按日期获取笔记列表
- **WHEN** frontend calls `GetNotesByDate(date string)` with `"2006-01-02"` format
- **THEN** return all non-deleted notes whose `created_at` falls within that date, ordered by `created_at DESC`

#### Scenario: 按月获取每日笔记数
- **WHEN** frontend calls `GetMonthNoteCounts(year int, month int)` with `2026, 7`
- **THEN** return a map of `{"1": 3, "5": 1, "15": 7, ...}` where keys are day-of-month and values are note counts
- **AND** only non-deleted notes are counted

### Requirement: 日历视图 UI

The system SHALL provide a calendar-based view for browsing notes by creation date.

#### Scenario: 初始加载
- **WHEN** user opens the calendar view via more menu
- **THEN** the calendar shows the current month
- **AND** dates with notes have ink-dot markers
- **AND** today's notes are pre-loaded in the right panel

#### Scenario: 月份切换
- **WHEN** user clicks the "◀" (prev month) button
- **THEN** calendar re-renders to show the previous month
- **AND** ink-dot markers update for the new month
- **AND** right panel clears (no auto-load)

#### Scenario: 选择日期查看笔记
- **WHEN** user clicks a date cell in the calendar
- **THEN** that date is visually highlighted (accent background + white text)
- **AND** the right panel loads notes created on that date
- **AND** the date header updates (e.g., "7月23日 星期四")

#### Scenario: 日期无笔记
- **WHEN** user clicks a date that has no notes
- **THEN** the right panel shows an empty state: "这天还没有笔记" with a subtle icon

#### Scenario: 点击笔记跳转
- **WHEN** user clicks a note in the right panel list
- **THEN** the system performs these sequential actions:
  1. Close any open editor
  2. `switchView('grid')`
  3. Set `state.activeNotebookId` to the note's notebook ID
  4. Click the sidebar notebook item to activate it
  5. `loadNotes()` to reload the grid
  6. `openEditor(note)` to open the note

### Requirement: 墨水圆点标记 (Ink Dot)

#### Scenario: 笔记数量视觉映射
- **WHEN** a date has 1 note
- **THEN** a small faint dot appears below the date number (4px diameter, `--accent` color, 30% opacity)
- **WHEN** a date has 2-5 notes
- **THEN** the dot is 6px diameter, 60% opacity
- **WHEN** a date has 6+ notes
- **THEN** the dot is 8px diameter, 90% opacity

## MODIFIED Requirements

### Requirement: 更多菜单增加日历入口

**Before**: 更多菜单功能列表缺少日历入口

**After**: 在"待办清单"上方新增日历菜单项：
```html
<div class="dropdown-item" data-action="calendar">
  <svg><!-- 日历图标 --></svg>日历
</div>
```

对应 `main.js` 的 handler 新增：
```javascript
else if (item.dataset.action === 'calendar') {
    switchView('calendar');
    initCalendarView();
}
```

## REMOVED Requirements

无
