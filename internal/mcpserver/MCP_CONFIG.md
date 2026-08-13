# Jot MCP 服务器配置（数据库存储）说明

> 本文件定义 Jot Agent 模式下外部 MCP 服务器的配置模型与存储方式。
> 对应实现：`internal/models/mcp_server.go`（数据模型）、`internal/services/mcp_server_service.go`（业务服务）、
> `internal/mcpserver/config.go`（读取与校验）、`internal/agent/agent.go`（工具装配）。

## 1. 存储位置

MCP 服务器配置记录存储于应用 SQLite 数据库的 **`mcp_servers` 表**（GORM 模型 `models.MCPServer`，已注册进 `database.AllModels`，随建表/迁移自动维护），不再使用任何配置文件。

| 字段 | 类型 | 说明 |
|------|------|------|
| `ID` | uint | 主键 |
| `Name` | string | 服务器唯一标识（数据库唯一索引） |
| `Transport` | string | `stdio` / `sse` / `http` |
| `Command` | string | stdio 传输的可执行命令 |
| `Args` | []string | stdio 传输的命令参数（JSON 序列化存储） |
| `Env` | map[string]string | stdio 传输注入的环境变量（JSON 序列化存储） |
| `URL` | string | sse / http 传输的服务器地址 |
| `Headers` | map[string]string | sse / http 传输注入的请求头（JSON 序列化存储，可选） |
| `Enabled` | bool | 是否启用，默认 `false`（安全考量） |
| `SortOrder` | int | 展示/装配顺序（升序） |
| `CreatedAt` / `UpdatedAt` | time.Time | 创建 / 更新时间戳 |

## 2. 管理方式

配置通过 `services.MCPServerService`（`List` / `Save` / `Delete`）读写，Wails 绑定方法暴露给前端：

- `GetMCPServers()`：按 `sort_order, id` 升序返回全部记录；
- `SaveMCPServer(server)`：新增（ID==0）或更新，校验失败返回中文错误；
- `DeleteMCPServer(id)`：按 ID 删除。

**设置页图形化 CRUD UI 在下一轮实现**，当前可通过绑定方法直接管理记录。

应用每次发起 Agent 对话时都会从数据库重新读取并连接启用的服务器（`mcpserver.LoadFromDB`），**修改配置后无需重启应用**，下一次对话即生效。每台服务器的连接与握手有 **10 秒超时**（`mcpserver.ConnectTimeout`），超时或连接失败仅跳过该服务器，不阻塞其他服务器与整轮对话。

## 3. 字段语义与校验规则

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 服务器唯一标识。用于工具名前缀 `mcp_{name}_{tool}`，不能为空 |
| `transport` | string | ✅ | 传输方式，取值 `stdio` / `sse` / `http` |
| `command` | string | 仅 stdio | stdio 传输的可执行命令（如 `npx`、`node`、本地 exe 路径） |
| `args` | string[] | 仅 stdio | stdio 传输的命令参数（可选） |
| `env` | object | 可选 | stdio 传输注入的环境变量（可选），`{"KEY": "value"}` |
| `url` | string | 仅 sse/http | sse / http 传输的服务器地址 |
| `headers` | object | 可选 | sse / http 传输注入的请求头（可选，如鉴权 Token），`{"Authorization": "Bearer xxx"}` |
| `enabled` | boolean | 可选 | 是否启用，默认 `false`（安全考量，未设置则视为停用） |

单条配置满足以下条件才视为合法：

- `name` 非空；
- `transport` 必须是 `stdio` / `sse` / `http` 之一；
- `stdio` 必须提供 `command`；
- `sse` / `http` 必须提供 `url`；
- `name` 全库唯一（更新时排除自身）。

校验失败会拒绝保存（`SaveMCPServer` 返回中文错误）；`mcpserver.LoadFromDB` 读取时对非法条目跳过并记录到 `Config.LoadErrors`，由调用方逐条告警。

## 4. 三种传输的配置示例

### 4.1 stdio（本地子进程，最常用）

| 字段 | 示例值 |
|------|--------|
| `name` | `filesystem` |
| `transport` | `stdio` |
| `command` | `npx` |
| `args` | `["-y", "@modelcontextprotocol/server-filesystem", "D:/docs"]` |
| `env` | `{"FOO": "bar"}` |
| `enabled` | `true` |

> 也可直接使用本地已编译的服务器程序：`command` 填 `playground/mcp-math/mcp-math.exe`，`args` 留空。

### 4.2 SSE（HTTP Server-Sent Events）

| 字段 | 示例值 |
|------|--------|
| `name` | `weather` |
| `transport` | `sse` |
| `url` | `http://127.0.0.1:8000/sse` |
| `enabled` | `true` |

### 4.3 Streamable HTTP（新版标准传输）

| 字段 | 示例值 |
|------|--------|
| `name` | `remote-api` |
| `transport` | `http` |
| `url` | `http://localhost:3000/mcp` |
| `enabled` | `true` |

## 5. 失败行为（重要）

- **单条配置非法**：仅跳过该条服务器，**其余合法服务器正常装配**。每次对话装配时逐条输出 `Warn` 告警，错误信息带索引定位，例如：
  ```
  MCP 服务器配置校验失败，该服务器已跳过  error="server[1]: MCP 服务器 配置非法: name 不能为空"
  ```
- **单台服务器连接失败**（命令不存在、地址不可达、握手失败）：仅跳过该服务器，日志 `Warn`，不影响其他服务器与内置工具。

## 6. 工具命名与装配行为

- 每个启用的服务器在连接握手后自动发现其全部工具，注册为 Agent 可调用工具。
- 工具名统一为 `mcp_{服务器名}_{工具名}`，例如 `math` 服务器的 `add` 工具 → `mcp_math_add`。
- 一台服务器可暴露多个工具（一个聚合入口、一组工具）。
- 服务器装配成功后输出 `Info` 日志：
  ```
  MCP 服务器工具已上线  server=math  count=3  tools=mcp_math_add, mcp_math_multiply, mcp_math_sqrt
  ```
- 所有 MCP 工具与内置工具同样遵循失败回填机制：调用失败会作为 `tool_error` 反馈给模型，不中断 Agent 循环。

## 7. 安全注意事项

- `enabled` 默认 `false`，服务器需显式开启才会被连接。
- `stdio` 会启动**任意可执行命令**，`sse/http` 可执行远程服务器上的任意工具操作，请仅配置可信来源。
- 环境变量 `env` 中的密钥明文存储在数据库中，注意库文件（`~/.jot/data/jot.db`）的访问权限。
- 启用的服务器工具**全量注入**模型上下文，工具过多会显著增加 token 消耗，建议只保留需要使用的服务器。

## 8. 常见问题排查

| 现象 | 排查方向 |
|------|----------|
| 日志无任何 MCP 记录 | `mcp_servers` 表为空或查询失败（`Debug` 日志） |
| `Warn 配置校验失败` | 按日志中的 `server[索引]` 与字段关键字修正对应记录 |
| `Warn 连接失败` | 命令是否可执行 / 路径是否正确 / 地址是否可达 / 服务器是否实现 MCP 握手 |
| 工具未出现在对话中 | 确认 `enabled` 为 true；确认服务器确实暴露了工具（`Info` 上线日志的 `tools` 字段）；模型是否主动调用（工具名已注入，必要时在提示词中引导） |
| 工具名冲突 | 内置工具与 MCP 工具不会冲突（MCP 统一加 `mcp_` 前缀）；`name` 重复的记录会被唯一索引拒绝 |
