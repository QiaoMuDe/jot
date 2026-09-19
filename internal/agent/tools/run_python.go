package tools

// 本文件实现 run_python Python 专用执行工具：在 ~/.jot/workspace 工作目录内执行
// Python 代码字符串（code）或工作区内脚本文件（path）。工具自动探测环境可用的
// 解释器（Windows: py→python→python3；UNIX: python3→python→py），对每个候选
// LookPath 后以 --version 冒烟验证（规避 Windows Microsoft Store 的 python stub
// 命中 PATH 但一跑弹商店），取第一个可用者；探测结果进程级缓存（只缓存成功，
// 失败不缓存——用户安装/修复解释器后下一次调用自动重新探测，无需重启）。
//
// 每次执行恒 critical=true 强制审批：Python 能执行任意代码（os.remove / shutil.
// rmtree / urllib 下载执行等），token 黑名单对其无效，审批语义与 run_command 中
// 解释器类命令命中黑名单保持一致。code 模式写入系统临时文件执行（规避 Windows
// 上 -c 的引号/换行/编码转义问题），执行后清理；超时与有界输出复用 run_command
// 的机制（runCommandTimeout + limitedBuffer）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// pythonSmokeTimeout 解释器冒烟验证（--version）超时：超时视为候选不可用，继续尝试下一个。
const pythonSmokeTimeout = 5 * time.Second

// pythonCandidates 返回候选解释器名（按平台排序）：Windows 优先 py（官方 launcher，
// 最可靠，python 可能是 Microsoft Store stub），UNIX 优先 python3。
func pythonCandidates() []string {
	if runtime.GOOS == "windows" {
		return []string{"py", "python", "python3"}
	}
	return []string{"python3", "python", "py"}
}

// smokeTestPython 对解释器候选执行 --version 冒烟验证：确认"能命中"≠"能运行"
// （LookPath 只查 PATH 存在性，Windows Store stub 命中但不弹出版本号）。
func smokeTestPython(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), pythonSmokeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, "--version")
	return cmd.Run() == nil
}

