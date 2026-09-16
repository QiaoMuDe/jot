package tools

// 本文件覆盖 edit_file 工具（片段替换 + 行级替换双模式）的核心行为。
// 测试通过 fsToolBase.workspaceRoot 注入临时目录，避免污染真实 ~/.jot/workspace。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestEditFile 构造注入临时工作目录的测试实例。
func newTestEditFile(dir string) *editFileTool {
	return &editFileTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}

// writeTestFile 在临时目录写入测试文件内容。
func writeTestFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readTestFile 读取临时目录中的测试文件内容。
func readTestFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestEditFileFragment 验证片段替换：精确匹配、第 N 次出现、全部替换、空白归一化兜底、删除片段、未找到报错。
func TestEditFileFragment(t *testing.T) {
	dir := t.TempDir()
	h := newTestEditFile(dir)
	const base = "甲 乙 丙\n甲 乙 丁\n甲 乙 戊"

	t.Run("精确匹配替换", func(t *testing.T) {
		writeTestFile(t, dir, "f.txt", base)
		out, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"丙","replace":"丙丙"}`)
		if err != nil {
			t.Fatalf("片段替换失败: %v", err)
		}
		if !strings.Contains(out, "第 1 处，精确匹配") {
			t.Errorf("应含匹配方式信息，实际: %q", out)
		}
		if got := readTestFile(t, dir, "f.txt"); got != "甲 乙 丙丙\n甲 乙 丁\n甲 乙 戊" {
			t.Errorf("替换后内容 = %q", got)
		}
	})

	t.Run("count 替换第 2 次出现", func(t *testing.T) {
		writeTestFile(t, dir, "f.txt", base)
		if _, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"乙","replace":"Y","count":2}`); err != nil {
			t.Fatalf("count 替换失败: %v", err)
		}
		if got := readTestFile(t, dir, "f.txt"); got != "甲 乙 丙\n甲 Y 丁\n甲 乙 戊" {
			t.Errorf("count=2 应替换第 2 行乙，实际: %q", got)
		}
	})

	t.Run("replace_all 全部替换", func(t *testing.T) {
		writeTestFile(t, dir, "f.txt", base)
		out, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"甲","replace":"A","replace_all":true}`)
		if err != nil {
			t.Fatalf("replace_all 失败: %v", err)
		}
		if !strings.Contains(out, "共 3 处") {
			t.Errorf("应含替换处数，实际: %q", out)
		}
		if got := readTestFile(t, dir, "f.txt"); got != "A 乙 丙\nA 乙 丁\nA 乙 戊" {
			t.Errorf("replace_all 后内容 = %q", got)
		}
	})

	t.Run("空白归一化兜底匹配", func(t *testing.T) {
		writeTestFile(t, dir, "f.txt", base)
		// 原文 "乙 丙" 为单空格，find 用双空格模拟空白差异，应经归一化命中第 1 处
		out, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"乙  丙","replace":"B"}`)
		if err != nil {
			t.Fatalf("归一化匹配失败: %v", err)
		}
		if !strings.Contains(out, "空白归一化匹配") {
			t.Errorf("应标注归一化匹配，实际: %q", out)
		}
		if got := readTestFile(t, dir, "f.txt"); got != "甲 B\n甲 乙 丁\n甲 乙 戊" {
			t.Errorf("归一化替换后内容 = %q", got)
		}
	})

	t.Run("删除片段（replace 空）", func(t *testing.T) {
		writeTestFile(t, dir, "f.txt", "甲乙丙\n甲乙丁")
		if _, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"丙","replace":""}`); err != nil {
			t.Fatalf("删除片段失败: %v", err)
		}
		if got := readTestFile(t, dir, "f.txt"); got != "甲乙\n甲乙丁" {
			t.Errorf("删除片段后内容 = %q", got)
		}
	})

	t.Run("未找到片段报错并提示", func(t *testing.T) {
		writeTestFile(t, dir, "f.txt", base)
		_, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"不存在的片段"}`)
		if err == nil {
			t.Fatal("未找到片段应报错")
		}
		if !strings.Contains(err.Error(), "未在文件中找到片段") {
			t.Errorf("错误应含未找到提示，实际: %v", err)
		}
		if !strings.Contains(err.Error(), "read_file") {
			t.Errorf("错误应指引 read_file 重试，实际: %v", err)
		}
	})

	t.Run("replace_all 归一化混合替换", func(t *testing.T) {
		// 文件中只有双空格分隔，find 用单空格，精确匹配全部落空后经归一化逐处替换
		writeTestFile(t, dir, "f.txt", "X  Y\nX  Y")
		out, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"X Y","replace":"XY","replace_all":true}`)
		if err != nil {
			t.Fatalf("归一化全量替换失败: %v", err)
		}
		if !strings.Contains(out, "共 2 处") {
			t.Errorf("应含替换处数，实际: %q", out)
		}
		if got := readTestFile(t, dir, "f.txt"); got != "XY\nXY" {
			t.Errorf("归一化全量替换后内容 = %q", got)
		}
	})

	t.Run("CRLF 文件空白归一化匹配", func(t *testing.T) {
		// find 用 LF 换行，文件是 CRLF：\r\n 经归一化折叠为单个空格后命中
		writeTestFile(t, dir, "f.txt", "a\r\nb\r\nc")
		if _, err := h.InvokableRun(context.Background(), `{"path":"f.txt","find":"a\nb","replace":"x"}`); err != nil {
			t.Fatalf("CRLF 归一化替换失败: %v", err)
		}
		if got := readTestFile(t, dir, "f.txt"); got != "x\r\nc" {
			t.Errorf("CRLF 归一化替换后内容 = %q", got)
		}
	})
}

// TestEditFileByLines 验证行级替换：单行、区间、删除行、末尾追加、行号越界报错。
func TestEditFileByLines(t *testing.T) {
	dir := t.TempDir()
	h := newTestEditFile(dir)
	const base = "第一行\n第二行\n第三行\n第四行"

	t.Run("单行替换", func(t *testing.T) {
		writeTestFile(t, dir, "l.txt", base)
		out, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":2,"line_end":2,"replace":"新二行"}`)
		if err != nil {
			t.Fatalf("单行替换失败: %v", err)
		}
		if !strings.Contains(out, "已替换第 2 行") {
			t.Errorf("应含替换行信息，实际: %q", out)
		}
		if got := readTestFile(t, dir, "l.txt"); got != "第一行\n新二行\n第三行\n第四行" {
			t.Errorf("单行替换后内容 = %q", got)
		}
	})

	t.Run("区间替换", func(t *testing.T) {
		writeTestFile(t, dir, "l.txt", base)
		if _, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":2,"line_end":3,"replace":"中间"}`); err != nil {
			t.Fatalf("区间替换失败: %v", err)
		}
		if got := readTestFile(t, dir, "l.txt"); got != "第一行\n中间\n第四行" {
			t.Errorf("区间替换后内容 = %q", got)
		}
	})

	t.Run("删除行（replace 空）", func(t *testing.T) {
		writeTestFile(t, dir, "l.txt", base)
		if _, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":2,"line_end":3,"replace":""}`); err != nil {
			t.Fatalf("删除行失败: %v", err)
		}
		if got := readTestFile(t, dir, "l.txt"); got != "第一行\n第四行" {
			t.Errorf("删除行后内容 = %q", got)
		}
	})

	t.Run("末尾追加", func(t *testing.T) {
		writeTestFile(t, dir, "l.txt", base)
		out, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":99,"replace":"第五行"}`)
		if err != nil {
			t.Fatalf("末尾追加失败: %v", err)
		}
		if !strings.Contains(out, "末尾追加") {
			t.Errorf("应含追加语义，实际: %q", out)
		}
		if got := readTestFile(t, dir, "l.txt"); got != "第一行\n第二行\n第三行\n第四行\n\n第五行" {
			t.Errorf("末尾追加后内容 = %q", got)
		}
	})

	t.Run("行号越界报错", func(t *testing.T) {
		writeTestFile(t, dir, "l.txt", base)
		_, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":3,"line_end":6,"replace":"x"}`)
		if err == nil {
			t.Fatal("行号越界应报错")
		}
		if !strings.Contains(err.Error(), "行号越界") {
			t.Errorf("错误应含行号越界，实际: %v", err)
		}
		if strings.Contains(err.Error(), "笔记") {
			t.Errorf("行级错误不应残留笔记字样，实际: %v", err)
		}
	})

	t.Run("反序行号（line_end < line_start）", func(t *testing.T) {
		// 反序时按单行处理（end 收敛为 start）：替换第 3 行
		writeTestFile(t, dir, "l.txt", base)
		if _, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":3,"line_end":1,"replace":"新三行"}`); err != nil {
			t.Fatalf("反序行号替换失败: %v", err)
		}
		if got := readTestFile(t, dir, "l.txt"); got != "第一行\n第二行\n新三行\n第四行" {
			t.Errorf("反序行号替换后内容 = %q", got)
		}
	})

	t.Run("空文件末尾追加", func(t *testing.T) {
		writeTestFile(t, dir, "l.txt", "")
		if _, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":1,"replace":"hello"}`); err != nil {
			t.Fatalf("空文件追加失败: %v", err)
		}
		if got := readTestFile(t, dir, "l.txt"); got != "hello" {
			t.Errorf("空文件追加后内容 = %q", got)
		}
	})

	t.Run("末尾追加空内容报错", func(t *testing.T) {
		writeTestFile(t, dir, "l.txt", base)
		if _, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":99,"replace":""}`); err == nil {
			t.Fatal("末尾追加空内容应报错")
		}
	})

	t.Run("CRLF 行级替换保留未替换行行尾符", func(t *testing.T) {
		// 第 1 行保留 \r\n，替换行与后续行为全新内容用 LF 分隔
		writeTestFile(t, dir, "l.txt", "a\r\nb\r\nc")
		if _, err := h.InvokableRun(context.Background(), `{"path":"l.txt","line_start":2,"replace":"B"}`); err != nil {
			t.Fatalf("CRLF 行级替换失败: %v", err)
		}
		if got := readTestFile(t, dir, "l.txt"); got != "a\r\nB\nc" {
			t.Errorf("CRLF 行级替换后内容 = %q", got)
		}
	})
}

