# MCP 服务器配置迁移至 SQLite（后端）Spec

## Why

MCP 服务器当前通过 `~/.jot/mcp/mcp-servers.json` 配置文件驱动（每次对话重新读取、手工编辑 JSON），无法通过页面管理，对普通用户不友好。本轮将配置来源从配置文件迁移到 SQLite 库，完成后端数据模型、业务逻辑与 Wails 绑定方法；前端设置页 CRUD 在下一轮实现。

## What Changes

- 新增 `models.MCPServer` 数据表并注册进 `AllModels`（AutoMigrate / ResetDatabase 自动同步）
- 新增 `services.MCPServerService`：`List` / `Save`（新增+更新）/ `Delete`，校验规则复用 mcpserver 现有 `validate`
- 重构 `internal/mcpserver` 包：删除全部文件读写逻辑（`Load` / `LoadDefault` / `EnsureConfig` / `EnsureConfigFileAt` / `DefaultConfigPath` / `DefaultConfigJSON`），新增 `LoadFromDB(db)`；保留 `Server` / `Config` 结构与 `validate`
- `internal/agent/agent.go`：`Deps` 移除 `MCPServerConfigPath`，注入数据库来源；`Run()` 装配处改为从库读取
- `app.go`：移除 startup 中 `EnsureConfig()`；新增 Wails 绑定 `GetMCPServers` / `SaveMCPServer` / `DeleteMCPServer`
- 重写 `config_test.go` 为数据库驱动测试，删除依赖文件路径的用例；适配受影响的其它测试
- 清理与文档：删除示例 `internal/mcpserver/mcp-servers.json`，检查并清理 `config.DirMCP` 残留引用，更新 `MCP_CONFIG.md` 与 `AGENTS.md`

**BREAKING**：`~/.jot/mcp/mcp-servers.json` 不再被读取；`agent.Deps.MCPServerConfigPath` 移除；`mcpserver` 包公开的文件相关函数删除。旧文件内容不做自动导入，需用户在设置页重新录入（下一轮提供 UI）。

## Impact

- Affected specs：`add-agent-mcp-server`（原配置文件驱动能力）
- Affected code：
  - `internal/models/mcp_server.go`（新增）、`internal/database/models.go`（注册）
  - `internal/services/mcp_server_service.go`（新增）
  - `internal/mcpserver/config.go`（重构）、`internal/mcpserver/client.go` / `tools.go`（不动）
  - `internal/agent/agent.go`（Deps + 装配）
  - `app.go`（绑定 + 初始化）
  - `internal/mcpserver/config_test.go`（重写）
  - `internal/config/config.go`（`DirMCP` 若残留则清理）

## ADDED Requirements

### Requirement: MCPServer 数据模型

系统 SHALL 提供 `MCPServer` 表（`internal/models/mcp_server.go`），字段：`Name`（唯一）、`Transport`（stdio/sse/http）、`Command`、`Args`（JSON 数组字符串）、`Env`（JSON 对象字符串）、`URL`、`Enabled`、`SortOrder`、`CreatedAt`、`UpdatedAt`；并注册进 `database.AllModels`，保证 `InitDB` 自动迁移、`ResetDatabase` 同步重建。

#### Scenario: 迁移建表
- **WHEN** 应用启动执行 `InitDB`
- **THEN** 自动创建 `mcp_servers` 表；恢复出厂重置时该表同步清空重建

### Requirement: MCPServerService CRUD

系统 SHALL 提供 `services.MCPServerService`：
- `List() ([]models.MCPServer, error)`：按 `SortOrder`、`ID` 排序返回全部服务器
- `Save(server *models.MCPServer) error`：`Name` 非空且唯一校验、`Transport` 合法、`stdio` 必须有 `Command`、`sse`/`http` 必须有 `URL`（复用 mcpserver `validate` 规则）；校验失败返回经 `ClassifyError` 翻译的用户友好中文错误；存在则更新、不存在则创建
- `Delete(id uint) error`：按 ID 删除

#### Scenario: 新增/更新服务器
- **WHEN** 调用 `Save` 且 `Name` 唯一、字段满足校验规则
- **THEN** 记录写入或更新成功

#### Scenario: 校验失败
- **WHEN** `Name` 为空、`Transport` 非法、或缺少对应传输必填字段
- **THEN** 返回中文错误信息，不写入数据库

### Requirement: Agent 装配从数据库读取

系统 SHALL 让 `agent.Run()` 每次对话从数据库查询启用状态的 MCP 服务器并装配工具；查询/装配失败仅记录日志跳过 MCP 装配，不中断内置工具与整轮对话。

#### Scenario: 无 MCP 服务器记录
- **WHEN** 数据库无任何 MCP 服务器记录
- **THEN** Agent 正常使用内置工具，日志记录 Debug 级跳过信息

#### Scenario: 单台服务器装配失败
- **WHEN** 某台启用服务器连接/握手失败
- **THEN** 仅跳过该台，输出 Warn 日志，其余服务器与内置工具正常装配

### Requirement: Wails 绑定方法

系统 SHALL 在 `app.go` 提供前端可调用的绑定：
- `GetMCPServers() []models.MCPServer`：返回全部服务器
- `SaveMCPServer(server models.MCPServer) error`：新增或更新（错误为用户友好中文）
- `DeleteMCPServer(id uint) error`：删除指定服务器

## MODIFIED Requirements

### Requirement: mcpserver 包重构

`internal/mcpserver/config.go` 从"文件解析"改为"数据库读取"：新增 `LoadFromDB(db *gorm.DB) (*Config, error)`，将库中记录转换为 `Config{Servers}` 并逐条校验（非法条目跳过并记录 `LoadErrors`，行为与原文件版一致）；`Server` / `Config` 结构与 `validate` 保留。删除所有文件相关公开函数与常量。

### Requirement: agent Deps 配置来源

`internal/agent.Deps` 移除 `MCPServerConfigPath string` 字段，新增数据库访问来源（如 `MCPServerStore` 接口或 `*gorm.DB`），装配处由"读文件"改为"查库"。

## REMOVED Requirements

### Requirement: 配置文件驱动 MCP 服务器

**Reason**：MCP 服务器需通过设置页页面管理（下一轮前端），配置文件无法支撑页面 CRUD，且手动编辑 JSON 门槛高、易错。
**Migration**：不再读取 `~/.jot/mcp/mcp-servers.json`，不自动导入旧文件；用户在设置页（下一轮）重新配置。示例文件 `internal/mcpserver/mcp-servers.json` 删除。

### Requirement: Deps.MCPServerConfigPath 测试覆盖机制

**Reason**：配置来源统一为数据库，文件路径概念彻底移除。
**Migration**：受影响的测试重写为数据库驱动（内存/临时 sqlite）。
