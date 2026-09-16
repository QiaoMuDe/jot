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

// TestIsDestructiveCommand 验证"破坏宿主系统"命令的黑名单判断覆盖各种形态
// （含 .exe 后缀、路径前缀）。此集合仅含破坏性命令（rm/del/rmdir/shutdown 等），
// 不含按子命令细分的 net/install 类逻辑（后者的覆盖见 TestCommandNeedsApproval）。
func TestIsDestructiveCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"rm", true},
		{"rm -rf /", true},
		{"del", true},
		{"/usr/bin/rmdir", true},
		{"formatsomething", false}, // 仅前缀不匹配（非黑名单基名）
		{"shutdown", true},
		{"sudo -u root cat /etc/passwd", true},
		{"git", false}, // git 非破坏宿主系统命令，按子命令细分见 CommandNeedsApproval
		{"pip", false}, // 同上
		{"go", false},  // 同上
		{"python", false},
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

// TestCommandNeedsApproval 验证 CommandNeedsApproval 按子命令细分 critical 判定。
func TestCommandNeedsApproval(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		args []string
		want bool
	}{
		// 破坏宿主系统：始终 true
		{"rm 破坏命令", "rm", []string{"-rf", "/"}, true},
		{"sudo 始终危险", "sudo", []string{"rm", "-rf", "/"}, true},
		// 解释器：始终 true
		{"cmd 解释器", "cmd", []string{"/c", "net user"}, true},
		{"powershell 解释器", "powershell", []string{"-Command", "x"}, true},
		// git：按子命令
		{"git status 只读", "git", []string{"status"}, false},
		{"git log 只读", "git", []string{"log"}, false},
		{"git diff 只读", "git", []string{"diff"}, false},
		{"git v 只读", "git", []string{"v"}, false},
		{"git config --get 只读", "git", []string{"config", "--get", "x"}, false},
		{"git remote -v 只读", "git", []string{"remote", "-v"}, false},
		{"git branch -l 只读", "git", []string{"branch", "-l"}, false},
		{"git tag -l 只读", "git", []string{"tag", "-l"}, false},
		{"git ls-files 只读", "git", []string{"ls-files"}, false},
		{"git clone 风险", "git", []string{"clone", "https://x"}, true},
		{"git push 风险", "git", []string{"push"}, true},
		{"git checkout 风险", "git", []string{"checkout", "x"}, true},
		{"git reset 风险", "git", []string{"reset", "--hard"}, true},
		{"git 无子命令风险", "git", nil, true},
		// npm
		{"npm ls 只读", "npm", []string{"ls"}, false},
		{"npm view 只读", "npm", []string{"view", "pkg"}, false},
		{"npm --version 只读", "npm", []string{"--version"}, false},
		{"npm search 只读", "npm", []string{"search", "x"}, false},
		{"npm install 风险", "npm", []string{"install", "pkg"}, true},
		{"npm run 风险", "npm", []string{"run", "build"}, true},
		// go
		{"go version 只读", "go", []string{"version"}, false},
		{"go list 只读", "go", []string{"list"}, false},
		{"go env 只读", "go", []string{"env"}, false},
		{"go fmt 只读", "go", []string{"fmt", "./..."}, false},
		{"go vet 只读", "go", []string{"vet", "./..."}, false},
		{"go install 风险", "go", []string{"install", "pkg@latest"}, true},
		{"go run 风险", "go", []string{"run", "main.go"}, true},
		{"go build 风险", "go", []string{"build", "./..."}, true},
		{"go test 风险", "go", []string{"test", "./..."}, true},
		{"go mod download 风险", "go", []string{"mod", "download"}, true},
		// pip / pip3
		{"pip list 只读", "pip", []string{"list"}, false},
		{"pip show 只读", "pip", []string{"show", "pkg"}, false},
		{"pip --version 只读", "pip", []string{"--version"}, false},
		{"pip freeze 只读", "pip", []string{"freeze"}, false},
		{"pip3 freeze 只读", "pip3", []string{"freeze"}, false},
		{"pip install 风险", "pip", []string{"install", "pkg"}, true},
		{"pip --upgrade 风险", "pip", []string{"--upgrade", "pkg"}, true},
		{"pip uninstall 风险", "pip", []string{"uninstall", "pkg"}, true},
		{"pip download 风险", "pip", []string{"download", "pkg"}, true},
		// curl / wget
		{"curl --version 只读", "curl", []string{"--version"}, false},
		{"curl -I url 只读", "curl", []string{"-I", "https://x"}, false},
		{"curl -O url 风险", "curl", []string{"-O", "https://x/file.zip"}, true},
		{"curl -o file url 风险", "curl", []string{"-o", "f", "https://x"}, true},
		{"curl 纯取回网页风险", "curl", []string{"https://x"}, true},
		{"wget url 风险", "wget", []string{"https://x/file"}, true},
		{"wget --version 只读", "wget", []string{"--version"}, false},
		// 普通命令
		{"python script 普通", "python", []string{"script.py"}, false},
		{"python --version 普通", "python", []string{"--version"}, false},
		{"非 net 命令放行", "my-tool", []string{"run"}, false},
		{"空命令放行", "   ", nil, false},
	}
	for _, c := range cases {
		if got := CommandNeedsApproval(c.cmd, c.args); got != c.want {
			t.Errorf("CommandNeedsApproval(%q, %v) = %v, want %v", c.cmd, c.args, got, c.want)
		}
	}
}
