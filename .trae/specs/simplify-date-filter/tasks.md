# Tasks - 简化时间筛选：去掉日历弹窗，保留快捷按钮

## Task Dependencies

- [Task 1] 无需依赖
- [Task 2] 无需依赖
- [Task 3] 依赖 Task 1-2（验证需要所有改动就位）

---

## Tasks

- [x] **Task 1**: 将 HTML 日历弹窗替换为下拉菜单
  - ✅ `index.html` 中 `.search-modal-date-picker` 替换为 `.search-modal-filter-dropdown`（空容器，由 JS 渲染）

- [x] **Task 2**: 简化前端 JS 逻辑
  - ✅ 保留 `state.searchModalDateStart` / `searchModalDateEnd` 状态变量
  - ✅ 移除函数：`renderDatePickerCalendar` / `handleDatePickerDateClick` / `toggleDatePicker` / `_updateDateLabel`
  - ✅ `closeSearchModalDatePicker` 保留为空函数（兼容引用）
  - ✅ 移除 `_datePickerYear` / `_datePickerMonth` 全局变量
  - ✅ `updateSearchModalFilterBtnActive()` 日期部分保持（状态变量名未变）
  - ✅ 新增 `renderDateFilterDropdownSelection()` 渲染 4 个下拉选项（今天/7天/30天/不限）
  - ✅ 移除日期按钮的独立 click 事件
  - ✅ 移除日历导航按钮（上月/下月）事件绑定
  - ✅ 移除日历快捷按钮事件绑定
  - ✅ `closeAllFilterDropdowns()` 中移除 `closeSearchModalDatePicker()` 调用
  - ✅ 选中选项后正确设置日期范围 → 关闭下拉 → `_triggerFilterSearch()`

- [x] **Task 3**: 清理 CSS
  - ✅ 移除全部 `.search-modal-date-picker-*` 选择器（原 4271-4474 行，~200 行）

- [x] **Task 4**: 验证
  - ✅ `npx vite build` 无错误
  - ✅ 时间筛选按钮通过通用按钮处理器触发下拉
  - ✅ 下拉选项通过 `_createFilterOption` 渲染，含 ✓ 勾选标记
  - ✅ 选中后 label 更新、搜索执行、下拉关闭
