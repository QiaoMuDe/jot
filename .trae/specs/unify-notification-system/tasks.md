# Tasks
- [ ] Task 1: 创建统一的右上角通知样式和容器结构
  - [ ] 在 `app.css` 或 `style.css` 中定义 `.notification-container` 样式（fixed 右上角，z-index:2000）
  - [ ] 定义通知条目 `.notification` 样式（圆角卡片、阴影、四种类型色板、入场/出场动画）
  - [ ] 在 `index.html` 中添加容器元素 `<div id="notificationContainer" class="notification-container"></div>`
  - [ ] 删除旧有的 `#undoToast` / `#undoToastMsg` / `#undoToastBtn` DOM 元素

- [ ] Task 2: 实现 `NotificationManager` 类
  - [ ] 在 `main.js` 顶部定义 `NotificationManager` 类（单例模式）
  - [ ] 实现 `constructor`：获取容器引用
  - [ ] 实现 `show(message, type, duration?)`：创建通知 DOM、添加动画、自动销毁
  - [ ] 实现 `showUndo(message, onUndo, duration?)`：创建带撤销按钮的通知
  - [ ] 实现内部 `_createEl(type, ...children)` 辅助方法

- [ ] Task 3: 替换所有旧通知调用点
  - [ ] 将 `showToast('xxx')` 全部替换为 `nm.show('xxx', type)`
  - [ ] 将 `showUndoToast(msg, noteIds)` 替换为 `nm.showUndo(msg, onUndo)`
  - [ ] 删除 `animateToast` / `showToast` / `hideToast` / `showUndoToast` / `hideUndoToast` 函数
  - [ ] 删除 `toastTimer` / `undoToastTimer` / `toastStack` / `undoNoteId` 状态变量

- [ ] Task 4: 更新 `showConfirmDialog` 视觉色板
  - [ ] 同步使用通知系统的 CSS 变量色板

- [ ] Task 5: 验证构建通过
  - [ ] Vite build 无错误
  - [ ] 应用运行无控制台报错
