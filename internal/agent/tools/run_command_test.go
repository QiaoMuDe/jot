package tools

// 本文件覆盖 run_command 命令执行工具与命令黑名单的关键行为：
// 空命令报错、LookPath 失败（环境无命令）、cwd 越界拒绝、黑名单判断。
// 超时测试会实际等待 30s 且依赖平台命令，此处省略以避免拖慢 CI
// （如需覆盖可在独立环境用 sleep 长命令 + 缩短超时验证 CommandContext 终止）。

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

// TestRunCommandSuccess 验证成功路径：go version 输出非空。
func TestRunCommandSuccess(t *testing.T) {
	dir := t.TempDir()
	h := &runCommandTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
	out, err := h.InvokableRun(context.Background(), `{"command":"go","args":["version"]}`)
	if err != nil {
		t.Fatalf("执行成功命令失败: %v", err)
	}
	if strings.Contains(out, "无输出") {
		t.Errorf("go version 不应无输出，实际: %.120s", out)
	}
	if !strings.Contains(strings.ToLower(out), "go") {
		t.Errorf("输出应含 go 字样，实际: %.120s", out)
	}
}

// TestRunCommandMissingCommand 验证 LookPath 失败：环境无命令时报友好错误。
func TestRunCommandMissingCommand(t *testing.T) {
	dir := t.TempDir()
	h := &runCommandTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
	// 用一个几乎不可能存在的命令名
	_, err := h.InvokableRun(context.Background(), `{"command":"jot-this-command-should-not-exist-9f3k"}`)
	if err == nil {
		t.Fatal("不存在的命令应报错")
	}
	if !strings.Contains(err.Error(), "环境中没有命令") {
		t.Errorf("错误应含环境中没有命令，实际: %v", err)
	}
}

// TestRunCommandCwdEscape 验证 cwd 越界被拒绝。
func TestRunCommandCwdEscape(t *testing.T) {
	dir := t.TempDir()
	h := &runCommandTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
	if _, err := h.InvokableRun(context.Background(), `{"command":"go","args":["version"],"cwd":"../outside"}`); err == nil {
		t.Fatal("cwd 越界应报错")
	}
}

// TestRunCommandEmptyCommand 验证空命令报错。
func TestRunCommandEmptyCommand(t *testing.T) {
	dir := t.TempDir()
	h := &runCommandTool{fsToolBase: fsToolBase{workspaceRoot: dir}}
	if _, err := h.InvokableRun(context.Background(), `{"command":"  "}`); err == nil {
		t.Fatal("空命令应报错")
	}
}

// TestIsDestructiveCommand 验证命令基名命中高危集合（含 .exe 后缀、路径前缀）。
// 参数中的危险子命令不在此函数覆盖，见 TestCommandNeedsApproval。
func TestIsDestructiveCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"rm", true},
		{"rm -rf /", true},
		{"del", true},
		{"/usr/bin/rmdir", true},
		{"formatsomething", false}, // 整词相等：formatsomething ≠ format
		{"shutdown", true},
		{"sudo -u root cat /etc/passwd", true},
		{"python", true}, // 解释器属高危
		{"cmd", true},
		{"curl", true}, // 网络下载工具
		{"git", false}, // 基名非高危，危险凭参数子命令判定（见 CommandNeedsApproval）
		{"pip", false},
		{"go", false},
		{"   ", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsDestructiveCommand(c.cmd); got != c.want {
			t.Errorf("IsDestructiveCommand(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
	// Windows .exe 后缀形态
	if runtime.GOOS == "windows" {
		if !IsDestructiveCommand("rm.exe") || !IsDestructiveCommand(`C:\tools\shutdown.exe`) {
			t.Error("Windows 下应识别 .exe 后缀破坏性命令")
		}
	}
}

// TestCommandNeedsApproval 验证 critical 判定：命令基名或任一参数 token 命中高危集合。
func TestCommandNeedsApproval(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		args []string
		want bool
	}{
		// 破坏性命令 / 解释器：基名命中即 true
		{"rm 破坏命令", "rm", []string{"-rf", "/"}, true},
		{"sudo 始终危险", "sudo", []string{"rm", "-rf", "/"}, true},
		{"cmd 解释器", "cmd", []string{"/c", "net user"}, true},
		{"powershell 解释器", "powershell", []string{"-Command", "x"}, true},
		{"python 解释器", "python", []string{"script.py"}, true},
		{"python 任意代码", "python", []string{"-c", "x"}, true},
		{"node 解释器", "node", []string{"app.js"}, true},
		{"bash 解释器", "bash", []string{"x.sh"}, true},
		{"curl 下载工具", "curl", []string{"https://x"}, true},
		{"wget 下载工具", "wget", []string{"https://x"}, true},
		// git：基名非高危，危险凭参数 token
		{"git status 只读", "git", []string{"status"}, false},
		{"git log 只读", "git", []string{"log"}, false},
		{"git diff 只读", "git", []string{"diff"}, false},
		{"git config --get 只读", "git", []string{"config", "--get", "x"}, false},
		{"git branch -l 只读", "git", []string{"branch", "-l"}, false},
		{"git tag -l 只读", "git", []string{"tag", "-l"}, false},
		{"git clone 风险", "git", []string{"clone", "https://x"}, true},
		{"git push 风险", "git", []string{"push"}, true},
		{"git checkout 风险", "git", []string{"checkout", "x"}, true},
		{"git reset --hard 风险", "git", []string{"reset", "--hard"}, true},
		{"git merge 风险", "git", []string{"merge", "x"}, true},
		{"git fetch 风险", "git", []string{"fetch"}, true},
		{"git 无参数放行", "git", nil, false},
		// npm / go / pip：基名非高危，危险凭参数 token
		{"npm ls 只读", "npm", []string{"ls"}, false},
		{"npm view 只读", "npm", []string{"view", "pkg"}, false},
		{"npm install 风险", "npm", []string{"install", "pkg"}, true},
		{"npm run 风险", "npm", []string{"run", "build"}, true},
		{"go version 只读", "go", []string{"version"}, false},
		{"go env 只读", "go", []string{"env"}, false},
		{"go install 风险", "go", []string{"install", "pkg@latest"}, true},
		{"go run 风险", "go", []string{"run", "main.go"}, true},
		{"go mod download 风险", "go", []string{"mod", "download"}, true},
		{"pip list 只读", "pip", []string{"list"}, false},
		{"pip show 只读", "pip", []string{"show", "pkg"}, false},
		{"pip freeze 只读", "pip", []string{"freeze"}, false},
		{"pip install 风险", "pip", []string{"install", "pkg"}, true},
		{"pip --upgrade 风险", "pip", []string{"--upgrade", "pkg"}, true},
		{"pip uninstall 风险", "pip", []string{"uninstall", "pkg"}, true},
		// 参数内嵌空白也能切出危险 token（拼接型参数兜底）
		{"拼接参数内 del 风险", "some-tool", []string{"/c", "del x"}, true},
		// 普通命令
		{"普通只读命令放行", "my-tool", []string{"list"}, false},
		{"普通命令 run 视为高危", "my-tool", []string{"run"}, true},
		{"空命令放行", "   ", nil, false},
	}
	for _, c := range cases {
		if got := CommandNeedsApproval(c.cmd, c.args); got != c.want {
			t.Errorf("CommandNeedsApproval(%q, %v) = %v, want %v", c.cmd, c.args, got, c.want)
		}
	}
}
