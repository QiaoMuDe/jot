# 优化编辑器拖拽行为 Spec

## Why

当前编辑器打开时（查看/编辑/新建模式），若非图片文件拖拽到编辑器上，会走 `handleFileDropPaths` 创建为新笔记，同时 CM6 原生 drop 事件又会把文件内容插入到编辑器光标处，导致"既新建了笔记又填入了内容"的冲突。需要统一行为：编辑器打开时不再从拖拽创建笔记，查看模式全部忽略，编辑/新建模式下区分图片（上传）、文本（插入内容）、二进制（忽略）。

## What Changes

- **CM6 编辑器注册 `drop` 事件 preventDefault** — 阻止 CM6 原生文件内容插入（在 target 阶段拦截）
- **Go 后端新增 `ReadTextFile` 方法** — 复用 `fs.IsBinaryPath` 检测，非二进制则返回内容
- **重写 `OnFileDrop` 路由逻辑** — 编辑器打开时（任何模式）不再创建笔记，编辑/新建模式下图片上传、文本插入、二进制忽略
- **编辑器外拖拽也受编辑器打开状态影响** — 编辑器打开时全局禁止拖入创建新笔记

## Impact

- Affected code: [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go)（新增 ReadTextFile）、[main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js)（CM6 drop 拦截 + OnFileDrop 路由重写）

## ADDED Requirements

### Requirement: CM6 原生 drop 拦截

系统 SHALL 阻止 CM6 编辑器在文件拖入时的默认文本插入行为（target 阶段）。

#### Scenario: 文件拖入编辑器
- **WHEN** 用户拖拽文件到 CM6 编辑器上
- **THEN** CM6 的 `drop` 事件被 `preventDefault()` 阻止，不插入文件内容

### Requirement: 编辑器已打开时全局禁止拖入建笔记

系统 SHALL 在编辑器打开时（任何模式），禁止从任何拖拽操作创建新笔记。

#### Scenario: 编辑器打开 + 拖到编辑器外
- **WHEN** 查看/编辑/新建模式下，用户拖拽文件到编辑器外的区域
- **THEN** 不做任何处理（不创建笔记、不上传图片）

### Requirement: 查看模式拖拽忽略

系统 SHALL 在查看模式下拖拽文件到编辑器时不执行任何操作。

#### Scenario: 查看模式 + 拖到编辑器上
- **WHEN** 查看模式下，用户拖拽任何文件到编辑器上
- **THEN** 不做任何处理（不创建笔记、不上传图片、不插入内容）

### Requirement: 编辑/新建模式拖拽文本文件插入内容

系统 SHALL 在编辑/新建模式下，拖拽文本文件到编辑器时读取内容并插入到光标处。

#### Scenario: 拖拽文本文件到编辑器
- **WHEN** 编辑/新建模式下，用户拖拽 `.txt`/`.md` 等文本文件到编辑器
- **THEN** Go 方法 `ReadTextFile` 调用 `fs.IsBinaryPath` 检查是非二进制文件
- **THEN** 返回文件内容字符串，前端在光标处插入
- **THEN** 二进制文件（如 `.zip`/`.exe`）被忽略，不操作

### Requirement: 编辑器未打开时保持原行为

系统 SHALL 在编辑器未打开（无笔记在查看/编辑）时，保持原有的拖拽导入笔记行为。

#### Scenario: 编辑器未打开 + 拖拽文件
- **WHEN** 编辑器未打开，用户拖拽文件到应用窗口
- **THEN** 调用 `handleFileDropPaths` 创建为新笔记

## MODIFIED Requirements

### Requirement: OnFileDrop 路由（从 add-editor-drag-drop-image 修改编辑器拖拽逻辑）

原编辑器拖拽路由只处理了图片路径分流到上传。现改为完整的三级判断：

1. **编辑器是否打开**（`cmEditor !== null`）：
   - 未打开 → `handleFileDropPaths`（原导入逻辑）
   - 已打开 → 不创建新笔记，继续判断
2. **编辑器是否只读**（查看模式）：
   - 只读 → 全部忽略
   - 非只读 → 继续判断
3. **文件类型**：
   - 图片 → `SaveImageFromPath` + 插入 `![](...)`
   - 非图片 → 调用 `ReadTextFile` 检查：
     - 文本文件 → 插入内容到光标处
     - 二进制文件 → 忽略

#### Scenario: 编辑模式拖拽混合文件
- **WHEN** 编辑模式下拖拽同时包含图片和文本文件到编辑器
- **THEN** 图片依次上传插入，文本文件依次读取插入
- **THEN** 二进制文件被忽略

## REMOVED Requirements

无。