// detectPython 真实解释器探测：按平台候选顺序，LookPath + 冒烟验证取第一个可用者。
// 返回裸错误，统一友好提示由 resolvePython 包装（工具边界契约）。
var detectPython = func() (string, error) {
	for _, name := range pythonCandidates() {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if smokeTestPython(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("已尝试 %s，均不可用", strings.Join(pythonCandidates(), " / "))
}

// pythonNotFoundMsg 探测失败统一友好提示（resolvePython 对任意探测失败包装此文案）。
// 候选顺序随平台动态拼接（Windows 为 py/python/python3，UNIX 为 python3/python/py）。
var pythonNotFoundMsg = "环境中未找到可用的 Python 解释器（已尝试 " + strings.Join(pythonCandidates(), " / ") + "），请先安装 Python 或告知用户"

// 进程级解释器缓存：只缓存成功结果；失败不缓存（用户安装/修复解释器后，
// 下一次调用自动重新探测，无需重启应用）。
var (
	pythonMu       sync.Mutex
	pythonResolved string
)

// resolvePython 带缓存的解释器探测：首次调用触发真实探测并缓存成功路径，后续直接命中。
// 探测失败统一包装为 pythonNotFoundMsg 友好提示，且失败不缓存。
func resolvePython() (string, error) {
	pythonMu.Lock()
	defer pythonMu.Unlock()
	if pythonResolved != "" {
		return pythonResolved, nil
	}
	p, err := detectPython()
	if err != nil {
		return "", errors.New(pythonNotFoundMsg) // 失败不缓存
	}
	pythonResolved = p
	return p, nil
}

// runPythonTool 执行工作目录内 Python 代码/脚本的工具。
type runPythonTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*runPythonTool)(nil)
var _ ActionTextProvider = (*runPythonTool)(nil)

// pythonApprovalPreviewMaxRunes 审批摘要中 code 预览的最大长度：让用户在审批弹窗内
// 看到代码主体判断安全性，又不至于撑爆面板（前端弹窗已支持滚动查看）。
const pythonApprovalPreviewMaxRunes = 2000

// pythonArgsMaxRunes 脚本参数数组整体最大长度：防止模型传入超长参数
// 撑爆 Windows 命令行长度限制（CreateProcess 上限约 32767 字符）。
const pythonArgsMaxRunes = 5000

// pythonRunSummary 构造动作文案（tool_start 记录展示）：code 模式取首行截断，
// path 模式取路径，避免整段代码刷屏。审批摘要请用 pythonApprovalSummary（完整预览）。
func pythonRunSummary(code, path string) string {
	if p := strings.TrimSpace(path); p != "" {
		return "执行 Python：" + TruncateRunes(p, 30)
	}
	if c := strings.TrimSpace(code); c != "" {
		first := c
		if i := strings.IndexAny(first, "\r\n"); i >= 0 {
			first = first[:i]
		}
		return "执行 Python：" + TruncateRunes(first, 30)
	}
	return "执行 Python"
}

// pythonApprovalSummary 构造审批专用摘要：code 模式返回完整代码截断到
// pythonApprovalPreviewMaxRunes（超长尾部追加截断提示），path 模式返回路径。
// 与 pythonRunSummary（动作文案）解耦：动作文案要简短、审批摘要要完整，
// 否则审批弹窗只能看到 code 首行（如导入库语句）而无法判断真实操作是否安全。
func pythonApprovalSummary(code, path string) string {
	if p := strings.TrimSpace(path); p != "" {
		return "执行 Python 脚本：" + TruncateRunes(p, 60)
	}
	if c := strings.TrimSpace(code); c != "" {
		s := "执行 Python 代码：\n" + c
		if n := len([]rune(s)); n > pythonApprovalPreviewMaxRunes {
			return TruncateRunes(s, pythonApprovalPreviewMaxRunes) + "\n[内容过长已截断，仅显示前 " + strconv.Itoa(pythonApprovalPreviewMaxRunes) + " 字]"
		}
		return s
	}
	return "执行 Python"
}

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *runPythonTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Code string `json:"code"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "执行 Python"
	}
	return pythonRunSummary(args.Code, args.Path)
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *runPythonTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run_python",
		Desc: "执行 Python 代码或脚本。自动探测环境中可用的解释器（py/python/python3，取第一个可用者，无需指定解释器名）。code 与 path 二选一：code 为 Python 代码字符串（适合一次性计算/数据处理），path 为工作区内脚本文件路径（适合已有脚本）。命令工作目录固定为 ~/.jot/workspace（或其中的子目录），cwd 超出工作目录将被拒绝。Python 可执行任意代码，每次执行都会请求用户确认。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code": {
				Type:     schema.String,
				Desc:     "Python 代码字符串（与 path 二选一）",
				Required: false,
			},
			"path": {
				Type:     schema.String,
				Desc:     "工作区内 Python 脚本路径（与 code 二选一，支持相对工作区的路径）",
				Required: false,
			},
			"args": {
				Type:     schema.Array,
				ElemInfo: &schema.ParameterInfo{Type: schema.String},
				Desc:     "传给脚本的参数数组",
				Required: false,
			},
			"cwd": {
				Type:     schema.String,
				Desc:     "命令工作目录（工作区内子目录或省略，缺省为工作目录根）；超出工作目录将被拒绝",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行 Python：参数校验 → cwd 边界 → 审批（恒 critical）→ 解释器探测
// （带缓存）→ 临时文件/脚本执行 → 输出规整回填模型。
func (t *runPythonTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Code string   `json:"code"`
		Path string   `json:"path"`
		Args []string `json:"args"`
		Cwd  string   `json:"cwd"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 run_python 参数失败: %w", err)
	}
	code := strings.TrimSpace(args.Code)
	path := strings.TrimSpace(args.Path)
	// code 与 path 必须且只能提供一个
	switch {
	case code == "" && path == "":
		return "", errors.New("run_python 缺少参数：code 与 path 必须二选一")
	case code != "" && path != "":
		return "", errors.New("run_python 参数冲突：code 与 path 只能提供一个")
	}
	if code != "" {
		if err := validateTextLen("code", code, maxToolLongText); err != nil {
			return "", err
		}
	} else if err := validateTextLen("path", path, maxToolShortText); err != nil {
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

	// args 整体长度校验：超长参数在 Windows 上会撑爆命令行长度限制
	//（CreateProcess 上限约 32767 字符），先校验后审批，防止无效内容触发审批
	if n := 0; len(args.Args) > 0 {
		for _, a := range args.Args {
			n += len([]rune(a)) + 1
		}
		if n > pythonArgsMaxRunes {
			return "", fmt.Errorf("脚本参数过长（最多 %d 字符，实际 %d）", pythonArgsMaxRunes, n)
		}
	}

	// 审批检查点：每次执行 Python 都请求审批，critical 恒 true（Python 可执行任意
	// 代码，token 黑名单对其无效；与 run_command 中解释器类命令命中黑名单的语义一致）。
	// 摘要用 pythonApprovalSummary（完整 code 预览），保证审批弹窗可见代码主体。
	if err := t.requestApproval(ctx, "run_python", pythonApprovalSummary(code, path), true); err != nil {
		return "", err
	}

	// 解释器探测（带缓存）：首次真实探测并缓存成功路径，后续直接命中
	interpreter, err := resolvePython()
	if err != nil {
		return "", err
	}

	// 脚本文件准备：code 模式写入系统临时文件（规避 -c 的引号/换行/编码转义），
	// path 模式经 resolvePath 沙箱校验后直接执行
	scriptPath := ""
	if code != "" {
		tmp, err := os.CreateTemp("", "jot_py_*.py")
		if err != nil {
			return "", fmt.Errorf("创建临时脚本失败: %w", err)
		}
		scriptPath = tmp.Name()
		// 延迟清理临时脚本：清理失败不影响主流程结果（errcheck 要求显式忽略）
		defer func() { _ = os.Remove(scriptPath) }()
		if _, err := tmp.WriteString(code); err != nil {
			_ = tmp.Close()
			return "", fmt.Errorf("写入临时脚本失败: %w", err)
		}
		if err := tmp.Close(); err != nil {
			return "", fmt.Errorf("关闭临时脚本失败: %w", err)
		}
	} else {
		scriptPath, err = t.resolvePath(path)
		if err != nil {
			return "", err
		}
		if st, err := os.Stat(scriptPath); err != nil {
			return "", fmt.Errorf("脚本文件不可用: %w", err)
		} else if st.IsDir() {
			return "", errors.New("path 指向的是目录而非脚本文件")
		}
	}

	// 带超时执行：CommandContext 在超时后终止子进程；输出经有界缓冲累计
	runCtx, cancel := context.WithTimeout(ctx, runCommandTimeout)
	defer cancel()
	allArgs := append([]string{scriptPath}, args.Args...)
	cmd := exec.CommandContext(runCtx, interpreter, allArgs...)
	cmd.Dir = cwd
	// 用带上限的 io.Writer 接管 stdout/stderr：超过上限即停止累加（但进程继续运行
	// 至完成/超时），保证内存有界，杜绝超大输出撑爆内存
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
			return "", errors.New("python 执行超时已终止")
		}
		return "", fmt.Errorf("python 执行失败: %w\n%s", runErr, TruncateRunes(string(out), MaxResultLen))
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "(执行成功，无输出)", nil
	}
	// 输出超长截断
	if n := len([]rune(text)); n > MaxResultLen {
		return TruncateRunes(text, MaxResultLen) + "\n[输出过长已截断]", nil
	}
	return text, nil
}

// NewRunPython 创建 Python 执行工具。
func NewRunPython(ctx *Context) tool.InvokableTool {
	return &runPythonTool{fsToolBase: fsToolBase{ctx: ctx}}
}
