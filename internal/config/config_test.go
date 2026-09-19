package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSubDir 验证 SubDir 返回 ~/.jot/<sub> 路径。
func TestSubDir(t *testing.T) {
	root, err := JotHomeDir()
	if err != nil {
		t.Fatalf("JotHomeDir() 意外错误: %v", err)
	}
	got, err := SubDir(DirData)
	if err != nil {
		t.Fatalf("SubDir(%q) 意外错误: %v", DirData, err)
	}
	want := filepath.Join(root, DirData)
	if got != want {
		t.Errorf("SubDir(%q) = %q, want %q", DirData, got, want)
	}
}

// TestWorkspaceFilePath 覆盖 WorkspaceFilePath 的边界校验：
// 相对路径合法、绝对路径在 workspace 内、../ 逃逸拒绝、绝对路径越界拒绝、
// 根目录自身视为合法。
func TestWorkspaceFilePath(t *testing.T) {
	// 构造一个真实绝对路径作工作区根（Abs 解析到当前盘符，Windows 下也正确）
	root, _ := filepath.Abs(filepath.Join(string(filepath.Separator), "tmp", "jot-ws"))
	sep := string(filepath.Separator)

	legal := []struct {
		name string
		p    string
		want string
	}{
		{"相对路径", "a.txt", filepath.Join(root, "a.txt")},
		{"相对子目录", "sub" + sep + "b.txt", filepath.Join(root, "sub"+sep+"b.txt")},
		{"绝对路径在工作区内", root + sep + "c.txt", root + sep + "c.txt"},
		{"工作区根目录自身", root, root},
		{"点相对", "." + sep + "x.txt", filepath.Join(root, "x.txt")},
	}
	for _, c := range legal {
		t.Run("合法_"+c.name, func(t *testing.T) {
			got, err := WorkspaceFilePath(root, c.p)
			if err != nil {
				t.Fatalf("WorkspaceFilePath(%q) 意外错误: %v", c.p, err)
			}
			if !filepath.IsAbs(got) || got != c.want {
				t.Errorf("WorkspaceFilePath(%q) = %q, want %q", c.p, got, c.want)
			}
		})
	}

	rejected := []string{
		"../escape.txt",
		"sub/../../escape.txt",
		root + sep + ".." + sep + "outside.txt",
		".." + sep + ".." + sep + "xxx",
		// 外来/畸形绝对路径：Windows 上 filepath.IsAbs("/home/...") 为 false，
		// 曾被当相对路径拼进沙箱根产生幽灵路径；现直接报格式错误
		"/home/user/.jot/workspace/x.txt",
		`\home\user\x.txt`,
	}
	for _, p := range rejected {
		// 子测试名清洗分隔符：路径含 "/" 会被 Go 测试输出按层级展开，替换为 "_" 保持可读
		name := strings.NewReplacer("/", "_", `\`, "_").Replace(p)
		t.Run("拒绝_"+name, func(t *testing.T) {
			if _, err := WorkspaceFilePath(root, p); err == nil {
				t.Errorf("WorkspaceFilePath(%q) 应返回越界错误", p)
			}
		})
	}
}

// TestWorkspaceFilePathSymlinkEscape 验证符号链接逃逸边界校验：
// 在临时 root 下创建指向外部临时目录的 symlink，经 symlink 访问外部文件应被拒绝，
// 直接访问 root 内文件仍放行。os.Symlink 不可用（如 Windows 需管理员权限）时跳过。
func TestWorkspaceFilePathSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	// 在外部目录放置一个"本不该被访问"的文件
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 在 root 内放一个正常文件，用于验证"直接访问仍放行"
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("os.Symlink 不可用（%v，Windows 需管理员权限时可能失败），跳过符号链接逃逸测试", err)
	}

	// 经 symlink 访问外部文件：目标本身存在（EvalSymlinks 直接解析到外部）
	t.Run("经 symlink 读外部现存文件拒绝", func(t *testing.T) {
		if _, err := WorkspaceFilePath(root, filepath.Join("link", "secret.txt")); err == nil {
			t.Fatal("经 symlink 访问外部文件应被拒绝")
		}
	})

	// 经 symlink 访问外部"尚不存在"的新文件（父目录是 symlink）：也应被拒绝
	t.Run("经 symlink 写外部新文件拒绝", func(t *testing.T) {
		if _, err := WorkspaceFilePath(root, filepath.Join("link", "new.txt")); err == nil {
			t.Fatal("经 symlink 目录写入外部新文件应被拒绝")
		}
	})

	// 直接访问 root 内文件：仍放行
	t.Run("直接访问 root 内文件放行", func(t *testing.T) {
		got, err := WorkspaceFilePath(root, "a.txt")
		if err != nil {
			t.Fatalf("直接访问 root 内文件应放行，got %v", err)
		}
		if got != filepath.Join(root, "a.txt") {
			t.Errorf("返回路径 = %q, want %q", got, filepath.Join(root, "a.txt"))
		}
	})
}
