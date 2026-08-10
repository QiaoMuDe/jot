# Tasks

- [x] Task 1: 创建 `agent-demo/` 独立模块骨架
  - [x] SubTask 1.1: 初始化独立 go.mod（`module agent-demo`，`go 1.26`），并 `go get github.com/cloudwego/eino@latest`、`go get github.com/cloudwego/eino-ext/components/model/openai@latest`
  - [x] SubTask 1.2: 添加 `main.go` 占位入口，验证 `go build ./...` 通过

- [x] Task 2: 实现三要素配置加载
  - [x] SubTask 2.1: 命令行 flag（`-base-url`、`-api-key`、`-model`）
  - [x] SubTask 2.2: 环境变量兜底（`AGENT_DEMO_BASE_URL`、`AGENT_DEMO_API_KEY`、`AGENT_DEMO_MODEL`），flag 优先于 env
  - [x] SubTask 2.3: 启动时打印最终生效的配置摘要（API key 脱敏）

- [x] Task 3: 实现 Agent ReAct 循环
  - [x] SubTask 3.1: 用 `openai.NewChatModel` 构建 ChatModel（读配置，BaseURL 指向兼容端点）
  - [x] SubTask 3.2: 实现自定义工具 `tool.BaseTool`（如 `get_current_time(city string)` 或 `web_search(query string)` 假实现，返回确定性结果），供模型调用
  - [x] SubTask 3.3: 用 `adk.NewChatModelAgent` 组装（Instruction + ToolsConfig），`adk.NewRunner` + `runner.Query` 消费事件流
  - [x] SubTask 3.4: 终端输出：工具调用名称/参数、工具返回结果、最终回答的流式文本

- [x] Task 4: 运行入口与文档
  - [x] SubTask 4.1: 支持命令行传入问题（如 `go run . "现在几点了"`），无参数时进入交互模式或输出使用说明
  - [x] SubTask 4.2: 编写 `agent-demo/README.md`：配置方式（env/flag）、运行示例、注意事项（模型需支持 function calling）

# Task Dependencies
- Task 2 依赖 Task 1（骨架就绪后才能加配置）
- Task 3 依赖 Task 2（Agent 组装需要配置）
- Task 4 依赖 Task 3（入口与文档基于可运行 demo）
