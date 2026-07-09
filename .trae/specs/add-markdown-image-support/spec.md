# Markdown 笔记图片粘贴与显示 Spec

## Why

当前 `.md` 笔记的预览模式无法显示本地图片路径，且用户无法通过粘贴方式从剪贴板插入图片。需要实现图片上传到后端统一存储、在笔记中以 HTTP URL 形式引用并显示的能力。

## What Changes

- **Go 后端**：新增图片存储服务，图片统一保存到 `~/.jot/images/`；新增 Wails AssetServer Handler 拦截 `/images/*` 请求以 HTTP 方式提供文件访问；新增 2 个前端可调用的 Go API（图片上传、本地路径导入）
- **前端编辑器**：在 CodeMirror 6 编辑器中监听粘贴事件，检测图片文件并自动上传后插入 `![](/images/xxx.png)` 格式的 Markdown 图片语法
- **预览渲染**：利用现有 marked 渲染 + Wails 文件服务器，图片自动以 HTTP URL 方式正常显示（预览/编辑/查看三种模式均生效）
- **图片存储目录**：在 `app.startup` 中自动初始化 `~/.jot/images/` 目录

## Impact

- Affected specs: add-md-editor, add-md-rendering, add-note-file-ext
- Affected code: `main.go`, `app.go`, `frontend/src/main.js`
- 不迁移现有笔记中的本地路径图片（本期范围限定在**新粘贴的图片**；本地路径兼容放在后续迭代）

## ADDED Requirements

### Requirement: Go 后端图片文件服务器

系统 SHALL 在 Wails AssetServer 中注册自定义 Handler，当请求路径以 `/images/` 开头时，从 `~/.jot/images/` 目录提供静态文件服务。

#### Scenario: 图片文件通过 HTTP 可访问
- **GIVEN** 图片文件 `abc.jpg` 存在于 `~/.jot/images/`
- **WHEN** 浏览器请求 `/images/abc.jpg`
- **THEN** 返回该图片文件，Content-Type 为对应 MIME 类型

#### Scenario: 图片不存在返回 404
- **GIVEN** 图片文件 `nonexist.png` 不存在于 `~/.jot/images/`
- **WHEN** 浏览器请求 `/images/nonexist.png`
- **THEN** 返回 HTTP 404

### Requirement: 图片存储目录自动初始化

系统 SHALL 在应用启动时自动创建 `~/.jot/images/` 目录（若不存在）。

#### Scenario: 首次启动
- **GIVEN** `~/.jot/images/` 目录不存在
- **WHEN** 应用启动
- **THEN** 自动创建 `~/.jot/images/` 目录

### Requirement: Go 后端图片上传 API

系统 SHALL 通过 Wails Bind 暴露 `SaveImage(name string, data []byte) (string, error)` 方法。

#### Scenario: 上传图片成功
- **GIVEN** 前端上传一张名为 `photo.png`、内容为二进制数据的图片
- **WHEN** 调用 `SaveImage`
- **THEN** 图片保存为 `~/.jot/images/uuid_photo.png`
- **THEN** 返回可访问 URL `/images/uuid_photo.png`

#### Scenario: 文件名冲突处理
- **GIVEN** `~/.jot/images/` 已存在同名文件
- **WHEN** 调用 `SaveImage`
- **THEN** 自动添加 UUID 前缀，不会覆盖已有文件

### Requirement: 前端粘贴图片上传

系统 SHALL 在 CodeMirror 6 编辑器中添加粘贴事件监听（仅 `.md` 后缀笔记生效），检测到剪贴板中图片文件时自动上传并插入 `![](/images/xxx.png)`。

#### Scenario: 在 Markdown 笔记中粘贴图片
- **GIVEN** 当前编辑的笔记后缀为 `.md`，编辑器处于编辑模式
- **WHEN** 用户使用 Ctrl+V 粘贴剪贴板中的图片
- **THEN** 调用 `SaveImage` 上传图片
- **THEN** 在光标位置插入 `![](/images/uuid_originalname.ext)`
- **THEN** 预览模式自动刷新显示图片

#### Scenario: 非图片粘贴不影响
- **GIVEN** 当前编辑的笔记后缀为 `.md`
- **WHEN** 用户粘贴纯文本或非图片文件
- **THEN** 不触发图片上传逻辑，保留默认粘贴行为

#### Scenario: 非 Markdown 笔记不影响
- **GIVEN** 当前编辑的笔记后缀为 `.txt`
- **WHEN** 用户粘贴图片
- **THEN** 不触发图片上传逻辑，保留默认粘贴行为

### Requirement: 预览模式图片显示

系统 SHALL 利用现有 `marked` 渲染管线 + Wails 文件服务器，使 `/images/xxx.png` 格式的 Markdown 图片语法在三种模式中正常显示：
- 查看模式（`.md` 自动预览）
- 编辑模式的预览切换
- 新建 `.md` 笔记后的预览

三种模式均走同一套 `updatePreview()` → `marked.parse()` → `<img src="/images/...">` → Wails Handler 提供文件的链路。

#### Scenario: 查看模式显示图片
- **GIVEN** 笔记内容为 `![图](/images/abc.jpg)`，且文件存在
- **WHEN** 以查看模式打开该笔记
- **THEN** 预览区 `<img>` 标签 src 正确指向 `/images/abc.jpg`
- **THEN** 图片在预览区正常渲染显示

#### Scenario: 编辑模式预览显示图片
- **GIVEN** 笔记内容为 `![图](/images/abc.jpg)`，且文件存在
- **WHEN** 在编辑模式点击"预览"切换
- **THEN** 预览区正常显示图片

## MODIFIED Requirements

### Requirement: 应用启动初始化（修改 `app.startup`）

系统 SHALL 在 `app.startup` 中原有初始化逻辑后增加图片存储目录创建步骤。

### Requirement: Wails AssetServer 配置（修改 `main.go`）

系统 SHALL 在 `main.go` 的 `AssetServer` 选项中增加 `Handler` 字段，拦截 `/images/*` 请求回退到 `~/.jot/images/` 目录的文件服务。

## REMOVED Requirements

（无删除项）
