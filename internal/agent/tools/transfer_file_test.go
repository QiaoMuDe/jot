package tools

// 本文件覆盖 transfer_file 工具的单元测试：
//  1. download 一律审批且 critical=true（纯新增与覆盖）、批准后落盘桌面；
//  2. upload 对齐 copy_file（纯新增免审批、覆盖审批 critical=false）；
//  3. 覆盖判定（overwrite=false 拒绝 / true 放行）与目录递归；
//  4. 双端边界校验（../ 逃逸拒绝）与参数校验；
//  5. 裸工具（ctx nil）放行 / Approver 缺失 fail-fast；
//  6. 目标端智能追加语义（嵌套落盘 / 整目录替换）与 ~ 前缀路径双端端到端。

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTransfer 构造测试用 transfer_file：注入工作目录根/家目录/桌面目录（桌面=
// home/Desktop），并创建两端目录。approver 为 nil 时用无审批器的 Context（覆盖
// 裸工具与 fail-fast 场景）。
func setupTransfer(t *testing.T, approver Approver) (*transferFileTool, *mockApprover, string, string) {
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
	ap := &mockApprover{}
	h := &transferFileTool{fsToolBase: fsToolBase{
		ctx:           &Context{Approver: approverOrMock(approver, ap)},
		workspaceRoot: ws,
		homeDir:       home,
		desktopDir:    desk,
	}}
	return h, ap, ws, desk
}

// approverOrMock approver 非 nil 时用之（如 rejectApprover），否则用记录型 mockApprover。
func approverOrMock(approver Approver, mock *mockApprover) Approver {
	if approver != nil {
		return approver
	}
	return mock
}

