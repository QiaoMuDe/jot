package mcpserver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jot/internal/mcpserver"
)

// writeTempConfig 将 JSON 内容写入 t.TempDir() 下的临时文件，返回文件路径。
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mcp-servers.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入临时配置文件失败: %v", err)
	}
	return path
}

// TestLoadValidConfig 验证 stdio + sse + http 三种传输的合法配置可正常解析，字段值正确。
func TestLoadValidConfig(t *testing.T) {
	path := writeTempConfig(t, `{
	  "servers": [
	    {"name": "math", "transport": "stdio", "command": "npx",
	     "args": ["-y", "@modelcontextprotocol/server-math"], "env": {"FOO": "1"}, "enabled": true},
	    {"name": "weather", "transport": "sse", "url": "http://127.0.0.1:8000/sse", "enabled": true},
	    {"name": "files", "transport": "http", "url": "http://127.0.0.1:8080/mcp", "enabled": false}
	  ]
	}`)

	cfg, err := mcpserver.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) 意外错误: %v", path, err)
	}
	if len(cfg.Servers) != 3 {
		t.Fatalf("服务器数量 = %d, want 3", len(cfg.Servers))
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

// TestLoadInvalidTransport 验证不支持的 transport（webrtc）被跳过并记录 LoadErrors。
func TestLoadInvalidTransport(t *testing.T) {
	path := writeTempConfig(t, `{"servers": [{"name": "bad", "transport": "webrtc", "enabled": true}]}`)

	cfg, err := mcpserver.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) 意外错误: %v", path, err)
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

// TestLoadEmptyName 验证 name 为空被跳过并记录 LoadErrors。
func TestLoadEmptyName(t *testing.T) {
	path := writeTempConfig(t, `{"servers": [{"name": "", "transport": "stdio", "command": "echo"}]}`)

	cfg, err := mcpserver.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) 意外错误: %v", path, err)
	}
	if len(cfg.LoadErrors) != 1 {
		t.Fatalf("LoadErrors 数量 = %d, want 1", len(cfg.LoadErrors))
	}
	if !strings.Contains(cfg.LoadErrors[0].Error(), "name") {
		t.Errorf("错误信息应提及 name 字段, got: %v", cfg.LoadErrors[0])
	}
}

// TestLoadStdioMissingCommand 验证 stdio 传输缺 command 被跳过并记录 LoadErrors。
func TestLoadStdioMissingCommand(t *testing.T) {
	path := writeTempConfig(t, `{"servers": [{"name": "math", "transport": "stdio"}]}`)

	cfg, err := mcpserver.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) 意外错误: %v", path, err)
	}
	if len(cfg.LoadErrors) != 1 {
		t.Fatalf("LoadErrors 数量 = %d, want 1", len(cfg.LoadErrors))
	}
	if !strings.Contains(cfg.LoadErrors[0].Error(), "command") {
		t.Errorf("错误信息应提及 command 字段, got: %v", cfg.LoadErrors[0])
	}
}

// TestLoadHTTPMissingURL 验证 http/sse 传输缺 url 被跳过并记录 LoadErrors。
func TestLoadHTTPMissingURL(t *testing.T) {
	cases := []struct {
		name      string
		transport string
	}{
		{"http 缺 url", "http"},
		{"sse 缺 url", "sse"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeTempConfig(t, `{"servers": [{"name": "x", "transport": "`+c.transport+`"}]}`)

			cfg, err := mcpserver.Load(path)
			if err != nil {
				t.Fatalf("Load(%q) 意外错误: %v", path, err)
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

// TestLoadSkipInvalidKeepsValid 验证单条非法不影响其余合法服务器装配。
func TestLoadSkipInvalidKeepsValid(t *testing.T) {
	path := writeTempConfig(t, `{
	  "servers": [
	    {"name": "good", "transport": "stdio", "command": "echo", "enabled": true},
	    {"name": "", "transport": "stdio", "command": "echo", "enabled": true},
	    {"name": "files", "transport": "http", "url": "http://127.0.0.1:8080/mcp", "enabled": false}
	  ]
	}`)

	cfg, err := mcpserver.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) 意外错误: %v", path, err)
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

// TestLoadMissingFile 验证不存在的配置文件路径返回错误且错误信息包含路径。
func TestLoadMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-exist.json")

	_, err := mcpserver.Load(missing)
	if err == nil {
		t.Fatal("Load 应返回错误, got nil")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("错误信息应包含路径 %q, got: %v", missing, err)
	}
}

// TestEnabledServers 验证只返回 enabled=true 的服务器。
func TestEnabledServers(t *testing.T) {
	path := writeTempConfig(t, `{
	  "servers": [
	    {"name": "a", "transport": "stdio", "command": "echo", "enabled": true},
	    {"name": "b", "transport": "sse", "url": "http://127.0.0.1:1/sse", "enabled": false},
	    {"name": "c", "transport": "http", "url": "http://127.0.0.1:2/mcp", "enabled": true}
	  ]
	}`)

	cfg, err := mcpserver.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) 意外错误: %v", path, err)
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

// TestEnsureConfigFile 验证 ensureConfigFile：目录+默认文件创建、幂等不覆盖、已存在文件保留。
func TestEnsureConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp", "mcp-servers.json")

	// 1. 首次调用：创建目录并写入默认配置
	if err := mcpserver.EnsureConfigFileAt(path); err != nil {
		t.Fatalf("ensureConfigFile 首次调用失败: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取默认配置文件失败: %v", err)
	}
	cfg, err := mcpserver.Load(path)
	if err != nil {
		t.Fatalf("默认配置应可被 Load 解析: %v", err)
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("默认配置 servers 应为空, got %d 条", len(cfg.Servers))
	}

	// 2. 二次调用：幂等，不报错且内容不变
	if err := mcpserver.EnsureConfigFileAt(path); err != nil {
		t.Fatalf("ensureConfigFile 二次调用失败: %v", err)
	}
	again, _ := os.ReadFile(path)
	if string(again) != string(data) {
		t.Error("重复调用不应改动已有配置文件内容")
	}

	// 3. 用户已填写配置时保留原文件（不被默认内容覆盖）
	custom := `{"servers":[{"name":"x","transport":"stdio","command":"echo","enabled":false}]}`
	if err := os.WriteFile(path, []byte(custom), 0644); err != nil {
		t.Fatalf("写入自定义配置失败: %v", err)
	}
	if err := mcpserver.EnsureConfigFileAt(path); err != nil {
		t.Fatalf("ensureConfigFile 对已存在文件调用失败: %v", err)
	}
	kept, _ := os.ReadFile(path)
	if string(kept) != custom {
		t.Error("已存在的用户配置不应被默认配置覆盖")
	}
}
