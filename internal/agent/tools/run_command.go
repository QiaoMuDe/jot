package tools

// 本文件实现 run_command 命令执行工具：在 AI 助手工作目录（~/.jot/workspace）
// 内执行可执行命令及其参数数组。它刻意不支持 shell 语法（管道/重定向/&&/;/
// 通配符等）——需要复杂逻辑应先 write_file 编写脚本再用对应解释器执行。
// cwd 仅允许工作目录内子目录或缺省（固定为工作目录根）；每次执行都先经
// Context.Approver 请求审批，critical 由是否命中破坏性命令（破坏宿主系统）
// / 高风险 net 类子命令（CommandNeedsApproval）决定——critical=true 时即使
// auto/review 模式也必须确认，不可绕过。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// runCommandTimeout 命令执行超时：超时则终止子进程并返回友好提示。
const runCommandTimeout = 30 * time.Second

// maxCommandOutputBytes run_command 输出有界累计上限（字节）：超过即停止累加
// 但进程继续运行至完成/超时，避免超大输出撑爆内存。
const maxCommandOutputBytes = 256 * 1024

// destructiveCmdPrefixes 破坏宿主系统的危险命令基名（命中即始终 critical=true，
// 即使 auto 模式也强制确认，作为不可绕过的最后防线）。
var destructiveCmdPrefixes = map[string]bool{
	"rm": true, "del": true, "rmdir": true, "shutdown": true, "reboot": true,
	"mkfs": true, "format": true, "dd": true, "sudo": true, "systemctl": true,
	"reg": true, "diskpart": true, "taskkill": true, "chmod": true, "chown": true,
	"cipher": true, "format-volume": true,
}

// shellCmdPrefixes 可执行任意脚本的解释器（始终 critical=true）。
var shellCmdPrefixes = map[string]bool{
	"powershell": true, "pwsh": true, "cmd": true,
}

// commandBaseName 提取命令基名的规范化小写形式：取第一个空白分隔 token 作为
// 命令名，再取 filepath.Base 并去除可执行后缀（.exe/.bat/.cmd）。
func commandBaseName(command string) string {
	name := strings.ToLower(strings.TrimSpace(command))
	if name == "" {
		return ""
	}
	if i := strings.IndexAny(name, " \t"); i >= 0 {
		name = name[:i]
	}
	base := filepath.Base(name)
	base = strings.TrimSuffix(base, ".exe")
	base = strings.TrimSuffix(base, ".bat")
	base = strings.TrimSuffix(base, ".cmd")
	return base
}

// IsDestructiveCommand 判断命令是否命中"破坏宿主系统"的危险命令集合
// （destructiveCmdPrefixes，始终 critical）。不含 net/install 类按子命令细分的逻辑，
// 后者见 CommandNeedsApproval。
func IsDestructiveCommand(command string) bool {
	return destructiveCmdPrefixes[commandBaseName(command)]
}

// CommandNeedsApproval 判断命令执行是否需要"不可绕过"的用户审批（critical=true）：
// 命中破坏性命令 / 可执行任意脚本的解释器为 true；net/install 类命令按子命令细分
// （见下文 *_NeedsApproval）；其余命令一律 false（仅在 confirm_every 模式确认）。
func CommandNeedsApproval(command string, args []string) bool {
	name := commandBaseName(command)
	if name == "" {
		return false
	}
	if destructiveCmdPrefixes[name] || shellCmdPrefixes[name] {
		return true
	}
	switch name {
	case "curl", "wget":
		return netDownloadNeedsApproval(name, args)
	case "pip", "pip3":
		return pipNeedsApproval(args)
	case "npm":
		return npmNeedsApproval(args)
	case "go":
		return goNeedsApproval(args)
	case "git":
		return gitNeedsApproval(args)
	}
	return false
}

