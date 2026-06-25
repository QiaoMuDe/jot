# Tasks - 搜索弹窗日历日期选择器

## Task Dependencies
- [Task 1] 必须先完成（后端改动，前端需依赖新的函数签名）
- [Task 2] 必须先完成（更新 Wails 绑定，否则前端调不到新 API）
- [Task 3] 依赖 Task 1-2（前端需要新的后端 API + 绑定）
- [Task 4] 依赖 Task 3（验证需要所有代码就位）

---

## Tasks

- [x] **Task 1**: 后端 `SearchNotes` 增加日期范围参数
  - [x] 1.1 在 `internal/services/note_service.go` 的 `Search()` 函数签名增加 `startDate, endDate string` 参数
  - [x] 1.2 在 `Search()` 中解析日期参数：若 `startDate` 和 `endDate` 均非空，在 WHERE 条件追加 `AND updated_at BETWEEN ? AND ?`
  - [x] 1.3 日期解析逻辑：`startDate 00:00:00` → `endDate 23:59:59`
  - [x] 1.4 同样的改动同步到 `SearchByNotebook()`（`SearchByNotebook` 需一致支持日期筛选）
  - [x] 1.5 在 `app.go` 的 `SearchNotes()` 绑定方法签名增加 `startDate, endDate string`，透传给 Service
  - [x] 1.6 构建验证：`go build ./...` 无错误

- [x] **Task 2**: 更新 Wails 自动生成的前端绑定文件
  - [x] 2.1 确认绑定文件已自动更新
  - [x] 2.2 确认新的 `SearchNotes` JS 函数接受 `(keyword, page, pageSize, notebookID, startDate, endDate)` 六个参数

- [x] **Task 3**: 前端日历日期选择器实现
  - [x] 3.1 状态管理：替换 `state.searchModalDateRange = 'all'` 为 `state.searchModalDateStart = ''` + `state.searchModalDateEnd = ''`
  - [x] 3.2 更新 `openSearchModal` 中日期状态重置逻辑
  - [x] 3.3 更新 `updateSearchModalFilterBtnActive()` 日期部分（判空代替判 `'all'`）
  - [x] 3.4 删除 `frontend/index.html` 中 `#searchModalDateDropdown` 的静态选项
  - [x] 3.5 在 `frontend/src/main.js` 新增日历相关函数（`renderDatePickerCalendar`/`handleDatePickerDateClick`/`_updateDateLabel`/`closeSearchModalDatePicker`/`toggleDatePicker`）
  - [x] 3.6 日期点击逻辑：开始→结束→关闭搜索全链路
  - [x] 3.7 更新日期按钮点击事件：改为打开日历弹窗（`toggleDatePicker`）
  - [x] 3.8 更新 `closeAllFilterDropdowns()` 同时关闭日历弹窗
  - [x] 3.9 更新 `searchModalLoadPage()`：收集日期参数传给后端
  - [x] 3.10 更新日期筛选按钮 label 显示逻辑
  - [x] 3.11 在 `frontend/src/style.css` 新增日历选择器样式（~180 行）
  - [x] 3.12 构建验证（`npx vite build`）

- [x] **Task 4**: 综合验证
  - [x] 4.1 空日期范围搜索（向后兼容：不传日期参数时返回全部）
  - [x] 4.2 选择日期的搜索只返回该日期范围的笔记
  - [x] 4.3 预设按钮（今天/7天/30天/不限）正确计算日期范围
  - [x] 4.4 日历月份切换 + 跨月选择
  - [x] 4.5 清除范围后按钮 label 恢复"不限"
  - [x] 4.6 选择范围后重新打开日历，已选范围正确高亮
  - [x] 4.7 点击日历外部区域正确关闭
  - [x] 4.8 键盘 ↑↓ Enter Esc 在日历打开时行为不受影响
  - [x] 4.9 构建验证（`npx vite build` + `go build ./...`）
