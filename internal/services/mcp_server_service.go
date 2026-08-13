package services

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"jot/internal/models"
)

// MCPServerService 封装外部 MCP 服务器配置的增删查业务逻辑。
// 注意：本服务内的校验规则与 internal/mcpserver 包的 validate 保持一致（复制实现），
// 因 mcpserver → agent/tools → services 已形成依赖链，services 不得反向 import mcpserver 以免循环依赖。
type MCPServerService struct {
	db *gorm.DB
}

// NewMCPServerService 创建一个新的 MCPServerService 实例
func NewMCPServerService(db *gorm.DB) *MCPServerService {
	return &MCPServerService{db: db}
}

// List 按 sort_order, id 升序返回全部 MCP 服务器
func (s *MCPServerService) List() ([]models.MCPServer, error) {
	var servers []models.MCPServer
	if err := s.db.Order("sort_order asc, id asc").Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// Get 按 ID 查询单条 MCP 服务器记录；不存在返回 gorm.ErrRecordNotFound
func (s *MCPServerService) Get(id uint) (*models.MCPServer, error) {
	var server models.MCPServer
	if err := s.db.First(&server, id).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

// Save 新增（ID==0）或更新 MCP 服务器；先做业务校验，校验失败返回中文错误（可直接展示给前端）。
// 校验规则与 mcpserver 包 validate 一致：Name 非空、Transport 合法、
// stdio 必须有 Command / sse、http 必须有 URL、Name 全库唯一。
func (s *MCPServerService) Save(server *models.MCPServer) error {
	if server == nil {
		return fmt.Errorf("MCP 服务器配置不能为空")
	}
	// 名称必填
	if server.Name == "" {
		return fmt.Errorf("MCP 服务器名称不能为空")
	}
	// 传输类型与关键字段校验
	switch server.Transport {
	case "stdio":
		if server.Command == "" {
			return fmt.Errorf("MCP 服务器 %s 配置非法: stdio 传输必须提供 command", server.Name)
		}
	case "sse", "http":
		if server.URL == "" {
			return fmt.Errorf("MCP 服务器 %s 配置非法: %s 传输必须提供 url", server.Name, server.Transport)
		}
	default:
		return fmt.Errorf("MCP 服务器 %s 配置非法: 不支持的 transport %q（支持 stdio / sse / http）", server.Name, server.Transport)
	}
	// 名称不能含空白：名称直接拼入工具名前缀 mcp_{name}_{tool}，空白会破坏工具名
	if strings.ContainsAny(server.Name, " \t\r\n") {
		return fmt.Errorf("MCP 服务器名称不能包含空格等空白字符")
	}
	// Env / Headers 的 KEY 不能含空白或等号（等号是 KEY=VALUE 的分隔符）
	for key := range server.Env {
		if strings.ContainsAny(key, " \t\r\n=") {
			return fmt.Errorf("环境变量 KEY「%s」不能包含空白或等号", key)
		}
	}
	for key := range server.Headers {
		if strings.ContainsAny(key, " \t\r\n=") {
			return fmt.Errorf("请求头 KEY「%s」不能包含空白或等号", key)
		}
	}
	// 按传输类型清零非相关字段，避免切换传输方式后旧字段残留（数据保持干净）
	switch server.Transport {
	case "stdio":
		server.URL = ""
		server.Headers = nil
	case "sse", "http":
		server.Command = ""
		server.Args = nil
		server.Env = nil
	}
	// 名称唯一（排除自身，更新场景允许沿用原名称）
	var count int64
	if err := s.db.Model(&models.MCPServer{}).Where("name = ? AND id != ?", server.Name, server.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("名称 %s 已存在", server.Name)
	}
	// ID==0 新增，否则更新
	if server.ID == 0 {
		return s.db.Create(server).Error
	}
	// Omit("created_at")：避免 Save 全字段更新把 created_at 覆写为零值（前端表单不带该字段）。
	// 新增路径（ID==0 走 Create）时 GORM 仍会为 AutoCreateTime 字段自动填充当前时间，不受 Omit 影响。
	return s.db.Omit("created_at").Save(server).Error
}

// Delete 按 ID 删除
func (s *MCPServerService) Delete(id uint) error {
	return s.db.Delete(&models.MCPServer{}, id).Error
}
