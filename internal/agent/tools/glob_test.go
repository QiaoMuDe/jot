package tools

// 本文件覆盖 glob 工具：通配符匹配（单层/递归 **）、字面前缀推导搜索根、
// 越权路径拒绝、空命中与搜索根不存在的友好返回。
// 测试通过 fsToolBase.workspaceRoot 注入临时目录，避免污染真实 ~/.jot/workspace。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestGlob 构造注入临时工作目录的测试实例。
func newTestGlob(dir string) *globTool {
	return &globTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}

// newTestWorkspace 创建临时工作目录并写入一组跨层级、多样式的样例文件。
func newTestWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"root.md":          "# root",
		"x.md":             "x",
		"xx.md":            "xx",
		"sub/a.py":         "a",
		"sub/b.py":         "b",
		"sub/note.txt":     "n",
		"scripts/a.py":     "sa",
		"scripts/note.md":  "sm",
		"scripts/note.txt": "st",
	}
	for rel, body := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestGlobBasics 验证 glob 的核心匹配：单层不跨目录、递归 **、字面后缀匹配。
func TestGlobBasics(t *testing.T) {
	dir := newTestWorkspace(t)
	h := newTestGlob(dir)

	t.Run("*.md 仅匹配当前层", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"*.md"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		for _, want := range []string{"root.md", "x.md", "xx.md"} {
			if !strings.Contains(out, want) {
				t.Errorf("应有 %q，实际: %.200s", want, out)
			}
		}
		for _, notWant := range []string{"scripts/note.md", "sub/a.py"} {
			if strings.Contains(out, notWant) {
				t.Errorf("不应含 %q（跨层），实际: %.200s", notWant, out)
			}
		}
	})

	t.Run("sub/*.py 单层匹配目标目录", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"sub/*.py"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		for _, want := range []string{"sub/a.py", "sub/b.py"} {
			if !strings.Contains(out, want) {
				t.Errorf("应有 %q，实际: %.200s", want, out)
			}
		}
		for _, notWant := range []string{"scripts/a.py", "sub/note.txt", "root.md"} {
			if strings.Contains(out, notWant) {
				t.Errorf("不应含 %q（超出范围/非 py），实际: %.200s", notWant, out)
			}
		}
	})

	t.Run("? 单字符匹配", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"?.md"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		if !strings.Contains(out, "x.md") || strings.Contains(out, "xx.md") || strings.Contains(out, "root.md") {
			t.Errorf("?.md 应只命中 x.md，实际: %.200s", out)
		}
	})
}

// TestGlobLiteralPrefix 验证由字面前缀推导的搜索根：只在该子树内查找。
func TestGlobLiteralPrefix(t *testing.T) {
	dir := newTestWorkspace(t)
	h := newTestGlob(dir)

	out, err := h.InvokableRun(context.Background(), `{"pattern":"scripts/*.md"}`)
	if err != nil {
		t.Fatalf("glob 失败: %v", err)
	}
	if !strings.Contains(out, "scripts/note.md") {
		t.Errorf("应命中 scripts/note.md，实际: %.200s", out)
	}
	for _, notWant := range []string{"scripts/note.txt", "scripts/a.py", "root.md"} {
		if strings.Contains(out, notWant) {
			t.Errorf("不应含 %q（超出 scripts/*.md 范围），实际: %.200s", notWant, out)
		}
	}

	out2, err := h.InvokableRun(context.Background(), `{"pattern":"sub/*"}`)
	if err != nil {
		t.Fatalf("glob 失败: %v", err)
	}
	if !strings.Contains(out2, "sub/a.py") || !strings.Contains(out2, "sub/b.py") {
		t.Errorf("sub/* 应命中 sub 下全部，实际: %.200s", out2)
	}
	if strings.Contains(out2, "root.md") {
		t.Errorf("sub/* 不应含跨层 root.md，实际: %.200s", out2)
	}
}

// TestGlobBoundaryAndEmpty 验证越权拒绝与空命中/搜索根不存在的友好返回。
func TestGlobBoundaryAndEmpty(t *testing.T) {
	dir := newTestWorkspace(t)
	h := newTestGlob(dir)

	t.Run("../ 逃逸拒绝", func(t *testing.T) {
		for _, pat := range []string{"../escape.md", "sub/../../escape.md"} {
			if _, err := h.InvokableRun(context.Background(), `{"pattern":`+jsonQuote(pat)+`}`); err == nil {
				t.Errorf("模式 %q 应报越权错误", pat)
			}
		}
	})

	t.Run("绝对路径拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"pattern":`+jsonQuote(filepath.Join(os.TempDir(), "*.md"))+`}`); err == nil {
			t.Error("绝对路径模式应报错")
		}
	})

	t.Run("空命中友好返回", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"*.zzz"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		if out != "未找到匹配项" {
			t.Errorf("应返回未找到匹配项，实际: %.100s", out)
		}
	})

	t.Run("字面前缀目录不存在友好返回", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"nope/*.md"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		if out != "未找到匹配项" {
			t.Errorf("应返回未找到匹配项，实际: %.100s", out)
		}
	})
}

// TestGlobTildePrefix 验证 ~/.jot/workspace/ 前缀模式剥离：完整形式命中、单独
// 出现视为 *、反斜杠变体经 toSlash 归一、其余 ~ 开头模式明确报错。
func TestGlobTildePrefix(t *testing.T) {
	dir := newTestWorkspace(t)
	h := newTestGlob(dir)

	t.Run("~/.jot/workspace/*.md 命中根层", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"~/.jot/workspace/*.md"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		for _, want := range []string{"root.md", "x.md"} {
			if !strings.Contains(out, want) {
				t.Errorf("应有 %q，实际: %.200s", want, out)
			}
		}
		if strings.Contains(out, "scripts/note.md") {
			t.Errorf("不应跨层命中，实际: %.200s", out)
		}
	})

	t.Run("~/.jot/workspace/scripts/*.md 命中子目录", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"~/.jot/workspace/scripts/*.md"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		if !strings.Contains(out, "scripts/note.md") {
			t.Errorf("应命中 scripts/note.md，实际: %.200s", out)
		}
	})

	t.Run("~/.jot/workspace 单独视为 *", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":"~/.jot/workspace"}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		if !strings.Contains(out, "root.md") || !strings.Contains(out, "sub") {
			t.Errorf("应列出根下条目，实际: %.200s", out)
		}
	})

	t.Run("反斜杠变体经 toSlash 归一", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"pattern":`+jsonQuote(`~\.jot\workspace\*.md`)+`}`)
		if err != nil {
			t.Fatalf("glob 失败: %v", err)
		}
		if !strings.Contains(out, "root.md") {
			t.Errorf("反斜杠变体应命中根层 .md，实际: %.200s", out)
		}
	})

	t.Run("其它 ~ 开头模式报错", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"pattern":"~/foo/*.md"}`); err == nil {
			t.Error("~/foo/*.md 应报错")
		}
	})
}
