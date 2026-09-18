package services

// 本文件覆盖工作区管理器（WorkspaceService）的单元测试：
//  1. 上传重名自动改名（文件/目录/无扩展名/隐藏文件保持原名）；
//  2. 文件树结构与排序与空目录隐藏（自底向上修剪）；
//  3. 目录递归上传/下载/删除（roundtrip，含下载重名自动改名）；
//  4. 工作区根目录删除被拒（空、.、/）；
//  5. ../ 逃逸路径被拒（删除与下载源）；
//  6. 下载到桌面重名自动改名（文件与嵌套路径）；
//  7. 批量操作单项失败不中断批次（上传/下载/删除）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupWorkspaceService 构造测试用 WorkspaceService：注入工作区根/家目录/桌面目录
// （桌面 = home/Desktop），并创建两端目录。
func setupWorkspaceService(t *testing.T) (*WorkspaceService, string, string) {
	t.Helper()
	home := t.TempDir()
	ws := filepath.Join(home, ".jot", "workspace")
	desk := filepath.Join(home, "Desktop")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(desk, 0o755); err != nil {
		t.Fatal(err)
	}
	svc := &WorkspaceService{workspaceRoot: ws, homeDir: home, desktopRoot: desk}
	return svc, ws, desk
}

// mustWrite 测试辅助：写文件，失败即终止。
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mustMkdir 测试辅助：创建目录，失败即终止。
func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// readFile 测试辅助：读取文件内容，失败即终止。
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件 %s 失败: %v", path, err)
	}
	return string(b)
}

// findEntry 在文件树中按 / 分隔相对路径查找节点，未找到返回 nil。
func findEntry(tree []WorkspaceFileEntry, rel string) *WorkspaceFileEntry {
	for i := range tree {
		if tree[i].RelPath == rel {
			return &tree[i]
		}
		if tree[i].IsDir {
			if e := findEntry(tree[i].Children, rel); e != nil {
				return e
			}
		}
	}
	return nil
}

