# 工作区管理器（Workspace Manager）Spec

## Why

工作区（`~/.jot/workspace`）是 AI Agent 的沙箱文件空间，但目前前端没有任何入口：用户只能通过对话让 AI 调用 `transfer_file` 等工具间接操作，无法快捷上传、浏览、下载、删除文件。本功能提供一个**工作区管理器 Modal**，让用户以「文件管理器」范式（先选内容、再执行操作）直接管理工作区，补齐 AI 工具链与用户主动操作之间的缺口。

## What Changes

* **顶栏入口**：AI 对话页顶栏（`view-header-left`，折叠侧栏与新建会话之间）新增工作区按钮（文件夹图标，lucide 风格 SVG），点击打开工作区管理器 Modal。加号菜单保持不变（保持「添加对话内容」的纯粹职责）。

*- **工作区管理器 Modal**（仿向量索引弹窗模式）：
  - **顶部操作栏（统一）**：`上传文件`（多选文件对话框）、`上传目录`（目录对话框）、`下载到桌面`、`删除`、`刷新` ＋ 右侧「已选 N 项」；下载/删除**未选中时禁用**
  - 中部：工作区**文件树**（目录可展开、checkbox 多选、相对路径/大小/修改时间）
  - 空状态引导上传；上传/下载/删除后刷新列表并保留选择

* **后端 services 层**：新增 `internal/services/workspace_service.go`，提供上传（文件/目录）、列表（树形）、下载、删除四个能力；全部路径经 `config.WorkspaceFilePath()` 沙箱校验（EvalSymlinks 防逃逸）；复制复用 go-kit `CopyEx`（原子性）。

* **Wails 绑定层**：`app.go` 新增 5 个绑定方法（含文件/目录对话框）。

* **重名策略**：目标重名时**自动改名** `xxx (1).ext`（上传与下载均适用），不打断批量流程。

* **审批**：用户手动操作**不触发 AI 审批门控**（操作者是用户本人，与约束 AI 的审批体系是两套体系；`transfer_file` 工具保持不变）。

* 新建 `frontend/src/js/workspace-manager.js` 模块 + 对应 CSS（沿用项目主题变量与既有 modal 模式）。

## Impact

* Affected specs：AI 聊天页（顶栏按钮区）、后端 Wails 绑定层、services 业务层

* Affected code：

  * 新增 `internal/services/workspace_service.go`（+ 单元测试 `workspace_service_test.go`）

  * 修改 `app.go`（5 个绑定方法 + 返回类型定义）

  * 修改 `frontend/index.html`（顶栏按钮 + Modal 结构）

  * 修改 `frontend/src/main.js`（入口绑定、ESC 收口）

  * 新增 `frontend/src/js/workspace-manager.js`

  * 新增/修改 `frontend/src/css/components/*`（Modal 样式）

  * 构建链：`npm run build` + `wails generate module` + `wails build`

## ADDED Requirements

### Requirement: 顶栏入口

系统 SHALL 在 AI 对话页顶栏（折叠侧栏与新建会话按钮之间）提供一个文件夹图标按钮，点击打开工作区管理器 Modal；按钮采用 lucide 风格 SVG（stroke-width 与现有 `ai-tool-btn` 一致），带 title 提示「工作区管理器」。

#### Scenario: 打开管理器

* **WHEN** 用户点击顶栏工作区按钮

* **THEN** 工作区管理器 Modal 以缩放+淡入动画打开（150–300ms、尊重 prefers-reduced-motion），并加载工作区文件树

### Requirement: 文件树浏览与多选

系统 SHALL 在 Modal 中部展示工作区文件树：目录行可展开/折叠、文件行显示相对路径/大小/修改时间；每行提供 checkbox 供多选；顶部操作栏右侧显示「已选 N 项」。

#### Scenario: 多选内容
- **WHEN** 用户勾选多个文件/目录
- **THEN** 顶部「已选 N 项」实时更新；顶部操作栏「下载到桌面」「删除」按钮变为可用

#### Scenario: 未选择时

* **WHEN** 无任何选中项

* **THEN** 下载/删除按钮保持禁用态（降透明度 + 不可点击），hover 显示「请先选择内容」提示

#### Scenario: 空工作区

* **WHEN** 工作区为空

* **THEN** 展示空状态文案（引导上传）与「上传文件/上传目录」操作按钮，不渲染空列表

### Requirement: 上传文件与目录

系统 SHALL 提供上传能力：`上传文件` 打开多选文件对话框，`上传目录` 打开目录对话框；选中后复制进工作区根目录（目录整体递归复制），重名自动改名；上传中按钮进入 loading 态并禁用重复触发；完成后刷新列表并通知结果（成功/失败条数）。

