# Tasks

- [x] Task 1: 创建后端 AI Service — 实现 `internal/services/ai_service.go`，支持 OpenAI 兼容格式 API 调用 + 配置管理 + 连通性检测 + 模型列表拉取
  - [x] 定义 `Message` 结构体（Role/Content 字段）和 `AIConfig` 结构体（BaseURL / APIKey / Model）
  - [x] 实现 `CallAI(prompts []Message) (string, error)` — 调用 `${base_url}/chat/completions`
  - [x] 实现 `TestBaseURL(baseURL string) (bool, error)` — 向 `${base_url}/models` 发 GET 请求检测连通性
  - [x] 实现 `FetchModels(baseURL, apiKey string) ([]string, error)` — 从 `/models` 获取模型 ID 列表
  - [x] 配置管理：从 `SettingService` 读写 `ai_base_url` / `ai_api_key` / `ai_model`

- [x] Task 2: 后端绑定层 — 在 `app.go` 中新增 AI 相关 Wails 绑定方法
  - [x] `GetAIConfig() (AIConfig, error)` — 读取 AI 配置
  - [x] `SaveAIConfig(config AIConfig) error` — 保存 AI 配置
  - [x] `TestAIBaseURL(baseURL string) (bool, error)` — 测试 URL 连通性
  - [x] `FetchAIModels(baseURL, apiKey string) ([]string, error)` — 获取模型列表
  - [x] `CallAI(prompts []Message) (string, error)` — 调用 AI 对话
  - [x] 在 `App` 结构体中注册 `AIService`

- [x] Task 3: 设置页 AI 配置 UI — 在 `index.html` 和 `main.js` 中新增 AI 配置 section
  - [x] HTML：在「编辑器选项」section 后新增「AI 助手」settings-section
  - [x] Base URL 输入框（placeholder: `https://api.openai.com/v1`）+ 「测试 URL」按钮 + 状态指示
  - [x] API Key 密码输入框（placeholder: `sk-...`，带显示/隐藏切换图标按钮）
  - [x] Model 下拉选择器（`#aiModelSelect`）+ 「获取模型列表」按钮
  - [x] CSS 样式（复用 settings-panel.css 现有样式模式）
  - [x] JS（在 main.js 中）：`initAISettings()` — 加载/测试 URL/获取模型列表/保存配置

- [x] Task 4: 创建 AI 对话页面 HTML + CSS — 独立页面视图 + QuikChat 容器 + 样式覆盖
  - [x] HTML：在 `index.html` 中新增 `#viewAiChat` 页面
  - [x] view-header：返回按钮 + 「AI 助手」标题 + 清空对话按钮
  - [x] QuikChat 挂载容器 `#aiChatContainer`（flex:1 撑满剩余高度）
  - [x] 底部「附加上下文」按钮 + 上下文指示器元素
  - [x] 新建 `ai-chat.css`：覆盖 QuikChat 默认样式，使用 Jot CSS 变量（`--bg`/`--text-primary`/`--accent`/`--border` 等），调整气泡样式/输入框/滚动条
  - [x] 更多菜单新增「AI 助手」条目 + Ctrl+Shift+F 快捷键注册（在 main.js 中）

- [x] Task 5: 创建 AI 对话页面 JS 逻辑 — 新建 `frontend/src/js/ai-chat.js` 独立模块
  - [x] 导入 QuikChat：`import quikchat from 'quikchat/md'` + `import 'quikchat/dist/quikchat.css'`
  - [x] `initAIChat()`：判断配置 → 初始化 QuikChat 实例 → 绑定事件
  - [x] QuikChat `onSend` 回调：构建 prompts 列表（含附加的上下文）→ `window.runtime.CallAI()` → 替换 typing indicator → 渲染回复
  - [x] 附加上下文：获取 `els.editorNoteTitle.value` + `getEditorContent()`，作为 system prompt
  - [x] 插入按钮：AI 回复渲染完成后，在消息 DOM 中附加「插入到编辑器」按钮
  - [x] 无配置时页面显示提示 + 「前往设置」链接
  - [x] 清空对话：确认后调用 `chat.historyClear()`
  - [x] QuikChat 主题与 Jot 主题变量联动
  - [x] 在 `main.js` 中 import 并调用 `initAIChat()`

# Task Dependencies
- [Task 1] 依赖无
- [Task 2] 依赖 [Task 1]
- [Task 3] 依赖 [Task 2]
- [Task 4] 依赖无（可与 Task 1/2/3/5 并行）
- [Task 5] 依赖 [Task 2]、[Task 4]
