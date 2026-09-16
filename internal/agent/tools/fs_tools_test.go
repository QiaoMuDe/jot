package tools

// 本文件覆盖工作目录文件工具（read_file / write_file / list_dir）的核心行为。
// 测试通过 fsToolBase.workspaceRoot 注入临时目录，避免污染真实 ~/.jot/workspace。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestReadFile / newTestWriteFile / newTestListDir 构造注入临时工作目录的测试实例。
func newTestReadFile(dir string) *readFileTool {
	return &readFileTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}
func newTestWriteFile(dir string) *writeFileTool {
	return &writeFileTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}
func newTestListDir(dir string) *listDirTool {
	return &listDirTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}

// TestWorkspaceFilePathUsedByRead 验证 read_file 的边界校验：相对路径合法读取，
// ../ 逃逸被拒绝（越权路径错误）。
func TestReadFileBasics(t *testing.T) {
	dir := t.TempDir()
	content := "hello 世界"
	rel := "sub/a.txt"
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newTestReadFile(dir)

	t.Run("相对路径合法读取", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"sub/a.txt"}`)
		if err != nil {
			t.Fatalf("读取失败: %v", err)
		}
		if out != content {
			t.Errorf("读取内容 = %q, want %q", out, content)
		}
	})
	t.Run("../ 逃逸拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"sub/../../escape.txt"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})
	t.Run("绝对路径越界拒绝", func(t *testing.T) {
		outside := os.TempDir()
		if _, err := h.InvokableRun(context.Background(), `{"path":`+jsonQuote(filepath.Join(outside, "x.txt"))+`}`); err == nil {
			t.Fatal("绝对路径越界应报错")
		}
	})
}

// TestReadFilePagination 验证 read_file 分页：offset+length 切片、未读完提示续读、
// offset 越界返回"已全部读取"。
func TestReadFilePagination(t *testing.T) {
	dir := t.TempDir()
	content := strings.Repeat("甲", 100) + strings.Repeat("乙", 100)
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newTestReadFile(dir)

	// length=50 从 offset=0 读，应读到前 50 个甲并提示续读
	out, err := h.InvokableRun(context.Background(), `{"path":"big.txt","length":50}`)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if !strings.Contains(out, strings.Repeat("甲", 50)) {
		t.Errorf("首段应含 50 个甲，实际: %.120s", out)
	}
	if !strings.Contains(out, "[文件未读完，可继续用 offset=50 读取后续内容]") {
		t.Errorf("未读完应提示续读 offset=50，实际: %.120s", out)
	}

	// 读到末尾
	out, err = h.InvokableRun(context.Background(), `{"path":"big.txt","offset":150,"length":100}`)
	if err != nil {
		t.Fatalf("末段读取失败: %v", err)
	}
	if strings.Contains(out, "[文件未读完") {
		t.Errorf("末段不应提示续读，实际: %.120s", out)
	}

	// offset 越界
	out, err = h.InvokableRun(context.Background(), `{"path":"big.txt","offset":100000}`)
	if err != nil {
		t.Fatalf("越界读取失败: %v", err)
	}
	if out != "已全部读取" {
		t.Errorf("越界应返回已全部读取，实际: %q", out)
	}

	// 空文件返回已全部读取
	if err := os.WriteFile(filepath.Join(dir, "empty.txt"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = h.InvokableRun(context.Background(), `{"path":"empty.txt"}`)
	if err != nil {
		t.Fatalf("空文件读取失败: %v", err)
	}
	if out != "已全部读取" {
		t.Errorf("空文件应返回已全部读取，实际: %q", out)
	}
}

// mockApprover 模拟审批器：记录收到的 toolName/summary/critical，returnErr 非空时拒绝，否则批准。
type mockApprover struct {
	gotTool     string
	gotSum      string
	gotCritical bool
}

func (m *mockApprover) RequestApproval(_ context.Context, toolName, summary string, critical bool) error {
	m.gotTool = toolName
	m.gotSum = summary
	m.gotCritical = critical
	return nil
}

// TestWriteFile 验证 write_file 的新建、覆盖、追加与越权拒绝。
func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	h := newTestWriteFile(dir)

	t.Run("新建文件", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"new.txt","content":"abc"}`)
		if err != nil {
			t.Fatalf("新建失败: %v", err)
		}
		if !strings.Contains(out, "已写入") {
			t.Errorf("返回应含已写入，实际: %q", out)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
		if string(data) != "abc" {
			t.Errorf("文件内容 = %q, want abc", string(data))
		}
	})

	t.Run("自动创建父目录", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"deep/nested/deep.txt","content":"x"}`); err != nil {
			t.Fatalf("写入带父目录文件失败: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "deep", "nested", "deep.txt")); err != nil {
			t.Errorf("父目录未自动创建: %v", err)
		}
	})

	t.Run("覆盖已存在文件", func(t *testing.T) {
		// 无审批钩子：直接覆盖
		if _, err := h.InvokableRun(context.Background(), `{"path":"new.txt","content":"replaced"}`); err != nil {
			t.Fatalf("覆盖失败: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
		if string(data) != "replaced" {
			t.Errorf("覆盖后内容 = %q, want replaced", string(data))
		}
	})

	t.Run("覆盖审批钩子生效", func(t *testing.T) {
		mock := &mockApprover{}
		hc := newTestWriteFile(dir)
		hc.ctx = &Context{Approver: mock}
		if _, err := hc.InvokableRun(context.Background(), `{"path":"new.txt","content":"approved"}`); err != nil {
			t.Fatalf("批准后覆盖失败: %v", err)
		}
		if mock.gotTool != "write_file" {
			t.Errorf("审批 toolName = %q, want write_file", mock.gotTool)
		}
		if !strings.Contains(mock.gotSum, "new.txt") {
			t.Errorf("审批摘要应含文件名，实际: %q", mock.gotSum)
		}
	})

	t.Run("追加写入", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"new.txt","content":"+suffix","append":true}`); err != nil {
			t.Fatalf("追加失败: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
		if string(data) != "approved+suffix" {
			t.Errorf("追加后内容 = %q, want approved+suffix", string(data))
		}
	})

	t.Run("越权路径拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"../evil.txt","content":"x"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})
}

