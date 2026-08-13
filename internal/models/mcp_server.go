package models

import "time"

// MCPServer 表示一台外部 MCP 服务器配置（原 ~/.jot/mcp/mcp-servers.json 条目）
type MCPServer struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Name      string            `gorm:"uniqueIndex;size:100" json:"name"`         // 服务器唯一标识，用于工具名前缀 mcp_{name}_{tool}
	Transport string            `gorm:"size:10" json:"transport"`                 // stdio | sse | http
	Command   string            `gorm:"size:500" json:"command"`                  // stdio 可执行命令
	Args      []string          `gorm:"type:text;serializer:json" json:"args"`    // stdio 命令参数（GORM json 序列化存储）
	Env       map[string]string `gorm:"type:text;serializer:json" json:"env"`     // stdio 环境变量注入（GORM json 序列化存储）
	URL       string            `gorm:"size:1000" json:"url"`                     // sse / http 地址
	Headers   map[string]string `gorm:"type:text;serializer:json" json:"headers"` // sse / http 请求头注入（可选）
	Enabled   bool              `json:"enabled"`                                  // 是否启用（前端新增默认 true；编辑沿用原值）
	SortOrder int               `json:"sort_order"`                               // 展示/装配顺序
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