// mustWrite 测试辅助：写文件，失败即终止。
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTransferFileDownload 验证下载：一律审批（critical=true）、批准后落盘桌面、
// 覆盖判定与摘要标记、目录递归、拒绝不落盘。
func TestTransferFileDownload(t *testing.T) {
	t.Run("新文件审批 critical=true 并落盘", func(t *testing.T) {
		h, ap, ws, desk := setupTransfer(t, nil)
		mustWrite(t, filepath.Join(ws, "out.txt"), "hello")
		out, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"out.txt"}`)
		if err != nil {
			t.Fatalf("下载失败: %v", err)
		}
		if !strings.Contains(out, "已下载") || !strings.Contains(out, "桌面") {
			t.Errorf("结果文案不符: %s", out)
		}
		got, err := os.ReadFile(filepath.Join(desk, "out.txt"))
		if err != nil || string(got) != "hello" {
			t.Errorf("桌面文件内容 = %q, err = %v, want hello", got, err)
		}
		if ap.gotTool != "transfer_file" || !ap.gotCritical {
			t.Errorf("download 应审批且 critical=true, got tool=%q critical=%v", ap.gotTool, ap.gotCritical)
		}
		if !strings.Contains(ap.gotSum, "下载到桌面") || !strings.Contains(ap.gotSum, "out.txt") {
			t.Errorf("审批摘要不符: %s", ap.gotSum)
		}
	})

	t.Run("目标已存在未设 overwrite 拒绝", func(t *testing.T) {
		h, _, ws, desk := setupTransfer(t, nil)
		mustWrite(t, filepath.Join(ws, "out.txt"), "new")
		mustWrite(t, filepath.Join(desk, "out.txt"), "old")
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"out.txt"}`); err == nil {
			t.Error("目标已存在且未设 overwrite=true 应报错")
		}
		if b, _ := os.ReadFile(filepath.Join(desk, "out.txt")); string(b) != "old" {
			t.Errorf("拒绝后桌面文件不应变更, got %q", b)
		}
	})

	t.Run("覆盖审批 critical=true 且摘要含覆盖标记", func(t *testing.T) {
		h, ap, ws, desk := setupTransfer(t, nil)
		mustWrite(t, filepath.Join(ws, "out.txt"), "new")
		mustWrite(t, filepath.Join(desk, "out.txt"), "old")
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"out.txt","overwrite":true}`); err != nil {
			t.Fatalf("覆盖下载失败: %v", err)
		}
		if b, _ := os.ReadFile(filepath.Join(desk, "out.txt")); string(b) != "new" {
			t.Errorf("覆盖后内容 = %q, want new", b)
		}
		if !ap.gotCritical || !strings.Contains(ap.gotSum, "（覆盖）") {
			t.Errorf("覆盖下载 critical=%v 摘要=%s, want true 且含（覆盖）", ap.gotCritical, ap.gotSum)
		}
	})

	t.Run("目录递归下载", func(t *testing.T) {
		h, _, ws, desk := setupTransfer(t, nil)
		if err := os.MkdirAll(filepath.Join(ws, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(ws, "sub", "a.txt"), "A")
		mustWrite(t, filepath.Join(ws, "sub", "b.txt"), "B")
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"sub"}`); err != nil {
			t.Fatalf("目录下载失败: %v", err)
		}
		for name, want := range map[string]string{"a.txt": "A", "b.txt": "B"} {
			got, err := os.ReadFile(filepath.Join(desk, "sub", name))
			if err != nil || string(got) != want {
				t.Errorf("桌面 sub/%s = %q, err = %v, want %s", name, got, err, want)
			}
		}
	})

	t.Run("拒绝后不落盘", func(t *testing.T) {
		h, _, ws, desk := setupTransfer(t, &rejectApprover{err: errors.New("用户拒绝了操作")})
		mustWrite(t, filepath.Join(ws, "out.txt"), "hello")
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"out.txt"}`); err == nil {
			t.Error("拒绝的下载应返回错误")
		}
		if _, err := os.Lstat(filepath.Join(desk, "out.txt")); !os.IsNotExist(err) {
			t.Error("拒绝后桌面不应出现文件")
		}
	})

	t.Run("目录下载到已存在同名目录嵌套落盘", func(t *testing.T) {
		h, ap, ws, desk := setupTransfer(t, nil)
		if err := os.MkdirAll(filepath.Join(ws, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(ws, "sub", "a.txt"), "A")
		// 桌面已存在同名目录 sub：智能追加语义下最终目标为 desk/sub/sub（不存在）
		if err := os.MkdirAll(filepath.Join(desk, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"sub"}`); err != nil {
			t.Fatalf("目录下载失败: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(desk, "sub", "sub", "a.txt"))
		if err != nil || string(got) != "A" {
			t.Errorf("桌面 sub/sub/a.txt = %q, err = %v, want A", got, err)
		}
		if strings.Contains(ap.gotSum, "（覆盖）") {
			t.Errorf("最终目标不存在时摘要不应含（覆盖）: %s", ap.gotSum)
		}
	})

	t.Run("嵌套目录已存在 overwrite=true 整目录替换", func(t *testing.T) {
		h, ap, ws, desk := setupTransfer(t, nil)
		if err := os.MkdirAll(filepath.Join(ws, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(ws, "sub", "new.txt"), "new")
		// 桌面已存在最终目标 desk/sub/sub（含旧文件 old.txt）
		if err := os.MkdirAll(filepath.Join(desk, "sub", "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(desk, "sub", "sub", "old.txt"), "old")
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"sub","overwrite":true}`); err != nil {
			t.Fatalf("覆盖目录下载失败: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(desk, "sub", "sub", "old.txt")); !os.IsNotExist(err) {
			t.Error("覆盖后旧文件 old.txt 应被替换移除")
		}
		if b, err := os.ReadFile(filepath.Join(desk, "sub", "sub", "new.txt")); err != nil || string(b) != "new" {
			t.Errorf("桌面 sub/sub/new.txt = %q, err = %v, want new", b, err)
		}
		if !ap.gotCritical || !strings.Contains(ap.gotSum, "（覆盖）") {
			t.Errorf("覆盖目录下载 critical=%v 摘要=%s, want true 且含（覆盖）", ap.gotCritical, ap.gotSum)
		}
	})
}

// TestTransferFileUpload 验证上传：纯新增免审批、覆盖审批 critical=false、落盘工作区。
func TestTransferFileUpload(t *testing.T) {
	t.Run("新文件免审批并落盘", func(t *testing.T) {
		h, ap, ws, desk := setupTransfer(t, nil)
		mustWrite(t, filepath.Join(desk, "in.txt"), "data")
		out, err := h.InvokableRun(context.Background(), `{"direction":"upload","path":"in.txt"}`)
		if err != nil {
			t.Fatalf("上传失败: %v", err)
		}
		if !strings.Contains(out, "已上传") {
			t.Errorf("结果文案不符: %s", out)
		}
		got, err := os.ReadFile(filepath.Join(ws, "in.txt"))
		if err != nil || string(got) != "data" {
			t.Errorf("工作区文件内容 = %q, err = %v, want data", got, err)
		}
		if ap.gotTool != "" {
			t.Errorf("upload 纯新增不应触发审批, got %q", ap.gotTool)
		}
	})

	t.Run("覆盖审批 critical=false", func(t *testing.T) {
		h, ap, ws, desk := setupTransfer(t, nil)
		mustWrite(t, filepath.Join(desk, "in.txt"), "data")
		mustWrite(t, filepath.Join(ws, "in.txt"), "old")
		if _, err := h.InvokableRun(context.Background(), `{"direction":"upload","path":"in.txt"}`); err == nil {
			t.Error("目标已存在且未设 overwrite=true 应报错")
		}
		if _, err := h.InvokableRun(context.Background(), `{"direction":"upload","path":"in.txt","overwrite":true}`); err != nil {
			t.Fatalf("覆盖上传失败: %v", err)
		}
		if b, _ := os.ReadFile(filepath.Join(ws, "in.txt")); string(b) != "data" {
			t.Errorf("覆盖后内容 = %q, want data", b)
		}
		if ap.gotCritical || !strings.Contains(ap.gotSum, "上传并覆盖") {
			t.Errorf("upload 覆盖 critical=%v 摘要=%s, want false 且含「上传并覆盖」", ap.gotCritical, ap.gotSum)
		}
	})
}

// TestTransferFileBoundary 验证双端边界校验与参数校验。
func TestTransferFileBoundary(t *testing.T) {
	t.Run("download 源端 ../ 越界拒绝", func(t *testing.T) {
		h, _, _, _ := setupTransfer(t, nil)
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"../escape.txt"}`); err == nil {
			t.Error("../ 逃逸应报错")
		}
	})

	t.Run("upload 源端 ../ 越界拒绝", func(t *testing.T) {
		h, _, _, _ := setupTransfer(t, nil)
		if _, err := h.InvokableRun(context.Background(), `{"direction":"upload","path":"../escape.txt"}`); err == nil {
			t.Error("../ 逃逸应报错")
		}
	})

	t.Run("源不存在报错且不审批", func(t *testing.T) {
		h, ap, _, _ := setupTransfer(t, nil)
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"nope.txt"}`); err == nil {
			t.Error("源不存在应报错")
		}
		if ap.gotTool != "" {
			t.Error("源不存在不应触发审批")
		}
	})

	t.Run("direction 非法与 path 缺失", func(t *testing.T) {
		h, _, _, _ := setupTransfer(t, nil)
		if _, err := h.InvokableRun(context.Background(), `{"direction":"sync","path":"a.txt"}`); err == nil {
			t.Error("direction 非法应报错")
		}
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download"}`); err == nil {
			t.Error("path 缺失应报错")
		}
	})

	t.Run("~ 前缀路径下载端到端", func(t *testing.T) {
		h, _, ws, desk := setupTransfer(t, nil)
		mustWrite(t, filepath.Join(ws, "out.txt"), "hello")
		// 源端以 ~ 形式给出：目标端按归一化后的相对位置解析，不再越界报错
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"~/.jot/workspace/out.txt"}`); err != nil {
			t.Fatalf("~ 形式下载失败: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(desk, "out.txt")); err != nil || string(b) != "hello" {
			t.Errorf("桌面 out.txt = %q, err = %v, want hello", b, err)
		}
	})

	t.Run("~ 前缀路径上传端到端", func(t *testing.T) {
		h, ap, ws, desk := setupTransfer(t, nil)
		mustWrite(t, filepath.Join(desk, "in.txt"), "data")
		if _, err := h.InvokableRun(context.Background(), `{"direction":"upload","path":"~/Desktop/in.txt"}`); err != nil {
			t.Fatalf("~ 形式上传失败: %v", err)
		}
		if b, err := os.ReadFile(filepath.Join(ws, "in.txt")); err != nil || string(b) != "data" {
			t.Errorf("工作区 in.txt = %q, err = %v, want data", b, err)
		}
		if ap.gotTool != "" {
			t.Errorf("upload 纯新增不应触发审批, got %q", ap.gotTool)
		}
	})
}

// TestTransferFileApprovalHooks 验证裸工具放行与 Approver 缺失 fail-fast。
func TestTransferFileApprovalHooks(t *testing.T) {
	t.Run("裸工具（ctx nil）放行", func(t *testing.T) {
		home := t.TempDir()
		ws := filepath.Join(home, ".jot", "workspace")
		desk := filepath.Join(home, "Desktop")
		if err := os.MkdirAll(ws, 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(ws, "out.txt"), "hello")
		h := &transferFileTool{fsToolBase: fsToolBase{workspaceRoot: ws, homeDir: home, desktopDir: desk}}
		if _, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"out.txt"}`); err != nil {
			t.Fatalf("裸工具下载应放行: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(desk, "out.txt")); err != nil {
			t.Errorf("裸工具下载应落盘: %v", err)
		}
	})

	t.Run("Approver 缺失 fail-fast", func(t *testing.T) {
		home := t.TempDir()
		ws := filepath.Join(home, ".jot", "workspace")
		desk := filepath.Join(home, "Desktop")
		if err := os.MkdirAll(ws, 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(ws, "out.txt"), "hello")
		h := &transferFileTool{fsToolBase: fsToolBase{
			ctx:           &Context{},
			workspaceRoot: ws,
			homeDir:       home,
			desktopDir:    desk,
		}}
		_, err := h.InvokableRun(context.Background(), `{"direction":"download","path":"out.txt"}`)
		if err == nil || !strings.Contains(err.Error(), "未配置审批机制") {
			t.Errorf("Approver 缺失应 fail-fast, got %v", err)
		}
		if _, lerr := os.Lstat(filepath.Join(desk, "out.txt")); !os.IsNotExist(lerr) {
			t.Error("fail-fast 后桌面不应出现文件")
		}
	})
}
