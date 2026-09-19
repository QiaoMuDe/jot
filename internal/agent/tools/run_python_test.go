package tools

// 本文件覆盖 run_python Python 执行工具的关键行为：code/path 二选一校验、
// cwd/path 沙箱越界拒绝、审批门控（critical 恒 true / 拒绝 / Approver 缺失
// fail-fast / 裸工具放行）、解释器探测失败友好错误、探测缓存（只探测一次、
// 失败不缓存），以及本机存在解释器时的真实执行。审批 mock 复用同包
// mockApprover（fs_tools_test.go）与 rejectApprover（manage_approval_test.go）。

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// resetPythonCache 清空解释器缓存态（测试间隔离，避免缓存污染跨测试断言）。
func resetPythonCache() {
	pythonMu.Lock()
	pythonResolved = ""
	pythonMu.Unlock()
}

// withDetectPython 临时替换解释器探测函数（包级变量），并重置缓存态；
// 返回恢复函数，调用方 defer 执行以还原探测函数与缓存。
func withDetectPython(fn func() (string, error)) func() {
	orig := detectPython
	detectPython = fn
	resetPythonCache()
	return func() {
		detectPython = orig
		resetPythonCache()
	}
}

// TestRunPythonParamValidation 验证参数校验：code/path 二选一、code 超长限制。
func TestRunPythonParamValidation(t *testing.T) {
	dir := t.TempDir()
	h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
	cases := []struct {
		name string
		args string
		want string
	}{
		{"code/path 都缺", `{}`, "必须二选一"},
		{"code/path 都给", `{"code":"x=1","path":"a.py"}`, "只能提供一个"},
	}
	for _, c := range cases {
		if _, err := h.InvokableRun(context.Background(), c.args); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err=%v, want 含 %q", c.name, err, c.want)
		}
	}
	// code 超长（超 maxToolLongText）
	long := strings.Repeat("x", maxToolLongText+1)
	if _, err := h.InvokableRun(context.Background(), `{"code":"`+long+`"}`); err == nil || !strings.Contains(err.Error(), "过长") {
		t.Errorf("code 超长应报错，实际: %v", err)
	}
}

// TestRunPythonCwdEscape 验证 cwd 越界被拒绝（cwd 校验先于审批与探测，不依赖解释器）。
func TestRunPythonCwdEscape(t *testing.T) {
	dir := t.TempDir()
	h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
	if _, err := h.InvokableRun(context.Background(), `{"code":"x=1","cwd":"../outside"}`); err == nil {
		t.Fatal("cwd 越界应报错")
	}
}

// TestRunPythonPathEscape 验证 path 越界被 resolvePath 沙箱校验拒绝（注入成功探测，
// 确保错误来自路径校验而非解释器探测）。
func TestRunPythonPathEscape(t *testing.T) {
	defer withDetectPython(func() (string, error) { return "/fake/python", nil })()
	dir := t.TempDir()
	h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
	_, err := h.InvokableRun(context.Background(), `{"path":"../outside.py"}`)
	if err == nil {
		t.Fatal("path 越界应报错")
	}
	if strings.Contains(err.Error(), "未找到可用的 Python 解释器") {
		t.Fatalf("越界应被 resolvePath 拦截而非探测报错，实际: %v", err)
	}
}

