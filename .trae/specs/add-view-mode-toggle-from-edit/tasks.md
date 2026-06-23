# Tasks

- [x] Task 1: HTML 添加"返回查看模式"按钮
  - 在 `frontend/index.html` 的 `.editor-header-actions` 中添加一个新按钮 `#editorViewBtn`
  - 使用"眼睛" SVG 图标，`title="返回查看模式"`
  - 默认 `style="display:none;"` 隐藏
  - 按钮排在 `#editorEditBtn` 之后、`#editorFullscreenBtn` 之前

- [x] Task 2: JS 状态管理 — `state.enteredFromViewMode` 标志
  - 在 `state` 对象中新增 `enteredFromViewMode: false`
  - 在 `els` 中新增 `editorViewBtn: $('editorViewBtn')`
  - 在 `editorEditBtn` 的 click 回调中设置 `state.enteredFromViewMode = true`（点击编辑按钮前）
  - 在 `closeEditor()` 的清理逻辑中重置 `state.enteredFromViewMode = false`

- [x] Task 3: JS 显示/隐藏逻辑 — `openEditor()` 中控制按钮可见性
  - 在 `openEditor()` 的只读模式 UI 配置区（约 L2507-L2518）新增：
    - 如果 `!isReadOnly && state.enteredFromViewMode`：显示 `#editorViewBtn`
    - 否则：隐藏 `#editorViewBtn`

- [x] Task 4: JS 点击事件 — `editorViewBtn` 切回查看模式
  - 新增 `editorViewBtn` 的 `click` 事件监听
  - 获取 `state.editingNoteId`，如果存在则调用 `openEditor(noteId, true)`
  - 设置 `state.enteredFromViewMode = false`（避免重复触发）

- [x] Task 5: CSS 按钮样式
  - `#editorViewBtn` 使用 `.editor-header-btn` 类（与现有其他 header 按钮一致：32x32 圆形透明按钮）
  - 无需额外特殊样式，现有 `.editor-header-btn` 样式覆盖

# Task Dependencies

- [Task 1] → [Task 2-5]（有 HTML 结构再写 JS/CSS）
- [Task 2] → [Task 3-4]（有状态变量才能用）
- [Task 3,4] → 各自独立（可并行编码）
- [Task 5] 独立
