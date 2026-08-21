// Package mcpserver 提供外部 MCP 服务器支持：配置读取（数据库）、客户端连接与工具发现包装。
//
// 职责：
//   - 从数据库读取与校验 MCP 服务器配置（stdio / sse / http 三种传输）：
//     config.go 定义 Server / Config 结构与 LoadFromDB。
//   - 基于 modelcontextprotocol/go-sdk 按传输类型构建客户端并完成握手
//     （协议版本自动协商，含降级到 2024-11-05）：
//     client.go 的 Connect。
//   - 基于 go-sdk 的 ListTools / CallTool 将服务器工具转为 eino tool.BaseTool，
//     统一加 mcp_{服务器名}_{工具名} 前缀防命名冲突，并提供 ActionText 文案：
//     tools.go 的 Session / OpenSession。
//
// 依赖方向：internal/agent import 本包，本包 import internal/agent/tools（仅用其
// ActionTextProvider 接口），不形成循环依赖。
package mcpserver

import (
	"fmt"

	"gorm.io/gorm"
	"jot/internal/models"
)

// Server 单台 MCP 服务器配置。
type Server struct {
	Name      string            `json:"name"`      // 服务器唯一标识，用于工具名前缀 mcp_{name}_{tool}
	Transport string            `json:"transport"` // stdio | sse | http
	Command   string            `json:"command"`   // stdio 传输的可执行命令
	Args      []string          `json:"args"`      // stdio 传输的命令参数
	Env       map[string]string `json:"env"`       // stdio 传输的环境变量注入（可选）
	URL       string            `json:"url"`       // sse / http 传输的服务器地址
	Headers   map[string]string `json:"headers"`   // sse / http 传输的请求头注入（可选）
	Enabled   bool              `json:"enabled"`   // 是否启用（默认 false，安全考量）
}

// Config MCP 服务器配置根结构。
type Config struct {
	Servers []Server `json:"servers"`
	// LoadErrors LoadFromDB 时单条服务器校验失败的记录（该条已跳过，不影响其余合法条目装配）
	LoadErrors []error `json:"-"`
}

// LoadFromDB 从数据库读取全部 MCP 服务器并校验：非法条目跳过并记录到 Config.LoadErrors，
// 其余合法条目正常返回，由调用方依据 LoadErrors 逐条输出告警（与原文件版 Load 行为一致）。
// 查询失败（含空库无表等整体性错误）返回 error；空库返回空 Servers 且 err 为 nil。
func LoadFromDB(db *gorm.DB) (*Config, error) {
	var records []models.MCPServer
	if err := db.Order("sort_order asc, id asc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("读取 MCP 服务器配置失败: %w", err)
	}
	cfg := &Config{Servers: make([]Server, 0, len(records))}
	for i := range records {
		rec := records[i]
		server := Server{
			Name:      rec.Name,
			Transport: rec.Transport,
			Command:   rec.Command,
			Args:      rec.Args,
			Env:       rec.Env,
			URL:       rec.URL,
			Headers:   rec.Headers,
			Enabled:   rec.Enabled,
		}
		if err := validate(&server); err != nil {
			cfg.LoadErrors = append(cfg.LoadErrors, fmt.Errorf("server[%d]: %w", i, err))
			continue
		}
		cfg.Servers = append(cfg.Servers, server)
	}
	return cfg, nil
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
