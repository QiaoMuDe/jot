// Package mcpserver 提供外部 MCP 服务器支持：配置文件解析、客户端连接与工具发现包装。
//
// 职责：
//   - 解析与校验 ~/.jot/mcp/mcp-servers.json（stdio / sse / http 三种传输）：
//     config.go 定义 Server / Config 结构与 Load / LoadDefault / EnabledServers。
//   - 基于 mark3labs/mcp-go 按传输类型构建客户端并完成握手（Start + Initialize）：
//     client.go 的 Connect。
//   - 基于 eino-ext mcp 组件将服务器工具转为 eino tool.BaseTool，
//     统一加 mcp_{服务器名}_{工具名} 前缀防命名冲突，并提供 ActionText 文案：
//     tools.go 的 Session / OpenSession。
//
// 依赖方向：internal/agent import 本包，本包 import internal/agent/tools（仅用其
// ActionTextProvider 接口），不形成循环依赖。
package mcpserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"jot/internal/config"
)

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

// DefaultConfigJSON 默认 MCP 服务器配置文件内容（空 servers 列表，按 MCP_CONFIG.md 规范填写后启用）。
const DefaultConfigJSON = `{
  "servers": []
}
`

// EnsureConfig 确保默认 MCP 配置目录（~/.jot/mcp/）与配置文件存在：
// 目录不存在时创建；配置文件不存在时写入 DefaultConfigJSON（已存在的文件不会被覆盖）。
func EnsureConfig() error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	return EnsureConfigFileAt(path)
}

// EnsureConfigFileAt 确保指定路径的 MCP 配置文件存在（含目录创建），供 EnsureConfig 及测试复用。
func EnsureConfigFileAt(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建 MCP 配置目录 %s 失败: %w", dir, err)
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查 MCP 配置文件 %s 失败: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(DefaultConfigJSON), 0644); err != nil {
		return fmt.Errorf("写入默认 MCP 配置文件 %s 失败: %w", path, err)
	}
	return nil
}

// Server 单台 MCP 服务器配置。
type Server struct {
	Name      string            `json:"name"`      // 服务器唯一标识，用于工具名前缀 mcp_{name}_{tool}
	Transport string            `json:"transport"` // stdio | sse | http
	Command   string            `json:"command"`   // stdio 传输的可执行命令
	Args      []string          `json:"args"`      // stdio 传输的命令参数
	Env       map[string]string `json:"env"`       // stdio 传输的环境变量注入（可选）
	URL       string            `json:"url"`       // sse / http 传输的服务器地址
	Enabled   bool              `json:"enabled"`   // 是否启用（默认 false，安全考量）
}

// Config MCP 服务器配置文件根结构。
type Config struct {
	Servers []Server `json:"servers"`
	// LoadErrors Load 时单条服务器校验失败的记录（该条已跳过，不影响其余合法条目装配）
	LoadErrors []error `json:"-"`
}

// Load 读取并校验 MCP 服务器配置文件。
// 整体性错误（文件缺失 / JSON 语法错误）返回 error；
// 单条服务器校验失败不中断整体加载：该条被跳过并记录到 Config.LoadErrors，
// 其余合法条目正常返回，由调用方依据 LoadErrors 逐条输出告警。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 MCP 服务器配置文件 %s 失败: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 MCP 服务器配置文件 %s 失败: %w", path, err)
	}
	valid := cfg.Servers[:0]
	for i := range cfg.Servers {
		if err := validate(&cfg.Servers[i]); err != nil {
			cfg.LoadErrors = append(cfg.LoadErrors, fmt.Errorf("server[%d]: %w", i, err))
			continue
		}
		valid = append(valid, cfg.Servers[i])
	}
	cfg.Servers = valid
	return &cfg, nil
}

// validate 校验单条服务器配置，失败返回含服务器名的错误。
func validate(s *Server) error {
	if s.Name == "" {
		return fmt.Errorf("MCP 服务器配置非法: name 不能为空")
	}
	switch s.Transport {
	case "stdio":
		if s.Command == "" {
			return fmt.Errorf("MCP 服务器 %s 配置非法: stdio 传输必须提供 command", s.Name)
		}
	case "sse", "http":
		if s.URL == "" {
			return fmt.Errorf("MCP 服务器 %s 配置非法: %s 传输必须提供 url", s.Name, s.Transport)
		}
	default:
		return fmt.Errorf("MCP 服务器 %s 配置非法: 不支持的 transport %q（支持 stdio / sse / http）", s.Name, s.Transport)
	}
	return nil
}

// EnabledServers 返回配置中启用（Enabled==true）的服务器列表。
func (c *Config) EnabledServers() []Server {
	if c == nil {
		return nil
	}
	var out []Server
	for _, s := range c.Servers {
		if s.Enabled {
			out = append(out, s)
		}
	}
	return out
}
