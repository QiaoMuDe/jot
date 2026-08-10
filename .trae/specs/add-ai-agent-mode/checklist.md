# Checklist

- [x] 根 `go.mod` 含 eino 与 eino-ext/openai 依赖，`go build ./...` 通过，现有 aicli（go-openai v1）编译无冲突
- [x] `AISessionConfig` / `SessionConfig` 均含 `agent_enabled` 字段，`GetSessionConfig` 能读出、`SaveSessionConfig` 能持久化，数据库迁移覆盖
- [x] `internal/agent` 包存在：`web_search`、`recall_notes` 两个读工具可被模型调用并返回格式化结果
- [x] Agent 逻辑全部位于 `internal/agent` 模块（`AgentService` + 工具注册 + 事件回调），`app.go` 仅注入依赖并薄封装 `CallAIAgentStream`，方法内无 Eino/Agent 实现细节
- [x] `CallAIAgentStream` 绑定方法存在，使用 Eino `adk.NewChatModelAgent` + `adk.NewRunner`，历史复用滑动窗口
- [x] Agent 模式普通问题不触发工具调用，直接流式回答
- [x] Agent 模式需搜索/召回的问题触发对应工具，回答带来源/召回信息
- [x] 工具调用过程通过 `ai:tool-status` 事件推送到前端并展示
- [x] 前端操作栏有"问答 / Agent"切换控件，切换持久化到会话配置
- [x] Agent 模式下深度思考/联网搜索/卡片召回三开关隐藏，技能/引用/上传保留
- [x] 切回问答模式后三开关恢复显示，`CallAIStream` 行为与改造前一致
- [x] Agent 模式回答写入 `ai_messages`，工具调用摘要写入 `search_sources`
- [x] Agent 模式停止按钮可取消，错误经 `ClassifyError` 转友好中文提示
- [x] 主项目其余功能（搜索、召回、深度思考、会话管理等）回归无异常
