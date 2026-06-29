# AI 助手（对话页面）Spec

## Why
Jot 当前是纯本地笔记工具，用户希望在笔记应用中集成 AI 能力，通过配置自己的 Base URL 和 API Key，在笔记应用内与 AI 进行对话。

## What Changes
- **新增**: 设置页 AI 配置面板（Base URL / API Key / Model）
- **新增**: AI 对话独立页面，基于 QuikChat 组件构建
- **新增**: 后端 AI Service 服务（调用 OpenAI 兼容格式 API）
- **新增**: 后端存储 AI 配置（复用 SettingService KV 存储）

## QuikChat 组件选型
选用 [QuikChat](https://www.npmjs.com/package/quikchat)（v^1.2.7）零依赖 Vanilla JS 聊天组件：
- 通过 npm 安装，ESM 导入 `import quikchat from 'quikchat/md'` + `import 'quikchat/dist/quikchat.css'`
- 内置 Markdown 渲染（加粗/斜体/代码/表格/列表）
- 原生流式输出支持（`messageAddNew` → `messageAppendContent`）
- 打字指示器 `messageAddTypingIndicator`
- 对话历史管理 `historyGet()` / `historyExport()`
- ～9KB gzipped（含 Markdown）

## Impact
- Affected code:
  - `frontend/index.html`：新增 `#viewAiChat` AI 对话页面结构、设置页 AI 配置面板
  - `frontend/src/js/ai-chat.js`：**新增独立模块**，初始化 QuikChat 实例 + 业务逻辑（附加上下文/插入编辑器/配置检测）
  - `frontend/src/main.js`：import `ai-chat.js`、注册视图切换/快捷键/更多菜单
  - `frontend/src/css/components/ai-chat.css`：**新增**，覆盖 QuikChat 默认样式，适配 Jot 主题变量
  - `frontend/src/css/components/settings-panel.css`：新增 AI 配置样式
  - `app.go`：新增 AI 相关绑定方法
  - `internal/services/`：新增 `ai_service.go` AI 服务
  - `frontend/package.json`：已添加 `quikchat: ^1.2.7`
- Affected specs: 无（全新功能，不修改现有 spec）

## ADDED Requirements

### Requirement: 后端 AI 服务
系统 SHALL 提供后端 AI 服务，支持调用 OpenAI 兼容格式的 API。

#### Scenario: 调用 AI 对话 API
- **WHEN** 前端发送 prompts 列表（对话历史）到后端
- **THEN** 后端使用配置的 Base URL + API Key + Model，按 OpenAI Chat Completion API 格式调用 `${base_url}/chat/completions`
- **AND** 返回 AI 响应文本

#### Scenario: 保存 AI 配置
- **WHEN** 用户在设置页保存 AI 配置
- **THEN** 后端通过 `SetSetting` 存储 `ai_base_url`、`ai_api_key`、`ai_model` 三个设置项

#### Scenario: 读取 AI 配置
- **WHEN** 前端需要调用 AI 时
- **THEN** 后端通过 `GetSetting` 读取配置
- **AND** 如果缺少必要配置（如 API Key 为空），返回明确的错误提示

#### Scenario: 测试 Base URL 连通性
- **WHEN** 用户点击「测试 URL」按钮
- **THEN** 后端向 `${base_url}/models` 发送 GET 请求
- **AND** 返回连通成功/失败状态

#### Scenario: 获取模型列表
- **WHEN** 用户点击「获取模型列表」按钮
- **THEN** 后端向 `${base_url}/models` 发送 GET 请求（携带 API Key）
- **AND** 返回可用模型 ID 列表（从响应 `data[].id` 字段提取）

### Requirement: 设置页 AI 配置面板
设置页 SHALL 提供 AI 配置区域，让用户填写 API 连接信息。

#### Scenario: AI 配置 section
- **WHEN** 用户打开设置页
- **THEN** 在「编辑器选项」section 之后显示「AI 助手」section
- **AND** 包含以下配置项：
  - Base URL 输入框 + 「测试 URL」按钮（检测连通性，状态提示）
  - API Key 密码输入框（type=password，带显示/隐藏切换按钮）
  - Model 下拉选择器 + 「获取模型列表」按钮（点击后拉取模型列表填充下拉菜单）
  - 保存按钮（保存所有配置）
  - 状态提示区

### Requirement: AI 对话页面
系统 SHALL 提供一个独立页面视图（类似设置页、数据管理页），让用户以聊天形式与 AI 交互。该页面使用 QuikChat 组件构建，业务逻辑放在 `frontend/src/js/ai-chat.js` 独立文件中。

#### Scenario: QuikChat 初始化
- **WHEN** AI 对话页面首次进入
- **THEN** `ai-chat.js` 的 `initAIChat()` 初始化 QuikChat 实例
- **AND** 传入 `onSend` 回调：构建 prompts → 调 `CallAI` → 渲染回复
- **AND** 选择 `quikchat/md` 版本以支持 Markdown 渲染
- **AND** 使用 QuikChat 内置的 typing indicator 展示加载态

#### Scenario: 进入 AI 对话页面
- **WHEN** 用户通过更多菜单或快捷键进入 AI 对话页面
- **THEN** 如果尚未配置 AI，页面中显示提示和「前往设置」快捷链接
- **AND** 已配置则进入 QuikChat 聊天界面

#### Scenario: 对话页面布局
- **WHEN** AI 对话页面打开
- **THEN** 页面包含：
  - 顶部 view-header：返回按钮 + 「AI 助手」标题 + 清空对话按钮
  - 中间 QuikChat 容器（`#aiChatContainer`，撑满剩余高度）
  - 底部「附加上下文」按钮（独立于 QuikChat 输入框之外）
  - 上下文指示器：显示当前是否附加了笔记内容
- **AND** QuikChat 默认样式通过 `ai-chat.css` 覆盖以适配 Jot 主题变量

#### Scenario: 发送消息
- **WHEN** 用户在 QuikChat 输入框输入内容并发送
- **THEN** QuikChat 的 `onSend` 回调触发
- **AND** 调用 `window.runtime.CallAI([...history, {role:'user', content: msg}])`
- **AND** QuikChat 显示打字指示器
- **AND** 收到回复后调用 `chat.messageReplaceContent(id, response)` 渲染

#### Scenario: 附加上下文
- **WHEN** 用户点击「附加上下文」按钮
- **THEN** 当编辑器打开且有内容时，当前笔记标题和内容作为 `system` role 消息附加到 prompts 列表中
- **AND** 上下文指示器显示「已附加: 笔记标题」

#### Scenario: 插入 AI 回复到编辑器
- **WHEN** AI 回复完成后，通过 QuikChat 事件回调在回复消息底部附加「插入到编辑器」按钮
- **THEN** 用户点击后，AI 回复内容插入到 CM6 编辑器的光标位置（或追加到内容末尾）
- **AND** 插入完成后页面保持当前视图不切换

#### Scenario: 清空对话
- **WHEN** 用户点击标题旁的「清空对话」按钮
- **THEN** 显示确认提示
- **AND** 确认后调用 QuikChat 的 `historyClear()` 清除所有消息

### Requirement: 更多菜单 + 快捷键入口
系统 SHALL 提供 AI 对话页面的快捷入口。

#### Scenario: 更多菜单入口
- **WHEN** 用户点击 topbar 的更多菜单按钮
- **THEN** 菜单中新增「AI 助手」项（快捷键提示 Ctrl+Shift+F）
- **AND** 放置在「快捷键说明」与「MD 语法」之间

#### Scenario: 快捷键
- **WHEN** 用户按下 Ctrl+Shift+F
- **THEN** 切换到 AI 对话页面

## REMOVED Requirements
无
