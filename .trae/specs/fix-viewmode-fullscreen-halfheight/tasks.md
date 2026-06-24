# Tasks

- [x] Task 0: 分析根因 — 通过代码阅读确认查看模式纯文本笔记未设置 `data-mode="edit"`，导致 `toggleEditorFullscreen()` Phase 3 清除内联样式后 `.md-rendered` 可见、占用 50% flex 空间。
- [x] Task 1: 修复 `openEditor()` 中查看模式纯文本分支 — 在 `openEditor()` 的 view mode + `state.noteType === 'text'` 分支，添加 `els.editorOverlay.dataset.mode = 'edit'`，确保 CSS `display: none !important` 在全屏切换后生效。
  - **文件**: `frontend/src/main.js`
  - **改动**: 在 `els.mdRendered.style.display = 'none';` 之前添加了 `els.editorOverlay.dataset.mode = 'edit';`
- [ ] Task 2: 验证修复 — 手动测试以下 4 个场景：
  1. 新建纯文本笔记 → 全屏 → 验证编辑器全高
  2. 编辑已有纯文本笔记 → 全屏 → 验证编辑器全高
  3. 查看纯文本笔记（只读）→ 全屏 → 验证编辑器全高（核心修复场景）
  4. Markdown 笔记查看模式 → 全屏 → 预览区域全高（无回归）

## Task Dependencies

无 — 这是一个单文件、单行修复，Task 1 可直接完成。
