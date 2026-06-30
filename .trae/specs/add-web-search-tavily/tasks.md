# Tasks

- [x] Task 1: 添加 Tavily Go 依赖并实现搜索服务
  - [x] `go get github.com/hekmon/tavily@v1.3.0`
  - [x] 新建 `internal/services/search_service.go`，实现 `SearchWeb(query string) (string, error)` 函数
  - [x] 函数内部：创建 Tavily client → 调用 Search (search_depth="advanced", max_results=5) → 提取标题+URL+content → 格式化为纯文本字符串返回
  - [x] 包含错误处理和超时控制（5秒超时）
- [x] Task 2: 修改后端 AI 配置和流式调用
  - [x] `ai_service.go`: `AIConfig` 新增 `TavilyAPIKey string` 字段
  - [x] `ai_service.go`: `GetConfig()` / `SaveConfig()` 同步读写 `tavily_api_key` 到 SettingService
  - [x] `ai_service.go`: `CallAIStream` 新增 `searchEnabled bool` 参数
  - [x] `ai_service.go`: `CallAIStream` 中当 `searchEnabled=true` 时，调 `SearchWeb` 获取结果并拼入 system message
  - [x] `app.go`: `CallAIStream` 签名同步新增 `searchEnabled bool`
  - [x] `app.go`: 新增 `TestTavilyConnection(apiKey string) (bool, error)` binding 用于设置页测试连接
- [x] Task 3: 前端设置页 — Tavily API Key 配置
  - [x] `index.html`: AI 设置区新增「Tavily API Key」输入框 +「测试连接」按钮 +「联网搜索默认开启」开关
  - [x] `main.js`: 对应的加载/保存/测试连接逻辑
  - [x] 复用现有 `.ai-setting-search-line` CSS 样式
- [x] Task 4: 前端 AI 聊天 — 联网搜索开关 + 流式参数传递
  - [x] `index.html`: 在 AI 聊天头部「深度思考」开关旁新增「联网搜索」开关（复用 toggle 样式，新 id）
  - [x] `ai-chat.js`: 新增 `enableSearch` 变量 + localStorage 持久化 + 开关事件绑定
  - [x] `ai-chat.js`: `onSend()` 中 `CallAIStream` 调用新增 `enableSearch` 参数
  - [x] `ai-chat.js`: 同步设置页开关状态
- [x] Task 5: AGENTS.md 更新记忆

## Task Dependencies

- [Task 1] → [Task 2]
- [Task 3] 与 [Task 4] 可并行
- [Task 2] 与 [Task 4] 都完成后端和前端即完成对接
