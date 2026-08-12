# MCP 连接超时控制 + 装配逻辑优化 + 日志补全

## Summary

针对 `agent.go` 中 MCP 服务器装配链路做三件事：

1. **核心修复**：单服务器连接无独立超时 → 每台服务器连接包 `context.WithTimeout`（10s），超时走现有「连接失败跳过」分支，防止远程服务器不可达时阻塞整轮对话。
2. **三个可选小优化**：for 循环内 defer 改为统一收集关闭；MCP 工具 `Info` 重复 JSON deepcopy 改为缓存；`LoadDefault` 失败日志 path 回填。
3. **日志补全**：梳理 MCP 链路中静默点，补充可观测日志（无启用服务器、连接耗时、关闭失败、单工具跳过）。

不引入连接缓存/并行连接等过度设计，保持现有「每轮重连、失败跳过、mcpserver 包不依赖 logger」的分层。

## Current State Analysis

* `agent.Run` 每轮对话装配 MCP 工具（[agent.go L118-182](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go#L118-L182)）：读配置 → 对每个 enabled 服务器 `OpenSession`（Connect 握手 + GetTools + 改名包装）→ 并入 `toolList`。

* 调用方 ctx 为 `context.WithCancel`（[app.go L2432](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2432)），**无超时**，只能用户手动停止。

* `Connect`（[client.go L15-54](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/mcpserver/client.go#L15-L54)）stdio/sse/http 的 Start + Initialize 全部直接使用调用方 ctx；服务器**串行**连接。

* for 循环内 `defer sess.Close()`（[agent.go L155](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go#L155)）功能正确（每次迭代 sess 为新变量）但可读性差。

* `mcpTool.Info`（[tools.go L76-90](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/mcpserver/tools.go#L76-L90)）每次调用都执行一次 `deepCopyToolInfo`（JSON 序列化+反序列化）；agent.go 中 L168（取名）与 L186（建 toolByName 索引）各触发一次。

* `LoadDefault` 失败时（家目录解析失败），[agent.go L132](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go#L132) 日志 `path=""`，排查不便。

* 日志缺口：无 enabled 服务器时无日志；单工具 Info 解析失败在 [tools.go L41-44](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/mcpserver/tools.go#L41-L44) 静默跳过；连接耗时不可观测；统一关闭失败被忽略（`_ = sess.Close()`）。

* agent.go 已 import `time`；mcpserver 包现有 `config_test.go`、`tools_test.go`（含 `TestOpenSessionOverSSE` 全链路测试）。

## Proposed Changes

### 1. 单服务器连接超时 — `internal/mcpserver/client.go`

* 新增导出常量：`const ConnectTimeout = 10 * time.Second`（供文档与日志引用）。

* `Connect` 开头：`connCtx, cancel := context.WithTimeout(ctx, ConnectTimeout); defer cancel()`，函数内 Start / Initialize 全部改用 `connCtx`。

* 错误增强：`errors.Is(err, context.DeadlineExceeded)` 时返回 `MCP 服务器 %s 连接超时（10s 内未完成握手）: %w`，否则保持现有通用文案。为此抽一个内部函数 `wrapConnectError(name string, err error) error` 便于单测。

* imports 增加 `errors`、`time`。

**为什么**：sse/http 远程不可达（DNS/TCP/TLS 挂起）时若无超时，串行装配会无限阻塞后续所有服务器与整轮对话；超时后复用现有「连接失败跳过」分支，语义不变。

### 2. defer 改统一关闭 — `internal/agent/agent.go`

* 装配循环前声明 `var mcpSessions []*mcpserver.Session`。

* 循环内去掉 `defer func() { _ = sess.Close() }()`，改为 `mcpSessions = append(mcpSessions, sess)`。

* 循环结束后注册一个 defer：遍历 `mcpSessions` 调用 `Close()`，失败记 `Warnw("MCP 会话关闭失败", server, error)`（注册位置在循环后、后续 return 之前，覆盖其余 return 路径）。

**为什么**：可读性 + 提前 return 时的确定性关闭 + 补上「关闭失败」日志。

### 3. MCP 工具 Info 缓存 — `internal/mcpserver/tools.go`

* `mcpTool` 增加字段 `cachedInfo *schema.ToolInfo`。

* `Info()` 首次调用时构建改名信息并缓存；后续直接使用缓存。为避免返回同一指针被调用方修改造成污染，返回时浅拷贝 struct（`copied := *m.cachedInfo; return &copied, nil`）——浅拷贝复制标量与 Name 字符串，保留指针字段只读语义，调用方改返回值的 Name 不影响缓存。

* 注释注明：工具定义在本轮会话内不变，缓存消除 agent.go 与 eino 框架多次调用 Info 时的重复 JSON deepcopy。

**为什么**：agent.go L168/L186 及框架内部多次调用 Info，每次触发一次 JSON round-trip；缓存后仅首次有开销。

### 4. LoadDefault 失败日志 path 回填 — `internal/agent/agent.go` L121-134

* `mcpPath` 为空且 `LoadDefault` 失败时，先用 `mcpserver.DefaultConfigPath()` 回填 `mcpPath`（再失败则回填字面量 `~/.jot/mcp/mcp-servers.json`），保证失败日志 path 可读。

### 5. 日志补全 — `internal/agent/agent.go` + `internal/mcpserver/tools.go`

* **无 enabled 服务器**：配置加载成功但 `EnabledServers()` 为空 → `Debugw("MCP 配置无启用的服务器")`。

* **连接耗时**：循环内 `OpenSession` 前后 `time.Since`；成功路径在「工具已上线」Info 日志附 `fastlog.Int64("duration_ms", ...)`；失败 Warn 同样附带耗时，便于定位慢服务器。

* **单工具跳过统计**：`Session` 增加字段 `Skipped int`（OpenSession 内 `t.Info` 失败/为空/名字为空时累加）；agent.go 服务器循环后若 `sess.Skipped > 0` → `Warnw("N 个工具因 Info 解析失败被跳过", server, count)`。不引入 logger 到 mcpserver（保持现有分层，agent.go 负责日志）。

### 6. 文档 — `internal/mcpserver/MCP_CONFIG.md`

* 补充一句：应用对每台服务器连接与握手有 10s 超时，超时自动跳过该服务器（不阻塞其他服务器与对话）。

### 7. 测试

* 新建 `internal/mcpserver/client_test.go`：

  * `TestWrapConnectError`：`context.DeadlineExceeded` → 断言含「连接超时」与服务器名；普通错误 → 断言含「连接失败」与服务器名。

  * `TestConnectWithCanceledCtx`：已取消的 ctx 传 `Connect`（stdio，不存在的命令）→ 返回错误且含服务器名（验证错误包装路径）。

* `internal/mcpserver/tools_test.go` 增加 `TestMCPToolInfoCached`：

  * 对同一 `mcpTool` 连续调 `Info` 两次，断言返回 Name 均为改名后名称且内容一致；

  * 修改第一次返回值的 Name，断言第二次返回值不受影响（验证浅拷贝防污染）。

## Assumptions & Decisions

* 超时固定 10s（用户建议值），导出常量便于后续调整；不做可配置化（避免过度设计）。

* 不引入连接缓存/长连接复用、不做多服务器并行连接（与「每轮重连、改配置即时生效」设计一致，现阶段串行简单有序）。

* `mcpserver` 包保持无 logger 依赖，日志统一在 `agent.go` 输出。

* `mcpTool.Info` 缓存返回浅拷贝而非共享指针，防止调用方修改污染。

## Verification

1. `go build ./...` — 编译通过。
2. `go test ./internal/mcpserver/...` — 新旧用例全部通过（含 `TestOpenSessionOverSSE`、新增 `TestWrapConnectError`、`TestConnectWithCanceledCtx`、`TestMCPToolInfoCached`）。
3. `go test ./internal/agent/...` — agent 包既有用例通过。
4. 人工核对：确认 `mcpSessions` 统一关闭 defer 覆盖后续所有 return 路径。

