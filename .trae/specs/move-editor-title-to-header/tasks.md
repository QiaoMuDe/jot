# Tasks

- [x] Task 1: HTML 结构迁移：将 `#editorNoteTitle` 移入 editor-header 左侧
  - [x] SubTask 1.1: 在 `frontend/index.html` 的 `.editor-header` 内、`.editor-header-actions` 之前新增标题容器 `<div class="editor-title-wrap">`，将原 editor-body 顶部的 `<input id="editorNoteTitle">` 移入其中（id、class 保留 `editor-input`）
  - [x] SubTask 1.2: 删除 editor-body 中原标题输入位置，确认标签选择器 `.editor-section` 成为 body 首个元素
- [x] Task 2: CSS 样式：顶栏布局 + 标题伪静态文字 + 编辑态
  - [x] SubTask 2.1: `.editor-header` 改为 `justify-content: space-between`，适当增加高度（如 `padding: 8px 12px 4px`），标题区 flex 占据剩余宽度、右操作区不被挤压
  - [x] SubTask 2.2: 新增 `.editor-title-wrap` 与标题默认态样式：透明背景、无边框、`font-size` 收敛为顶栏尺度（约 1rem）、单行 ellipsis、`max-width` 约 50% 防止挤压操作区
  - [x] SubTask 2.3: 编辑/新建模式 hover 提示：淡下划线（`border-bottom: 1px dashed var(--text-muted)` 或类似）、`cursor: text`、tooltip「双击编辑标题」；编辑态（focus）显示 accent 下划线
  - [x] SubTask 2.4: 查看模式（`.editor-input-readonly`）在顶栏场景下：无 hover 提示、`cursor: default`；沿用现有 class 机制
  - [x] SubTask 2.5: 补偿 editor-body 间距：标签选择器 `.editor-section` 顶部间距调整，使标题移除后视觉协调
- [x] Task 3: JS 交互：双击编辑 / Enter / Esc / blur 收尾
  - [x] SubTask 3.1: 在 `frontend/src/main.js` 编辑器相关区域新增交互逻辑：dblclick 时（仅当非 readOnly）进入编辑态并 focus；记录进入编辑态前的标题值用于取消
  - [x] SubTask 3.2: keydown 处理：Enter 提交并 blur（阻止换行/表单默认行为）；Esc 恢复原标题并退出编辑态
  - [x] SubTask 3.3: blur 处理：提交新标题；若 trim 后为空则恢复原标题；退出编辑态样式
  - [x] SubTask 3.4: `switchEditorReadOnly` 中同步：查看模式时若处于编辑态则强制退出编辑态；编辑态 class 随 readOnly 切换
  - [x] SubTask 3.5: 确认 `openEditor`（新建默认标题/编辑填充/缓存未命中清空）、`closeEditor` 中事件监听清理路径完整，无重复绑定（沿用现有 removeEventListener 模式）
- [x] Task 4: 构建与人工验证
  - [x] SubTask 4.1: 执行 `npm run build` 重新构建前端资源
  - [x] SubTask 4.2: 执行 `wails build` 重新编译应用
  - [x] SubTask 4.3: 按 checklist.md 逐项人工验证三种模式下的标题行为

# Task Dependencies
- Task 2 依赖 Task 1（DOM 位置先迁移）
- Task 3 依赖 Task 1（事件绑定目标位置）
- Task 3 与 Task 2 可并行（不同文件：css vs js）
- Task 4 依赖 Task 1-3 全部完成
