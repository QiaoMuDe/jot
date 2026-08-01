# Tasks

- [x] Task 1: 安装 `marked-alert` 依赖包
  - 在 `frontend/` 目录执行 `npm install marked-alert`
  - 验证 package.json 和 package-lock.json 更新

- [x] Task 2: 在预览模式中注册 marked-alert 扩展
  - 在 `main.js` 中导入 `marked-alert` 并调用 `marked.use(alert())`
  - 在 `preview-worker.js` 中做同样的注册

- [x] Task 3: 移除 CM6 Alert 高亮插件
  - 删除 `cm6-syntax-highlight.js` 中的 `markdownAlertPlugin` 及相关代码
  - 删除 `.md` 分支中对该插件的引用
  - 删除 `editor.css` 中的 CM6 相关 `.cm-alert-*` 样式

- [x] Task 4: 添加 Alert 预览样式
  - 在 `editor.css` 中新增 `.markdown-alert` 系列样式
  - 在 `ai-chat.css` 中补充 `.ai-msg-assistant` 内的 alert 样式
  - 在 `md-reference.css` 中补充 `.md-ref-preview` 内的 alert 样式

## Task Dependencies

- Task 2 依赖 Task 1
- Task 4 依赖 Task 2