// TestWorkspaceUploadAutoRename 上传重名自动改名：文件、目录、无扩展名文件、
// 隐藏文件（保持原名再追加序号）。
func TestWorkspaceUploadAutoRename(t *testing.T) {
	t.Run("文件重名自动改名", func(t *testing.T) {
		svc, ws, _ := setupWorkspaceService(t)
		mustWrite(t, filepath.Join(ws, "a.txt"), "old1")
		src := filepath.Join(t.TempDir(), "a.txt")
		mustWrite(t, src, "new")
		res, err := svc.UploadFiles([]string{src})
		if err != nil {
			t.Fatalf("上传失败: %v", err)
		}
		if len(res) != 1 || res[0].Error != "" {
			t.Fatalf("上传结果不符: %+v", res)
		}
		if res[0].Name != "a.txt" || res[0].Target != "a (1).txt" {
			t.Errorf("结果 Name=%q Target=%q, want a.txt / a (1).txt", res[0].Name, res[0].Target)
		}
		if got := readFile(t, filepath.Join(ws, "a (1).txt")); got != "new" {
			t.Errorf("改名后内容 = %q, want new", got)
		}
		if got := readFile(t, filepath.Join(ws, "a.txt")); got != "old1" {
			t.Errorf("原文件被改动 = %q, want old1", got)
		}
		// 再次上传同名文件 → a (2).txt
		if _, err := svc.UploadFiles([]string{src}); err != nil {
			t.Fatalf("二次上传失败: %v", err)
		}
		if got := readFile(t, filepath.Join(ws, "a (2).txt")); got != "new" {
			t.Errorf("二次改名后内容 = %q, want new", got)
		}
	})

	t.Run("目录重名自动改名", func(t *testing.T) {
		svc, ws, _ := setupWorkspaceService(t)
		mustMkdir(t, filepath.Join(ws, "dir"))
		mustWrite(t, filepath.Join(ws, "dir", "old.txt"), "old")
		src := filepath.Join(t.TempDir(), "dir")
		mustWrite(t, filepath.Join(src, "sub", "new.txt"), "new")
		res, err := svc.UploadDirectory(src)
		if err != nil {
			t.Fatalf("上传目录失败: %v", err)
		}
		if res.Error != "" || res.Target != "dir (1)" {
			t.Fatalf("上传结果不符: %+v", res)
		}
		if got := readFile(t, filepath.Join(ws, "dir (1)", "sub", "new.txt")); got != "new" {
			t.Errorf("改名目录内容 = %q, want new", got)
		}
		if _, err := os.Lstat(filepath.Join(ws, "dir", "old.txt")); err != nil {
			t.Errorf("原目录应保留: %v", err)
		}
	})

	t.Run("无扩展名文件自动改名", func(t *testing.T) {
		svc, ws, _ := setupWorkspaceService(t)
		mustWrite(t, filepath.Join(ws, "README"), "old")
		src := filepath.Join(t.TempDir(), "README")
		mustWrite(t, src, "new")
		res, err := svc.UploadFiles([]string{src})
		if err != nil || len(res) != 1 || res[0].Error != "" {
			t.Fatalf("上传失败: err=%v res=%+v", err, res)
		}
		if res[0].Target != "README (1)" {
			t.Errorf("Target = %q, want README (1)", res[0].Target)
		}
		if got := readFile(t, filepath.Join(ws, "README (1)")); got != "new" {
			t.Errorf("改名后内容 = %q, want new", got)
		}
	})

	t.Run("隐藏文件保持原名追加序号", func(t *testing.T) {
		svc, ws, _ := setupWorkspaceService(t)
		mustWrite(t, filepath.Join(ws, ".gitignore"), "old")
		src := filepath.Join(t.TempDir(), ".gitignore")
		mustWrite(t, src, "new")
		res, err := svc.UploadFiles([]string{src})
		if err != nil || len(res) != 1 || res[0].Error != "" {
			t.Fatalf("上传失败: err=%v res=%+v", err, res)
		}
		if res[0].Target != ".gitignore (1)" {
			t.Errorf("Target = %q, want .gitignore (1)", res[0].Target)
		}
		if got := readFile(t, filepath.Join(ws, ".gitignore (1)")); got != "new" {
			t.Errorf("改名后内容 = %q, want new", got)
		}
	})
}

// TestWorkspaceListTree 文件树：目录在前、名称升序、/ 分隔相对路径、文件大小与
// 修改时间、空目录隐藏（自底向上修剪，含"子目录全被修剪"的父目录）。
func TestWorkspaceListTree(t *testing.T) {
	svc, ws, _ := setupWorkspaceService(t)
	mustWrite(t, filepath.Join(ws, "b.txt"), "bb")
	mustWrite(t, filepath.Join(ws, "a.txt"), "a")
	mustMkdir(t, filepath.Join(ws, "dir"))
	mustWrite(t, filepath.Join(ws, "dir", "z.txt"), "zzzz")
	mustWrite(t, filepath.Join(ws, "dir", "a.txt"), "aa")
	mustMkdir(t, filepath.Join(ws, "dir2", "empty")) // 嵌套空目录链
	mustMkdir(t, filepath.Join(ws, "empty"))         // 顶层空目录

	tree, err := svc.ListWorkspaceFiles()
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}

	// 顶层排序：目录在前（dir），文件按名升序（a.txt, b.txt）；dir2/empty 被修剪
	var topRel []string
	for _, e := range tree {
		topRel = append(topRel, e.RelPath)
	}
	wantTop := []string{"dir", "a.txt", "b.txt"}
	if strings.Join(topRel, ",") != strings.Join(wantTop, ",") {
		t.Errorf("顶层条目 = %v, want %v", topRel, wantTop)
	}
	if findEntry(tree, "empty") != nil || findEntry(tree, "dir2") != nil {
		t.Error("空目录（含子目录全被修剪的 dir2）不应出现在结果中")
	}

	// 子目录内容与排序、/ 分隔相对路径
	dir := findEntry(tree, "dir")
	if dir == nil || !dir.IsDir || dir.Size != 0 {
		t.Fatalf("dir 节点不符: %+v", dir)
	}
	if len(dir.Children) != 2 || dir.Children[0].Name != "a.txt" || dir.Children[1].Name != "z.txt" {
		t.Errorf("dir.Children = %+v, want [a.txt, z.txt]", dir.Children)
	}
	if dir.Children[0].RelPath != "dir/a.txt" {
		t.Errorf("子节点 RelPath = %q, want dir/a.txt", dir.Children[0].RelPath)
	}

	// 文件大小与修改时间（unix 秒）
	fa := findEntry(tree, "a.txt")
	if fa == nil || fa.IsDir || fa.Size != 1 || fa.ModTime <= 0 {
		t.Errorf("a.txt 节点不符: %+v", fa)
	}
	fz := findEntry(tree, "dir/z.txt")
	if fz == nil || fz.Size != 4 || fz.ModTime <= 0 {
		t.Errorf("dir/z.txt 节点不符: %+v", fz)
	}
}

