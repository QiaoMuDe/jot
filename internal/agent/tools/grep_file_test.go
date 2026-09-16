package tools

// 本文件覆盖 grep_file 工具（单文件 + 目录递归内容搜索）的核心行为。
// 测试通过 fsToolBase.workspaceRoot 注入临时目录，复用 edit_file_test.go 的
// writeTestFile / readTestFile 写读测试文件。

import (
	"context"
	"strings"
	"testing"
)

// newTestGrepFile 构造注入临时工作目录的测试实例。
func newTestGrepFile(dir string) *grepFileTool {
	return &grepFileTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}

// TestGrepFileSingle 验证单文件搜索：行号正确、「行 N: 」格式、无匹配文案、空文件、二进制跳过。
func TestGrepFileSingle(t *testing.T) {
	dir := t.TempDir()
	h := newTestGrepFile(dir)
	const content = "hello 世界\nfoo bar\n第二行 foo\n结尾"
	writeTestFile(t, dir, "a.txt", content)

	t.Run("匹配行号与格式", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"a.txt","pattern":"foo"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "行 2: foo bar") || !strings.Contains(out, "行 3: 第二行 foo") {
			t.Errorf("输出应含带行号的命中行，实际:\n%s", out)
		}
		if strings.Contains(out, "行 1:") || strings.Contains(out, "行 4:") {
			t.Errorf("不应输出未命中行，实际:\n%s", out)
		}
	})

	t.Run("未命中返回未找到匹配行", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"a.txt","pattern":"不存在的"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if out != "未找到匹配行" {
			t.Errorf("输出 = %q, want 未找到匹配行", out)
		}
	})

	t.Run("空文件无匹配", func(t *testing.T) {
		writeTestFile(t, dir, "empty.txt", "")
		out, err := h.InvokableRun(context.Background(), `{"path":"empty.txt","pattern":"x"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if out != "未找到匹配行" {
			t.Errorf("输出 = %q, want 未找到匹配行", out)
		}
	})

	t.Run("二进制文件跳过", func(t *testing.T) {
		// 含 NUL 字节判为二进制
		writeTestFile(t, dir, "bin.dat", "abc\x00def")
		out, err := h.InvokableRun(context.Background(), `{"path":"bin.dat","pattern":"abc"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "二进制") {
			t.Errorf("二进制文件应提示跳过，实际: %q", out)
		}
	})
}

// TestGrepFileCaseInsensitive 验证忽略大小写：缺省大小写敏感、case_insensitive 命中。
func TestGrepFileCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	h := newTestGrepFile(dir)
	writeTestFile(t, dir, "ci.txt", "Hello\nWORLD\nmix")

	t.Run("缺省大小写敏感", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"ci.txt","pattern":"hello"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if out != "未找到匹配行" {
			t.Errorf("大小写敏感应不命中，实际: %q", out)
		}
	})
	t.Run("case_insensitive 命中", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"ci.txt","pattern":"WORLD","case_insensitive":true}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "行 2: WORLD") {
			t.Errorf("忽略大小写应命中第 2 行，实际:\n%s", out)
		}
	})
}

// TestGrepFileRegex 验证正则模式：命中、大小写不敏感正则、非法正则报错。
func TestGrepFileRegex(t *testing.T) {
	dir := t.TempDir()
	h := newTestGrepFile(dir)
	writeTestFile(t, dir, "re.txt", "abc123\nabcXYZ\ndef")

	t.Run("正则命中", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"re.txt","pattern":"abc\\d+","regex":true}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "行 1: abc123") {
			t.Errorf("正则应命中第 1 行，实际:\n%s", out)
		}
		if strings.Contains(out, "abcXYZ") {
			t.Errorf("正则不应命中大小写不符行，实际:\n%s", out)
		}
	})
	t.Run("正则忽略大小写", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"re.txt","pattern":"abcxyz","regex":true,"case_insensitive":true}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "行 2: abcXYZ") {
			t.Errorf("忽略大小写正则应命中第 2 行，实际:\n%s", out)
		}
	})
	t.Run("非法正则应报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"re.txt","pattern":"[","regex":true}`)
		if err == nil {
			t.Fatal("非法正则应报错")
		}
		if !strings.Contains(err.Error(), "正则编译失败") {
			t.Errorf("错误应含编译失败提示，实际: %v", err)
		}
	})
}