// TestListDir 验证 list_dir 的目录结构列举与深度截断。
func TestListDir(t *testing.T) {
	dir := t.TempDir()
	// 结构：a.txt, sub/b.txt, sub/deep/c.txt
	mustWrite := func(p string) {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("a.txt")
	mustWrite("sub/b.txt")
	mustWrite("sub/deep/c.txt")

	t.Run("默认深度列出", func(t *testing.T) {
		out, err := newTestListDir(dir).InvokableRun(context.Background(), `{}`)
		if err != nil {
			t.Fatalf("list_dir 失败: %v", err)
		}
		if !strings.Contains(out, "文件: a.txt") {
			t.Errorf("应包含文件 a.txt，实际:\n%s", out)
		}
		if !strings.Contains(out, "目录: sub") {
			t.Errorf("应包含目录 sub 与文件 b.txt，实际:\n%s", out)
		}
		if !strings.Contains(out, "文件: sub"+string(filepath.Separator)+"b.txt") {
			t.Errorf("应包含 sub/b.txt，实际:\n%s", out)
		}
		// 默认深度 2：sub/deep/c.txt 不显示
		if strings.Contains(out, "c.txt") {
			t.Errorf("默认深度不应列出深层 c.txt，实际:\n%s", out)
		}
	})

	t.Run("depth=3 展开深层", func(t *testing.T) {
		out, err := newTestListDir(dir).InvokableRun(context.Background(), `{"depth":3}`)
		if err != nil {
			t.Fatalf("list_dir 失败: %v", err)
		}
		if !strings.Contains(out, "文件: sub"+string(filepath.Separator)+"deep"+string(filepath.Separator)+"c.txt") {
			t.Errorf("depth=3 应列出 c.txt，实际:\n%s", out)
		}
	})

	t.Run("越权路径拒绝", func(t *testing.T) {
		if _, err := newTestListDir(dir).InvokableRun(context.Background(), `{"path":"../secret"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})

	t.Run("depth 上限截断", func(t *testing.T) {
		out, err := newTestListDir(dir).InvokableRun(context.Background(), `{"depth":999}`)
		if err != nil {
			t.Fatalf("list_dir 失败: %v", err)
		}
		if !strings.Contains(out, "c.txt") {
			t.Errorf("depth 超上限应按 5 展开，应列出 c.txt，实际:\n%s", out)
		}
	})
}

// jsonQuote 给字符串添加 JSON 双引号引号包裹（测试用简单转义）。
func jsonQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}
