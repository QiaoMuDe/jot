package mcpserver_test

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"jot/internal/mcpserver"
	"jot/internal/models"
)

// newTestDB 打开内存 SQLite（单连接）并迁移 mcp_servers 表，随后插入种子数据。
func newTestDB(t *testing.T, seeds ...models.MCPServer) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1) // 内存库必须单连接，否则各连接库相互独立
	if err := db.AutoMigrate(&models.MCPServer{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	for _, s := range seeds {
		if err := db.Create(&s).Error; err != nil {
			t.Fatalf("插入种子数据失败: %v", err)
		}
	}
	return db
}

// TestLoadFromDBEmpty 验证空库（表存在但无记录）返回空 Servers 且 err 为 nil。
func TestLoadFromDBEmpty(t *testing.T) {
	db := newTestDB(t)

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 空库不应报错: %v", err)
	}
	if cfg == nil || len(cfg.Servers) != 0 {
		t.Fatalf("空库 Servers 应为空, got %+v", cfg)
	}
	if len(cfg.LoadErrors) != 0 {
		t.Fatalf("空库 LoadErrors 应为空, got %v", cfg.LoadErrors)
	}
}

// TestLoadFromDBValidRecords 验证 stdio + sse + http 三种传输的合法记录可正常读取，字段值正确。
func TestLoadFromDBValidRecords(t *testing.T) {
	db := newTestDB(t,
		models.MCPServer{Name: "math", Transport: "stdio", Command: "npx",
			Args: []string{"-y", "@modelcontextprotocol/server-math"}, Env: map[string]string{"FOO": "1"}, Enabled: true, SortOrder: 1},
		models.MCPServer{Name: "weather", Transport: "sse", URL: "http://127.0.0.1:8000/sse", Enabled: true, SortOrder: 2},
		models.MCPServer{Name: "files", Transport: "http", URL: "http://127.0.0.1:8080/mcp", Enabled: false, SortOrder: 3},
	)

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 意外错误: %v", err)
	}
	if len(cfg.Servers) != 3 {
		t.Fatalf("服务器数量 = %d, want 3", len(cfg.Servers))
	}
	if len(cfg.LoadErrors) != 0 {
		t.Fatalf("合法记录不应有 LoadErrors, got %v", cfg.LoadErrors)
	}

	// stdio 服务器
	s0 := cfg.Servers[0]
	if s0.Name != "math" || s0.Transport != "stdio" || s0.Command != "npx" {
		t.Errorf("stdio 服务器字段错误: %+v", s0)
	}
	if len(s0.Args) != 2 || s0.Args[0] != "-y" || s0.Args[1] != "@modelcontextprotocol/server-math" {
		t.Errorf("stdio 服务器 args 错误: %v", s0.Args)
	}
	if s0.Env["FOO"] != "1" {
		t.Errorf("stdio 服务器 env 错误: %v", s0.Env)
	}
	if !s0.Enabled {
		t.Errorf("stdio 服务器 enabled 应为 true")
	}

	// sse 服务器
	s1 := cfg.Servers[1]
	if s1.Name != "weather" || s1.Transport != "sse" || s1.URL != "http://127.0.0.1:8000/sse" {
		t.Errorf("sse 服务器字段错误: %+v", s1)
	}
	if !s1.Enabled {
		t.Errorf("sse 服务器 enabled 应为 true")
	}

	// http 服务器
	s2 := cfg.Servers[2]
	if s2.Name != "files" || s2.Transport != "http" || s2.URL != "http://127.0.0.1:8080/mcp" {
		t.Errorf("http 服务器字段错误: %+v", s2)
	}
	if s2.Enabled {
		t.Errorf("http 服务器 enabled 应为 false")
	}
}

// TestLoadFromDBSortOrder 验证按 sort_order, id 升序返回。
func TestLoadFromDBSortOrder(t *testing.T) {
	db := newTestDB(t,
		models.MCPServer{Name: "b", Transport: "stdio", Command: "echo", SortOrder: 2},
		models.MCPServer{Name: "a", Transport: "stdio", Command: "echo", SortOrder: 1},
		models.MCPServer{Name: "c", Transport: "stdio", Command: "echo", SortOrder: 1},
	)

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 意外错误: %v", err)
	}
	got := names(cfg.Servers)
	if len(got) != 3 || got[0] != "a" || got[1] != "c" || got[2] != "b" {
		t.Errorf("排序结果 = %v, want [a c b]（sort_order 升序，同序按 id 升序）", got)
	}
}

// TestLoadFromDBInvalidTransport 验证不支持的 transport（webrtc）被跳过并记录 LoadErrors。
func TestLoadFromDBInvalidTransport(t *testing.T) {
	db := newTestDB(t, models.MCPServer{Name: "bad", Transport: "webrtc", Enabled: true})

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 意外错误: %v", err)
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("非法服务器应被跳过, Servers = %v", cfg.Servers)
	}
	if len(cfg.LoadErrors) != 1 {
		t.Fatalf("LoadErrors 数量 = %d, want 1", len(cfg.LoadErrors))
	}
	msg := cfg.LoadErrors[0].Error()
	if !strings.Contains(msg, "webrtc") {
		t.Errorf("错误信息应包含非法 transport %q, got: %v", "webrtc", msg)
	}
	if !strings.Contains(msg, "server[0]") {
		t.Errorf("错误信息应包含服务器索引定位, got: %v", msg)
	}
}

