# Tasks - 日历日期选择器保持打开支持多选

## Task Dependencies

无依赖，仅为前端 JS 改动。

---

## Tasks

- [x] **Task 1**: 移除结束日期选择后的自动关闭
  - 在 `handleDatePickerDateClick()` 中，当结束日期被选中时（`start && !end && dateStr >= start` 分支），移除 `closeSearchModalDatePicker()` 调用
  - 保留 `_updateDateLabel()`、`updateSearchModalFilterBtnActive()`、`_triggerFilterSearch()` 调用
  - 日历保持打开，用户可继续调整

- [x] **Task 2**: 移除快捷预设按钮的自动关闭
  - 在快捷按钮（今天/7天/30天/不限）点击事件中，移除 `closeSearchModalDatePicker()` 调用
  - 保留 `_updateDateLabel()`、`updateSearchModalFilterBtnActive()`、`_triggerFilterSearch()` 调用
  - 日历保持打开，用户可查看选中效果

- [x] **Task 3**: 确认日历关闭方式正常
  - 验证点击日历外部区域 (`document` 的 `click` 事件) 能关闭日历 ✅
  - 验证再次点击"时间"筛选按钮 `toggleDatePicker()` 能关闭日历 ✅
  - 验证点击其他筛选器（笔记本/标签）下拉时，日历被 `closeAllFilterDropdowns()` 关闭 ✅
  - 以上行为无需代码变更（现有逻辑已覆盖）

## 验证

- [x] 构建验证：`npx vite build` 无错误
- [x] 选择开始日期 → 日历保持打开，提示变为"请选择结束日期"
- [x] 选择结束日期 → 日历保持打开，范围高亮，搜索立即执行
- [x] 已有范围后再次点击 → 范围清空，变为新开始日期，日历保持打开
- [x] 点击快捷预设 → 范围切换，搜索执行，日历保持打开
- [x] 点击外部区域 → 日历关闭（现有关闭代码不变）
- [x] 再次点击时间按钮 → 日历关闭（现有关闭代码不变）
