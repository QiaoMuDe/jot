package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPToolInfoCached 验证 mcpTool.Info 缓存行为：
// 多次调用返回改名后名称且内容一致；修改某次返回值不影响后续调用（浅拷贝防污染）。
func TestMCPToolInfoCached(t *testing.T) {
	mt := &mcpTool{
		serverName:   "svr",
		originalName: "orig",
		toolDef: &mcp.Tool{
			Name:        "orig",
			Description: "desc",
			InputSchema: map[string]any{"type": "object"},
		},
	}

	info1, err := mt.Info(context.Background())
	if err != nil {
		t.Fatalf("首次 Info 失败: %v", err)
	}
	if info1.Name != "mcp_svr_orig" {
		t.Errorf("改名后 Name = %q, want mcp_svr_orig", info1.Name)
	}
	if info1.Desc != "desc" {
		t.Errorf("Desc = %q, want desc", info1.Desc)
	}

	info2, err := mt.Info(context.Background())
	if err != nil {
		t.Fatalf("二次 Info 失败: %v", err)
	}
	if info2.Name != info1.Name {
		t.Errorf("二次 Name = %q, 应与首次一致 %q", info2.Name, info1.Name)
	}

	// 修改首次返回值，后续调用不受污染（验证浅拷贝而非共享指针）
	info1.Name = "HACKED"
	info3, err := mt.Info(context.Background())
	if err != nil {
		t.Fatalf("三次 Info 失败: %v", err)
	}
	if info3.Name != "mcp_svr_orig" {
		t.Errorf("修改返回值后缓存被污染: Name = %q, want mcp_svr_orig", info3.Name)
	}
}

// TestInputSchemaToParamsOneOf 验证 InputSchema 转换：
// 合法 object Schema → 返回 JSONSchema 形式 ParamsOneOf；空/非法输入 → nil（降级无参数）。
func TestInputSchemaToParamsOneOf(t *testing.T) {
	// 合法 object schema
	po := inputSchemaToParamsOneOf(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"q": map[string]any{"type": "string"},
		},
		"required": []any{"q"},
	})
	if po == nil {
		t.Fatal("合法 schema 转换结果不应为 nil")
	}

	// nil → nil
	if inputSchemaToParamsOneOf(nil) != nil {
		t.Error("nil 输入应返回 nil")
	}

	// 无法 marshal 的值 → nil
	if inputSchemaToParamsOneOf(make(chan int)) != nil {
		t.Error("非法输入应返回 nil")
	}
}
