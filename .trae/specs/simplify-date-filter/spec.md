# 简化时间筛选：去掉日历弹窗，保留快捷按钮 Spec

## Why

当前时间筛选功能包含一个完整的日历日期选择器（月视图、月导航、范围选择），交互复杂（需理解"选开始→选结束"两步操作），维护成本高（~520 行 JS+CSS）。用户实际最常用的是四个快捷预设按钮（今天/最近7天/最近30天/不限）。改为和笔记本/标签筛选器一致的下拉菜单，降低交互门槛，减少代码量。

## What Changes

1. **HTML**：将 `.search-modal-date-picker` 日历弹窗替换为标准的 `.search-modal-filter-dropdown` 下拉菜单，内含 4 个选项
2. **CSS**：移除全部 `.search-modal-date-picker-*` 样式（~200 行），复用已有的 `.search-modal-filter-dropdown` 和 `.search-modal-filter-option` 样式
3. **JS**：移除日历相关函数（`renderDatePickerCalendar`/`handleDatePickerDateClick`/`toggleDatePicker`/`closeSearchModalDatePicker`/`_updateDateLabel`），替换为简单下拉渲染
4. **状态管理**：保留 `state.searchModalDateStart`/`searchModalDateEnd`（后端仍需它们）
5. **后端**：不变 — 后端 `SearchNotes` 已接收 `startDate/endDate` 参数，快捷按钮在前端计算出具体日期后通过同一接口传入，SQL `BETWEEN` 条件逻辑不变，后端无感知

## Impact

- Affected specs: add-calendar-date-picker（回退日历 UI）、fix-date-picker-auto-close（被本 spec 替代）
- Affected code:
  - `frontend/index.html` — 替换日历 HTML 结构为下拉菜单
  - `frontend/src/style.css` — 移除 ~200 行日历专属样式
  - `frontend/src/main.js` — 移除日历函数，替换为简单下拉选项渲染
  - 后端无变更

## ADDED Requirements

无（仅简化，不新增功能）

## MODIFIED Requirements

### Requirement: 时间筛选交互

**Before:**
- 点击"时间"筛选按钮 → 弹出日历弹窗（月视图 + 月导航 + 日期网格）
- 点击日期 → 设为开始/结束
- 选完结束日期 → 搜索执行
- 底部有 4 个快捷按钮

**After:**
- 点击"时间"筛选按钮 → 弹出下拉菜单（和笔记本/标签筛选器完全一致）
- 4 个选项：今天 / 最近7天 / 最近30天 / 不限
- 选中后立即执行搜索
- 选中的选项显示 ✓ 勾选标记

### Requirement: 筛选状态同步

**Before:**
- 选中日期范围 → 按钮 label 显示 "2026-06-01 ~ 2026-06-25"
- 按钮 active 态样式

**After:**
- 选中快捷选项 → 按钮 label 显示对应的中文名称（"今天"/"最近7天"/"最近30天"）
- 不限 → 显示"不限"
- 按钮 active 态样式保留

## REMOVED Requirements

### Requirement: 日历日期选择器 UI

**Reason**: 交互复杂，用户觉得麻烦，改用快捷下拉菜单替代

**Migration**:
- 移除 HTML 中 `.search-modal-date-picker` 及其子元素
- 移除 CSS 中所有 `.search-modal-date-picker-*` 选择器（~200 行）
- 移除 JS 中 `renderDatePickerCalendar()`/`handleDatePickerDateClick()`/`toggleDatePicker()`/`closeSearchModalDatePicker()`/`_updateDateLabel()` 及相关逻辑
