# Checklist - 简化时间筛选：去掉日历弹窗，保留快捷按钮

## HTML

- [x] Task 1: 日历弹窗 HTML 替换为下拉菜单

## CSS

- [x] Task 3: 日历专属 CSS 已移除（~200 行）

## JS

- [x] Task 2: 日历渲染/选择/切换函数已移除
- [x] Task 2: 下拉选项渲染函数已实现
- [x] Task 2: 选中快捷选项后正确设置日期范围并触发搜索
- [x] Task 2: `openSearchModal` 日期重置逻辑正确
- [x] Task 2: `closeAllFilterDropdowns` 中不再引用日历关闭

## 功能验证

- [x] Task 4: 点击时间按钮 → 弹出下拉菜单（通过通用按钮处理器触发）
- [x] Task 4: 选中"今天" → label 更新为"今天"，搜索执行
- [x] Task 4: 选中"最近7天" → label 更新，搜索执行
- [x] Task 4: 选中"最近30天" → label 更新，搜索执行
- [x] Task 4: 选中"不限" → label 恢复"不限"，范围清空
- [x] Task 4: 笔记本/标签筛选器不受影响

## 构建

- [x] Task 4: `npx vite build` 无错误