// netDownloadNeedsApproval 判断 curl/wget 是否属高风险下载（critical=true）。
// 纯只读（HEAD/--version）放行；带了输出落盘类 flag 或携带 http(s) URL 且意图
// 下载（非纯 HEAD 探测）视为风险。
func netDownloadNeedsApproval(command string, args []string) bool {
	if command == "wget" {
		// wget 默认落盘：除纯 --version/--help 外一律视为风险
		for _, a := range args {
			if a == "--version" || a == "-V" || a == "--help" {
				return false
			}
		}
		return true
	}
	// curl
	hasOutput, hasHead, hasURL := false, false, false
	for _, a := range args {
		switch a {
		case "-o", "--output", "-O", "--output-document", "--remote-name", "-f", "--force":
			hasOutput = true
		case "-I", "--head":
			hasHead = true
		case "--version", "-V":
			hasHead = true // 视作只读探测
		}
		if strings.HasPrefix(a, "http://") || strings.HasPrefix(a, "https://") {
			hasURL = true
		}
	}
	if hasOutput {
		return true
	}
	if hasURL && !hasHead {
		return true
	}
	return false
}

// pipNeedsApproval 判断 pip/pip3 子命令是否风险（install/uninstall/download/wheel/--upgrade）。
// 只读子命令（list/show/--version/freeze）放行。
func pipNeedsApproval(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "list", "show", "--version", "-V", "freeze", "check", "help":
		return false
	}
	// install / download / wheel / uninstall / --upgrade 及默认 → true
	return true
}

// npmNeedsApproval 判断 npm 子命令是否风险。install/add/init/run/exec/uninstall/
// remove/ci/rebuild/link/--force 会安装或执行任意脚本 → true；
// 只读（ls/view/--version/search/outdated）放行。
func npmNeedsApproval(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "ls", "view", "--version", "-v", "search", "outdated", "ping", "help":
		return false
	}
	return true
}

// goNeedsApproval 判断 go 子命令是否风险。install/run/get/build/test/mod 执行编译、
// 联网下载或运行外部代码 → true；只读（version/list/env/fmt/vet/doc）放行。
// go build 会执行外部代码编译，保守视为 true。
func goNeedsApproval(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "version", "list", "env", "fmt", "vet", "doc", "help":
		return false
	}
	// install/run/get/build/test/mod 及默认 → true
	return true
}

// gitNeedsApproval 判断 git 子命令是否风险。会修改仓库状态的子命令
// (clone/init/fetch/pull/push/reset/clean/checkout/switch/merge/rebase/cherry-pick/
// apply/am/restore/revert/submodule/update 等) → true；
// 只读（status/log/diff/v/--version/ls-files）放行；branch/tag/remote/config 按
// 是否带只读 flag 细分。
func gitNeedsApproval(args []string) bool {
	if len(args) == 0 {
		return true
	}
	sub := args[0]
	switch sub {
	case "status", "log", "diff", "show", "v", "--version", "--help", "ls-files", "rev-parse", "help":
		return false
	case "branch":
		return !gitHasFlag(args[1:], "-l", "--list", "-a", "--all", "-r", "--remotes", "-vv", "-v")
	case "tag":
		return !gitHasFlag(args[1:], "-l", "--list")
	case "remote":
		return !gitHasFlag(args[1:], "-v", "show")
	case "config":
		return !gitHasFlag(args[1:], "--get", "-g", "--get-all", "--list", "--null")
	}
	// 其余子命令（含 clone/init/fetch/pull/push/reset/checkout/merge 等）→ true
	return true
}

