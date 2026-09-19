# Plan：工作区管理器拖拽上传文件/目录

## Context

「工作区管理器」面板（AI Agent 沙箱 `~/.jot/workspace`）目前只能通过「上传文件 / 上传目录」按钮弹系统对话框上传。本次为面板增加**拖拽上传文件/目录**能力：

1. **功能**：拖拽文件或目录到面板内即可上传。
2. **交互（方案 A）**：拖拽悬停在文件树的**目录行上即代表目标为该目录**（悬停亮起即目标）；悬停其他区域（空白 / 文件行）目标为工作区根目录。
3. **屏蔽**：面板打开时，其他拖拽功能（AI 聊天拖拽上传、CM6 编辑器图片/文本插入、笔记导入）全部屏蔽，只服务面板上传。

关键现状：项目使用 Wails OS 级文件拖拽（`EnableFileDrop: true`），所有拖入窗口的文件被统一拦截，把**绝对路径数组 + 释放坐标(x,y)** 交到 main.js 的 `window.runtime.OnFileDrop` 回调，回调内 `elementFromPoint(x,y)` 按落点路由。HTML5 的 document dragenter/dragleave 只做「释放以导入文件」遮罩视觉，不处理文件。面板 `#workspaceModal` 是 body 级 fixed 节点（z-index 1100），打开时铺满视口——因此**面板打开时 elementFromPoint 任何坐标都命中面板内**，天然只需在 OnFileDrop 最前面加一条「面板打开 → 路由到工作区上传」的判断即可完成全部屏蔽。

## 一、后端（internal/services/workspace\_service.go + app.go）

### 1. `uploadOne` 重构支持目标目录

`uploadOne(root, srcPath)` → `uploadOne(root, targetDir, srcPath)`：

* 目标从 root 改为 targetDir：`finalDst, err := config.WorkspaceFilePath(targetDir, uniqueTarget(targetDir, name))`

* 「防止上传工作区自身」的比对仍以 `root` 为准（不变）

* 现有调用点 `UploadFiles` / `UploadDirectory` 改为 `s.uploadOne(root, root, p)`，行为完全不变（现有测试不受影响）

### 2. 新增 `UploadPathsToWorkspace(paths []string, targetRel string) ([]WorkspaceTransferResult, error)`

* `targetRel` 为空串或 `/` → 目标 = root（根目录）

* 非空：`config.WorkspaceFilePath(root, filepath.FromSlash(targetRel))` 做沙箱校验（防 `../` 逃逸 / symlink 逃逸），再 `os.Stat` 确认存在且为目录，不合法直接整体返回错误

* 循环 `s.uploadOne(root, targetDir, p)`，复用全部既有逻辑（重名自动改名 / 单项失败不中断批次 / CopyEx 原子复制 / 沙箱校验）

### 3. app.go 新增绑定

`UploadPathsToWorkspace(paths []string, targetRel string) ([]services.WorkspaceTransferResult, error)`，风格对齐现有工作区绑定（fastlog `Debugw/Errorw/Infow`，错误透传 services）。

### 4. 测试（workspace\_service\_test.go）

复用 `setupWorkspaceService` / `findEntry` / `mustWrite`，新增用例：

* 上传文件到子目录（`a/b/` 下重名自动改名）

* 上传整个目录到子目录（递归 roundtrip）

* `targetRel=".."` / `"../x"` 逃逸被整体拒绝

* `targetRel` 不存在被拒绝

* `targetRel=""` 行为与 `UploadFiles` 等价（根目录）

## 二、前端

### 1. index.html：面板内新增常驻拖拽遮罩节点

`.workspace-body` 内、`.workspace-tree` 之上新增 `<div id="workspaceDropOverlay" class="workspace-drop-overlay" style="display:none;">`，内含一行文案 `<span>`（「释放到目录 xxx」/「释放到工作区根目录」）。常驻节点，仅切 display，不动态创建。

### 2. workspace-manager.js

* 新增模块级状态 `workspaceDropTargetRel = null`（null = 根目录；非空 = 最后悬停的目录 RelPath）

