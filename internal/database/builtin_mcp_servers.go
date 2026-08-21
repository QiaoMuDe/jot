package database

import (
	"jot/internal/models"

	"gorm.io/gorm"
)

// InitBuiltinMCPServers 增量插入内置 MCP 服务器（仅插入 Name 不存在的）
// 用户可在 builtinMCPServers 切片中按相同格式逐条添加更多服务器。
//
// 注意：
//   - 内置服务器默认 Enabled=false：多为含密钥占位符的模板，用户需在设置页
//     「MCP 服务器」填入真实 API Key / Access Secret 并启用后，才会被 Agent 装配连接；
//   - 项目未实现 ${ENV} 环境变量展开，请求头/参数中的占位符需直接替换为真实值；
//   - 知乎三服务（zhihu_search / zhihu_global / zhihu_hot）为 MCP over SSE（Bearer 鉴权），
//     填入知乎 Access Secret 后即可使用（端点见知乎开发者文档）。
func InitBuiltinMCPServers(db *gorm.DB) error {
	// 查询所有已有服务器名称
	var existingNames []string
	db.Model(&models.MCPServer{}).Pluck("name", &existingNames)
	existing := make(map[string]bool, len(existingNames))
	for _, n := range existingNames {
		existing[n] = true
	}

	builtinMCPServers := []models.MCPServer{
		// Tavily 官方 MCP（HTTP，API Key 通过 query 参数传递；占位符 <your-api-key> 需替换为真实 Key）
		{
			Name:      "tavily",
			Transport: "http",
			URL:       "https://mcp.tavily.com/mcp/?tavilyApiKey=<your-api-key>",
			Enabled:   false,
			SortOrder: 1,
		},
		// AnySearch MCP（HTTP + 请求头 Bearer 认证；占位符 <your-api-key> 需替换为真实 Key）
		{
			Name:      "anysearch",
			Transport: "http",
			URL:       "https://api.anysearch.com/mcp",
			Headers:   map[string]string{"Authorization": "Bearer <your-api-key>"},
			Enabled:   false,
			SortOrder: 2,
		},
		// 知乎 MCP 系列（MCP over SSE，Bearer 鉴权；占位符 <your_access_secret> 需替换为知乎 Access Secret）
		{
			Name:      "zhihu_search",
			Transport: "sse",
			URL:       "https://developer.zhihu.com/api/mcp/zhihu_search/v1/sse",
			Headers:   map[string]string{"Authorization": "Bearer <your_access_secret>"},
			Enabled:   false,
			SortOrder: 3,
		},
		{
			Name:      "zhihu_global",
			Transport: "sse",
			URL:       "https://developer.zhihu.com/api/mcp/global_search/v1/sse",
			Headers:   map[string]string{"Authorization": "Bearer <your_access_secret>"},
			Enabled:   false,
			SortOrder: 4,
		},
		{
			Name:      "zhihu_hot",
			Transport: "sse",
			URL:       "https://developer.zhihu.com/api/mcp/hot_list/v1/sse",
			Headers:   map[string]string{"Authorization": "Bearer <your_access_secret>"},
			Enabled:   false,
			SortOrder: 5,
		},
		// ↓ 用户可在下面继续添加更多内置 MCP 服务器 ↓
	}

	var toInsert []models.MCPServer
	for _, s := range builtinMCPServers {
		if !existing[s.Name] {
			toInsert = append(toInsert, s)
		}
	}
	if len(toInsert) == 0 {
		return nil
	}
	return db.Create(&toInsert).Error
}
