# 日历日期选择器保持打开支持多选 Spec

## Why

当前日历日期选择器在用户选择结束日期后自动关闭（`closeSearchModalDatePicker()`），用户无法在日历上反复调整/查看已选范围，体验不便。需要改为**点击日期后日历保持打开**，用户可继续点击调整选择，直到主动关闭日历（点击外部或再次点击时间筛选按钮）。

## What Changes

1. **移除结束日期选择后的自动关闭行为**：`handleDatePickerDateClick()` 中选完结束日期后不再调用 `closeSearchModalDatePicker()`，改为保持日历打开
2. **移除快捷按钮点击后的自动关闭行为**：快捷预设点击后不再自动关闭日历，用户可查看选中效果后再手动关闭
3. **新增"确认"或关闭机制**：用户通过点击日历外部区域或再次点击时间筛选按钮来关闭日历
4. **多选/调整支持**：已有完整范围时点击日期→清空范围并以该日期为新开始（现有逻辑保留），但日历保持打开

## Impact

- Affected specs: add-calendar-date-picker（功能变更）
- Affected code:
  - `frontend/src/main.js` — `handleDatePickerDateClick()` 移除 close 调用；`toggleDatePicker()` 不变；快捷按钮点击移除 close
  - 无需 CSS 变更，无需后端变更

## ADDED Requirements

### Requirement: 日历选择后保持打开

#### Scenario: 选择结束日期后日历保持打开
- **WHEN** user has selected a start date
- **AND** user clicks a date >= start date as end date
- **THEN** the end date is set and visually marked
- **AND** the range between start and end is highlighted
- **AND** the filter button label updates to "YYYY-MM-DD ~ YYYY-MM-DD"
- **AND** the calendar stays open (不再自动关闭)
- **AND** `_triggerFilterSearch()` fires with the date range (搜索立即执行)

#### Scenario: 已有完整范围后再次点击调整
- **WHEN** both start and end dates are selected (完整范围)
- **AND** calendar is still open
- **AND** user clicks a new date
- **THEN** the existing range is cleared
- **AND** the clicked date becomes the new start date
- **AND** prompt updates to "请选择结束日期"
- **AND** calendar stays open

#### Scenario: 使用快捷预设后日历保持打开
- **WHEN** user clicks a quick preset button (今天/7天/30天/不限)
- **THEN** the date range is applied and label updates
- **AND** `_triggerFilterSearch()` fires
- **AND** the calendar stays open (allowing user to see the selected range)

### Requirement: 日历关闭方式

#### Scenario: 点击外部区域关闭日历
- **WHEN** the calendar date picker is open
- **AND** user clicks outside the `.search-modal-filter` container
- **THEN** the calendar closes (通过现有的 `document.addEventListener('click')` 处理)

#### Scenario: 再次点击时间筛选按钮关闭日历
- **WHEN** the calendar date picker is open
- **AND** user clicks the "时间" filter button again
- **THEN** the calendar closes (通过 `toggleDatePicker()` 的切换逻辑)

## MODIFIED Requirements

### Requirement: handleDatePickerDateClick 行为

**Before:**
- 选完结束日期 → 调用 `closeSearchModalDatePicker()` → 日历关闭
- 已有完整范围时点击 → 重置为开始 → 日历保持打开

**After:**
- 选完结束日期 → **不调用** `closeSearchModalDatePicker()` → 日历保持打开
- 已有完整范围时点击 → 重置为开始 → 日历保持打开（不变）

### Requirement: 快捷按钮行为

**Before:**
- 点击快捷预设 → 设置范围 → `closeSearchModalDatePicker()` → 日历关闭 → `_triggerFilterSearch()`

**After:**
- 点击快捷预设 → 设置范围 → `_triggerFilterSearch()` → 日历保持打开（移除 close 调用）

## REMOVED Requirements

无