// TestRunPythonApproval 验证审批门控：批准放行（critical 恒 true）、拒绝返回拒绝
// 错误、Approver 缺失 fail-fast、裸工具（ctx nil）放行。
func TestRunPythonApproval(t *testing.T) {
	// 注入探测失败：审批通过后应继续走到探测阶段，可用"探测错误"证明未被审批拦截
	defer withDetectPython(func() (string, error) { return "", errors.New("mock no python") })()

	t.Run("批准放行且 critical=true", func(t *testing.T) {
		mock := &mockApprover{}
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: t.TempDir()}}
		h.ctx = &Context{Approver: mock}
		// 多行 code：首行为导入语句，主体（第二行起）须进入审批摘要，否则用户看不到真实操作
		_, err := h.InvokableRun(context.Background(), "{\"code\":\"import json\\nprint('dangerous op')\"}")
		if mock.gotTool != "run_python" {
			t.Errorf("审批 toolName = %q, want run_python", mock.gotTool)
		}
		if !mock.gotCritical {
			t.Error("run_python 审批 critical 应恒为 true")
		}
		if !strings.Contains(mock.gotSum, "执行 Python") {
			t.Errorf("审批摘要应含动作前缀，实际: %q", mock.gotSum)
		}
		// 审批摘要须包含 code 主体（非仅首行导入语句）
		if !strings.Contains(mock.gotSum, "dangerous op") {
			t.Errorf("审批摘要应含 code 主体内容，实际: %q", mock.gotSum)
		}
		// 审批通过后应继续走到解释器探测（返回探测错误而非审批错误）
		if err == nil || !strings.Contains(err.Error(), "未找到可用的 Python 解释器") {
			t.Errorf("批准后应继续探测解释器，实际 err=%v", err)
		}
	})

	t.Run("拒绝返回拒绝错误", func(t *testing.T) {
		rej := &rejectApprover{err: errors.New("用户拒绝了执行")}
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: t.TempDir()}}
		h.ctx = &Context{Approver: rej}
		_, err := h.InvokableRun(context.Background(), `{"code":"x=1"}`)
		if err == nil || !strings.Contains(err.Error(), "用户拒绝了执行") {
			t.Errorf("拒绝应返回拒绝错误，实际: %v", err)
		}
	})

	t.Run("Approver 缺失 fail-fast", func(t *testing.T) {
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: t.TempDir()}}
		h.ctx = &Context{} // Approver 为 nil：装配遗漏应报错而非静默放行
		_, err := h.InvokableRun(context.Background(), `{"code":"x=1"}`)
		if err == nil || !strings.Contains(err.Error(), "未配置审批机制") {
			t.Errorf("Approver 缺失应 fail-fast，实际: %v", err)
		}
	})

	t.Run("裸工具放行", func(t *testing.T) {
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: t.TempDir()}}
		_, err := h.InvokableRun(context.Background(), `{"code":"x=1"}`)
		// ctx nil 审批放行 → 继续走到探测错误
		if err == nil || !strings.Contains(err.Error(), "未找到可用的 Python 解释器") {
			t.Errorf("裸工具应审批放行并继续探测，实际: %v", err)
		}
	})
}

// TestRunPythonDetectFailure 验证解释器探测失败返回友好中文错误。
func TestRunPythonDetectFailure(t *testing.T) {
	defer withDetectPython(func() (string, error) { return "", errors.New("mock no python") })()
	h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: t.TempDir()}}
	_, err := h.InvokableRun(context.Background(), `{"code":"x=1"}`)
	if err == nil || !strings.Contains(err.Error(), "未找到可用的 Python 解释器") {
		t.Errorf("探测失败应返回友好错误，实际: %v", err)
	}
}

// TestPythonApprovalSummary 验证审批摘要函数：code 模式含完整代码主体（非仅首行）、
// 超长截断带提示、path 模式含路径；并验证动作文案（pythonRunSummary）仍只取首行。
func TestPythonApprovalSummary(t *testing.T) {
	// code 模式：多行代码主体须进入审批摘要（旧逻辑只取首行导致审批看不到真实操作）
	s := pythonApprovalSummary("import json\nprint('dangerous op')", "")
	if !strings.Contains(s, "执行 Python 代码") {
		t.Errorf("摘要应含 code 前缀，实际: %q", s)
	}
	if !strings.Contains(s, "dangerous op") {
		t.Errorf("摘要应含 code 主体第二行，实际: %q", s)
	}
	// 超长 code：截断到上限并带截断提示
	long := strings.Repeat("x", pythonApprovalPreviewMaxRunes+100)
	s = pythonApprovalSummary(long, "")
	if !strings.HasSuffix(s, "[内容过长已截断，仅显示前 "+strconv.Itoa(pythonApprovalPreviewMaxRunes)+" 字]") {
		t.Errorf("超长 code 摘要应带截断提示，实际: %.80q", s)
	}
	// path 模式：含脚本路径
	s = pythonApprovalSummary("", "scripts/analyze.py")
	if !strings.Contains(s, "scripts/analyze.py") {
		t.Errorf("path 摘要应含脚本路径，实际: %q", s)
	}
	// 动作文案保持简短：仍只取首行（不影响审批摘要的完整预览）
	act := pythonRunSummary("import json\nprint('dangerous op')", "")
	if strings.Contains(act, "dangerous op") {
		t.Errorf("动作文案应只取首行，实际: %q", act)
	}
}

