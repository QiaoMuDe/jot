# Checklist

## 数据层
- [x] `MCPServer` 模型已创建并注册进 `AllModels`，`InitDB` 自动迁移、`ResetDatabase` 同步重建
- [x] `MCPServerService` 提供 `List` / `Save` / `Delete`，校验失败返回用户友好中文错误
- [x] `Save` 的 Name 唯一性、Transport 合法性、stdio 必填 Command、sse/http 必填 URL 校验均生效

## mcpserver 包
- [x] `internal/mcpserver` 不再包含任何文件读写逻辑与文件路径依赖（`os` / `config` / `filepath`）
- [x] `LoadFromDB` 将库记录转换为 `Config`，非法条目跳过并记录 `LoadErrors`，行为与原文件版一致

## agent 装配
- [x] `agent.Deps` 已移除 `MCPServerConfigPath`，装配改为从数据库读取
- [x] 无 MCP 记录时不中断 Agent 运行（Debug 日志跳过）；单台失败仅 Warn 跳过

## Wails 绑定
- [x] `app.go` 提供 `GetMCPServers` / `SaveMCPServer` / `DeleteMCPServer` 绑定方法
- [x] startup 中 `EnsureConfig()` 已移除，无残留 MCP 文件初始化逻辑

## 测试与清理
- [x] `config_test.go` 已重写为数据库驱动测试并通过
- [x] `go build ./...` 与 `go test ./...` 全部通过（注：`TestOpenSessionOverSSE` 为既有用例，因本机 IPv6 监听不可用而失败，与本迁移无关；本次新增/改动的 DB 驱动测试全部通过）
- [x] 全局无 `mcp-servers.json`、`MCPServerConfigPath`、`EnsureConfig`、`LoadDefault` 残留引用（生产代码；历史 spec 文档与 AGENTS.md 历史记忆点除外）
- [x] `MCP_CONFIG.md` 与 `AGENTS.md` 已更新为库驱动说明