// TestWorkspaceDirectoryRoundtrip 目录递归上传/下载/删除（roundtrip），下载到
// 桌面重名时自动改名（源目录仍在桌面的场景顺带覆盖）。
func TestWorkspaceDirectoryRoundtrip(t *testing.T) {
	svc, ws, desk := setupWorkspaceService(t)
	// 源目录放独立临时目录（避免与桌面下载目标冲突）
	srcDir := filepath.Join(t.TempDir(), "srcDir")
	mustWrite(t, filepath.Join(srcDir, "a.txt"), "A")
	mustWrite(t, filepath.Join(srcDir, "sub", "b.txt"), "B")

	// 上传目录（递归）
	res, err := svc.UploadDirectory(srcDir)
	if err != nil {
		t.Fatalf("上传目录失败: %v", err)
	}
	if res.Error != "" || res.Target != "srcDir" {
		t.Fatalf("上传结果不符: %+v", res)
	}
	if got := readFile(t, filepath.Join(ws, "srcDir", "a.txt")); got != "A" {
		t.Errorf("工作区 srcDir/a.txt = %q, want A", got)
	}
	if got := readFile(t, filepath.Join(ws, "srcDir", "sub", "b.txt")); got != "B" {
		t.Errorf("工作区 srcDir/sub/b.txt = %q, want B", got)
	}

	// 下载目录到桌面（递归）
	resD, err := svc.DownloadFiles([]string{"srcDir"})
	if err != nil || len(resD) != 1 || resD[0].Error != "" {
		t.Fatalf("下载目录失败: err=%v res=%+v", err, resD)
	}
	if resD[0].Target != "srcDir" {
		t.Errorf("下载 Target = %q, want srcDir", resD[0].Target)
	}
	if got := readFile(t, filepath.Join(desk, "srcDir", "sub", "b.txt")); got != "B" {
		t.Errorf("桌面 srcDir/sub/b.txt = %q, want B", got)
	}

	// 再次下载同名目录 → 桌面自动改名 srcDir (1)
	resD2, err := svc.DownloadFiles([]string{"srcDir"})
	if err != nil || len(resD2) != 1 || resD2[0].Error != "" {
		t.Fatalf("二次下载目录失败: err=%v res=%+v", err, resD2)
	}
	if resD2[0].Target != "srcDir (1)" {
		t.Errorf("二次下载 Target = %q, want srcDir (1)", resD2[0].Target)
	}
	if got := readFile(t, filepath.Join(desk, "srcDir (1)", "a.txt")); got != "A" {
		t.Errorf("桌面 srcDir (1)/a.txt = %q, want A", got)
	}

	// 目录删除：recursive=false 拒绝且不删
	resDel, err := svc.DeleteFiles([]string{"srcDir"}, false)
	if err != nil || len(resDel) != 1 {
		t.Fatalf("删除目录失败: err=%v res=%+v", err, resDel)
	}
	if resDel[0].Error == "" || !strings.Contains(resDel[0].Error, "recursive=true") {
		t.Errorf("非递归删除目录应报错: %+v", resDel[0])
	}
	if _, err := os.Lstat(filepath.Join(ws, "srcDir")); err != nil {
		t.Errorf("拒绝后目录不应被删除: %v", err)
	}

	// 目录删除：recursive=true 成功
	resDel2, err := svc.DeleteFiles([]string{"srcDir"}, true)
	if err != nil || len(resDel2) != 1 || resDel2[0].Error != "" {
		t.Fatalf("递归删除目录失败: err=%v res=%+v", err, resDel2)
	}
	if resDel2[0].Target != "srcDir" {
		t.Errorf("删除 Target = %q, want srcDir", resDel2[0].Target)
	}
	if _, err := os.Lstat(filepath.Join(ws, "srcDir")); !os.IsNotExist(err) {
		t.Error("递归删除后目录应不存在")
	}
}

