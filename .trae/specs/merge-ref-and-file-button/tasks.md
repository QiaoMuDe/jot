# Tasks

- [ ] Task 1: 创建"+"菜单按钮 HTML 结构
  - 在 `index.html` 的 `.ai-chat-toolbar` 中，用 `#aiChatAddBtn` + `.ai-chat-add-wrap` 包裹的菜单结构替换原有的 `#aiChatRefBtn` 和 `#aiChatFileBtn`
  - 触发按钮：圆形"+" SVG + "添加"文字 + 下拉箭头 SVG
  - 菜单：`#aiChatAddDropdown`，包含两个 `.ai-chat-add-item`（引用笔记 + 上传文件），每个含 SVG 图标 + 文字
  - 菜单弹出方向：向上（`bottom: calc(100% + 4px)`）

- [ ] Task 2: 新增 CSS 样式
  - 在 `ai-chat.css` 中新增：
    - `.ai-chat-add-wrap`（`position: relative`）
    - `.ai-chat-add-dropdown`（向上弹出、动画复用现有 dropdown 模式：opacity + translateY + scale transition）
    - `.ai-chat-add-item`（flex 布局、hover 高亮、与 ref 按钮相同的 `.has-ref` 状态切换）
    - 菜单项逐个滑入动画（`nth-child` delayed transitions）

- [ ] Task 3: 重构 JS 逻辑
  - 新增 `addBtn`/`addDropdown` 变量声明和 DOM 获取
  - 事件绑定：
    - `addBtn` click → `toggle` open class（e.stopPropagation）
    - `addDropdown` click → e.stopPropagation
    - `document` click → 关闭菜单
    - 菜单内"引用"项 click → `openNoteRefModal()` + 关闭菜单
    - 菜单内"上传"项 click → 原有的 `SelectAIChatFiles` 异步逻辑 + 关闭菜单
  - 高亮状态迁移：
    - `updateRefChips()` 中原 `refBtn.classList` 操作改为 `addBtn.classList`
    - `renderFileChips()` 中原 `fileBtn.classList` 操作改为 `addBtn.classList`
    - 删除 `refBtn` 和 `fileBtn` 的 `.has-ref` 操作引用

- [ ] Task 4: 清理遗留引用
  - 删除 `ai-chat.js` 中 `refBtn` 和 `fileBtn` 的事件绑定代码块（引用和上传逻辑已迁移到菜单项）
  - 保留 `refBtn`/`fileBtn` 变量声明（可能其他位置仍有引用，需全局检查，确认无引用后再删除变量声明）

# Task Dependencies
- Task 1 → Task 2（HTML 结构存在后样式才能生效）
- Task 1 → Task 3（HTML 结构存在后 JS 才能操作 DOM）
- Task 3 → Task 4（JS 逻辑迁移后才能安全删除旧的绑定代码）
