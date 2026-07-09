# AI 聊天拖拽上传文件 Spec

## Why
AI 聊天模块只支持点击"上传"按钮从对话框选择文件，不支持拖拽文件直接上传。用户需要打开文件选择器才能上传文件，交互路径较长。通过拖拽文件到 AI 聊天区域，可直接复用已有的验证/读取/截断逻辑，提升操作效率。

## What Changes
- **后端**: 将 `SelectAIChatFiles` 中的文件校验/读取/截断逻辑提取为内部方法 `readAIChatFiles(paths []string) []AIChatFileResult`，让按钮上传和拖拽上传共用
- **HTML**: 在 `.ai-chat-content` 内新增 `#aiChatDropOverlay` 遮罩层（初始隐藏），显示"释放以上传文件"文字
- **CSS**: 新增 AI 聊天专属遮罩样式（`position: absolute` 锚定在 `.ai-chat-content`），复用全局遮罩的视觉风格
- **JS (main.js)**: 
  - 全局 `initFileDrop()` 中 `dragenter/dragover/dragleave/drop` 增加 AI 聊天区判断，在 AI 聊天区内不操作全局遮罩和 `_dragCounter`
  - 全局 `OnFileDrop` 回调中增加 AI 聊天区判断，路由到 AI 聊天文件处理逻辑
- **JS (ai-chat.js)**: 
  - 在 `.ai-chat-content` 上注册 `dragenter/dragover/dragleave/drop` 事件
  - AI 专属拖拽计数器控制 `#aiChatDropOverlay` 显隐
  - 拖拽文件处理通过 `window.go.main.App.ReadAIChatFiles(paths)` 获取结果，加入 `uploadedFiles[]` 并渲染 chips

## Impact
- Affected specs: AI Chat 文件上传功能、全局拖拽导入功能
- Affected code: `app.go`（提取内部方法）、`index.html`（新增遮罩）、`ai-chat.css`（遮罩样式）、`ai-chat.js`（拖拽事件+处理）、`main.js`（全局拖拽路由）

## ADDED Requirements

### Requirement: 后端提取内部方法 readAIChatFiles
The system SHALL extract the file validation/reading/truncation logic from `SelectAIChatFiles` into a reusable internal method.

#### Scenario: 共用逻辑
- **WHEN** 用户通过按钮上传文件（SelectAIChatFiles）
- **THEN** 调用内部方法 `readAIChatFiles(paths)` 处理文件
- **WHEN** 用户拖拽文件到 AI 聊天区
- **THEN** 调用同一个内部方法 `readAIChatFiles(paths)` 处理文件
- **THEN** 两种入口共享相同的校验规则：目录拒绝、10MB 大小限制、IsBinaryPath 二进制检测、GetAIRefMaxChars 内容截断

### Requirement: AI 聊天专属拖拽遮罩
The system SHALL provide a drag overlay within the AI chat content area.

#### Scenario: 遮罩显示
- **WHEN** 用户拖拽文件进入 `.ai-chat-content` 区域
- **THEN** `#aiChatDropOverlay` 显示，文字为"释放以上传文件"
- **WHEN** 用户拖拽文件离开 `.ai-chat-content` 区域
- **THEN** `#aiChatDropOverlay` 隐藏
- **WHEN** 用户在 `.ai-chat-content` 区域内释放文件
- **THEN** `#aiChatDropOverlay` 隐藏，文件开始处理

### Requirement: 全局拖拽屏蔽 AI 聊天区
The system SHALL skip the global drag overlay when dragging over the AI chat content area.

#### Scenario: 全局遮罩不覆盖 AI 聊天区
- **WHEN** 用户拖拽文件进入 AI 聊天区域
- **THEN** 全局 `#dropOverlay` 不显示
- **THEN** 全局 `_dragCounter` 不递增
- **WHEN** 用户拖拽文件离开 AI 聊天区域进入其他区域
- **THEN** 全局拖拽逻辑恢复正常

### Requirement: OnFileDrop 路由至 AI 聊天
The system SHALL route file drops in the AI chat area to the AI chat file processing logic.

#### Scenario: 路由判断
- **WHEN** `window.runtime.OnFileDrop` 触发且释放坐标在 `.ai-chat-content` 内
- **THEN** 调用 `window.go.main.App.ReadAIChatFiles(paths)` 获取文件结果
- **THEN** 将结果添加到 `uploadedFiles[]`（失败项显示错误通知，成功项加入列表）
- **THEN** 调用 `renderFileChips()` 渲染文件 chips

## MODIFIED Requirements
### Requirement: 全局拖拽事件处理
- `initFileDrop()` 中的 `dragenter/dragover/dragleave/drop` 事件增加 AI 聊天区域判断
- 当 `e.target.closest('.ai-chat-content')` 存在时，不处理全局遮罩状态

### Requirement: OnFileDrop 回调
- 在现有 `if (cmEditor !== null)` 和 `else` 分支之前增加 AI 聊天区域判断
- AI 聊天区域判断优先级高于编辑器判断

## REMOVED Requirements
（无移除项）
