package tools

// 本文件实现 run_command 命令执行工具：在 AI 助手工作目录（~/.jot/workspace）
// 内执行可执行命令及其参数数组。它刻意不支持 shell 语法（管道/重定向/&&/;/
// 通配符等）——需要复杂逻辑应先 write_file 编写脚本再用对应解释器执行。
// cwd 仅允许工作目录内子目录或缺省（固定为工作目录根）；每次执行都先经
// Context.Approver 请求审批，critical（是否不可绕过、任何审批模式都必须确认）
// 由 highRiskTokens 判定：命令基名或任一参数 token 命中高危关键字即 critical=true。
//
// 高危判定为"护栏"语义而非隔离：它是让用户对危险命令/脚本执行保留最后否决权，
// 对抗性（改名、脚本包裹、python -c 等）可绕过；真正硬边界依赖 confirm_every 模式。

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

// highRiskTokens 高危关键字集合（map 成员判定，O(1)）：命令基名或任一参数 token
// 命中（大小写不敏感、整词相等）即判为 critical=true（任何审批模式都强制确认）。
// 三类合一：
//  1. 破坏宿主系统的危险命令；
//  2. 可执行任意逻辑的 Shell / 脚本解释器（重开 shell 语义或跑任意代码）；
//  3. 高危动词 / 网络下载工具（安装、删除、下载、执行、改仓库状态等）。
//
// 这是护栏而非隔离：对抗性可改名/脚本包裹绕过，真正硬边界靠 confirm_every 模式。
var highRiskTokens = map[string]bool{
	// 破坏宿主系统的危险命令（含磁盘/文件系统摧毁）
	"rm": true, "del": true, "rmdir": true, "shutdown": true, "reboot": true,
	"mkfs": true, "format": true, "dd": true, "sudo": true, "systemctl": true,
	"reg": true, "diskpart": true, "taskkill": true, "chmod": true, "chown": true,
	"fdisk": true, "parted": true, "cfdisk": true, "wipefs": true, "shred": true,
	"mkfs.ext4": true, "mkfs.xfs": true, "mkfs.btrfs": true, "mkfs.fat": true,
	// Linux 系统/权限管理
	"halt": true, "poweroff": true, "init": true, "killall": true, "pkill": true,
	"userdel": true, "usermod": true, "groupdel": true, "passwd": true, "chgrp": true,
	// 可执行任意逻辑的 Shell / 脚本解释器（重开 shell 语义或跑任意代码）
	"cmd": true, "powershell": true, "pwsh": true, "bash": true, "dash": true,
	"zsh": true, "sh": true, "ksh": true, "csh": true, "fish": true,
	"python": true, "python3": true, "py": true,
	"node": true, "deno": true, "bun": true, "perl": true, "ruby": true, "php": true,
	"lua": true, "luajit": true, "groovy": true, "tclsh": true, "Rscript": true,
	"julia": true, "wscript": true, "cscript": true, "mshta": true, "expect": true,
	// 网络下载工具（默认落盘/下载执行）
	"curl": true, "wget": true,
	// Windows 系统管理 / LOLBin
	"certutil": true, "bitsadmin": true, "wmic": true, "bcdedit": true,
	"vssadmin": true, "fsutil": true, "takeown": true, "icacls": true,
	"cacls": true, "runas": true, "psexec": true,
	// 高危动词 / 子命令（写、删、下载、执行、改仓库状态、自动确认等）
	"install": true, "uninstall": true, "upgrade": true, "remove": true, "unlink": true,
	"erase": true, "delete": true, "clone": true, "checkout": true, "reset": true,
	"clean": true, "merge": true, "rebase": true, "push": true, "pull": true, "fetch": true,
	"exec": true, "eval": true, "run": true, "download": true, "purge": true, "wipe": true,
	"--force": true, "--hard": true, "--yes": true, "-y": true, "--assume-yes": true, "--upgrade": true,
}

// hasHighRiskToken 判断单个 token 是否命中高危集合（整词相等、大小写不敏感）。
func hasHighRiskToken(token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	return highRiskTokens[token]
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

// IsDestructiveCommand 判断命令基名是否命中高危集合。注意：参数中的高危子命令
// 由 CommandNeedsApproval 兜底，此处只看命令本身，供只想查命令名的场景复用。
func IsDestructiveCommand(command string) bool {
	return hasHighRiskToken(commandBaseName(command))
}

// matchArgTokens 遍历参数并切分出 token，命中高危关键字即返回 true。
// 参数内嵌空白（如 cmd 的 /c "del x"）也会被切词，避免拼接型参数漏网。
func matchArgTokens(args []string) bool {
	for _, a := range args {
		for _, f := range strings.Fields(a) {
			if hasHighRiskToken(f) {
				return true
			}
		}
	}
	return false
}

// CommandNeedsApproval 判断命令执行是否需要"不可绕过"的用户审批（critical=true）：
// 命令基名命中高危集合，或任一参数 token 命中高危集合（覆盖"基名无害、参数藏
// 危险子命令"的绕过，如 git clone / pip install / python -m pip install）。
func CommandNeedsApproval(command string, args []string) bool {
	if hasHighRiskToken(commandBaseName(command)) {
		return true
	}
	return matchArgTokens(args)
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
	// （命令基名或任一参数 token 命中 highRiskTokens）决定。
	// critical=true=破坏/解释器/高危子命令，任何审批模式都必须确认（不可绕过）；
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