// TestWorkspaceDeleteRootRejected 删除工作区根目录本身被拒（空、.、/）。
func TestWorkspaceDeleteRootRejected(t *testing.T) {
	svc, ws, _ := setupWorkspaceService(t)
	mustWrite(t, filepath.Join(ws, "keep.txt"), "keep")
	for _, rel := range []string{"", ".", "/"} {
		res, err := svc.DeleteFiles([]string{rel}, true)
		if err != nil {
			t.Fatalf("删除根 rel=%q 调用失败: %v", rel, err)
		}
		if len(res) != 1 || !strings.Contains(res[0].Error, "不允许删除工作区根目录") {
			t.Errorf("rel=%q 应拒绝删除根目录, got %+v", rel, res)
		}
	}
	if _, err := os.Lstat(filepath.Join(ws, "keep.txt")); err != nil {
		t.Error("拒绝删除根后工作区内容不应受影响")
	}
}

// TestWorkspaceEscapeRejected ../ 逃逸路径被拒：删除与下载源均不得越过工作区边界，
// 工作区外文件不受影响。
func TestWorkspaceEscapeRejected(t *testing.T) {
	svc, ws, desk := setupWorkspaceService(t)
	home := filepath.Dir(filepath.Dir(ws)) // ws = home/.jot/workspace
	outside := filepath.Join(home, "escape.txt")
	mustWrite(t, outside, "secret")

	// 删除逃逸拒绝
	res, err := svc.DeleteFiles([]string{"../escape.txt"}, true)
	if err != nil || len(res) != 1 {
		t.Fatalf("删除逃逸调用失败: err=%v res=%+v", err, res)
	}
	if !strings.Contains(res[0].Error, "边界") {
		t.Errorf("删除逃逸应报边界错误, got %q", res[0].Error)
	}
	if got := readFile(t, outside); got != "secret" {
		t.Errorf("工作区外文件不应受影响, got %q", got)
	}

	// 下载逃逸拒绝
	resD, err := svc.DownloadFiles([]string{"../escape.txt"})
	if err != nil || len(resD) != 1 {
		t.Fatalf("下载逃逸调用失败: err=%v res=%+v", err, resD)
	}
	if !strings.Contains(resD[0].Error, "边界") {
		t.Errorf("下载逃逸应报边界错误, got %q", resD[0].Error)
	}
	if _, err := os.Lstat(filepath.Join(desk, "escape.txt")); !os.IsNotExist(err) {
		t.Error("逃逸下载不应落盘桌面")
	}
}

