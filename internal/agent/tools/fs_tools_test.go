package tools

// 本文件覆盖工作目录文件工具（read_file / write_file / ls_dir）的核心行为。
// 测试通过 fsToolBase.workspaceRoot 注入临时目录，避免污染真实 ~/.jot/workspace。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestReadFile / newTestWriteFile / newTestLsDir 构造注入临时工作目录的测试实例。
func newTestReadFile(dir string) *readFileTool {
	return &readFileTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}
func newTestWriteFile(dir string) *writeFileTool {
	return &writeFileTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}
func newTestLsDir(dir string) *lsDirTool {
	return &lsDirTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
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
	t.Run("二进制文件跳过", func(t *testing.T) {
		// 含 NUL 字节的文件经 go-kit fs 判为二进制，读取应跳过并提示而非输出乱码
		if err := os.WriteFile(filepath.Join(dir, "bin.dat"), []byte("PK\x00\x01\x02data"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := h.InvokableRun(context.Background(), `{"path":"bin.dat"}`)
		if err != nil {
			t.Fatalf("二进制文件读取失败: %v", err)
		}
		if !strings.Contains(out, "文件为二进制，已跳过读取") {
			t.Errorf("二进制文件应返回跳过提示，实际: %q", out)
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

// TestLsDir 验证 ls_dir 的单层列表（像 ls）与按 path 钻取、越权拒绝。
func TestLsDir(t *testing.T) {
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

	t.Run("缺省列出根的直接内容（单层）", func(t *testing.T) {
		out, err := newTestLsDir(dir).InvokableRun(context.Background(), `{}`)
		if err != nil {
			t.Fatalf("ls_dir 失败: %v", err)
		}
		if !strings.Contains(out, "文件: a.txt") {
			t.Errorf("应包含文件 a.txt，实际:\n%s", out)
		}
		if !strings.Contains(out, "目录: sub") {
			t.Errorf("应包含目录 sub，实际:\n%s", out)
		}
		// 首行路径锚点：根目录显示 "."
		if !strings.Contains(out, "当前目录: .") {
			t.Errorf("根目录应显示当前目录锚点 当前目录: .，实际:\n%s", out)
		}
		// 单层语义：不列出 sub 及其子目录内的任何文件
		if strings.Contains(out, "sub"+string(filepath.Separator)+"b.txt") || strings.Contains(out, "c.txt") {
			t.Errorf("单层列表不应含深层文件，实际:\n%s", out)
		}
	})

	t.Run("以子目录为 path 钻取", func(t *testing.T) {
		out, err := newTestLsDir(dir).InvokableRun(context.Background(), `{"path":"sub"}`)
		if err != nil {
			t.Fatalf("ls_dir 失败: %v", err)
		}
		if !strings.Contains(out, "文件: b.txt") {
			t.Errorf("path=sub 应列出 b.txt，实际:\n%s", out)
		}
		if !strings.Contains(out, "目录: deep") {
			t.Errorf("path=sub 应列出子目录 deep，实际:\n%s", out)
		}
		// 首行路径锚点：path=sub 显示 "sub"
		if !strings.Contains(out, "当前目录: sub") {
			t.Errorf("path=sub 应显示当前目录锚点 当前目录: sub，实际:\n%s", out)
		}
		if strings.Contains(out, "c.txt") {
			t.Errorf("path=sub 单层不应列出 deep/c.txt，实际:\n%s", out)
		}
	})

	t.Run("多层子目录钻取", func(t *testing.T) {
		// 回归：Windows 上 filepath.Rel 返回反斜杠 rel（如 sub\deep），os.Root.FS()
		// 适配器仅接受 "/" 分隔，曾致 readdir invalid argument（见 tools_log 50 次路径失败）
		out, err := newTestLsDir(dir).InvokableRun(context.Background(), `{"path":"sub/deep"}`)
		if err != nil {
			t.Fatalf("ls_dir 失败: %v", err)
		}
		if !strings.Contains(out, "文件: c.txt") {
			t.Errorf("path=sub/deep 应列出 c.txt，实际:\n%s", out)
		}
		// 首行路径锚点：显示 "sub/deep"（relDisplayPath 输出 OS 原生分隔符）
		if !strings.Contains(out, "当前目录: "+filepath.Join("sub", "deep")) {
			t.Errorf("多层钻取应显示当前目录锚点 sub/deep，实际:\n%s", out)
		}
	})

	t.Run("越权路径拒绝", func(t *testing.T) {
		if _, err := newTestLsDir(dir).InvokableRun(context.Background(), `{"path":"../secret"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})

	t.Run("detail=true 附大小与修改时间", func(t *testing.T) {
		out, err := newTestLsDir(dir).InvokableRun(context.Background(), `{"detail":true}`)
		if err != nil {
			t.Fatalf("ls_dir 失败: %v", err)
		}
		if !strings.Contains(out, "文件: a.txt  大小: 1 B  修改: ") {
			t.Errorf("详情输出应含文件大小与修改时间前缀，实际:\n%s", out)
		}
		if !strings.Contains(out, "目录: sub  大小: -  修改: ") {
			t.Errorf("目录详情输出大小应为 -，实际:\n%s", out)
		}
	})

	t.Run("缺省 detail 不含大小与修改时间", func(t *testing.T) {
		out, err := newTestLsDir(dir).InvokableRun(context.Background(), `{"path":"sub"}`)
		if err != nil {
			t.Fatalf("ls_dir 失败: %v", err)
		}
		if strings.Contains(out, "大小:") || strings.Contains(out, "修改:") {
			t.Errorf("未指定 detail 不应出现大小/修改时间，实际:\n%s", out)
		}
	})
}

// TestReadFileLineNumbers 验证 read_file 的 line_numbers 输出：缺省无前缀、
// line_numbers=true 带「行 N: 」全局行号前缀、与分页组合时起始行号正确。
func TestReadFileLineNumbers(t *testing.T) {
	dir := t.TempDir()
	content := "第一行\n第二行\n第三行"
	if err := os.WriteFile(filepath.Join(dir, "ln.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newTestReadFile(dir)

	t.Run("缺省无行号前缀", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"ln.txt"}`)
		if err != nil {
			t.Fatalf("读取失败: %v", err)
		}
		if strings.Contains(out, "行 1:") {
			t.Errorf("缺省不应带行号前缀，实际: %q", out)
		}
	})
	t.Run("line_numbers=true 带全局行号", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"ln.txt","line_numbers":true}`)
		if err != nil {
			t.Fatalf("读取失败: %v", err)
		}
		if !strings.Contains(out, "行 1: 第一行") || !strings.Contains(out, "行 3: 第三行") {
			t.Errorf("行号输出不符，实际:\n%s", out)
		}
	})
	t.Run("分页组合起始行号正确（offset 落在行中间）", func(t *testing.T) {
		// offset=2 跳过 "第一" 两个 rune，从第一行剩余 "行" 起读，起始行号应为 1
		out, err := h.InvokableRun(context.Background(), `{"path":"ln.txt","offset":2,"length":10,"line_numbers":true}`)
		if err != nil {
			t.Fatalf("读取失败: %v", err)
		}
		if !strings.Contains(out, "行 1: 行") {
			t.Errorf("起始行号应为 1，实际:\n%s", out)
		}
	})
	t.Run("offset 跨行起始行号正确", func(t *testing.T) {
		// offset=4 跳过 "第一行\n" 4 个 rune，从第 2 行起读，起始行号应为 2
		out, err := h.InvokableRun(context.Background(), `{"path":"ln.txt","offset":4,"length":10,"line_numbers":true}`)
		if err != nil {
			t.Fatalf("读取失败: %v", err)
		}
		if !strings.Contains(out, "行 2: 第二行") {
			t.Errorf("起始行号应为 2，实际:\n%s", out)
		}
	})
	t.Run("结尾换行文件行号与续读一致", func(t *testing.T) {
		// 文件以 \n 结尾：全读时 splitNoteLines 剔除末尾空行，只编号真实内容行；
		// 从 offset=4（跳过 "a\nb\n"）续读时起始行号 3 与全读编号保持自洽
		content := "a\nb\nc\n"
		if err := os.WriteFile(filepath.Join(dir, "lnlf.txt"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		hh := newTestReadFile(dir)
		full, err := hh.InvokableRun(context.Background(), `{"path":"lnlf.txt","line_numbers":true}`)
		if err != nil {
			t.Fatalf("全读失败: %v", err)
		}
		if !strings.Contains(full, "行 1: a") || !strings.Contains(full, "行 2: b") || !strings.Contains(full, "行 3: c") {
			t.Errorf("全读行号不符，实际:\n%s", full)
		}
		if strings.Contains(full, "行 4:") {
			t.Errorf("结尾空行不应编号，实际:\n%s", full)
		}
		next, err := hh.InvokableRun(context.Background(), `{"path":"lnlf.txt","offset":4,"length":10,"line_numbers":true}`)
		if err != nil {
			t.Fatalf("续读失败: %v", err)
		}
		if !strings.Contains(next, "行 3: c") {
			t.Errorf("续读起始行号应为 3，实际:\n%s", next)
		}
	})
}

// jsonQuote 给字符串添加 JSON 双引号引号包裹（测试用简单转义）。
func jsonQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}

// TestResolvePathTildeExpansion 验证 ~ 前缀路径展开：~/.jot/workspace/... 形式
// 可直接命中工作区内文件，~/ 其它目录与 ~ 单独越界拒绝。
func TestResolvePathTildeExpansion(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".jot", "workspace")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "tilde ok"
	if err := os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &readFileTool{fsToolBase: fsToolBase{workspaceRoot: root, homeDir: home}}

	t.Run("~/.jot/workspace/ 完整形式命中", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"~/.jot/workspace/sub/a.txt"}`)
		if err != nil {
			t.Fatalf("读取失败: %v", err)
		}
		if out != content {
			t.Errorf("读取内容 = %q, want %q", out, content)
		}
	})

	t.Run("~/ 其它子目录越界拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"~/other.txt"}`); err == nil {
			t.Error("~/other.txt 越界应报错")
		}
	})

	t.Run("~ 单独越界拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"~"}`); err == nil {
			t.Error("~ 单独越界应报错")
		}
	})
}

// TestExpandTilde 验证 expandTilde 的展开规则：仅前导 ~（~、~/、~\）展开为家
// 目录，路径中间的 ~ 与普通相对路径原样保留。
func TestExpandTilde(t *testing.T) {
	home := t.TempDir()
	b := fsToolBase{homeDir: home}

	cases := []struct {
		in   string
		want string
	}{
		{"~", home},
		{"~/.jot/workspace/foo.txt", filepath.Join(home, ".jot", "workspace", "foo.txt")},
		{"~\\.jot\\workspace\\a.txt", filepath.Join(home, ".jot\\workspace\\a.txt")},
		{"notes/~x.txt", "notes/~x.txt"},
		{"sub/a.txt", "sub/a.txt"},
	}
	for _, c := range cases {
		got, err := b.expandTilde(c.in)
		if err != nil {
			t.Fatalf("expandTilde(%q) 意外错误: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("expandTilde(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
