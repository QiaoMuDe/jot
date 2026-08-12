# MCP 配置迁移 + `~/.jot` 统一路径配置：`./mcp-servers.json` → `~/.jot/mcp/mcp-servers.json`

## 摘要

1. MCP 服务器配置文件读取位置从进程工作目录（项目根）的 `mcp-servers.json` 改为 `~/.jot/mcp/mcp-servers.json`，**不迁移旧配置**（用户已确认）。
2. 新建统一路径配置包 `internal/config`，集中定义 `~/.jot` 根目录与子目录解析，所有散落的 `filepath.Join(home, ".jot", ...)`（数据库、日志、图片、备份、MCP）改为引用它（用户要求）。

## 当前状态分析

`~/.jot` 路径当前散落在多处硬编码：

| 文件 | 行号 | 用途 |
| --- | --- | --- |
| `internal/database/db.go` | L24、L93 | `data/jot.db`、`backup` |
| `app.go` | L133、L3337、L3916 | `logs` |
| `app.go` | L210、L373、L418、L460、L510 | `images` |
| `main.go` | L57 | `images` |
| `internal/mcpserver/config.go` | L22-23 | `DefaultConfigFile = "mcp-servers.json"`（相对工作目录） |

其它关键点：
- `internal/agent/agent.go` L119-126：`Deps.MCPServerConfigPath` 为空时回退 `mcpserver.DefaultConfigFile`；`app.go` 两处构造 Deps（L187、L3931）均未设置该字段 → 实际走回退路径。
- `internal/mcpserver/config_test.go` 全部用例通过显式临时路径调用 `Load(path)`，不受影响。
- 前端无任何 MCP 引用，无需改动。
- `main.go` 的 `os`、`filepath` 仅在 L56-57 使用，收敛后将变为未使用 import，需一并清理。

## 改动方案

### 1. 新建 `internal/config/config.go`（统一路径配置包）

```go
// Package config 提供 Jot 应用在用户家目录下的统一根目录（~/.jot）路径解析。
// 所有读写 ~/.jot 下文件的模块都应通过本包获取路径，避免硬编码散落各处。
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ~/.jot 下子目录名常量
const (
	DirData   = "data"   // 数据库目录
	DirBackup = "backup" // 备份目录
	DirImages = "images" // 图片目录
	DirLogs   = "logs"   // 日志目录
	DirMCP    = "mcp"    // MCP 配置目录
)

// JotHomeDir 返回应用根目录: ~/.jot
func JotHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户家目录失败: %w", err)
	}
	return filepath.Join(home, ".jot"), nil
}

// SubDir 返回应用根目录下的子目录路径，如 SubDir(DirMCP) -> ~/.jot/mcp
func SubDir(sub string) (string, error) {
	root, err := JotHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, sub), nil
}
```

同包新增 `internal/config/config_test.go`：验证 `SubDir(DirMCP)` 返回 `JotHomeDir()/mcp`。

### 2. `internal/mcpserver/config.go` — 默认路径改到家目录

- 删除 `DefaultConfigFile` 常量，新增（需 import `path/filepath`、`jot/internal/config`）：

```go
// DefaultConfigPath 返回默认 MCP 服务器配置文件路径: ~/.jot/mcp/mcp-servers.json
func DefaultConfigPath() (string, error) {
	dir, err := config.SubDir(config.DirMCP)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcp-servers.json"), nil
}

// LoadDefault 读取默认路径（~/.jot/mcp/mcp-servers.json）的 MCP 服务器配置。
func LoadDefault() (*Config, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return Load(path)
}
```

- 更新包注释 L4 中 `mcp-servers.json` 的表述。

### 3. `internal/agent/agent.go` — 回退逻辑改用 `LoadDefault()`

L119-126 改为（`err` 已在 Run 上文声明）：

```go
mcpPath := s.deps.MCPServerConfigPath
var mcpCfg *mcpserver.Config
if mcpPath == "" {
    // 默认读取用户家目录 ~/.jot/mcp/mcp-servers.json（可由 Deps.MCPServerConfigPath 覆盖）
    mcpCfg, err = mcpserver.LoadDefault()
} else {
    mcpCfg, err = mcpserver.Load(mcpPath)
}
if err != nil {
    // 原有 Debug 跳过分支保持不变（含路径解析失败的兜底）
}
```

`MCPServerConfigPath` 注入覆盖机制保留；`app.go` 两处 Deps 构造无需改动。

### 4. `internal/database/db.go` — 引用统一配置

- `DefaultDBPath()`（L18-25）改为 `config.SubDir(config.DirData)` + `jot.db`。
- `BackupDir()`（L87-94）改为 `return config.SubDir(config.DirBackup)`。
- import `jot/internal/config`；`os`/`filepath`/`fmt` 在 InitDB 中仍被使用，保留。

### 5. `app.go` — 图片/日志目录引用统一配置

import 增加 `jot/internal/config`。逐处替换：

| 位置 | 原代码 | 改为 |
| --- | --- | --- |
| L133 | `logDir := filepath.Join(home, ".jot", "logs")` | `logDir, err := config.SubDir(config.DirLogs)`（结合上文 err 作用域，见实现） |
| L207-210 | `home, _ := os.UserHomeDir()` + `filepath.Join(home, ".jot", "images")` | `imageDir, _ := config.SubDir(config.DirImages)`（保留忽略错误语义） |
| L366-373 | `home, err := os.UserHomeDir()` + join | `imageDir, err := config.SubDir(config.DirImages)` |
| L414-418 | 同上 | 同上 |
| L455-459 | 同上 | 同上 |
| L504-510 | `imageDirPath()` 主体 | `return config.SubDir(config.DirImages)` |
| L3332-3337 | `homeDir, err := os.UserHomeDir()` + join | `dir, err := config.SubDir(config.DirLogs)` + `logDir = dir` |
| L3914-3915 | `home, _ := os.UserHomeDir()` + join | `logDir, err := config.SubDir(config.DirLogs)` |

### 6. `main.go` — 图片目录引用统一配置

- L55-57：`home, _ := os.UserHomeDir(); imageDir := filepath.Join(...)` → `imageDir, err := config.SubDir(config.DirImages)`（L59 `err :=` 改为 `err =`）。
- 移除不再使用的 `os`、`path/filepath` import，新增 `jot/internal/config`。

### 7. 文档同步

- `internal/mcpserver/MCP_CONFIG.md`：第 1 节文件位置、第 6 节日志示例、第 9 节排查表（L129）中的路径改为 `~/.jot/mcp/mcp-servers.json`。
- `internal/agent/doc.go` L24-25：`mcp-servers.json` → `~/.jot/mcp/mcp-servers.json`。

## 不做的事

- ❌ 不迁移项目根 `internal/mcpserver/mcp-servers.json`（用户已确认），保留原地作示例。
- ❌ 不新增前端改动（前端无 MCP 引用）。
- ❌ 不改 `MCPServerConfigPath` 注入覆盖机制。

## 验证

1. `go build ./...` 编译通过。
2. `go test ./internal/config/... ./internal/mcpserver/...` 通过。
3. 手动验证：`~/.jot/mcp/mcp-servers.json` 放置启用服务器配置 → Agent 对话出现 `MCP 服务器工具已上线`；删除文件 → `Debug MCP 服务器配置不可用`，内置工具不受影响。