// TestGrepFileDirectory 验证目录递归：多文件相对路径前缀、file_pattern 过滤、子目录、整体无匹配。
func TestGrepFileDirectory(t *testing.T) {
	dir := t.TempDir()
	h := newTestGrepFile(dir)
	writeTestFile(t, dir, "a.go", "package main\nvar x = 1")
	writeTestFile(t, dir, "b.md", "hello world")
	writeTestFile(t, dir, "sub/c.go", "package sub\nvar x = 2")
	writeTestFile(t, dir, "sub/d.txt", "no match here")

	t.Run("递归命中带相对路径前缀", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":".","pattern":"var x"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "a.go:行 2: var x = 1") || !strings.Contains(out, "sub/c.go:行 2: var x = 2") {
			t.Errorf("应含相对路径前缀的命中行，实际:\n%s", out)
		}
		if strings.Contains(out, "b.md") || strings.Contains(out, "d.txt") {
			t.Errorf("不应输出未命中文件，实际:\n%s", out)
		}
	})

	t.Run("file_pattern 过滤", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":".","pattern":"var x","file_pattern":"*.go"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "a.go:行 2: var x = 1") || !strings.Contains(out, "sub/c.go:行 2: var x = 2") {
			t.Errorf("file_pattern=*.go 应命中两个 go 文件，实际:\n%s", out)
		}
	})

	t.Run("子目录为 path", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"sub","pattern":"var x"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "sub/c.go:行 2: var x = 2") {
			t.Errorf("path=sub 应命中 c.go 且前缀含 sub，实际:\n%s", out)
		}
		if strings.Contains(out, "a.go") {
			t.Errorf("path=sub 不应命中 sub 外文件，实际:\n%s", out)
		}
	})

	t.Run("整体无匹配", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":".","pattern":"zzz"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if out != "未找到匹配项" {
			t.Errorf("输出 = %q, want 未找到匹配项", out)
		}
	})
}

// TestGrepFileContextLines 验证 context_lines：缺省无上下文、前置/后置上下文与去重。
func TestGrepFileContextLines(t *testing.T) {
	dir := t.TempDir()
	h := newTestGrepFile(dir)
	// 命中行2 与行4：行3 同时是行2 的后置上下文与行4 的前置上下文，只应输出一次
	writeTestFile(t, dir, "ctx.txt", "one\ntwo-HIT\nthree\nfour-HIT\nfive")

	t.Run("缺省无上下文", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"ctx.txt","pattern":"HIT"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if strings.Contains(out, "one") || strings.Contains(out, "three") || strings.Contains(out, "five") {
			t.Errorf("缺省不应输出上下文行，实际:\n%s", out)
		}
	})
	t.Run("context_lines=1 前置后置", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"ctx.txt","pattern":"HIT","context_lines":1}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		for _, want := range []string{"行 1: one", "行 2: two-HIT", "行 3: three", "行 4: four-HIT", "行 5: five"} {
			if !strings.Contains(out, want) {
				t.Errorf("缺上下文行 %q，实际:\n%s", want, out)
			}
		}
		if n := strings.Count(out, "行 3: three"); n != 1 {
			t.Errorf("上下文行不应重复，实际出现 %d 次:\n%s", n, out)
		}
	})
}

// TestGrepFileBoundary 验证边界：路径逃逸/不存在、context_lines 非法值、超长行截断、结果超长截断。
func TestGrepFileBoundary(t *testing.T) {
	dir := t.TempDir()
	h := newTestGrepFile(dir)
	writeTestFile(t, dir, "a.txt", "short\n"+strings.Repeat("长", 300))

	t.Run("../ 逃逸拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"../evil.txt","pattern":"x"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})
	t.Run("路径不存在报错", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"missing.txt","pattern":"x"}`); err == nil {
			t.Fatal("路径不存在应报错")
		}
	})
	t.Run("context_lines 负数报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"a.txt","pattern":"short","context_lines":-1}`)
		if err == nil || !strings.Contains(err.Error(), "不可为负数") {
			t.Errorf("负数 context_lines 应报错，实际: %v", err)
		}
	})
	t.Run("context_lines 非整数报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"a.txt","pattern":"short","context_lines":1.5}`)
		if err == nil || !strings.Contains(err.Error(), "必须为 0-50 的整数") {
			t.Errorf("非整数 context_lines 应报错，实际: %v", err)
		}
	})
	t.Run("context_lines 超上限报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"a.txt","pattern":"short","context_lines":51}`)
		if err == nil || !strings.Contains(err.Error(), "过大") {
			t.Errorf("超上限 context_lines 应报错，实际: %v", err)
		}
	})
	t.Run("超长行截断", func(t *testing.T) {
		// 第 2 行 300 个"长"，命中后应截断到 maxGrepLineRunes 并带 …
		out, err := h.InvokableRun(context.Background(), `{"path":"a.txt","pattern":"长长"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "…") {
			t.Errorf("超长行应截断并带省略号，实际: %.120q", out)
		}
	})
	t.Run("结果超长提前停止", func(t *testing.T) {
		// 单文件多命中：输出逼近 maxGrepRunes 时提前中断并附提示
		var b strings.Builder
		for i := 0; i < 3000; i++ {
			b.WriteString("匹配行内容填充 " + string(rune('A'+i%26)) + "\n")
		}
		writeTestFile(t, dir, "big.txt", b.String())
		out, err := h.InvokableRun(context.Background(), `{"path":"big.txt","pattern":"匹配行内容填充"}`)
		if err != nil {
			t.Fatalf("搜索失败: %v", err)
		}
		if !strings.Contains(out, "[结果超长，已提前停止]") {
			t.Errorf("结果超长应提示提前停止，实际输出 %d 字符", len([]rune(out)))
		}
	})
}
