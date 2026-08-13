# Tasks

- [x] Task 1: 新增 MCPServer 数据模型并注册
  - [x] 1.1 新建 `internal/models/mcp_server.go`：`MCPServer` 结构（Name 唯一索引、Transport、Command、Args/Env 为 JSON 字符串 TEXT、URL、Enabled、SortOrder、时间戳）
  - [x] 1.2 在 `internal/database/models.go` 的 `AllModels` 注册 `&models.MCPServer{}`
  - [x] 验证：`go build ./...` 通过

- [x] Task 2: 新增 MCPServerService
  - [x] 2.1 新建 `internal/services/mcp_server_service.go`：`MCPServerService{db}`、`NewMCPServerService`
  - [x] 2.2 实现 `List`（按 SortOrder, ID 排序）、`Save`（Name 唯一校验 + transport 校验 + 必填字段校验，复用 mcpserver `validate` 规则）、`Delete`
  - [x] 2.3 校验失败错误输出用户友好中文（直接返回中文 fmt 错误，可展示给前端）
  - [x] 验证：`go build ./...` 通过

- [x] Task 3: 重构 internal/mcpserver 包为数据库读取
  - [x] 3.1 `config.go` 删除文件相关公开函数与常量（`Load` / `LoadDefault` / `EnsureConfig` / `EnsureConfigFileAt` / `DefaultConfigPath` / `DefaultConfigJSON`）及 `os` / `config` / `filepath` 依赖
  - [x] 3.2 新增 `LoadFromDB(db *gorm.DB) (*Config, error)`：库记录 → `Config.Servers`，逐条校验，非法条目记录 `LoadErrors` 并跳过（行为与原文件版一致）
  - [x] 3.3 保留 `Server` / `Config` 结构与 `validate`
  - [x] 验证：`go build ./...` 通过

- [x] Task 4: 改造 agent 装配来源
  - [x] 4.1 `internal/agent/agent.go` `Deps` 移除 `MCPServerConfigPath`，新增数据库来源注入（`MCPServerDB *gorm.DB`）
  - [x] 4.2 `Run()` 装配处改为调用 `mcpserver.LoadFromDB`，无记录/失败时 Debug 日志跳过，单条非法 Warn（行为保持一致）
  - [x] 4.3 `app.go` `NewAgentService` 调用处同步注入新依赖
  - [x] 验证：`go build ./...` 通过

- [x] Task 5: app.go Wails 绑定方法
  - [x] 5.1 移除 startup 中 `mcpserver.EnsureConfig()` 调用及 `internal/mcpserver` import
  - [x] 5.2 新增 `GetMCPServers() []models.MCPServer`、`SaveMCPServer(server models.MCPServer) error`、`DeleteMCPServer(id uint) error`（错误信息为用户友好中文）
  - [x] 5.3 `App` 结构注入 `mcpServerService`（`NewApp` 装配 `NewMCPServerService(db)`）
  - [x] 验证：`go build ./...` 通过

- [x] Task 6: 测试重写与适配
  - [x] 6.1 重写 `internal/mcpserver/config_test.go` 为数据库驱动测试（内存 sqlite，覆盖：空库、多台服务器、非法条目跳过并记 LoadErrors）
  - [x] 6.2 检查 `client_test.go` / `tools_test.go` / `client_internal_test.go` / `tools_internal_test.go` 是否依赖文件路径或 `Deps.MCPServerConfigPath`，如有则适配为 DB 来源（确认无依赖，未改动）
  - [x] 6.3 为 `MCPServerService` 补充单元测试（List/Save 校验/Delete）
  - [x] 验证：`go test ./internal/mcpserver/... ./internal/services/... ./internal/agent/...` 全部通过（注：`TestOpenSessionOverSSE` 为本机 IPv6 环境问题，与本迁移无关）

- [x] Task 7: 清理与文档
  - [x] 7.1 删除示例文件 `internal/mcpserver/mcp-servers.json`
  - [x] 7.2 检查 `internal/config/config.go` 中 `DirMCP` 是否仍有引用，无引用则删除该常量（已删除，`config_test.go` 改用 `DirData`）
  - [x] 7.3 改写 `internal/mcpserver/MCP_CONFIG.md` 为"数据库存储 + 后端绑定 API"说明（前端设置页操作说明留待下一轮）
  - [x] 7.4 更新 `AGENTS.md` 记忆点：MCP 配置来源 文件驱动 → 库驱动，记录新增模型/Service/绑定（追加记忆点 10，按维护规范重编号 1-10）
  - [x] 验证：`go build ./...` 与 `go test ./...` 全绿（除既有 IPv6 环境用例）；全局搜索确认无 `mcp-servers.json`、`MCPServerConfigPath`、`EnsureConfig`、`LoadDefault` 残留引用

# Task Dependencies

- Task 2 依赖 Task 1（模型先注册）
- Task 3 依赖 Task 1（LoadFromDB 读取模型）
- Task 4 依赖 Task 3（装配调用 LoadFromDB）
- Task 5 依赖 Task 2（绑定调用 MCPServerService）与 Task 4（app.go 注入处同步改动）
- Task 6 依赖 Task 3 / Task 4 / Task 5
- Task 7 依赖 Task 5（清理与文档需在代码迁移完成后进行）
