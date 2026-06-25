# Checklist - 搜索弹窗开启动画优化

## CSS

- [x] Task 1: 遮罩 opacity + backdrop-filter 过渡已实现
- [x] Task 2: 内容卡片错峰入场（transition-delay: 0.05s）
- [x] Task 2: `.search-modal` 容器 `transition` 已移除
- [x] Task 3: 结果项出场动画（.closing .search-modal-item）
- [x] Task 3: prefers-reduced-motion 降级

## JS

- [x] Task 4: 聚焦使用 transitionend 事件替代 setTimeout
- [x] Task 4: closeSearchModal 中 closing class 管理
- [x] Task 4: 退出动画完成后 cleanup

## 功能验证

- [x] Task 5: 打开动画层次正确（遮罩→内容→结果项）
- [x] Task 5: 关闭动画流畅优雅（结果项→内容→遮罩）
- [x] Task 5: reduced-motion 无动画
- [x] Task 5: 输入框聚焦时机正确

## 构建

- [x] Task 5: `npx vite build` 无错误