// TestEditFileValidation 验证参数校验：path 缺失、模式互斥、无编辑内容、replace_all 与 count 互斥、长度超限。
func TestEditFileValidation(t *testing.T) {
	dir := t.TempDir()
	h := newTestEditFile(dir)
	writeTestFile(t, dir, "v.txt", "x")

	cases := []struct {
		name, args, wantErr string
	}{
		{"缺少 path", `{"find":"x","replace":"y"}`, "缺少 path"},
		{"模式混用", `{"path":"v.txt","find":"x","line_start":1,"replace":"y"}`, "不可混用"},
		{"无编辑内容", `{"path":"v.txt"}`, "无可编辑内容"},
		{"replace_all 与 count 互斥", `{"path":"v.txt","find":"x","replace":"y","replace_all":true,"count":2}`, "互斥"},
		{"负 line_start", `{"path":"v.txt","line_start":-1,"replace":"y"}`, "不可为负数"},
		{"非整数 count", `{"path":"v.txt","find":"x","replace":"y","count":1.9}`, "必须为整数"},
		{"find 超长", `{"path":"v.txt","find":"` + strings.Repeat("a", maxToolFindLen+1) + `","replace":"y"}`, "过长"},
		{"replace 超长", `{"path":"v.txt","find":"x","replace":"` + strings.Repeat("a", maxToolLongText+1) + `"}`, "过长"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := h.InvokableRun(context.Background(), c.args)
			if err == nil {
				t.Fatal("应报错")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("错误应含 %q，实际: %v", c.wantErr, err)
			}
		})
	}
}

