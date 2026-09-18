# Tasks

## Task 1: 后端 services 层（workspace_service.go）

实现 `internal/services/workspace_service.go`，提供四个核心能力 + 类型定义 + 单元测试：

- [x] 1.1 定义 `WorkspaceFileEntry` / `WorkspaceTransferResult` 类型
- [x] 1.2 `ListWorkspaceFiles()`：递归收集工作区文件树（目录在前、名称升序、隐藏空目录，相对路径 `/` 分隔，大小/修改时间）
- [x] 1.3 `UploadFilesToWorkspace(paths)` / `UploadDirectoryToWorkspace(dirPath)`：复制进工作区根，重名自动改名 `xxx (1).ext`，逐项返回结果，单项失败不中断批次
- [x] 1.4 `DownloadWorkspaceFiles(relPaths)`：批量复制到桌面根同名相对路径（目录递归），桌面重名自动改名
- [x] 1.5 `DeleteWorkspaceFiles(relPaths)`：批量删除，目录需显式 recursive=true，拒绝工作区根目录，单项失败不中断
- [x] 1.6 全部路径经 `config.WorkspaceFilePath()` 沙箱校验（EvalSymlinks 防 `../`/symlink 逃逸），复用 go-kit `CopyEx`
- [x] 1.7 单元测试 `workspace_service_test.go`：上传重名自动改名 / 列表树形结构与排序 / 目录递归上传下载删除 / 根目录删除拒绝 / `../` 与 symlink 逃逸拒绝 / 单项失败不中断

**验证**：`go build ./...` + `go vet ./internal/services/` + `go test ./internal/services/ -run Workspace -count=1` 全绿，`gofmt` 无输出

## Task 2: Wails 绑定层（app.go）

- [x] 2.1 新增绑定 `UploadFilesToWorkspace(paths)` / `UploadDirectoryToWorkspace(dirPath)`：内嵌 `OpenMultipleFilesDialog` / `OpenDirectoryDialog` 对话框 + 转发 services
- [x] 2.2 新增绑定 `ListWorkspaceFiles()` / `DownloadWorkspaceFiles(relPaths)` / `DeleteWorkspaceFiles(relPaths, recursive)`
- [x] 2.3 对话框取消返回空结果不报错；logger 按项目规范记 Debugw/Infow/Errorw

**验证**：`go build ./...` + `go vet` 全绿

## Task 3: 前端（入口按钮 + Modal + 交互）

- [x] 3.1 `index.html`：顶栏 `view-header-left`（折叠侧栏与新建会话之间）新增工作区按钮（lucide 文件夹 SVG，`ai-tool-btn` 样式，title「工作区管理器」）
- [x] 3.2 `index.html`：新增工作区管理器 Modal 结构（**顶部操作栏统一放置**：上传文件/上传目录/下载到桌面/删除/刷新 + 右侧「已选 N 项」；中部文件树容器；空状态容器；关闭按钮），仿向量索引弹窗模式
- [x] 3.3 新增 `frontend/src/js/workspace-manager.js`：打开/关闭、文件树渲染（目录展开折叠、checkbox 多选、已选计数）、上传文件/目录（对话框 + loading 态 + 结果通知）、下载、删除（二次确认 danger）、刷新（保留选择）、空状态、代际序号 `workspaceLoadSeq` 竞态防护、无残留监听器
- [x] 3.4 `main.js` 集成：入口按钮事件、ESC 收口进 `handleKeyboardNavigation`、模块导出/暴露遵循既有模式
- [x] 3.5 新增 Modal 样式（沿用主题 CSS 变量；SVG 图标 stroke-width 统一；禁用态降透明度；删除按钮 danger 色与下载分离；开合缩放+淡入 150–300ms + `prefers-reduced-motion` 禁用；文件树行 hover/选中态即时响应）

**验证**：`npm run build` 通过；`wails generate module` 后绑定无缺漏

## Task 4: 构建与整体验证

- [x] 4.1 全量构建：`go build ./...`、`go vet ./...`、`go test ./internal/... -count=1` 全绿
- [x] 4.2 前端 `npm run build` 通过
- [x] 4.3 `wails generate module` 生成 wailsjs 绑定无缺漏；按 checklist.md 逐项验证
- [x] 4.4 更新 AGENTS.md 临时记忆（按维护规范三步法）

# Task Dependencies

- Task 2 依赖 Task 1（绑定转发 services）
- Task 3 依赖 Task 2（前端调用绑定方法）
- Task 4 依赖 Task 1/2/3
