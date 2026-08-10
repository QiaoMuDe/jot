# Tasks

- [x] Task 1: 主项目添加 Eino 依赖
  - [ ] SubTask 1.1: 在根 `go.mod` 添加 `github.com/cloudwego/eino v0.9.13` 与 `github.com/cloudwego/eino-ext/components/model/openai v0.1.13`（复用 agent-demo 已下载的模块缓存），运行 `go mod tidy`（如网络受限则手工补全 go.sum 所需条目）
  - [ ] SubTask 1.2: 验证 `go build ./...` 通过，且现有 `aicli`（go-openai v1）不受影响

- [x] Task 2: 会话配置新增 AgentEnabled 字段
  - [ ] SubTask 2.1: `internal/models/ai_session_config.go` 的 `AISessionConfig` 新增 `AgentEnabled bool`（gorm/json: agent_enabled）
  - [ ] SubTask 2.2: `internal/services/ai_service.go` 的 `SessionConfig` 新增对应字段，并透传：读取 `GetSessionConfig` 填入、保存 `SaveSessionConfig` 写库
  - [ ] SubTask 2.3: 确认数据库迁移（AutoMigrate）覆盖新字段（跟随 `internal/database/models.go` 现有机制）

- [x] Task 3: 新建 internal/agent 独立模块（核心链路，未来功能均在此扩展）
  - [ ] SubTask 3.1: 定义 `AgentService` 结构：构造函数 `NewAgentService(aiSvc, searchSvc, vectorSvc, settingSvc, logger)` 注入现有服务依赖；对外接口 `Run(ctx, Request, Emit)`（`Request` 为入参结构，`Emit func(event, data string)` 为事件回调，由调用方注入 Wails runtime，模块内不直接 import Wails）
  - [ ] SubTask 3.2: `tools.go`——封装两个读工具（`tool.InvokableTool`）：
    * `web_search(query, sources)`：复用现有搜索服务（Tavily/知乎多源），返回格式化结果
    * `recall_notes(query)`：复用 `VectorRecall`，笔记本过滤参数从会话配置注入，结果受 `ai_card_recall_limit` 约束
  - [ ] SubTask 3.3: `agent.go`——模块内部组装 `openai.NewChatModel`（BaseURL/APIKey/Model 来自现有 AI 配置）+ `adk.NewChatModelAgent`（Instruction 复用现有 baseSystemPrompt + 技能 + 角色扮演，MaxIterations 设上限），`adk.NewRunner`；工具按需注册，未来新增工具在此追加
  - [ ] SubTask 3.4: 事件桥接——消费 `runner.Run(ctx, messages)`（历史复用 `truncateAIMessages` 滑动窗口）返回的 AgentEvent：文本块经 `Emit("ai:stream-chunk", ...)`；工具调用/结果经 `Emit("ai:tool-status", ...)`；结束经 `Emit("ai:stream-done", ...)`
  - [ ] SubTask 3.5: 取消支持——`Run` 接收带 cancel 的 ctx，停止按钮触发 cancel 终止循环

- [x] Task 4: app.go 注入依赖并薄封装 CallAIAgentStream
  - [ ] SubTask 4.1: `App` 结构体新增字段 `AgentSvc *agent.AgentService`，在初始化处构造注入（与其他服务一致）
  - [ ] SubTask 4.2: 新增 `CallAIAgentStream` 绑定方法：只做参数收集（sessionID、userText、skillIds、referencedNoteIDs、roleplayNoteIDs、followUp、uploadedFiles、userMsgID、recallNotebookIDs，不再接收搜索/召回/思考开关）→ 构造 `agent.Request` → 调用 `AgentSvc.Run(ctx, req, func(ev, data){ runtime.EventsEmit(a.ctx, ev, data) })`；**方法内不出现任何 Eino/Agent 实现细节**
  - [ ] SubTask 4.3: 复用现有消息保存逻辑（assistant 消息写入 `ai_messages`，工具调用摘要写入 `search_sources` 字段），会话摘要/历史刷新一致
  - [ ] SubTask 4.4: 模型不支持工具调用或调用失败时，经 `ClassifyError` 输出友好中文错误并发射错误事件

- [x] Task 5: 前端 Agent 模式 UI
  - [ ] SubTask 5.1: AI 对话操作栏新增"问答 / Agent"模式切换控件（会话级，样式参考现有主题体系），切换调用现有会话配置保存接口写入 `agent_enabled`
  - [ ] SubTask 5.2: Agent 模式下隐藏深度思考、联网搜索、卡片召回三个开关（CSS/JS 控制），技能、引用笔记、上传等保留
  - [ ] SubTask 5.3: 消息发送按会话配置选择 `CallAIAgentStream` 或现有 `CallAIStream`；新增 `ai:tool-status` 事件处理（展示工具调用过程，复用现有搜索状态样式）
  - [ ] SubTask 5.4: Agent 模式下"停止"按钮行为与现有链路一致

- [x] Task 6: 编译验证与回归
  - [ ] SubTask 6.1: `go build ./...`、`go vet ./...` 通过
  - [ ] SubTask 6.2: 前端构建/语法检查通过，问答模式（开关显示、CallAIStream 链路、搜索/召回/深度思考行为）回归无变化
  - [ ] SubTask 6.3: 手工冒烟（可选）：Agent 模式提问普通问题不触发工具；提问需搜索的问题触发 web_search 并展示过程

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 1]
- [Task 4] depends on [Task 3]
- [Task 5] depends on [Task 2]、[Task 4]
- [Task 6] depends on [Task 4]、[Task 5]