* `bindWorkspaceEvents` 新增面板 `#workspaceModal` 的 dragenter / dragover / dragleave / drop 四个监听（懒绑定常驻范式）：

  * dragenter / dragover：`e.preventDefault()`；`dataTransfer.types.includes('Files')` 才处理（防文本拖拽误导）；`elementFromPoint(e.clientX, e.clientY).closest('.workspace-node.is-dir')` 取悬停目录行

  * **防抖**：仅当悬停目标 rel 变化时才更新 `workspaceDropTargetRel`、目录行高亮（`.ws-drop-target`）与遮罩文案（dragover 高频触发，避免无谓 DOM 操作）

  * dragleave：清理高亮（计数器防抖，参考 `_aiDragCounter` 范式）

  * drop：仅清理 UI（真实文件由 OnFileDrop 处理）

* 新增 `handleWorkspaceDrop(paths)`（挂 `window`，供 main.js OnFileDrop 调用）：

  * `paths` 为空直接返回；`setWorkspaceBusy(true, null)` 置忙

  * `window.go.main.App.UploadPathsToWorkspace(paths, workspaceDropTargetRel)`（带 `workspaceDropTargetRel || ''`）

  * 成功后通知结果 + 刷新；**目标目录祖先链全部加入** **`workspaceExpanded`**（如 `a/b/c` 需展开 `a`、`a/b`、`a/b/c`），刷新后用户能直接看到新上传文件

  * `finally` 复位 busy、隐藏遮罩、清高亮

* `closeWorkspaceManager` 复位 `workspaceDropTargetRel` + 移除所有 `.ws-drop-target` + 隐藏遮罩（双保险）

### 3. main.js OnFileDrop 路由（最前面插入）

```js
const wsModal = document.getElementById('workspaceModal');
if (wsModal && wsModal.style.display !== 'none') {
    if (typeof window.handleWorkspaceDrop === 'function') await window.handleWorkspaceDrop(paths);
    return; // 面板打开时屏蔽一切其他拖拽功能
}
```

面板打开时铺满视口，任何落点都在面板内，无需再按坐标细分。面板关闭（display:none）时走原逻辑，互不干扰。

### 4. 全局拖拽守卫（关键：不拦截冒泡，改统一守卫）

**不在面板节点上 stopPropagation**（会破坏 document 级 `_dragCounter` 进出平衡导致遮罩残留）。改为在 document 的 dragenter / dragleave / drop 三个处理器（main.js `initFileDrop` 内）**最前面**加守卫：

```js
const wsModal = document.getElementById('workspaceModal');
if (wsModal && wsModal.style.display !== 'none') { e.preventDefault(); return; }
```

面板打开期间全局 `_dragCounter` 恒为 0，天然无残留，「释放以导入文件」遮罩也不会误亮。dragover 守卫注意先 `preventDefault` 防浏览器导航。

### 5. ai-chat.js `initAiChatFileDrop` 同款守卫

dragenter / dragover / dragleave / drop 四个处理器最前面加面板打开判断，面板打开时直接 `return`（dragover 需 `preventDefault`），避免 AI 聊天遮罩在面板场景下误亮或计数残留。

### 6. workspace.css

* `.workspace-drop-overlay`：absolute 覆盖 `.workspace-body`，半透明主题变量背景（`--overlay-bg` 或 accent 淡色）+ 居中大字文案 + 圆角，开合淡入

* `.workspace-node.is-dir > .workspace-row.ws-drop-target`：accent 色高亮背景 + 左边界色条（优先级高于 `.workspace-row:hover`）

## 三、验证

1. `gofmt` 无输出 + `go build ./...` + `go vet ./...` + `go test ./internal/services/ -count=1` 全绿（新增测试覆盖子目录/逃逸/根目录）
2. `npm run build` 通过（Vite）
3. `wails generate module` 重新生成 frontend/wailsjs 绑定（新增 `UploadPathsToWorkspace`）
4. 手动验证：

   * 打开面板，拖文件到空白区 → 上传到根目录，通知成功，树刷新可见

   * 拖文件悬停到某目录行（高亮亮起）→ 释放 → 上传到该目录，且祖先链目录自动展开可见

   * 拖整个文件夹悬停目录行 → 递归上传，重名自动改名

   * 面板打开时拖到面板外 / AI 聊天区 → 无任何旧功能触发，仅面板上传

   * 拖拽中 ESC 关闭面板 → 释放后走主界面原路由（降级合理），面板状态干净无残留

   * 拖文本（非文件）进面板 → 无误导文案
5. 纯前后端改动，需 `wails build` 出新二进制生效