// gitHasFlag 判断 args 是否命中任一 flag（用于区分 git 只读/写子命令）。
func gitHasFlag(args []string, flags ...string) bool {
	for _, a := range args {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

// runCommandTool 执行工作目录内命令的工具。
type runCommandTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*runCommandTool)(nil)
var _ ActionTextProvider = (*runCommandTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *runCommandTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "执行命令"
	}
	if cmd := strings.TrimSpace(args.Command); cmd != "" {
		return "执行命令：" + TruncateRunes(cmd, 30)
	}
	return "执行命令"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *runCommandTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run_command",
		Desc: "执行工作目录内的可执行命令（如 python、git 等系统命令）。只接受可执行文件及其参数数组，不支持 shell 语法（管道 |、重定向 >、&&、; 、通配符等）——需要复杂逻辑请先 write_file 编写脚本再用对应解释器执行。命令工作目录固定为 ~/.jot/workspace（或其中的子目录），cwd 超出工作目录将被拒绝。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "要执行的可执行命令（不含参数），如 python、git",
				Required: true,
			},
			"args": {
				Type:     schema.Array,
				ElemInfo: &schema.ParameterInfo{Type: schema.String},
				Desc:     "传给命令的参数数组（不含 shell 语法）",
				Required: false,
			},
			"cwd": {
				Type:     schema.String,
				Desc:     "命令工作目录（工作目录内子目录或省略，缺省为工作目录根）；超出工作目录将被拒绝",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行命令：校验 → cwd 边界校验 → 黑名单审批 → LookPath 定位 →
// 超时执行，规整输出回填模型。
func (t *runCommandTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Cwd     string   `json:"cwd"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 run_command 参数失败: %w", err)
	}
	command := strings.TrimSpace(args.Command)
	if command == "" {
		return "", errors.New("run_command 参数缺少 command")
	}
	if err := validateTextLen("command", command, maxToolShortText); err != nil {
		return "", err
	}

	// cwd 边界校验：提供时须为工作目录内子目录（根目录自身合法），缺省为工作目录根
	cwd, err := t.wsRoot()
	if err != nil {
		return "", err
	}
	if idx := strings.TrimSpace(args.Cwd); idx != "" {
		cwd, err = t.resolvePath(idx)
		if err != nil {
			return "", err
		}
	}

	// 审批检查点：每次命令执行都先请求审批，是否不可绕过由 CommandNeedsApproval
	// （命中破坏性命令 / 高风险 net 类子命令）决定。
	// critical=true=破坏宿主系统或高风险子命令，任何审批模式都必须确认（不可绕过）；
	// critical=false=普通命令，仅 confirm_every 模式确认，review/auto 自动放行。
	summary := strings.TrimSpace(strings.Join(args.Args, " "))
	if err := t.requestApproval(ctx, "run_command", "执行命令："+command+" "+summary, CommandNeedsApproval(command, args.Args)); err != nil {
		return "", err
	}

	// 用 exec.LookPath 先定位命令，找不到给出友好中文错误
	if _, lookErr := exec.LookPath(command); lookErr != nil {
		return "", fmt.Errorf("环境中没有命令 “%s”，请改用系统已有命令或先 write_file 脚本再执行", command)
	}

	// 带超时执行：CommandContext 在超时后终止子进程；输出经有界缓冲累计
	runCtx, cancel := context.WithTimeout(ctx, runCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, command, args.Args...)
	cmd.Dir = cwd
	// 用带上限的 io.Writer 接管 stdout/stderr：超过 maxCommandOutputBytes 即停止
	// 累加（但进程继续运行至完成/超时），保证内存有界，杜绝超大输出撑爆内存。
	var outBuf, errBuf limitedBuffer
	outBuf.max, errBuf.max = maxCommandOutputBytes, maxCommandOutputBytes
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	out := append(outBuf.buf, errBuf.buf...)
	if runErr != nil {
		// 用户取消优先
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// 超时终止
		if runCtx.Err() == context.DeadlineExceeded {
			return "", errors.New("命令执行超时已终止")
		}
		return "", fmt.Errorf("命令执行失败: %w\n%s", runErr, TruncateRunes(string(out), MaxResultLen))
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "(命令执行成功，无输出)", nil
	}
	// 输出超长截断
	if n := len([]rune(text)); n > MaxResultLen {
		return TruncateRunes(text, MaxResultLen) + "\n[输出过长已截断]", nil
	}
	return text, nil
}

// limitedBuffer 有界输出缓冲：累计写入至多 max 字节，达到上限后返回写入成功
// （让进程继续运行）但不再实际累加，保证内存占用有界。
type limitedBuffer struct {
	buf  []byte
	max  int
	full bool
}

// Write 实现 io.Writer：在未达上限时累计，达上限后丢弃剩余输入但返回 len(p)，
// 避免阻塞/干扰子进程写入。
func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.max <= 0 || b.full {
		return len(p), nil
	}
	remaining := b.max - len(b.buf)
	if remaining <= 0 {
		b.full = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf = append(b.buf, p[:remaining]...)
		b.full = true
	} else {
		b.buf = append(b.buf, p...)
	}
	return len(p), nil
}

// NewRunCommand 创建命令执行工具。
func NewRunCommand(ctx *Context) tool.InvokableTool {
	return &runCommandTool{fsToolBase: fsToolBase{ctx: ctx}}
}
