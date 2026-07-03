# Tasks

- [x] Task 1: 后端数据模型 — 为 `AISession` 模型新增 `IsPinned` 字段，更新 `AISessionSummary` 结构体
  - 在 `internal/models/ai_session.go` 的 `AISession` 结构体添加 `IsPinned bool` 字段
  - 在 `internal/services/ai_service.go` 的 `AISessionSummary` 结构体添加 `IsPinned bool` 字段
  - 在 `GetAISessions()` 中填充 `IsPinned` 字段值

- [x] Task 2: 后端排序逻辑 — 修改 `GetAISessions()` 排序为先置顶（按标题 ASC）再按 `updated_at DESC`
  - 使用 GORM 的 `Order` 实现自定义排序（`CASE WHEN is_pinned = 1 THEN 0 ELSE 1 END, title ASC, updated_at DESC`）
  - 确保 SQLite 兼容

- [x] Task 3: 后端新增 API — 新增 `TogglePinAISession(id uint)` 方法并绑定到 Wails
  - 在 `AIService` 中添加 `TogglePinAISession(id uint) error` 方法
  - 在 `app.go` 中添加 `TogglePinAISession(id uint) error` 绑定方法
  - 调用 `loadSessionList()` 刷新列表

- [x] Task 4: 前端交互 — 将删除按钮替换为"更多"按钮 + 下拉菜单
  - 在 `renderSessionList()` 中，将 `delBtn`（✕）替换为 `moreBtn`（⋯）
  - 点击 `moreBtn` 弹出下拉菜单（`<div class="ai-session-more-menu">`），包含"置顶/取消置顶"和"删除会话"两个菜单项
  - 实现菜单显示/隐藏逻辑（点击按钮切换、点击外部关闭）
  - 菜单定位：确保不超出侧栏边界
  - 删除操作逻辑不变
  - 新增置顶操作：调用 `TogglePinAISession()` 后端 API，刷新列表

- [x] Task 5: 前端置顶状态渲染 — 置顶会话显示置顶图标和视觉区分
  - 在 `renderSessionList()` 中，对 `s.is_pinned` 为 `true` 的会话：
    - 在标题前添加置顶小图标（pin SVG）
    - 添加 `pinned` CSS 类名（浅色背景标识）
  - 在置顶会话与普通会话之间添加分隔线（如果两种类型都存在）

- [x] Task 6: CSS 样式 — 下拉菜单和置顶状态样式
  - 新增 `.ai-session-more-btn` 样式（隐藏，hover 时显示）
  - 新增 `.ai-session-more-menu` 样式（下拉菜单容器、定位、动画）
  - 新增 `.ai-session-more-menu-item` 样式（菜单项，含 hover 状态）
  - 新增 `.ai-session-more-menu-item.danger` 样式（删除按钮危险色）
  - 新增 `.ai-session-more-menu-divider` 样式（分隔线）
  - 新增 `.ai-session-item.pinned` 样式（置顶会话背景色）
  - 新增 `.ai-session-item-pin-icon` 样式（置顶图标）
  - 新增 `.ai-session-pin-divider` 样式（置顶/普通会话分隔线）
  - 确保下拉菜单位置不超出侧栏边界

# Task Dependencies
- Task 4 依赖 Task 3（前端需要后端 `TogglePinAISession` API 才能工作）
- Task 5 依赖 Task 4（置顶状态渲染需要新的 render 逻辑）
- Task 6 可与 Task 4 并行进行
