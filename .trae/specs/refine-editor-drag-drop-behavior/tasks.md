# Tasks

- [x] Task 1: CM6 编辑器注册 `drop` 事件 — 文件拖入时 `preventDefault()` 阻止原生行为
  - [x] SubTask 1.1: 在 `createCmEditor` 函数中找到粘贴事件注册处（`cmEditor.dom.addEventListener('paste', handlePaste)`），在附近加 `cmEditor.dom.addEventListener('drop', ...)` 判断 `e.dataTransfer?.files?.length > 0` 则 `e.preventDefault()`
  - [x] SubTask 1.2: `npx vite build` 验证

- [x] Task 2: Go 后端新增 `ReadTextFile` 方法
  - [x] SubTask 2.1: 在 `app.go` 的图片方法区附近添加 `ReadTextFile(localPath string) (string, error)`：先 `fs.IsBinaryPath` 检查 → 非二进制则 `os.ReadFile` 返回内容，二进制返回空字符串+错误
  - [x] SubTask 2.2: `go build ./...` 验证

- [x] Task 3: 重写 `OnFileDrop` 路由逻辑
  - [x] SubTask 3.1: 修改 `initFileDrop` 中的 `OnFileDrop` 回调，按三级判断路由：
    - `cmEditor === null` → `handleFileDropPaths`（原逻辑）
    - `cmEditor !== null` + `readOnly` → 全部忽略
    - `cmEditor !== null` + 非 `readOnly` → 图片上传、文本插入、二进制忽略
  - [x] SubTask 3.2: `npx vite build` 验证

- [x] Task 4: 编译与验证
  - [x] SubTask 4.1: `go build ./...` + `npx vite build` 均通过
  - [ ] SubTask 4.2: 运行时手动验证场景矩阵

# Task Dependencies

- Task 2 无依赖（纯 Go 后端）
- Task 1 和 Task 3 可并行开发（均修改 main.js 但修改位置不冲突）
- Task 4 在所有 task 之后
