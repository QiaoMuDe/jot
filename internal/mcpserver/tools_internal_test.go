package mcpserver

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// fakeInnerTool 最小假工具：仅实现 BaseTool.Info，供 mcpTool 缓存测试使用。
type fakeInnerTool struct {
	info *schema.ToolInfo
}

func (f *fakeInnerTool) Info(context.Context) (*schema.ToolInfo, error) {
	return f.info, nil
}

var _ tool.BaseTool = (*fakeInnerTool)(nil)

// TestMCPToolInfoCached 验证 mcpTool.Info 缓存行为：
// 多次调用返回改名后名称且内容一致；修改某次返回值不影响后续调用（浅拷贝防污染）。
func TestMCPToolInfoCached(t *testing.T) {
	mt := &mcpTool{
		serverName:   "svr",
		originalName: "orig",
		inner: &fakeInnerTool{info: &schema.ToolInfo{
			Name: "orig",
			Desc: "desc",
		}},
	}

	info1, err := mt.Info(context.Background())
	if err != nil {
		t.Fatalf("首次 Info 失败: %v", err)
	}
	if info1.Name != "mcp_svr_orig" {
		t.Errorf("改名后 Name = %q, want mcp_svr_orig", info1.Name)
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
