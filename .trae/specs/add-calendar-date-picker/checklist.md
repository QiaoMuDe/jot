# Checklist - 搜索弹窗日历日期选择器

## 后端

- [x] Task 1.1-1.3: `Search()` 函数签名增加 `startDate, endDate string`，WHERE 条件按 spec 追加 `updated_at BETWEEN`
- [x] Task 1.4: `SearchByNotebook()` 同步改动
- [x] Task 1.5: `app.go` 的 `SearchNotes` 绑定方法接收并透传日期参数
- [x] Task 1.6: `go build ./...` 无错误

## Wails 绑定

- [x] Task 2.1-2.2: Wails JS 绑定文件自动更新，`SearchNotes` JS 函数接受 6 个参数

## 前端状态管理

- [x] Task 3.1: `state.searchModalDateRange` 替换为 `searchModalDateStart` + `searchModalDateEnd`
- [x] Task 3.2: `openSearchModal` 中日期重置逻辑更新
- [x] Task 3.3: `updateSearchModalFilterBtnActive()` 日期判空逻辑

## 前端 UI — 日历选择器

- [x] Task 3.4: HTML 中删除旧日期下拉静态选项
- [x] Task 3.5: `renderDatePickerCalendar()` 函数实现（日历网格/月切换/选中范围高亮/预设按钮）
- [x] Task 3.6: `handleDatePickerDateClick()` 函数实现（开始/结束选择逻辑）
- [x] Task 3.7: 日期按钮点击事件改为打开日历弹窗
- [x] Task 3.8: `closeAllFilterDropdowns()` 关闭日历弹窗
- [x] Task 3.9: `searchModalLoadPage()` 传递日期参数到后端
- [x] Task 3.10: 日期按钮 label 更新逻辑
- [x] Task 3.11: CSS 样式全部到位（容器280px/grid/选中/范围/今天/预设按钮/弹出动画/~180 行）
- [x] Task 3.12: `npx vite build` 无错误

## 综合验证

- [x] Task 4.1: 不选日期范围仍返回全部结果（向后兼容）
- [x] Task 4.2: 选择日期范围后只返回该范围笔记
- [x] Task 4.3: 预设按钮正确计算日期边界
- [x] Task 4.4: 月份切换 + 跨月选择正常
- [x] Task 4.5: 清除范围后 label 恢复"不限"
- [x] Task 4.6: 重新打开日历已选范围高亮正确
- [x] Task 4.7: 点击日历外部正确关闭
- [x] Task 4.8: 日历打开时键盘导航不影响
- [x] Task 4.9: 全量构建（`npx vite build` + `go build ./...`）通过
