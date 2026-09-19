package tools

// 本文件覆盖复制/移动工具（copy_item / move_item）的核心行为。
// 测试通过 fsToolBase.workspaceRoot 注入临时目录，避免污染真实 ~/.jot/workspace。

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// newTestCopyItem / newTestMoveItem / newTestDeleteItem 构造注入临时工作目录的测试实例。
func newTestCopyItem(dir string) *copyItemTool {
	return &copyItemTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}
func newTestMoveItem(dir string) *moveItemTool {
	return &moveItemTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}
func newTestDeleteItem(dir string) *deleteItemTool {
	return &deleteItemTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}
func newTestMkdirDir(dir string) *mkdirDirTool {
	return &mkdirDirTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
}

// TestCopyItem 验证复制：新建复制、复制到已存在目录（自动追加文件名）、
// 缺省拒绝覆盖、overwrite=true 覆盖并审批、源不存在、路径逃逸、目录递归复制。
func TestCopyItem(t *testing.T) {
	dir := t.TempDir()
	h := newTestCopyItem(dir)

	t.Run("复制文件到新位置", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		out, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"backup.txt"}`)
		if err != nil {
			t.Fatalf("复制失败: %v", err)
		}
		if !strings.Contains(out, "a.txt") || !strings.Contains(out, "backup.txt") {
			t.Errorf("反馈应含源与目标，实际: %q", out)
		}
		if got := readTestFile(t, dir, "backup.txt"); got != "hello" {
			t.Errorf("复制后内容 = %q", got)
		}
	})

	t.Run("复制到已存在目录自动追加文件名", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"sub/"}`); err != nil {
			t.Fatalf("复制到目录失败: %v", err)
		}
		if got := readTestFile(t, dir, "sub/a.txt"); got != "hello" {
			t.Errorf("目录内副本内容 = %q", got)
		}
	})

	t.Run("目标已存在缺省拒绝覆盖", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		writeTestFile(t, dir, "b.txt", "keep")
		_, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"b.txt"}`)
		if err == nil {
			t.Fatal("目标已存在未设 overwrite 应报错")
		}
		if !strings.Contains(err.Error(), "已存在") {
			t.Errorf("错误应含已存在提示，实际: %v", err)
		}
		if got := readTestFile(t, dir, "b.txt"); got != "keep" {
			t.Errorf("拒绝后目标不应被改动 = %q", got)
		}
	})

	t.Run("overwrite=true 覆盖并触发审批", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "new")
		writeTestFile(t, dir, "b.txt", "old")
		mock := &mockApprover{}
		hc := newTestCopyItem(dir)
		hc.ctx = &Context{Approver: mock}
		if _, err := hc.InvokableRun(context.Background(), `{"source":"a.txt","dest":"b.txt","overwrite":true}`); err != nil {
			t.Fatalf("覆盖复制失败: %v", err)
		}
		if mock.gotTool != "copy_item" {
			t.Errorf("覆盖复制应触发审批，实际 toolName = %q", mock.gotTool)
		}
		if mock.gotCritical {
			t.Error("copy_item 审批 critical 应为 false")
		}
		if got := readTestFile(t, dir, "b.txt"); got != "new" {
			t.Errorf("覆盖后目标内容 = %q", got)
		}
	})

	t.Run("新建复制不触发审批", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		mock := &mockApprover{}
		hc := newTestCopyItem(dir)
		hc.ctx = &Context{Approver: mock}
		if _, err := hc.InvokableRun(context.Background(), `{"source":"a.txt","dest":"c.txt"}`); err != nil {
			t.Fatalf("新建复制失败: %v", err)
		}
		if mock.gotTool != "" {
			t.Errorf("新建复制不应触发审批，实际 toolName = %q", mock.gotTool)
		}
	})

	t.Run("源不存在报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"source":"nope.txt","dest":"c.txt"}`)
		if err == nil {
			t.Fatal("源不存在应报错")
		}
		if !strings.Contains(err.Error(), "不存在") {
			t.Errorf("错误应含不存在提示，实际: %v", err)
		}
	})

	t.Run("路径逃逸拒绝", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		if _, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"../evil.txt"}`); err == nil {
			t.Fatal("目标越权路径应报错")
		}
		if _, err := h.InvokableRun(context.Background(), `{"source":"../evil.txt","dest":"c.txt"}`); err == nil {
			t.Fatal("源越权路径应报错")
		}
	})

	t.Run("复制目录递归", func(t *testing.T) {
		writeTestFile(t, dir, "srcDir/x.txt", "x")
		writeTestFile(t, dir, "srcDir/sub/y.txt", "y")
		if _, err := h.InvokableRun(context.Background(), `{"source":"srcDir","dest":"dstDir"}`); err != nil {
			t.Fatalf("复制目录失败: %v", err)
		}
		if got := readTestFile(t, dir, "dstDir/x.txt"); got != "x" {
			t.Errorf("目录内文件 x 内容 = %q", got)
		}
		if got := readTestFile(t, dir, "dstDir/sub/y.txt"); got != "y" {
			t.Errorf("递归子目录文件 y 内容 = %q", got)
		}
	})

	t.Run("目录复制到自身子目录拒绝", func(t *testing.T) {
		writeTestFile(t, dir, "sdir/x.txt", "x")
		_, err := h.InvokableRun(context.Background(), `{"source":"sdir","dest":"sdir/sub"}`)
		if err == nil {
			t.Fatal("复制目录到自身子目录应报错")
		}
		if !strings.Contains(err.Error(), "子目录") {
			t.Errorf("错误应含子目录提示，实际: %v", err)
		}
	})

	t.Run("源与目标相同拒绝", func(t *testing.T) {
		writeTestFile(t, dir, "same.txt", "x")
		_, err := h.InvokableRun(context.Background(), `{"source":"same.txt","dest":"same.txt","overwrite":true}`)
		if err == nil {
			t.Fatal("源与目标相同应报错")
		}
		if !strings.Contains(err.Error(), "相同") {
			t.Errorf("错误应含相同提示，实际: %v", err)
		}
	})

	t.Run("大小写变体自复制拒绝(Windows)", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("仅 Windows 大小写不敏感文件系统")
		}
		writeTestFile(t, dir, "casedir/x.txt", "x")
		upper := strings.ToUpper(filepath.Join(dir, "casedir"))
		_, err := h.InvokableRun(context.Background(), `{"source":"casedir","dest":`+strconv.Quote(upper+"/sub")+`}`)
		if err == nil {
			t.Fatal("大小写变体路径的目录自复制应被拦截")
		}
		if !strings.Contains(err.Error(), "子目录") {
			t.Errorf("错误应含子目录提示，实际: %v", err)
		}
	})
}

// TestMoveItem 验证移动：移动到新位置（源消失）、移动到已存在目录（自动追加
// 文件名）、缺省拒绝覆盖、overwrite=true 覆盖、源不存在、路径逃逸、目录移动、
// 无论是否覆盖始终触发审批。
func TestMoveItem(t *testing.T) {
	dir := t.TempDir()
	h := newTestMoveItem(dir)

	t.Run("移动文件到新位置", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		out, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"moved.txt"}`)
		if err != nil {
			t.Fatalf("移动失败: %v", err)
		}
		if !strings.Contains(out, "a.txt") || !strings.Contains(out, "moved.txt") {
			t.Errorf("反馈应含源与目标，实际: %q", out)
		}
		if got := readTestFile(t, dir, "moved.txt"); got != "hello" {
			t.Errorf("移动后目标内容 = %q", got)
		}
		if _, err := os.Lstat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
			t.Errorf("移动后源应消失，实际 Lstat err = %v", err)
		}
	})

	t.Run("移动到已存在目录自动追加文件名", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"sub/"}`); err != nil {
			t.Fatalf("移动到目录失败: %v", err)
		}
		if got := readTestFile(t, dir, "sub/a.txt"); got != "hello" {
			t.Errorf("目录内目标内容 = %q", got)
		}
		if _, err := os.Lstat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
			t.Errorf("移动后源应消失，实际 Lstat err = %v", err)
		}
	})

	t.Run("目标已存在缺省拒绝覆盖", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		writeTestFile(t, dir, "b.txt", "keep")
		_, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"b.txt"}`)
		if err == nil {
			t.Fatal("目标已存在未设 overwrite 应报错")
		}
		if !strings.Contains(err.Error(), "已存在") {
			t.Errorf("错误应含已存在提示，实际: %v", err)
		}
		// 拒绝后源与目标都应保持原样
		if got := readTestFile(t, dir, "a.txt"); got != "hello" {
			t.Errorf("拒绝后源不应被移除 = %q", got)
		}
		if got := readTestFile(t, dir, "b.txt"); got != "keep" {
			t.Errorf("拒绝后目标不应被改动 = %q", got)
		}
	})

	t.Run("overwrite=true 覆盖", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "new")
		writeTestFile(t, dir, "b.txt", "old")
		if _, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"b.txt","overwrite":true}`); err != nil {
			t.Fatalf("覆盖移动失败: %v", err)
		}
		if got := readTestFile(t, dir, "b.txt"); got != "new" {
			t.Errorf("覆盖后目标内容 = %q", got)
		}
		if _, err := os.Lstat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
			t.Errorf("移动后源应消失，实际 Lstat err = %v", err)
		}
	})

	t.Run("移动始终触发审批（含目标不存在）", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		mock := &mockApprover{}
		hc := newTestMoveItem(dir)
		hc.ctx = &Context{Approver: mock}
		if _, err := hc.InvokableRun(context.Background(), `{"source":"a.txt","dest":"c.txt"}`); err != nil {
			t.Fatalf("移动失败: %v", err)
		}
		if mock.gotTool != "move_item" {
			t.Errorf("移动应始终触发审批，实际 toolName = %q", mock.gotTool)
		}
		if mock.gotCritical {
			t.Error("move_item 审批 critical 应为 false")
		}
	})

	t.Run("源不存在报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"source":"nope.txt","dest":"c.txt"}`)
		if err == nil {
			t.Fatal("源不存在应报错")
		}
	})

	t.Run("路径逃逸拒绝", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		if _, err := h.InvokableRun(context.Background(), `{"source":"a.txt","dest":"../evil.txt"}`); err == nil {
			t.Fatal("目标越权路径应报错")
		}
		if _, err := h.InvokableRun(context.Background(), `{"source":"../evil.txt","dest":"c.txt"}`); err == nil {
			t.Fatal("源越权路径应报错")
		}
	})

	t.Run("移动目录", func(t *testing.T) {
		writeTestFile(t, dir, "srcDir/x.txt", "x")
		if _, err := h.InvokableRun(context.Background(), `{"source":"srcDir","dest":"dstDir"}`); err != nil {
			t.Fatalf("移动目录失败: %v", err)
		}
		if got := readTestFile(t, dir, "dstDir/x.txt"); got != "x" {
			t.Errorf("移动后目录内文件内容 = %q", got)
		}
		if _, err := os.Lstat(filepath.Join(dir, "srcDir")); !os.IsNotExist(err) {
			t.Errorf("移动后源目录应消失，实际 Lstat err = %v", err)
		}
	})

	t.Run("移动到自身子目录拒绝", func(t *testing.T) {
		writeTestFile(t, dir, "mdir/x.txt", "x")
		_, err := h.InvokableRun(context.Background(), `{"source":"mdir","dest":"mdir/sub"}`)
		if err == nil {
			t.Fatal("移动到自身子目录应报错")
		}
		if !strings.Contains(err.Error(), "子目录") {
			t.Errorf("错误应含子目录提示，实际: %v", err)
		}
	})

	t.Run("缺少参数报错", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"dest":"c.txt"}`); err == nil {
			t.Fatal("缺少 source 应报错")
		}
		if _, err := h.InvokableRun(context.Background(), `{"source":"a.txt"}`); err == nil {
			t.Fatal("缺少 dest 应报错")
		}
	})
}

// TestDeleteItem 验证删除：删除文件、删除不存在、目录需 recursive 才递归删除、
// 空目录缺省可删、递归删除、逃逸拒绝、拒绝删工作区根、强制审批 critical=true。
func TestDeleteItem(t *testing.T) {
	dir := t.TempDir()
	h := newTestDeleteItem(dir)

	t.Run("删除文件", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		out, err := h.InvokableRun(context.Background(), `{"path":"a.txt"}`)
		if err != nil {
			t.Fatalf("删除失败: %v", err)
		}
		if !strings.Contains(out, "a.txt") {
			t.Errorf("反馈应含路径，实际: %q", out)
		}
		if _, err := os.Lstat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
			t.Errorf("删除后文件应消失，实际 Lstat err = %v", err)
		}
	})

	t.Run("删除不存在报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"nope.txt"}`)
		if err == nil {
			t.Fatal("删除不存在应报错")
		}
		if !strings.Contains(err.Error(), "不存在") {
			t.Errorf("错误应含不存在提示，实际: %v", err)
		}
	})

	t.Run("非空目录缺省拒绝递归删除", func(t *testing.T) {
		writeTestFile(t, dir, "d/x.txt", "x")
		_, err := h.InvokableRun(context.Background(), `{"path":"d"}`)
		if err == nil {
			t.Fatal("非空目录未设 recursive 应报错")
		}
		if !strings.Contains(err.Error(), "recursive") {
			t.Errorf("错误应引导 recursive，实际: %v", err)
		}
		if got := readTestFile(t, dir, "d/x.txt"); got != "x" {
			t.Errorf("拒绝后目录内容应保留，实际: %q", got)
		}
	})

	t.Run("空目录缺省可删", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := h.InvokableRun(context.Background(), `{"path":"empty"}`); err != nil {
			t.Fatalf("删除空目录失败: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(dir, "empty")); !os.IsNotExist(err) {
			t.Errorf("删除后空目录应消失，实际 Lstat err = %v", err)
		}
	})

	t.Run("recursive=true 递归删除非空目录", func(t *testing.T) {
		writeTestFile(t, dir, "d/sub/x.txt", "x")
		if _, err := h.InvokableRun(context.Background(), `{"path":"d","recursive":true}`); err != nil {
			t.Fatalf("递归删除失败: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(dir, "d")); !os.IsNotExist(err) {
			t.Errorf("递归删除后目录应消失，实际 Lstat err = %v", err)
		}
	})

	t.Run("路径逃逸拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"../evil.txt"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})

	t.Run("拒绝删除工作区根", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":".","recursive":true}`)
		if err == nil {
			t.Fatal("删除工作区根应报错")
		}
		if !strings.Contains(err.Error(), "根目录") {
			t.Errorf("错误应含根目录提示，实际: %v", err)
		}
	})

	t.Run("大小写变体根保护拒绝(Windows)", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("仅 Windows 大小写不敏感文件系统")
		}
		hc := newTestDeleteItem(strings.ToUpper(dir))
		_, err := hc.InvokableRun(context.Background(), `{"path":`+strconv.Quote(strings.ToLower(dir))+`,"recursive":true}`)
		if err == nil {
			t.Fatal("大小写变体的工作区根应被拒绝删除")
		}
		if !strings.Contains(err.Error(), "根目录") {
			t.Errorf("错误应含根目录提示，实际: %v", err)
		}
	})

	t.Run("删除触发强制审批 critical=true", func(t *testing.T) {
		writeTestFile(t, dir, "a.txt", "hello")
		mock := &mockApprover{}
		hc := newTestDeleteItem(dir)
		hc.ctx = &Context{Approver: mock}
		if _, err := hc.InvokableRun(context.Background(), `{"path":"a.txt"}`); err != nil {
			t.Fatalf("删除失败: %v", err)
		}
		if mock.gotTool != "delete_item" {
			t.Errorf("删除应触发审批，实际 toolName = %q", mock.gotTool)
		}
		if !mock.gotCritical {
			t.Error("delete_item 审批 critical 应为 true（不可绕过）")
		}
	})

	t.Run("缺少 path 报错", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"recursive":true}`); err == nil {
			t.Fatal("缺少 path 应报错")
		}
	})
}

// TestMkdirDir 验证建目录：单层/递归创建、已存在幂等成功、同名文件报错、
// 逃逸拒绝、不触发审批。
func TestMkdirDir(t *testing.T) {
	dir := t.TempDir()
	h := newTestMkdirDir(dir)

	t.Run("创建单层目录", func(t *testing.T) {
		out, err := h.InvokableRun(context.Background(), `{"path":"docs"}`)
		if err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if !strings.Contains(out, "docs") {
			t.Errorf("反馈应含路径，实际: %q", out)
		}
		st, err := os.Lstat(filepath.Join(dir, "docs"))
		if err != nil || !st.IsDir() {
			t.Errorf("docs 应为已创建目录，Lstat err = %v", err)
		}
	})

	t.Run("默认单级父目录不存在报错", func(t *testing.T) {
		_, err := h.InvokableRun(context.Background(), `{"path":"x/y"}`)
		if err == nil {
			t.Fatal("父目录不存在且未递归应报错")
		}
		if !strings.Contains(err.Error(), "父目录不存在") {
			t.Errorf("错误应含父目录不存在提示，实际: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(dir, "x")); !os.IsNotExist(err) {
			t.Errorf("失败时不应残留父级目录，Lstat err = %v", err)
		}
	})

	t.Run("默认单级父目录已存在成功", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(dir, "parent"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := h.InvokableRun(context.Background(), `{"path":"parent/child"}`); err != nil {
			t.Fatalf("父目录已存在的单级创建应成功: %v", err)
		}
		st, err := os.Lstat(filepath.Join(dir, "parent", "child"))
		if err != nil || !st.IsDir() {
			t.Errorf("parent/child 应为已创建目录，Lstat err = %v", err)
		}
	})

	t.Run("recursive 递归创建多层", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"a/b/c","recursive":true}`); err != nil {
			t.Fatalf("递归创建失败: %v", err)
		}
		for _, p := range []string{"a", "a/b", "a/b/c"} {
			st, err := os.Lstat(filepath.Join(dir, p))
			if err != nil || !st.IsDir() {
				t.Errorf("%s 应为已创建目录，Lstat err = %v", p, err)
			}
		}
	})

	t.Run("已存在目录幂等成功", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(dir, "exist"), 0o755); err != nil {
			t.Fatal(err)
		}
		out, err := h.InvokableRun(context.Background(), `{"path":"exist"}`)
		if err != nil {
			t.Fatalf("已存在目录应幂等成功: %v", err)
		}
		if !strings.Contains(out, "已存在") {
			t.Errorf("已存在目录反馈应含已存在提示，实际: %q", out)
		}
	})

	t.Run("同名文件报错", func(t *testing.T) {
		writeTestFile(t, dir, "afile", "hello")
		_, err := h.InvokableRun(context.Background(), `{"path":"afile"}`)
		if err == nil {
			t.Fatal("目标为已存在文件应报错")
		}
		if !strings.Contains(err.Error(), "不是目录") {
			t.Errorf("错误应含不是目录提示，实际: %v", err)
		}
	})

	t.Run("路径逃逸拒绝", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{"path":"../evil"}`); err == nil {
			t.Fatal("越权路径应报错")
		}
	})

	t.Run("缺少 path 报错", func(t *testing.T) {
		if _, err := h.InvokableRun(context.Background(), `{}`); err == nil {
			t.Fatal("缺少 path 应报错")
		}
	})

	t.Run("建目录不触发审批", func(t *testing.T) {
		mock := &mockApprover{}
		hc := newTestMkdirDir(dir)
		hc.ctx = &Context{Approver: mock}
		if _, err := hc.InvokableRun(context.Background(), `{"path":"noapprove"}`); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if mock.gotTool != "" {
			t.Errorf("建目录不应触发审批，实际 toolName = %q", mock.gotTool)
		}
	})
}
