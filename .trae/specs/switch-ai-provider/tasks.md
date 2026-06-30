# Tasks

- [x] Task 1: 添加 LangChainGo 依赖并构建通过
  - 运行 `go get github.com/tmc/langchaingo` 拉取依赖
  - 确保项目能正常编译，不影响现有功能
  - **不依赖其他任务**

- [x] Task 2: 扩展 `AIConfig` 结构体，新增 `Provider` 字段
  - `internal/services/types.go` 中 AIConfig 加 `Provider string \`json:"provider"\``
  - 更新后端设置存储逻辑，支持 `ai_provider` key（启动时如果没有则默认 `"openai"`）
  - 前端 `models.ts` 和 `App.d.ts` 等 Wails 绑定文件不需要手动修改，后续 wails build 会自动生成
  - **依赖 Task 1**

- [x] Task 3: 重写 `ai_service.go` 使用 LangChainGo 统一接口
  - 引入 `github.com/tmc/langchaingo/llms` 和对应 sub-packages
  - 新建 `createLLM(cfg AIConfig) (llms.Model, error)` 工厂函数，根据 `cfg.Provider` 路由：
    - `"openai"` → `openai.New(...)`，支持自定义 BaseURL
    - `"anthropic"` → `anthropic.New(...)`，支持 thinking 模式
    - `"google"` → `googleai.New(...)`
    - `"ollama"` → `ollama.New(...)`，支持自定义 ServerURL
  - 重写 `CallAIStream` 使用 `llms.GenerateContent` + `WithStreamingFunc`
  - 保持 `ai:stream-chunk` / `ai:stream-thinking` / `ai:stream-done` / `ai:stream-error` 推送逻辑不变
  - 保留 Message 结构体和 Session 持久化逻辑
  - **依赖 Task 2**

- [x] Task 4: 更新 `app.go` 中 AI 相关方法适配 Provider
  - `TestAIBaseURL` → 改为调用 `TestConnection(cfg AIConfig)`，用 LangChainGo 的统一调用测试
  - `FetchAIModels` → 只对 `openai` provider 生效，其他 provider 返回空列表
  - **依赖 Task 3**

- [x] Task 5: 更新前端 AI 设置页面 — 添加「服务商」下拉选择器
  - `index.html` 中 AI 设置区域新增 provider 下拉选择框，4 个选项
  - JavaScript: 根据选择动态控制 Base URL 行的显示/隐藏
  - JavaScript: 非 OpenAI 兼容时禁用「获取模型列表」按钮
  - JavaScript: 切换 provider 时重置 model 下拉
  - **依赖 Task 2**（需要 AIConfig.Provider 字段就绪后才联调）

# Task Dependencies
- [Task 1] 无依赖
- [Task 2] 依赖 [Task 1]
- [Task 3] 依赖 [Task 2]
- [Task 4] 依赖 [Task 3]
- [Task 5] 依赖 [Task 2]（前端联调依赖 Task 2 的 API 变更）
