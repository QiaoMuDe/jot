# 重构：提取工具加载逻辑为独立函数

## 背景

当前 `Run()` 方法中，MCP 工具加载逻辑（\~110行代码）直接嵌入在 `agent.go` 的第 464-577 行，导致：

1. `generatePlan()` 无法复用 MCP 工具加载逻辑，只能看到内置工具
2. 代码可读性差，`Run()` 方法过长

## 当前状态

* **内置工具加载**: `buildTools()` in `registry.go` - 已独立，可复用

* **MCP 工具加载**: 嵌入在 `Run()` 中，不可复用

* **计划生成**: `generatePlan()` 只能用 `tools.BuiltinTools()` 获取内置工具元信息

## 重构方案

### 1. 新增 `loadMCPTools` 方法

**文件**: `internal/agent/registry.go`

提取 `Run()` 中第 464-577 行的 MCP 工具加载逻辑为独立函数：

```go
// loadMCPTools 从数据库加载 MCP 服务器配置，建立连接并获取工具。
// 返回工具列表（已过滤禁用工具）和工具元信息列表。
// 失败仅记录日志，不中断调用方。
func loadMCPTools(
    ctx context.Context,
    deps Deps,
    toolCtx *tools.Context,
    disabled map[string]bool,
) ([]tool.BaseTool, []tools.ToolMeta) {
    // ... 现有逻辑（从 Run() 中提取）
}
```

### 2. 新增 `buildToolMetas` 函数

**文件**: `internal/agent/registry.go`

从 `tool.BaseTool` 列表生成元信息列表（供 `generatePlan` 使用）：

```go
// buildToolMetas 从工具列表提取元信息（名称和描述）。
func buildToolMetas(ctx context.Context, toolList []tool.BaseTool) []tools.ToolMeta {
    metas := make([]tools.ToolMeta, 0, len(toolList))
    for _, t := range toolList {
        if info, err := t.Info(ctx); err == nil && info != nil {
            metas = append(metas, tools.ToolMeta{
                Name:  info.Name,
                Label: info.Desc,
            })
        }
    }
    return metas
}
```

### 3. 修改 `generatePlan` 函数

**文件**: `internal/agent/agent.go` (第 271-362 行)

* 移除 `disabledTools` 参数

* 新增 `toolMetas []tools.ToolMeta` 参数

* 移除内部的工具列表生成逻辑（第 278-294 行）

* 直接使用传入的 `toolMetas` 生成工具描述字符串

### 4. 修改 `Run()` 方法

**文件**: `internal/agent/agent.go` (第 458-595 行)

调整代码顺序：

1. 构建 `disabledTools` map（第 458-461 行，保持不变）
2. 加载内置工具：`buildTools()`（第 462 行，保持不变）
3. **新增**: 调用 `loadMCPTools()` 加载 MCP 工具（替代第 464-577 行）
4. 合并工具列表：`toolList = append(toolList, mcpTools...)`
5. 生成工具元信息：`toolMetas := buildToolMetas(runCtx, toolList)`
6. 调用 `generatePlan()`，传入 `toolMetas`（替代 `disabledTools`）
7. 构建 `toolByName` 索引（第 579-586 行，保持不变）
8. 注入 Agent Runner（第 591-627 行，保持不变）

## 涉及文件

| 文件                           | 修改内容                                          |
| ---------------------------- | --------------------------------------------- |
| `internal/agent/registry.go` | 新增 `loadMCPTools` 和 `buildToolMetas` 函数       |
| `internal/agent/agent.go`    | 修改 `generatePlan` 签名和实现；重构 `Run()` 方法中的工具加载流程 |

## 验证步骤

1. 编译通过：`go build ./internal/agent/...`
2. 运行测试：`go test ./internal/agent/...`
3. 功能验证：Plan 模式下，模型应能看到 MCP 工具（如有配置）

