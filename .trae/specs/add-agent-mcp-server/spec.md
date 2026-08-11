# Agent 模式接入外部 MCP 服务器（配置文件驱动·测试阶段）Spec

## Why

Agent 模式目前仅支持内置工具，无法调用外部 MCP 服务器（文件系统、数据库、第三方 API 等）扩展能力。当前处于测试阶段：**不做数据库建表、不做前端管理 UI**，通过读取当前目录下的 JSON 配置文件驱动 MCP 服务器连接与工具注册，先把后端核心的使用和调用逻辑跑通。

## What Changes

- 定义 MCP 服务器 JSON 配置文件格式，并提供示例文件 `mcp-servers.json`（放置于进程工作目录，即项目根目录）。
- 新增后端包 `internal/mcpserver`，封装三件事：
  - **配置解析**：读取/解析/校验 `mcp-servers.json`（stdio / sse / http 三种传输）。
  - **客户端连接**：基于 `mark3labs/mcp-go` 按传输类型构建客户端并完成握手（Start + Initialize）。
  - **工具发现与包装**：`GetTools` 将 MCP 服务器工具转为 eino `tool.BaseTool`，统一加 `mcp_{服务器名}_{工具名}` 前缀防命名冲突，并包装 ActionText 文案（"调用 {服务器} 的 {工具}"）供前端状态条展示。
- Agent 集成：`agent.AgentService.Run()` 启动时读取配置文件，为每个 `enabled` 服务器连接并发现工具，追加进现有工具注册表（`buildTools`），经现有 `tools.WrapWithError` 包装（失败回填模型不中断循环）。
- 依赖引入：`github.com/cloudwego/eino-ext/components/tool/mcp`（底层 `mark3labs/mcp-go`）。
- **不做**（测试阶段明确排除）：数据库建表（MCPServer 模型）、app.go 绑定方法、前端设置页管理 UI、配置文件在线编辑。

## Impact

- 受影响代码：
  - `go.mod` / `go.sum`：新增 eino-ext mcp 组件依赖
  - `internal/agent/agent.go`：Deps 增加配置路径字段，Run() 装配 MCP 工具
  - `internal/agent/registry.go`：buildTools 预留 MCP 工具接入点（或由 agent.go 统一追加）
  - 新增 `internal/mcpserver/` 包（config.go / client.go / tools.go）
  - 新增 `mcp-servers.json` 示例配置文件（项目根目录）
- 受影响能力：Agent 模式工具链（新增外部工具来源，原有 10 个内置工具行为不变）
- 风险点：eino-ext/components/tool/mcp 依赖的 eino 版本可能与当前 v0.9.13 冲突（Task 1 优先验证）

## ADDED Requirements

### Requirement: JSON 配置文件格式与解析

系统 SHALL 从进程工作目录读取 `mcp-servers.json` 定义 MCP 服务器列表。

配置格式定义如下：

```json
{
  "servers": [
    {
      "name": "math",
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-everything"],
      "env": {},
      "url": "",
      "enabled": true
    },
    {
      "name": "weather",
      "transport": "http",
      "url": "http://localhost:3000/mcp",
      "command": "",
      "args": [],
      "env": {},
      "enabled": true
    }
  ]
}
```

字段语义：
- `name`：服务器唯一标识，非空，用于工具名前缀 `mcp_{name}_{tool}`。
- `transport`：`stdio` | `sse` | `http`，必填。
- `command` / `args`：stdio 传输的可执行命令与参数（如 `npx -y @modelcontextprotocol/server-everything`）。
- `env`：可选，字符串键值对环境变量注入。
- `url`：sse / http 传输的服务器地址。
- `enabled`：是否启用，默认 false（安全考量）。

#### Scenario: 配置文件存在且合法
- **WHEN** 进程工作目录存在 `mcp-servers.json` 且至少一个服务器 `enabled=true`
- **THEN** 解析成功，启用的服务器参与本轮 Agent 工具装配

#### Scenario: 配置文件缺失
- **WHEN** 工作目录不存在 `mcp-servers.json`
- **THEN** 解析返回明确错误，Agent 按无 MCP 工具运行（行为与现状一致），不 panic

#### Scenario: 单条配置非法
- **WHEN** 某服务器 transport 非法 / name 为空 / stdio 缺 command / url 传输缺 url
- **THEN** 该服务器被跳过并记录日志，其余合法服务器正常装配

### Requirement: MCP 客户端连接（三传输 + 握手）

系统 SHALL 按传输类型构建 mcp-go 客户端并完成标准握手（`Start` + `Initialize`）。

- `stdio`：`client.NewStdioMCPClient(command, envSlice, args...)`，env 从配置 `env` map 构造。
- `sse`：`client.NewSSEMCPClient(url)`。
- `http`：`client.NewStreamableHTTPClient(url, opts...)`。
- 连接与握手全程受调用方传入的 `ctx` 控制（复用 Agent 取消上下文，停止按钮可中断）。

#### Scenario: 服务器不可达
- **WHEN** 连接超时或握手失败
- **THEN** 返回错误，调用方记录日志并跳过该服务器，不中断本轮 Agent

### Requirement: 工具发现、改名与包装

系统 SHALL 将每个启用的 MCP 服务器上发现到的工具注册为 eino 工具：

- 工具名统一改为 `mcp_{服务器名}_{工具名}`（如 `mcp_math_add`），避免与内置工具（`web_search` 等）或跨服务器重名冲突。
- 每个 MCP 工具经 `tools.WrapWithError` 包装：调用失败错误文本回填模型继续 ReAct 循环，并发射 `tool_error` 事件。
- 每个 MCP 工具提供 ActionText 文案（"调用 {服务器名} 的 {工具名}"），使前端 `ai:tool-status` 状态条显示友好文案。
- 工具发现（tools/list）结果按轮刷新（每轮 Run 重新连接与发现），保证工具列表与服务器实际状态一致。

#### Scenario: 工具调用成功
- **WHEN** 模型决定调用 `mcp_{name}_{tool}` 且服务器正常返回
- **THEN** 工具结果作为 Tool 消息回填模型，前端收到 `tool_start` / `tool_result` 事件

#### Scenario: 工具调用失败
- **WHEN** MCP 服务器调用返回错误
- **THEN** 经 WrapWithError 回填错误文本，模型可调整策略继续，前端收到 `tool_error` 事件

### Requirement: Agent 装配 MCP 工具

系统 SHALL 在 `agent.AgentService.Run()` 的工具装配阶段追加 MCP 工具：

- 读取配置路径（`Deps.MCPServerConfigPath`，默认 `"mcp-servers.json"`）。
- 仅装配 `enabled=true` 的服务器。
- 单个服务器连接/发现失败不中断其他服务器装配与整个 Agent 运行。
- 配置中无启用服务器时，Agent 行为与现状完全一致（零回归）。

## MODIFIED Requirements

### Requirement: Agent 工具注册表扩展

原有 `buildTools` 注册 10 个内置工具的行为不变；MCP 工具作为**追加来源**在 `Run()` 中并入 `toolList`，原有 `toolByName` 索引、ActionTextProvider 断言、WrapWithError 包装逻辑全部复用。

## REMOVED Requirements

无移除项（本轮为增量功能，不删除既有能力）。
