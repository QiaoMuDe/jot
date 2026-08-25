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

// ImportMCPServerItem 单条 MCP 服务器导入结果
// 用于 App.ImportMCPServers 返回每条处理结果，Wails 绑定需用 struct 传复杂数据
type ImportMCPServerItem struct {
	Index int    `json:"index"` // 1-based 条目序号
	Name  string `json:"name"`  // 服务器名(若可解析)
	OK    bool   `json:"ok"`    // 是否成功入库
	Error string `json:"error"` // 失败原因
}

// ParseMCPServersResult MCP 服务器导入的解析+校验结果(不入库)
// OK=false 时 Error 必填(整体解析失败或全部条目校验失败);Items 描述每条详情
type ParseMCPServersResult struct {
	OK    bool                  `json:"ok"`    // 整体是否可入库(所有条目都通过校验)
	Error string                `json:"error"` // 整体错误(空=无)
	Items []ImportMCPServerItem `json:"items"` // 每条校验结果(OK=true 表示通过字段校验)
}
