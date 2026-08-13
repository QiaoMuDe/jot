package services

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"jot/internal/models"
)

// newMCPServerTestDB 打开内存 SQLite（单连接）并迁移 mcp_servers 表。
func newMCPServerTestDB(t *testing.T) *gorm.DB {
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
	return db
}

// TestListEmpty 验证空库返回空列表且无错误。
func TestListEmpty(t *testing.T) {
	db := newMCPServerTestDB(t)
	svc := NewMCPServerService(db)

	servers, err := svc.List()
	if err != nil {
		t.Fatalf("List 空库不应报错: %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("空库应返回空列表, got %+v", servers)
	}
}

// TestListSortOrder 验证按 sort_order, id 升序返回。
func TestListSortOrder(t *testing.T) {
	db := newMCPServerTestDB(t)
	svc := NewMCPServerService(db)

	if err := svc.Save(&models.MCPServer{Name: "b", Transport: "stdio", Command: "echo", SortOrder: 2}); err != nil {
		t.Fatalf("保存 b 失败: %v", err)
	}
	if err := svc.Save(&models.MCPServer{Name: "a", Transport: "stdio", Command: "echo", SortOrder: 1}); err != nil {
		t.Fatalf("保存 a 失败: %v", err)
	}
	if err := svc.Save(&models.MCPServer{Name: "c", Transport: "stdio", Command: "echo", SortOrder: 1}); err != nil {
		t.Fatalf("保存 c 失败: %v", err)
	}

	servers, err := svc.List()
	if err != nil {
		t.Fatalf("List 意外错误: %v", err)
	}
	if len(servers) != 3 {
		t.Fatalf("列表长度 = %d, want 3", len(servers))
	}
	got := []string{servers[0].Name, servers[1].Name, servers[2].Name}
	if got[0] != "a" || got[1] != "c" || got[2] != "b" {
		t.Errorf("排序结果 = %v, want [a c b]（sort_order 升序，同序按 id 升序）", got)
	}
}

// TestSaveCreate 验证 ID==0 新增成功，字段可正确回读（含 Args/Env 的 JSON 序列化往返）。
func TestSaveCreate(t *testing.T) {
	db := newMCPServerTestDB(t)
	svc := NewMCPServerService(db)

	server := &models.MCPServer{
		Name:      "math",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-math"},
		Env:       map[string]string{"FOO": "1", "BAR": "2"},
		Enabled:   true,
		SortOrder: 3,
	}
	if err := svc.Save(server); err != nil {
		t.Fatalf("Save 新增失败: %v", err)
	}
	if server.ID == 0 {
		t.Fatal("Save 新增后应回填 ID")
	}

	// 从库重新读取，验证字段（含 Args/Env JSON 序列化往返）正确还原
	var got models.MCPServer
	if err := db.First(&got, server.ID).Error; err != nil {
		t.Fatalf("重新读取失败: %v", err)
	}
	if got.Name != "math" || got.Transport != "stdio" || got.Command != "npx" {
		t.Errorf("基础字段错误: %+v", got)
	}
	if len(got.Args) != 2 || got.Args[0] != "-y" || got.Args[1] != "@modelcontextprotocol/server-math" {
		t.Errorf("args JSON 序列化往返还原错误: %v", got.Args)
	}
	if got.Env["FOO"] != "1" || got.Env["BAR"] != "2" {
		t.Errorf("env JSON 序列化往返还原错误: %v", got.Env)
	}
	if !got.Enabled || got.SortOrder != 3 {
		t.Errorf("enabled/sort_order 字段错误: %+v", got)
	}
}

// TestSaveUpdate 验证给定已存在记录的 ID 与修改后的字段，更新成功且不新增行。
func TestSaveUpdate(t *testing.T) {
	db := newMCPServerTestDB(t)
	svc := NewMCPServerService(db)

	server := &models.MCPServer{Name: "math", Transport: "stdio", Command: "npx", Enabled: true}
	if err := svc.Save(server); err != nil {
		t.Fatalf("Save 新增失败: %v", err)
	}
	id := server.ID

	// 携带已存在 ID 与修改后的字段进行更新
	server.Command = "node"
	server.Args = []string{"server.js"}
	if err := svc.Save(server); err != nil {
		t.Fatalf("Save 更新失败: %v", err)
	}

	var count int64
	if err := db.Model(&models.MCPServer{}).Count(&count).Error; err != nil {
		t.Fatalf("统计行数失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("更新不应新增行, 当前行数 = %d, want 1", count)
	}

	var got models.MCPServer
	if err := db.First(&got, id).Error; err != nil {
		t.Fatalf("重新读取失败: %v", err)
	}
	if got.Command != "node" || len(got.Args) != 1 || got.Args[0] != "server.js" {
		t.Errorf("更新后的字段未生效: %+v", got)
	}
}

// TestSaveValidate 验证各类非法配置均返回中文错误且不写入库。
func TestSaveValidate(t *testing.T) {
	cases := []struct {
		name    string
		server  models.MCPServer
		wantMsg string
	}{
		{"Name 为空", models.MCPServer{Transport: "stdio", Command: "echo"}, "名称不能为空"},
		{"Transport 非法", models.MCPServer{Name: "bad", Transport: "webrtc"}, "不支持的 transport"},
		{"stdio 缺 Command", models.MCPServer{Name: "math", Transport: "stdio"}, "必须提供 command"},
		{"sse 缺 URL", models.MCPServer{Name: "weather", Transport: "sse"}, "必须提供 url"},
		{"http 缺 URL", models.MCPServer{Name: "files", Transport: "http"}, "必须提供 url"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := newMCPServerTestDB(t)
			svc := NewMCPServerService(db)

			err := svc.Save(&c.server)
			if err == nil {
				t.Fatal("Save 应返回校验错误, got nil")
			}
			if !strings.Contains(err.Error(), c.wantMsg) {
				t.Errorf("错误信息 = %q, 应包含 %q", err.Error(), c.wantMsg)
			}
			// 校验失败不应写入库
			var count int64
			if err := db.Model(&models.MCPServer{}).Count(&count).Error; err != nil {
				t.Fatalf("统计行数失败: %v", err)
			}
			if count != 0 {
				t.Fatalf("校验失败不应写入, 当前行数 = %d, want 0", count)
			}
		})
	}
}

// TestSaveNameUnique 验证重复 Name 新增返回「已存在」错误；更新自身（同 ID 同 Name）不报错。
func TestSaveNameUnique(t *testing.T) {
	db := newMCPServerTestDB(t)
	svc := NewMCPServerService(db)

	first := &models.MCPServer{Name: "math", Transport: "stdio", Command: "npx"}
	if err := svc.Save(first); err != nil {
		t.Fatalf("首次保存失败: %v", err)
	}

	// 重复 Name 新增应报错
	dup := &models.MCPServer{Name: "math", Transport: "stdio", Command: "node"}
	err := svc.Save(dup)
	if err == nil {
		t.Fatal("重复 Name 应返回错误, got nil")
	}
	if !strings.Contains(err.Error(), "已存在") {
		t.Errorf("错误信息 = %q, 应包含「已存在」", err.Error())
	}

	// 更新自身（同 ID 同 Name）不应报错
	first.Command = "node"
	if err := svc.Save(first); err != nil {
		t.Fatalf("更新自身（同名）不应报错: %v", err)
	}
}

// TestDelete 验证删除后 List 不再包含该记录。
func TestDelete(t *testing.T) {
	db := newMCPServerTestDB(t)
	svc := NewMCPServerService(db)

	s1 := &models.MCPServer{Name: "math", Transport: "stdio", Command: "npx"}
	s2 := &models.MCPServer{Name: "files", Transport: "http", URL: "http://127.0.0.1:8080/mcp"}
	if err := svc.Save(s1); err != nil {
		t.Fatalf("保存 s1 失败: %v", err)
	}
	if err := svc.Save(s2); err != nil {
		t.Fatalf("保存 s2 失败: %v", err)
	}

	if err := svc.Delete(s1.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	servers, err := svc.List()
	if err != nil {
		t.Fatalf("List 意外错误: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("删除后列表长度 = %d, want 1", len(servers))
	}
	if servers[0].ID != s2.ID {
		t.Errorf("删除后剩余记录应为 s2, got %+v", servers[0])
	}
}
