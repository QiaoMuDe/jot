# Tasks

- [x] Task 1: 在 `EDITOR_ACTIONS` 中新增「编码解码」分组，含 3 个子分组 6 个操作项
  - Base64 编码: `btoa(text)`，catch 时抛 `new Error('invalid base64')`
  - Base64 解码: `atob(text)`，catch 时抛 `new Error('invalid base64')`
  - URL 编码: `encodeURIComponent(text)`
  - URL 解码: `decodeURIComponent(text)`，catch 时抛 `new Error('invalid url encoding')`
  - HTML 编码: 替换 `&` `<` `>` `"` `'` 为 HTML 实体
  - HTML 解码: 创建 `DOMParser` 解析 HTML，读取 `document.body.textContent`

- [x] Task 2: 运行 `npm run build` 验证构建无错误

# Task Dependencies
- Task 1 是独立任务
- Task 2 依赖 Task 1