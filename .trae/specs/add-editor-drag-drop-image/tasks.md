# Tasks

- [ ] Task 1: 重构粘贴图片逻辑 — 将 paste 事件中的图片上传+插入代码抽取为共享函数 `uploadImage(file, view)`
  - [ ] SubTask 1.1: 在 `main.js` 中创建 `uploadImage(file, view)` 函数，包含 FileReader 读取、SaveImage 调用、光标处插入 `![](...)` 以及光标定位逻辑
  - [ ] SubTask 1.2: 将现有的 paste 事件处理改为调用 `uploadImage()`，确保功能不变
  - [ ] SubTask 1.3: 验证 `npx vite build` 编译通过

- [ ] Task 2: 实现拖拽图片上传 — 在 CM6 编辑器中添加 `dragover`/`drop` 事件监听
  - [ ] SubTask 2.1: 在 CM6 编辑器根 DOM 上监听 `dragover` 事件：`e.preventDefault()` + 过滤图片文件 + 添加 `dragover` 类（仅 .md 编辑模式）
  - [ ] SubTask 2.2: 监听 `dragleave` 事件：移除 `dragover` 类
  - [ ] SubTask 2.3: 监听 `drop` 事件：`e.preventDefault()` → 提取 `dataTransfer.files` 中的图片文件 → 对每个文件调用 `uploadImage()`

- [ ] Task 3: 拖拽悬停视觉反馈
  - [ ] SubTask 3.1: 在 [style.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/style.css) 中新增 `.cm-editor.dragover { outline: 2px dashed var(--accent-color); }` 样式
  - [ ] SubTask 3.2: 可选为 drag-over 状态添加过渡动画

- [ ] Task 4: 编译与验证
  - [ ] SubTask 4.1: `npx vite build` 前端构建通过
  - [ ] SubTask 4.2: 手动验证拖拽一张图片到编辑器：检查上传结果、Markdown 插入、光标位置、视觉反馈

# Task Dependencies

- Task 1 是 Task 2 的前置依赖（先抽取共享函数，再新增拖拽调用）
- Task 3 可独立完成（纯 CSS）
- Task 4 是最终验证
