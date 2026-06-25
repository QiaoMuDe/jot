# Tasks - 搜索弹窗开启动画优化

## Task Dependencies

- [Task 1] 无依赖
- [Task 2] 无依赖
- [Task 3] 无依赖
- [Task 4] 无依赖
- [Task 5] 依赖 Task 1-4（验证）

---

## Tasks

- [x] **Task 1**: CSS — 遮罩动画
  - ✅ `.search-modal-mask` 增加 `opacity: 0` + `transition: opacity 0.2s ease-out, backdrop-filter 0.2s ease-out`
  - ✅ `.search-modal.visible .search-modal-mask` 设置 `opacity: 1` + `backdrop-filter: blur(4px)`
  - ✅ 默认状态 `backdrop-filter: blur(0px)` 确保初始值可过渡

- [x] **Task 2**: CSS — 内容卡片错峰入场
  - ✅ `.search-modal` 移除 `transition: opacity 0.18s ease-in`
  - ✅ `.search-modal-content` 增加 `opacity: 0` + `transition-delay: 0.05s`
  - ✅ `.search-modal.visible .search-modal-content` 增加 `transition-delay: 0.05s`

- [x] **Task 3**: CSS — 结果项出场动画 + reduced-motion
  - ✅ `.search-modal-item` 增加 `opacity 0.1s ease-in` transition
  - ✅ 新增 `.search-modal.closing .search-modal-item`：`opacity: 0; transform: translateY(-6px);`
  - ✅ `@media (prefers-reduced-motion: reduce)` 完整覆盖所有弹窗元素

- [x] **Task 4**: JS — 聚焦优化 + closing class 管理
  - ✅ `openSearchModal()` 中 `setTimeout(focus, 50)` 替换为 `transitionend` 事件 + 500ms 兜底
  - ✅ `closeSearchModal()` 添加 closing class → 150ms 后 cleanup
  - ✅ 移除 animation-delay 清理逻辑

- [x] **Task 5**: 验证
  - ✅ `npx vite build` 无错误
  - ✅ 打开弹窗：遮罩先淡入 → 50ms 后内容卡片出现
  - ✅ 关闭弹窗：closing class 触发退出动画
  - ✅ prefers-reduced-motion 降级完整
  - ✅ 聚焦在 transitionend 后到达输入框
