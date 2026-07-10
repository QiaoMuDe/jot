# Tasks

- [ ] Task 1: 修改 `createMsgActions` 函数 — 将右侧一排按钮替换为单个三点按钮
  - 移除逐个创建按钮的代码（copy, edit, resend, save, regen, followUp, delete）
  - 保留横排三点按钮（MORE_ICON，cx 水平排列），点击时显示垂直菜单
- [ ] Task 2: 创建三点按钮弹出菜单逻辑
  - 新函数 `showMsgActionsMenu(content, role, msgEl, btn)` — 创建与右键菜单相同的菜单项
  - 复用 `handleCopy`, `handleDeleteMsg`, `handleResend` 等已存在的事件处理
  - 菜单使用 `.context-menu` 类的样式（与右键菜单样式一致）
  - 垂直弹出（非水平），固定在三点按钮附近
  - 点击菜单项后执行对应操作并关闭菜单
  - 点击菜单外区域关闭菜单
- [ ] Task 3: 清理旧代码 — 删除不再需要的函数和逻辑
  - 删除 `collapseActionsIfNeeded` 函数及其所有调用
  - 删除 `toggleActionPopup` 函数
- [ ] Task 4: 更新 CSS — 适配常驻显示和新的三点按钮布局
  - 移除 `.action-buttons` 的 `opacity: 0` + `transition` 悬停渐显，改为 `opacity: 1`
  - 移除 `.ai-msg:hover .ai-msg-actions .action-buttons` 规则
  - 移除 `.ai-msg-user .action-buttons` 的 `opacity: 0` + `transition` + hover 渐显规则
  - 移除 `.action-popup` 全部样式
  - 移除 `.action-buttons.collapsed .more-btn` 和 `.action-buttons.collapsed > button:not(.more-btn)` 规则
  - 更新 `.more-btn` 样式，始终显示（`display: flex`）
  - 移除 `.ai-msg-user.narrow-mode` / `.wide-mode` 按钮定位相关规则
  - 调整 `.user-tokens` 样式适配常驻显示
  - 三点按钮菜单使用 `.context-menu` 已有样式（利用 `#aiMsgContextMenu` 元素）

# Task Dependencies
- 无依赖，可顺序执行 Task 1 → Task 2 → Task 3 → Task 4