// TestLoadFromDBEmptyName 验证 name 为空被跳过并记录 LoadErrors。
func TestLoadFromDBEmptyName(t *testing.T) {
	db := newTestDB(t, models.MCPServer{Name: "", Transport: "stdio", Command: "echo"})

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 意外错误: %v", err)
	}
	if len(cfg.LoadErrors) != 1 {
		t.Fatalf("LoadErrors 数量 = %d, want 1", len(cfg.LoadErrors))
	}
	if !strings.Contains(cfg.LoadErrors[0].Error(), "name") {
		t.Errorf("错误信息应提及 name 字段, got: %v", cfg.LoadErrors[0])
	}
}

// TestLoadFromDBStdioMissingCommand 验证 stdio 传输缺 command 被跳过并记录 LoadErrors。
func TestLoadFromDBStdioMissingCommand(t *testing.T) {
	db := newTestDB(t, models.MCPServer{Name: "math", Transport: "stdio"})

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 意外错误: %v", err)
	}
	if len(cfg.LoadErrors) != 1 {
		t.Fatalf("LoadErrors 数量 = %d, want 1", len(cfg.LoadErrors))
	}
	if !strings.Contains(cfg.LoadErrors[0].Error(), "command") {
		t.Errorf("错误信息应提及 command 字段, got: %v", cfg.LoadErrors[0])
	}
}

// TestLoadFromDBMissingURL 验证 http/sse 传输缺 url 被跳过并记录 LoadErrors。
func TestLoadFromDBMissingURL(t *testing.T) {
	cases := []struct {
		name      string
		transport string
	}{
		{"http 缺 url", "http"},
		{"sse 缺 url", "sse"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := newTestDB(t, models.MCPServer{Name: "x", Transport: c.transport})

			cfg, err := mcpserver.LoadFromDB(db)
			if err != nil {
				t.Fatalf("LoadFromDB 意外错误: %v", err)
			}
			if len(cfg.LoadErrors) != 1 {
				t.Fatalf("LoadErrors 数量 = %d, want 1", len(cfg.LoadErrors))
			}
			if !strings.Contains(cfg.LoadErrors[0].Error(), "url") {
				t.Errorf("错误信息应提及 url 字段, got: %v", cfg.LoadErrors[0])
			}
		})
	}
}

// TestLoadFromDBSkipInvalidKeepsValid 验证单条非法不影响其余合法服务器装配。
func TestLoadFromDBSkipInvalidKeepsValid(t *testing.T) {
	db := newTestDB(t,
		models.MCPServer{Name: "good", Transport: "stdio", Command: "echo", Enabled: true},
		models.MCPServer{Name: "", Transport: "stdio", Command: "echo", Enabled: true},
		models.MCPServer{Name: "files", Transport: "http", URL: "http://127.0.0.1:8080/mcp", Enabled: false},
	)

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 意外错误: %v", err)
	}
	if len(cfg.Servers) != 2 {
		t.Errorf("合法服务器数量 = %d, want 2 (非法条目被跳过), got %v", len(cfg.Servers), names(cfg.Servers))
	}
	if cfg.Servers[0].Name != "good" || cfg.Servers[1].Name != "files" {
		t.Errorf("合法服务器列表 = %v, want [good files]", names(cfg.Servers))
	}
	if len(cfg.LoadErrors) != 1 {
		t.Errorf("LoadErrors 数量 = %d, want 1", len(cfg.LoadErrors))
	}
	if !strings.Contains(cfg.LoadErrors[0].Error(), "server[1]") {
		t.Errorf("错误信息应定位到 server[1], got: %v", cfg.LoadErrors[0])
	}
	// 合法服务器仍可正常取启用列表
	if got := cfg.EnabledServers(); len(got) != 1 || got[0].Name != "good" {
		t.Errorf("EnabledServers = %v, want [good]", names(got))
	}
}

// TestLoadFromDBTableMissing 验证表缺失（整体性错误，如未迁移）返回错误且信息可定位。
func TestLoadFromDBTableMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	// 不做 AutoMigrate，直接查询应报错

	_, err = mcpserver.LoadFromDB(db)
	if err == nil {
		t.Fatal("LoadFromDB 应返回错误, got nil")
	}
	if !strings.Contains(err.Error(), "读取 MCP 服务器配置失败") {
		t.Errorf("错误信息应包含「读取 MCP 服务器配置失败」, got: %v", err)
	}
}

// TestEnabledServers 验证只返回 enabled=true 的服务器。
func TestEnabledServers(t *testing.T) {
	db := newTestDB(t,
		models.MCPServer{Name: "a", Transport: "stdio", Command: "echo", Enabled: true},
		models.MCPServer{Name: "b", Transport: "sse", URL: "http://127.0.0.1:1/sse", Enabled: false},
		models.MCPServer{Name: "c", Transport: "http", URL: "http://127.0.0.1:2/mcp", Enabled: true},
	)

	cfg, err := mcpserver.LoadFromDB(db)
	if err != nil {
		t.Fatalf("LoadFromDB 意外错误: %v", err)
	}
	got := cfg.EnabledServers()
	if len(got) != 2 {
		t.Fatalf("EnabledServers 数量 = %d, want 2", len(got))
	}
	if got[0].Name != "a" || got[1].Name != "c" {
		t.Errorf("EnabledServers 结果 = %v, want [a c]", names(got))
	}

	// nil 接收者安全返回
	var nilCfg *mcpserver.Config
	if got := nilCfg.EnabledServers(); got != nil {
		t.Errorf("nil Config 的 EnabledServers 应为 nil, got %v", got)
	}
}

// names 提取服务器名列表，便于断言输出。
func names(servers []mcpserver.Server) []string {
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		out = append(out, s.Name)
	}
	return out
}