// TestResolvePythonCache 验证解释器缓存机制：成功结果缓存（第二次命中不重复探测）、
// 失败不缓存（下次重新探测）。
func TestResolvePythonCache(t *testing.T) {
	t.Run("成功结果缓存，第二次命中不重复探测", func(t *testing.T) {
		calls := 0
		defer withDetectPython(func() (string, error) {
			calls++
			return "/fake/python", nil
		})()
		p1, err := resolvePython()
		if err != nil || p1 != "/fake/python" {
			t.Fatalf("首次探测 p=%q err=%v", p1, err)
		}
		p2, err := resolvePython()
		if err != nil || p2 != "/fake/python" {
			t.Fatalf("第二次 p=%q err=%v", p2, err)
		}
		if calls != 1 {
			t.Errorf("真实探测应只执行一次，实际 %d 次", calls)
		}
	})

	t.Run("失败不缓存，下次重新探测", func(t *testing.T) {
		calls := 0
		orig := detectPython
		detectPython = func() (string, error) {
			calls++
			if calls == 1 {
				return "", errors.New("no python yet")
			}
			return "/ok/python", nil
		}
		defer func() {
			detectPython = orig
			resetPythonCache()
		}()
		resetPythonCache()
		if _, err := resolvePython(); err == nil {
			t.Fatal("首次探测应失败")
		}
		p, err := resolvePython()
		if err != nil || p != "/ok/python" {
			t.Fatalf("失败后应重新探测成功，p=%q err=%v", p, err)
		}
		if calls != 2 {
			t.Errorf("探测调用次数 = %d, want 2", calls)
		}
	})
}

// TestRunPythonExecute 验证真实执行（code 与 path 两模式）：本机无可用解释器时跳过。
func TestRunPythonExecute(t *testing.T) {
	interp := hostPython()
	if interp == "" {
		t.Skip("本机无可用的 Python 解释器，跳过真实执行测试")
	}
	defer withDetectPython(func() (string, error) { return interp, nil })()
	dir := t.TempDir()

	t.Run("code 模式", func(t *testing.T) {
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
		out, err := h.InvokableRun(context.Background(), `{"code":"print('ok')"}`)
		if err != nil {
			t.Fatalf("code 执行失败: %v", err)
		}
		if !strings.Contains(out, "ok") {
			t.Errorf("输出应含 ok，实际: %q", out)
		}
	})

	t.Run("path 模式", func(t *testing.T) {
		script := filepath.Join(dir, "hello.py")
		if err := os.WriteFile(script, []byte("print('path-ok')"), 0o644); err != nil {
			t.Fatalf("写脚本失败: %v", err)
		}
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
		out, err := h.InvokableRun(context.Background(), `{"path":"hello.py"}`)
		if err != nil {
			t.Fatalf("path 执行失败: %v", err)
		}
		if !strings.Contains(out, "path-ok") {
			t.Errorf("输出应含 path-ok，实际: %q", out)
		}
	})
}

// TestRunPythonExecuteArgsCwdTruncate 补充真实执行边界：args 参数透传、cwd 实际生效、
// 输出超长截断（均依赖真实解释器，不可用时跳过）。
func TestRunPythonExecuteArgsCwdTruncate(t *testing.T) {
	interp := hostPython()
	if interp == "" {
		t.Skip("本机无可用的 Python 解释器，跳过真实执行测试")
	}
	defer withDetectPython(func() (string, error) { return interp, nil })()
	dir := t.TempDir()

	t.Run("args 参数透传", func(t *testing.T) {
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
		// sys.argv[0] 为脚本路径，argv[1:] 应为透传参数
		out, err := h.InvokableRun(context.Background(), `{"code":"import sys; print(' '.join(sys.argv[1:]))","args":["甲","b","c"]}`)
		if err != nil {
			t.Fatalf("args 透传执行失败: %v", err)
		}
		if out != "甲 b c" {
			t.Errorf("args 透传 = %q, want 甲 b c", out)
		}
	})

	t.Run("cwd 生效", func(t *testing.T) {
		sub := filepath.Join(dir, "sub")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
		out, err := h.InvokableRun(context.Background(), `{"code":"import os; print(os.getcwd())","cwd":"sub"}`)
		if err != nil {
			t.Fatalf("cwd 执行失败: %v", err)
		}
		// Windows 盘符大小写可能与构造路径不一致，用 Clean + EqualFold 比较
		if !strings.EqualFold(filepath.Clean(strings.TrimSpace(out)), filepath.Clean(sub)) {
			t.Errorf("cwd = %q, want %q", strings.TrimSpace(out), sub)
		}
	})

	t.Run("输出超长截断", func(t *testing.T) {
		h := &runPythonTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
		out, err := h.InvokableRun(context.Background(), `{"code":"print('x'*2000)"}`)
		if err != nil {
			t.Fatalf("超长输出执行失败: %v", err)
		}
		if !strings.HasSuffix(out, "[输出过长已截断]") {
			t.Errorf("超长输出应带截断提示，实际: %.120q", out)
		}
	})
}

// hostPython 探测本机真实可用的解释器（独立于包级缓存与 detectPython 替换，
// 供真实执行测试判断是否可运行）。
func hostPython() string {
	for _, name := range pythonCandidates() {
		if p, err := exec.LookPath(name); err == nil && smokeTestPython(p) {
			return p
		}
	}
	return ""
}
