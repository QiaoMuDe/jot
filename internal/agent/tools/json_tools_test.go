package tools

// 本文件测试 JSON 工具三件套（json_validate / json_format / json_extract）：
// 纯函数逻辑（合法/非法 JSON、行列号定位、点路径提取、参数边界），全部离线可跑。

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

// runInvokable 便捷执行工具并返回输出与错误。
func runInvokable(t *testing.T, tl tool.InvokableTool, argsJSON string) (string, error) {
	t.Helper()
	return tl.InvokableRun(context.Background(), argsJSON)
}

func TestJSONValidateValid(t *testing.T) {
	tool, err := NewJSONValidateTool()
	if err != nil {
		t.Fatalf("NewJSONValidateTool: %v", err)
	}
	got, err := runInvokable(t, tool, `{"text": "{\"a\": 1, \"b\": [true, null]}"}`)
	if err != nil {
		t.Fatalf("InvokableRun err: %v", err)
	}
	if !strings.Contains(got, "JSON 合法") || !strings.Contains(got, "object") {
		t.Fatalf("合法 JSON 应报告合法与顶层类型，got %q", got)
	}
}

func TestJSONValidateInvalidReportsPosition(t *testing.T) {
	tool, _ := NewJSONValidateTool()
	// 第二行存在语法错误：{"a": 1,\n"b": 2,} 的尾逗号
	got, err := runInvokable(t, tool, `{"text": "{\"a\": 1,\n\"b\": 2,}"}`)
	if err != nil {
		t.Fatalf("InvokableRun err: %v", err)
	}
	if !strings.Contains(got, "不合法") {
		t.Fatalf("非法 JSON 应报告不合法，got %q", got)
	}
}

func TestJSONValidateEmptyAndOverlong(t *testing.T) {
	tool, _ := NewJSONValidateTool()
	if got, _ := runInvokable(t, tool, `{"text": "  "}`); !strings.Contains(got, "不能为空") {
		t.Fatalf("空文本应提示不能为空，got %q", got)
	}
	long := `{"text": "` + strings.Repeat("a", maxJSONLen+1) + `"}`
	if got, _ := runInvokable(t, tool, long); !strings.Contains(got, "过长") {
		t.Fatalf("超长文本应提示过长，got %.40q", got)
	}
}

func TestJSONFormat(t *testing.T) {
	tool, _ := NewJSONFormatTool()
	got, err := runInvokable(t, tool, `{"text": "{\"a\":1}", "indent": "  "}`)
	if err != nil {
		t.Fatalf("InvokableRun err: %v", err)
	}
	if !strings.Contains(got, "\n  \"a\": 1") {
		t.Fatalf("应输出带缩进的美化 JSON，got %q", got)
	}
	// 非法 indent
	got, _ = runInvokable(t, tool, `{"text": "{}", "indent": "x"}`)
	if !strings.Contains(got, "indent 只能由空格或 tab") {
		t.Fatalf("非法 indent 应提示，got %q", got)
	}
	// 非法 JSON
	got, _ = runInvokable(t, tool, `{"text": "not json"}`)
	if !strings.Contains(got, "不合法") {
		t.Fatalf("非法 JSON 应提示，got %q", got)
	}
}

func TestJSONExtract(t *testing.T) {
	tool, _ := NewJSONExtractTool()
	payload := `{"data":{"items":[{"title":"A","meta":{"n":3}},{"title":"B"}]},"name":"root"}`
	// 嵌套 + 数组下标（[0] 括号写法与 .0 点写法）+ 顶层 + # 通配
	cases := []struct {
		path string
		want string
	}{
		{"data.items[0].title", "A"},  // JSONPath 括号写法（归一化）
		{"data.items.0.title", "A"},   // gjson 原生点写法
		{"data.items[1].title", "B"},  // 括号写法
		{"data.items[0].meta.n", "3"}, // 数字字段
		{"name", "root"},              // 顶层字段
		{"$.name", "root"},            // $ 前缀（归一化）
		{"$", payload},                // $ 顶层：返回原文
		{"data.items", `[{"title":"A","meta":{"n":3}},{"title":"B"}]`}, // 数组：原始键序
		{"data.items.#.title", `["A","B"]`},                            // gjson # 通配符
	}
	for _, c := range cases {
		// escapeJSONArg 返回带引号的完整 JSON 字符串字面量，直接作为 text 字段值
		args := `{"text": ` + escapeJSONArg(payload) + `, "path": "` + c.path + `"}`
		got, err := runInvokable(t, tool, args)
		if err != nil {
			t.Fatalf("path %q InvokableRun err: %v", c.path, err)
		}
		if got != c.want {
			t.Fatalf("path %q: got %q, want %q", c.path, got, c.want)
		}
	}
	// 路径不存在
	args := `{"text": ` + escapeJSONArg(payload) + `, "path": "data.missing"}`
	got, _ := runInvokable(t, tool, args)
	if !strings.Contains(got, "不存在") {
		t.Fatalf("路径不存在应提示，got %q", got)
	}
	// 缺 path
	got, _ = runInvokable(t, tool, `{"text": `+escapeJSONArg(payload)+`}`)
	if !strings.Contains(got, "path 不能为空") {
		t.Fatalf("缺 path 应提示，got %q", got)
	}
	// null 值
	got, _ = runInvokable(t, tool, `{"text": `+escapeJSONArg(`{"a":null}`)+`, "path": "a"}`)
	if got != "null" {
		t.Fatalf("null 值应返回 null，got %q", got)
	}
}

// TestNormalizeGJSONPath 覆盖路径归一化：$ 前缀、括号下标、连续下标、开头下标。
func TestNormalizeGJSONPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"data.items[0].title", "data.items.0.title"},
		{"$.name", "name"},
		{"$", ""},
		{"name[0][1]", "name.0.1"},
		{"[0].b", "0.b"},
		{"data.items.#.title", "data.items.#.title"}, // # 通配原样透传
		{"  a.b  ", "a.b"},
	}
	for _, c := range cases {
		if got := normalizeGJSONPath(c.in); got != c.want {
			t.Fatalf("normalizeGJSONPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// escapeJSONArg 把测试负载转义为 JSON 字符串字面量（供工具参数 text 使用）。
func escapeJSONArg(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
