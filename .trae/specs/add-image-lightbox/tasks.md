# Tasks

- [x] Task 1: 灯箱 CSS — 在 editor.css 末尾新增 `.image-lightbox` 样式
  - [x] SubTask 1.1: 在 `d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\editor.css` 末尾追加 `.image-lightbox` 样式（fixed/flex 居中/深色背景/大图约束）
  - [x] SubTask 1.2: `npx vite build` 验证

- [x] Task 2: 图片尺寸调整 + 光标样式 — 修改 editor.css 中 `.md-rendered img`
  - [x] SubTask 2.1: 将 `.md-rendered img` 的 `max-width: 100%` 改为 `max-width: 85%`，增加 `display: block; margin: 0.5em auto; cursor: zoom-in`
  - [x] SubTask 2.2: `npx vite build` 验证

- [x] Task 3: 灯箱 JS — 在 `_applyPreviewDOMHelpers` 中新增图片点击处理
  - [x] SubTask 3.1: 在 `d:\资源池\下水道\Dev\本地项目\jot\frontend\src\main.js` 的 `_applyPreviewDOMHelpers` 函数末尾（代码块处理之后）加 `els.mdRendered.querySelectorAll('img').forEach(...)` 点击事件：创建 `.image-lightbox` 遮罩、设置内容为 `<img src="...">`、点击遮罩移除
  - [x] SubTask 3.2: 加 `img.style.cursor = 'zoom-in'`（与 CSS 冗余但确保所有环境生效）
  - [x] SubTask 3.3: `npx vite build` 验证

- [x] Task 4: 编译与验证
  - [x] SubTask 4.1: `go build ./...` + `npx vite build` 均通过
  - [ ] SubTask 4.2: 运行时手动验证灯箱功能

# Task Dependencies

- Task 1 和 Task 2 均可独立编写（均修改 editor.css，但修改位置不冲突，可并行）
- Task 3 在 Task 1 完成后编写（灯箱 JS 依赖 CSS 类名 `.image-lightbox`）
- Task 4 在所有 task 之后