// TestWorkspaceDownloadAutoRename 下载到桌面重名自动改名：顶层文件与嵌套路径
// （针对最后一段基名改名，父目录相对位置保持不变）。
func TestWorkspaceDownloadAutoRename(t *testing.T) {
	svc, ws, desk := setupWorkspaceService(t)

	// 顶层文件重名
	mustWrite(t, filepath.Join(ws, "a.txt"), "ws")
	mustWrite(t, filepath.Join(desk, "a.txt"), "old")
	res, err := svc.DownloadFiles([]string{"a.txt"})
	if err != nil || len(res) != 1 || res[0].Error != "" {
		t.Fatalf("下载失败: err=%v res=%+v", err, res)
	}
	if res[0].Name != "a.txt" || res[0].Target != "a (1).txt" {
		t.Errorf("结果 Name=%q Target=%q, want a.txt / a (1).txt", res[0].Name, res[0].Target)
	}
	if got := readFile(t, filepath.Join(desk, "a (1).txt")); got != "ws" {
		t.Errorf("改名后内容 = %q, want ws", got)
	}
	if got := readFile(t, filepath.Join(desk, "a.txt")); got != "old" {
		t.Errorf("桌面原文件被改动 = %q, want old", got)
	}

	// 嵌套路径重名：桌面 dir/b.txt 已存在 → dir/b (1).txt
	mustWrite(t, filepath.Join(ws, "dir", "b.txt"), "B")
	mustMkdir(t, filepath.Join(desk, "dir"))
	mustWrite(t, filepath.Join(desk, "dir", "b.txt"), "oldB")
	resN, err := svc.DownloadFiles([]string{"dir/b.txt"})
	if err != nil || len(resN) != 1 || resN[0].Error != "" {
		t.Fatalf("嵌套下载失败: err=%v res=%+v", err, resN)
	}
	if resN[0].Target != "dir/b (1).txt" {
		t.Errorf("嵌套下载 Target = %q, want dir/b (1).txt", resN[0].Target)
	}
	if got := readFile(t, filepath.Join(desk, "dir", "b (1).txt")); got != "B" {
		t.Errorf("嵌套改名内容 = %q, want B", got)
	}
}

// TestWorkspaceBatchNonInterrupt 批量操作单项失败不中断批次：上传/删除/下载各含
// 一个不存在项，成功项正常完成、失败项 Error 非空。
func TestWorkspaceBatchNonInterrupt(t *testing.T) {
	svc, ws, desk := setupWorkspaceService(t)

	// 上传：一个存在、一个源不存在
	srcOK := filepath.Join(t.TempDir(), "ok.txt")
	mustWrite(t, srcOK, "ok")
	res, err := svc.UploadFiles([]string{srcOK, filepath.Join(t.TempDir(), "missing.txt")})
	if err != nil {
		t.Fatalf("批量上传调用失败: %v", err)
	}
	if len(res) != 2 || res[0].Error != "" || res[1].Error == "" {
		t.Fatalf("批量上传结果不符: %+v", res)
	}
	if got := readFile(t, filepath.Join(ws, "ok.txt")); got != "ok" {
		t.Errorf("成功项内容 = %q, want ok", got)
	}

	// 删除：一个存在、一个不存在
	mustWrite(t, filepath.Join(ws, "del.txt"), "d")
	resDel, err := svc.DeleteFiles([]string{"del.txt", "nope.txt"}, true)
	if err != nil {
		t.Fatalf("批量删除调用失败: %v", err)
	}
	if len(resDel) != 2 || resDel[0].Error != "" || resDel[1].Error == "" {
		t.Fatalf("批量删除结果不符: %+v", resDel)
	}
	if _, err := os.Lstat(filepath.Join(ws, "del.txt")); !os.IsNotExist(err) {
		t.Error("成功项 del.txt 应被删除")
	}

	// 下载：一个存在、一个不存在
	mustWrite(t, filepath.Join(ws, "dl.txt"), "dl")
	resD, err := svc.DownloadFiles([]string{"dl.txt", "ghost.txt"})
	if err != nil {
		t.Fatalf("批量下载调用失败: %v", err)
	}
	if len(resD) != 2 || resD[0].Error != "" || resD[1].Error == "" {
		t.Fatalf("批量下载结果不符: %+v", resD)
	}
	if got := readFile(t, filepath.Join(desk, "dl.txt")); got != "dl" {
		t.Errorf("成功项内容 = %q, want dl", got)
	}
}
