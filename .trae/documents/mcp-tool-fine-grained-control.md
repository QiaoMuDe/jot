# MCP 服务器工具精细化控制

## 概述

为 MCP 服务器提供工具级别的启用/禁用开关，让用户可以精细化控制 MCP 服务器暴露的单个工具是否注册到 Agent 工具列表，而非目前的"服务器一启用，所有工具全量注册"。

## 当前状态分析

### 后端

- `MCPServer` 模型（[models/mcp_server.go](file:///d:/峡谷/Dev/本地项目/jot/internal/models/mcp_server.go)）只有 `Enabled` 字段，控制整台服务器，无工具级控制
- `GetAgentTools()`（[app.go#L2242](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2242)）只返回内置工具，不包含 MCP 工具
- `agent.go` 的 MCP 工具装配循环（[agent.go#L427-L442](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go#L427-L442)）遍历 `r.sess.Tools` 全部追加，不做过滤
- `disabledTools` map 已在 `agent.go` 第 340-343 行构建，但 MCP 装配处未使用
- 内置工具的禁用机制已存在：通过 `ai_agent_tools_disabled` 设置键 + `buildTools()` 过滤

### 前端

- `agentToolsMeta` 列表（[main.js#L9053](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9053)）只包含内置工具
- `renderAgentToolsMgrList()`（[main.js#L9130](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9130)）和 `createAgentToolRow()`（[main.js#L9184](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9184)）是通用的，可渲染任意 `ToolMeta` 条目
- MCP 服务器列表（`mcpServers`，[main.js#L9237](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9237)）只显示服务器级信息，不包含工具列表
- `WarmupResult.ToolTotal` 仅在通知中展示，不持久化到设置页

### 核心依赖方向

```
internal/agent → internal/mcpserver
internal/agent → internal/agent/tools
internal/mcpserver → internal/agent/tools（仅 ActionTextProvider 接口）
app.go → internal/agent + internal/mcpserver + internal/agent/tools
```

`mcpserver` 包不能导入 `internal/agent`（否则循环依赖），但可以导入 `internal/agent/tools`。

## 设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 禁用名单存储位置 | 复用 `ai_agent_tools_disabled` 设置键 | 不改 schema，与内置工具机制一致 |
| MCP 工具名格式 | 现有 `mcp_{serverName}_{toolName}` | 已在用，无需改动 |
| 工具列表获取方式 | 从 MCP 池（`Pool`）读取已预热会话的快照 | 零额外网络开销，设置页秒开 |
| 未预热时的行为 | 不显示 MCP 工具，只显示内置工具 | 无需阻塞等待，自然降级 |
| 前端分组 | 不加分组，MCP 工具直接混入内置工具列表 | 前端渲染逻辑完全通用，零改动 |
| MCP 工具 Label | `"{serverName} 的 {originalName}"` | 与 ActionText 文案风格一致 |

## 变更清单

### 1. `internal/mcpserver/pool.go` - 新增 `ListToolMetas()`

新增 `SessionToolMeta` 结构体和 `ListToolMetas()` 方法，暴露池中已预热服务器的工具信息。

```go
// SessionToolMeta 单条 MCP 工具元信息（供设置页展示与开关控制）
type SessionToolMeta struct {
    ServerName string // 服务器名，用于前端分组
    FullName   string // 完整工具名，格式 mcp_{serverName}_{toolName}
}

// ListToolMetas 返回池中所有已预热会话的工具元信息。
// 未预热的服务器不在此列；调用方（GetAgentTools）据此展示，不阻塞等待。
func (p *Pool) ListToolMetas() []SessionToolMeta {
    p.mu.Lock()
    defer p.mu.Unlock()
    var metas []SessionToolMeta
    for _, entry := range p.entries {
        if entry.sess == nil {
            continue
        }
        for _, t := range entry.sess.Tools {
            info, err := t.Info(context.Background())
            if err != nil || info == nil {
                continue
            }
            metas = append(metas, SessionToolMeta{
                ServerName: entry.sess.ServerName,
                FullName:   info.Name,
            })
        }
    }
    return metas
}
```

### 2. `app.go` - 扩展 `GetAgentTools()` 返回 MCP 工具

在现有内置工具列表后追加 MCP 工具。MCP 工具的 `Enabled` 状态同样由 `disabledTools` 决定。

关键位置：[app.go#L2242-L2258](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2242-L2258)

```go
func (a *App) GetAgentTools() []agent.ToolMeta {
    // 1. 读取禁用名单（不变）
    // 2. 获取内置工具（不变）
    // 3. 追加 MCP 工具（新增）
    if a.mcpPool != nil {
        mcpTools := a.mcpPool.ListToolMetas()
        for _, mt := range mcpTools {
            label := mt.ServerName + " 的 " + strings.TrimPrefix(mt.FullName, "mcp_"+mt.ServerName+"_")
            result = append(result, agent.ToolMeta{
                Name:    mt.FullName,
                Label:   label,
                Enabled: !disabledSet[mt.FullName],
            })
        }
    }
    return result
}
```

### 3. `internal/agent/agent.go` - MCP 装配时增加过滤

在 MCP 工具装配循环中，对已禁用的工具跳过注册。

关键位置：[agent.go#L427-L442](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go#L427-L442)

在 `toolNames = append(toolNames, mcpToolName)` 之前增加：

```go
// 检查是否被禁用
if disabledTools[mcpToolName] {
    continue
}
```

**注意**：`disabledTools` map 已在第 340-343 行构建，本处直接使用。

### 4. `frontend/src/main.js` - 预热后刷新工具列表

在 `warmupMCPServers()` 完成后，重新调用 `GetAgentTools()` 刷新 `agentToolsMeta`，使工具开关列表中的 MCP 工具显示出来。

```js
// warmupMCPServers 函数末尾追加
async function warmupMCPServers() {
    // ... 现有逻辑不变 ...
    // 预热完成后刷新 Agent 工具列表（MCP 工具可能已变化）
    await refreshAgentToolsMeta();
}

// 新增：重新加载 Agent 工具元信息
async function refreshAgentToolsMeta() {
    try {
        agentToolsMeta = (await window.go.main.App.GetAgentTools()) || [];
    } catch (e) {
        agentToolsMeta = [];
    }
    updateAgentToolsButtonText();
    // 如果工具管理面板已展开，重新渲染
    if (agentToolsMgrExpanded) {
        renderAgentToolsMgrList();
    }
}
```

同时在 `loadAllSettings()` 中已有的 `GetAgentTools()` 调用保持不变（设置页首次打开时自动获取）。

### 5. 无需改动的文件

| 文件 | 理由 |
|------|------|
| `internal/models/mcp_server.go` | 不改 schema |
| `internal/mcpserver/config.go` | 不改配置结构 |
| `internal/mcpserver/tools.go` | `mcpTool.Info()` 已提供 `FullName` |
| `internal/services/mcp_server_service.go` | 不改 CRUD 逻辑 |
| `internal/agent/registry.go` | `buildTools()` 已用 `disabledTools` 过滤 |
| `internal/agent/tools/meta.go` | `BuiltinTools()` 只返回内置工具，不变 |
| `frontend/src/main.js` 的 `renderAgentToolsMgrList()` | 通用渲染，无需改动 |
| `frontend/src/main.js` 的 `createAgentToolRow()` | 通用渲染，无需改动 |
| `frontend/index.html` | 无需改 HTML 结构 |
| `frontend/src/css/components/settings-panel.css` | 无需改样式 |

## 数据流完整时序

### 设置页打开（已预热）

```
loadAllSettings()
  → GetAgentTools()
    → 读取 ai_agent_tools_disabled → ["mcp_filesystem_write"]
    → 内置工具: [summarize_text, ...] (14个)
    → mcpPool.ListToolMetas() → [{ServerName: "filesystem", FullName: "mcp_filesystem_read"}, {ServerName: "filesystem", FullName: "mcp_filesystem_write"}, ...]
    → 返回: [内置工具14个 + MCP工具N个]，其中 mcp_filesystem_write.Enabled=false
  → frontend 渲染: "已启用 17/18"
```

### 用户开关 MCP 工具

```
用户取消勾选 mcp_filesystem_read
  → agentToolsDisabled.push("mcp_filesystem_read")
  → saveSettings() → ai_agent_tools_disabled = ["mcp_filesystem_write", "mcp_filesystem_read"]
  → updateAgentToolsButtonText() → "已启用 16/18"
```

### 对话时工具装配

```
agent.Run()
  → DisabledTools = ["mcp_filesystem_write", "mcp_filesystem_read"]
  → buildTools() → 过滤内置工具
  → 连接 MCP 池 → 遍历 r.sess.Tools
  → 遇到 mcp_filesystem_read → disabledTools["mcp_filesystem_read"] = true → continue
  → 只注册未禁用的 MCP 工具
```

## 验证步骤

1. 启动应用，打开设置页 → Agent 工具开关列表应显示内置工具 + 已预热 MCP 服务器的工具
2. 禁用某个 MCP 工具 → 保存 → 按钮文字数字更新
3. 在 AI 对话中发起 Agent 模式 → 被禁用的 MCP 工具不应被模型调用
4. 在设置页中启用/停用 MCP 服务器 → 预热后 Agent 工具列表自动刷新
5. 新增 MCP 服务器 → 预热后 Agent 工具列表出现该服务器的工具
6. 删除 MCP 服务器 → 预热后 Agent 工具列表移除该服务器的工具