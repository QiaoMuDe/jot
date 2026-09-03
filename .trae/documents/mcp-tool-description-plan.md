# Agent 工具设置：从 MCP 服务器获取工具描述

## 概述

设置页"对话与搜索"中 Agent 工具列表的 MCP 工具描述，目前使用硬编码拼接 `"{服务器名} 的 {工具名}"`。改为优先从 MCP 服务器获取原始描述，不可用时才 fallback 到兜底文案。

## 现状分析

### 数据流

```
MCP 服务器 ListTools → mcp.Tool {Name, Description, ...}
                           ↓
mcpTool.Info() → {Name: "mcp_{server}_{tool}", Desc: toolDef.Description}
                           ↓
ListToolMetas() → SessionToolMeta {ServerName, FullName}
                           ↓  (Description 丢失)
GetAgentTools() → 硬编码拼接 "{ServerName} 的 {toolName}" 作为 Label
```

### 关键文件

| 文件                            | 行号        | 说明                                                     |
| ----------------------------- | --------- | ------------------------------------------------------ |
| `internal/mcpserver/pool.go`  | L39-L43   | `SessionToolMeta` 结构体 — 缺少 `Description` 字段            |
| `internal/mcpserver/pool.go`  | L332-L355 | `ListToolMetas()` — 未提取 `info.Desc`                    |
| `internal/mcpserver/tools.go` | L172-L189 | `mcpTool.Info()` 已返回 `Desc`（来自 `mcp.Tool.Description`） |
| `app.go`                      | L2578     | 硬编码拼接 `"{ServerName} 的 {toolName}"`                    |
| `internal/agent/types.go`     | L32-L37   | `ToolMeta` 结构体 — `Label` 字段用于前端展示                      |

### 可用信息

* `mcp.Tool.Description` 是 MCP 协议标准字段，由每个 MCP 服务器自行提供

* 类型为 `string`，位于 `github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/protocol.go:1910`

* 内容通常是英文，描述工具的实际用途（如 `"Read a file from the filesystem"`）

## 修改方案

### 1. `internal/mcpserver/pool.go` — `SessionToolMeta` 增加 `Description` 字段

**文件**: `internal/mcpserver/pool.go` L39-L42

**改动**: 在 `SessionToolMeta` 结构体中增加 `Description string` 字段。

**原因**: 从 MCP 工具的 `Info()` 可以获取 `Desc`，但当前结构体没有字段承载这个信息，导致在 `ListToolMetas()` 中丢弃。

### 2. `internal/mcpserver/pool.go` — `ListToolMetas()` 提取描述

**文件**: `internal/mcpserver/pool.go` L348-L351

**改动**: 在 `ListToolMetas()` 中，从 `info.Desc` 提取描述填入 `SessionToolMeta.Description`。

**原因**: `t.Info(ctx)` 已返回完整的 `ToolInfo`（含 `Desc`），只需赋值即可。

### 3. `app.go` — `GetAgentTools()` 两段式构造 Label

**文件**: `app.go` L2578

**当前代码**:

```go
label := mt.ServerName + " 的 " + strings.TrimPrefix(mt.FullName, "mcp_"+mt.ServerName+"_")
```

**改为**:

```go
label := mt.ServerName + " 的 " + strings.TrimPrefix(mt.FullName, "mcp_"+mt.ServerName+"_")
if mt.Description != "" {
    // 截断：取前 N 个字符，过长时截断 + "..."
    // 如果描述本身较短则直接使用
    desc := strings.TrimSpace(mt.Description)
    if len([]rune(desc)) > maxDescLen {
        desc = string([]rune(desc)[:maxDescLen]) + "..."
    }
    label = desc
}
```

**截断策略**:

* 使用 `maxDescLen = 40`（40 个字符，中英文均适用）

* 用 `[]rune` 处理，避免中文字符被截断成乱码

* 超长时截断并追加 `"..."`，不超长时直接使用

**两段式逻辑**:

1. 优先：`mt.Description` 非空且截断后 → 作为 Label
2. 兜底：`mt.Description` 为空时 → 使用当前的 `"{ServerName} 的 {toolName}"` 拼接

## 验证步骤

1. 确认 `go build ./...` 编译通过
2. 启动应用，打开设置页"对话与搜索" → Agent 工具管理面板
3. 确认已预热 MCP 服务器的工具显示原始描述（截断后），而非"XX 的 XX"
4. 确认未提供描述的 MCP 工具仍显示"XX 的 XX"兜底文案
5. 确认内置工具（非 MCP）显示不受影响