// TestEditFilePathSafety 验证路径边界：../ 逃逸、绝对路径越界、文件不存在报错。
func TestEditFilePathSafety(t *testing.T) {
	dir := t.TempDir()
	h := newTestEditFile(dir)
	writeTestFile(t, dir, "safe.txt", "x")

	t.Run("../ 逃逸拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"../evil.txt","find":"x","replace":"y"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})
	t.Run("绝对路径越界拒绝", func(t *testing.T) {
		outside := filepath.Join(os.TempDir(), "outside_edit.txt")
		if _, err := h.InvokableRun(context.Background(), `{"path":`+jsonQuote(outside)+`,"find":"x","replace":"y"}`); err == nil {
			t.Fatal("绝对路径越界应报错")
		}
	})
	t.Run("文件不存在报错并引导创建", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"nope.txt","find":"x","replace":"y"}`)
		if err == nil {
			t.Fatal("文件不存在应报错")
		}
		if !strings.Contains(err.Error(), "write_file") {
			t.Errorf("错误应引导 write_file 创建，实际: %v", err)
		}
	})
}

// TestEditFileApproval 验证审批钩子：批准放行、Approver 缺失 fail-fast、裸工具直接放行。
func TestEditFileApproval(t *testing.T) {
	dir := t.TempDir()

	t.Run("审批钩子生效", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "x")
		mock := &mockApprover{}
		hc := newTestEditFile(dir)
		hc.ctx = &Context{Approver: mock}
		if _, err := hc.InvokableRun(context.Background(), `{"path":"a.txt","find":"x","replace":"y"}`); err != nil {
			t.Fatalf("批准后编辑失败: %v", err)
		}
		if mock.gotTool != "edit_file" {
			t.Errorf("审批 toolName = %q, want edit_file", mock.gotTool)
		}
		if !strings.Contains(mock.gotSum, "a.txt") {
			t.Errorf("审批摘要应含文件名，实际: %q", mock.gotSum)
		}
		if mock.gotCritical {
			t.Error("edit_file 审批 critical 应为 false，实际 true")
		}
	})
	t.Run("Approver 缺失报错", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "x")
		hc := newTestEditFile(dir)
		hc.ctx = &Context{} // 非 nil 但缺 Approver：装配遗漏应 fail-fast
		if _, err := hc.InvokableRun(context.Background(), `{"path":"a.txt","find":"x","replace":"y"}`); err == nil {
			t.Fatal("Approver 缺失应报错")
		}
	})
	t.Run("ctx 为 nil 直接放行", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "x")
		h := newTestEditFile(dir)
		if _, err := h.InvokableRun(context.Background(), `{"path":"a.txt","find":"x","replace":"y"}`); err != nil {
			t.Fatalf("裸工具应放行: %v", err)
		}
	})
}

// TestEditFileLargeFile 验证大文件防护：超过 maxEditFileSize 拒绝。
func TestEditFileLargeFile(t *testing.T) {
	dir := t.TempDir()
	h := newTestEditFile(dir)
	writeTestFile(t, dir, "big.txt", strings.Repeat("x", maxEditFileSize+1))
	_, err := h.InvokableRun(context.Background(), `{"path":"big.txt","find":"y","replace":"z"}`)
	if err == nil {
		t.Fatal("超大文件应被拒绝")
	}
	if !strings.Contains(err.Error(), "过大") {
		t.Errorf("错误应含过大提示，实际: %v", err)
	}
}

// TestEditFileBinary 验证二进制防护：含 NUL 的文件按文本编辑会破坏内容，应被拒绝。
func TestEditFileBinary(t *testing.T) {
	dir := t.TempDir()
	h := newTestEditFile(dir)
	// 含 NUL 字节的文件经 go-kit fs 判为二进制，find/replace 按文本处理会破坏内容
	writeTestFile(t, dir, "bin.dat", "PK\x00\x01\x02data")

	t.Run("片段模式拒绝", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"bin.dat","find":"data","replace":"x"}`)
		if err == nil {
			t.Fatal("二进制文件编辑应被拒绝")
		}
		if !strings.Contains(err.Error(), "二进制") {
			t.Errorf("错误应含二进制提示，实际: %v", err)
		}
	})
	t.Run("行级模式拒绝", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"bin.dat","line_start":1,"replace":"x"}`)
		if err == nil {
			t.Fatal("二进制文件行级编辑应被拒绝")
		}
		if !strings.Contains(err.Error(), "二进制") {
			t.Errorf("错误应含二进制提示，实际: %v", err)
		}
	})
}