#### Scenario: 上传文件

* **WHEN** 用户点击「上传文件」并选择若干文件

* **THEN** 文件被复制到工作区根目录；若目标已存在则自动改名为 `xxx (1).ext`；列表刷新，通知「已上传 N 个文件」

* **AND** 上传期间上传按钮禁用，完成后恢复

#### Scenario: 上传目录

* **WHEN** 用户点击「上传目录」并选择目录

* **THEN** 目录整体递归复制到工作区根；重名自动改名；完成后刷新列表

### Requirement: 下载到桌面

系统 SHALL 将选中的工作区内容复制到**桌面根同名相对路径**（对齐 `transfer_file` 语义），支持批量；桌面目标重名时自动改名；完成后刷新列表（下载不改变工作区内容，刷新保留选择）并通知结果。

#### Scenario: 批量下载

* **WHEN** 用户多选若干文件/目录并点击「下载到桌面」

* **THEN** 各内容按相对路径复制到 `~/Desktop`（目录递归）；重名自动改名；通知「已下载 N 项」

### Requirement: 删除

系统 SHALL 支持删除选中内容：点击「删除」弹出二次确认（danger 样式、说明删除不可恢复）；含目录时确认文案提示将递归删除；**禁止删除工作区根目录**（根目录永远不可勾选/删除）；目录删除遵循「显式 recursive」约束（服务端校验）；完成后刷新列表并通知。

#### Scenario: 删除文件

* **WHEN** 用户选中文件并确认删除

* **THEN** 文件被删除，列表刷新，通知删除结果

#### Scenario: 删除目录

* **WHEN** 用户选中目录并确认删除（确认文案含「递归删除」）

* **THEN** 目录及其内容被递归删除，列表刷新

#### Scenario: 取消删除

* **WHEN** 用户在确认框中取消

* **THEN** 不执行删除，列表与选择保持不变

### Requirement: 沙箱安全边界

系统 SHALL 保证所有操作仅发生工作区内与桌面目标端：上传目标、下载源、删除目标均经 `config.WorkspaceFilePath()` 沙箱解析（Clean + EvalSymlinks），`../` 逃逸与 symlink/junction 逃逸一律拒绝并返回中文错误；下载目标限制在 `~/Desktop` 根下同名相对路径。

#### Scenario: 路径逃逸

* **WHEN** 任一输入路径尝试越出工作区或桌面根

* **THEN** 操作被拒绝，返回明确的中文错误提示，不产生任何文件副作用

### Requirement: 交互与可访问性（UI/UX 准则）

* Modal 提供显式关闭按钮 + 遮罩点击关闭 + ESC 关闭（ESC 收口在 `main.js` `handleKeyboardNavigation`，遵循项目约束）

* 所有图标使用 SVG（lucide 风格、stroke-width 统一），不用 emoji

* 删除按钮使用 danger 语义色并与下载按钮视觉分离（destructive-emphasis）

* 操作按钮 loading 态 / disabled 态明确（降透明度 + cursor + 语义属性）

* 动画遵循 `prefers-reduced-motion`；展开/折叠与 Modal 开合 150–300ms，仅用 transform/opacity

* 遵循项目竞态防护范式：列表加载用代际序号 `workspaceLoadSeq` 丢弃过期响应；定时器/异步统一清理防泄漏

* 通知复用 `NotificationManager`

#### Scenario: 关闭 Modal

* **WHEN** 用户点击关闭按钮 / 遮罩 / 按 ESC

* **THEN** Modal 关闭且恢复打开前焦点状态，无残留监听器

### Requirement: 后端返回类型

系统 SHALL 定义并暴露以下类型（经 Wails 绑定序列化到前端）：

```go
// WorkspaceFileEntry 工作区文件树节点（相对路径为展示键）
type WorkspaceFileEntry struct {
    Name     string               // 显示名
    RelPath  string               // 相对工作区根路径（/ 分隔）
    IsDir    bool
    Size     int64                // 目录为 0
    ModTime  int64                // unix 秒
    Children []WorkspaceFileEntry // 目录才有
}

// WorkspaceTransferResult 单次上传/下载结果
type WorkspaceTransferResult struct {
    Name   string // 原文件名/目录名
    Target string // 实际写入的相对路径（含自动改名后的新名）
    Error  string // 失败原因，空表示成功
}
```

#### Scenario: 列表返回

* **WHEN** 前端调用 `ListWorkspaceFiles()`

* **THEN** 返回按目录分组的树形结构，目录在前、名称升序，隐藏空目录

