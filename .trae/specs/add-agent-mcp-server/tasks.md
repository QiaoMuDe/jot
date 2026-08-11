# Tasks

## Task 1: 引入 eino-ext MCP 组件依赖并验证版本兼容性
- [x] 执行 `go get github.com/cloudwego/eino-ext/components/tool/mcp` 并 `go mod tidy`
- [x] 确认依赖解析后 eino 版本仍为 v0.9.13（或兼容版本），`go build ./...` 通过，不破坏现有 openai 模型组件与 adk.ChatModelAgent
- [x] 查看 `eino-ext/components/tool/mcp` 包源码，确认 `Config` 结构（Cli / ToolNameList）与 `GetTools` 签名，以及其依赖的 mcp-go 版本与客户端构造 API

## Task 2: 定义并解析 mcp-servers.json 配置文件
- [x] 新建 `internal/mcpserver/config.go`：定义 `Config`（servers 列表）与 `Server` 结构体（name / transport / command / args / env / url / enabled）
- [x] 实现 `Load(path string) (*Config, error)`：读取文件、JSON 解析、逐条校验（transport 枚举、name 非空、stdio 缺 command / url 传输缺 url 判定非法）
- [x] 提供默认配置路径常量 `DefaultConfigFile = "mcp-servers.json"`（相对进程工作目录解析）
- [x] 在项目根目录新增示例文件 `mcp-servers.json`（含 1 个 stdio 示例 + 1 个 http 示例，stdio 示例 enabled=true 便于验证）

## Task 3: 实现 MCP 客户端连接层（三传输 + 握手）
- [x] 新建 `internal/mcpserver/client.go`：实现 `Connect(ctx context.Context, s Server) (client.MCPClient, error)`
- [x] 按 transport 分发：stdio（`NewStdioMCPClient`，env map 转 slice）/ sse（`NewSSEMCPClient`）/ http（`NewStreamableHttpClient`）
- [x] 完成握手：`cli.Start(ctx)` + `cli.Initialize(ctx, mcp.InitializeRequest{...})`，ClientInfo 标识 jot
- [x] 连接失败/握手失败返回包装错误（含服务器名），供调用方跳过

## Task 4: 实现工具发现与包装
- [x] 新建 `internal/mcpserver/tools.go`：实现 `DiscoverTools(ctx, cli, serverName) ([]tool.BaseTool, error)`，内部调用 eino-ext 的 `GetTools(ctx, &mcpp.Config{Cli: cli})`
- [x] 实现改名包装器 `mcpToolWrapper`：`Info()` 返回 `mcp_{serverName}_{原始名}`，`InvokableRun` 委托内层
- [x] 让包装器实现 ActionTextProvider（文案 "调用 {serverName} 的 {原始工具名}"），保证经 `tools.WrapWithError` 包装后父包能断言到该文案
- [x] 导出便捷方法 `BuildTools(ctx, s Server) ([]tool.BaseTool, error)`：Connect → Discover → 改名包装，一步完成

## Task 5: Agent 装配 MCP 工具（核心接入）
- [x] `internal/agent/agent.go` 的 `Deps` 增加字段 `MCPServerConfigPath string`
- [x] `Run()` 工具装配阶段：读取配置（路径为空回退默认值），对每个 enabled 服务器调 `mcpserver.OpenSession`，结果经 `tools.WrapWithError` 追加进 `toolList`
- [x] 单个服务器失败：记录日志（用现有 `Deps.Logger`）后 continue，不中断其他服务器与整体运行
- [x] 配置文件读取失败：Debug 日志并继续（无 MCP 工具运行，零回归）
- [x] 确认 `toolByName` 索引与 `emitToolStart` 的 ActionText 断言对新 MCP 工具生效

## Task 6: 单元验证与构建验证
- [x] 新建 `internal/mcpserver/config_test.go`：覆盖合法配置解析、非法 transport / 缺 command / 缺 url 校验、文件缺失错误
- [x] 新建 `internal/mcpserver/tools_test.go`：用 mark3labs/mcp-go 的 `server.NewMCPServer` + `NewSSEServer` 起内存 MCP 服务器，验证 Connect 握手 → DiscoverTools 工具发现 → 工具名带 `mcp_{name}_` 前缀 → InvokableRun 真实调用返回正确结果
- [x] `go vet ./...` 与 `go build ./...` 通过，无新增编译告警
- [x] 确认未改动任何数据库模型 / app.go 绑定 / 前端代码（测试阶段边界）

# Task Dependencies
- [Task 2] 依赖 [Task 1]（需要 eino-ext 依赖就绪后确认 Config 结构）
- [Task 3] 依赖 [Task 1]
- [Task 4] 依赖 [Task 2] [Task 3]
- [Task 5] 依赖 [Task 4]
- [Task 6] 依赖 [Task 2] [Task 3] [Task 4] [Task 5]（构建验证依赖全部代码就绪；测试可在 Task 2/3/4 完成后先行编写）
