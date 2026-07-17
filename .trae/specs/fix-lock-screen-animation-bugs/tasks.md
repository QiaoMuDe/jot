# Tasks

- [x] Task 1: 修复退出动画背景 opacity 无 transition
  - `modals.css` L848：`.lock-screen-backdrop` 的 `transition` 添加 `opacity 0.4s ease-out`

- [x] Task 2: 修复入场动画尾部闪烁（entering 超时延长）
  - `main.js` L7129：entering 清除超时从 500ms 改为 700ms

- [x] Task 3: 修复 screenLockPasswordRow 过渡无效（CSS + JS 联动）
  - `settings-panel.css`：重写密码行显示/隐藏规则，使用 visibility 方案替代 `[style*="display: none"]`
  - `main.js` L2261-L2264：toggle 点击改用 class 切换
  - `main.js` L7968-L7970：设置加载改用 class 切换

- [x] Task 4: 关闭锁屏密码时添加确认弹窗
  - `main.js` L2265-L2268：调用 confirm 弹窗确认后再清空密码

# Task Dependencies
- Task 3 涉及 JS 和 CSS 联动，需要同时处理两个文件
- Task 1、2、4 相互独立，可并行
