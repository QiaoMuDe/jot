package tools

// 本文件实现 AI 助手工作目录（~/.jot/workspace）的文件操作工具：
// read_file 分页读取、write_file 创建/覆盖/追加写入、list_dir 列出目录结构。
// 三者通过 config.WorkspaceFilePath 做路径边界校验，仅允许操作工作目录内的文件，
// 越权路径一律拒绝。写入与危险命令由 Context.Approver 审批钩子接管
// （覆盖已存在文件=常规审批 critical=false；命令黑名单命中=不可绕过 critical=true）。

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/config"
)

// readFilePageSize read_file 缺省读取页大小（按 rune 计），超长文件可分页续读。
const readFilePageSize = 8000

// maxListDepth list_dir 最大递归深度上限。
const maxListDepth = 5

// defaultListDepth list_dir 缺省递归深度。
const defaultListDepth = 2

// maxListDirRunes list_dir 累计输出 rune 上限：遍历中途达到该值（剩余不足
// listDirHeadroomRunes）即提前中断，避免先全量构建再截断造成的内存浪费。
const maxListDirRunes = 20000

// listDirHeadroomRunes list_dir 提前中断的剩余阈值：距上限不足此数量即停止收集。
const listDirHeadroomRunes = 2000

// errListTooLarge list_dir 内部哨兵错误：用于使 WalkDir 提前中断遍历。
var errListTooLarge = errors.New("list_dir 结果超长，提前中断遍历")

// fsToolBase 文件工具共享基础：持有执行上下文与可注入的工作目录根（测试用）。
type fsToolBase struct {
	ctx           *Context
	workspaceRoot string // 测试注入用，空则取 config.WorkspaceDir()
}

// wsRoot 返回工作目录根路径：注入优先，否则取 config.WorkspaceDir()。
func (b *fsToolBase) wsRoot() (string, error) {
	if b.workspaceRoot != "" {
		return b.workspaceRoot, nil
	}
	return config.WorkspaceDir()
}

// resolvePath 把用户传入路径解析为工作目录内的绝对路径并做边界校验。
func (b *fsToolBase) resolvePath(p string) (string, error) {
	root, err := b.wsRoot()
	if err != nil {
		return "", err
	}
	return config.WorkspaceFilePath(root, p)
}

// requestApproval 通过 Context.Approver 请求用户审批：宿主未注入时视为无审批
// 机制直接放行（返回 nil）；拒绝时返回拒绝错误文本，调用方不执行写操作。
// critical 表示是否为不可绕过的危险操作（如命令黑名单命中），透传给审批实现。
func (b *fsToolBase) requestApproval(ctx context.Context, toolName, summary string, critical bool) error {
	if b.ctx == nil || b.ctx.Approver == nil {
		return nil
	}
	return b.ctx.Approver.RequestApproval(ctx, toolName, summary, critical)
}

// readFileTool 读取工作目录内文件的工具。
type readFileTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*readFileTool)(nil)
var _ ActionTextProvider = (*readFileTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *readFileTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "读取文件"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "读取文件：" + TruncateRunes(p, 30)
	}
	return "读取文件"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *readFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: "读取工作目录内的文件，支持分页；offset 指定起始字符偏移、length 指定读取字符数（默认 8000）。文件内容超长时可通过递增 offset 续读。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要读取的文件路径，相对 ~/.jot/workspace 或为其内绝对路径",
				Required: true,
			},
			"offset": {
				Type:     schema.Number,
				Desc:     "起始字符偏移（从 0 开始），缺省 0；续读时传上一段返回的结尾 offset",
				Required: false,
			},
			"length": {
				Type:     schema.Number,
				Desc:     "本次读取的字符数（按 rune 计），缺省 8000",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行读取：边界校验 → 有界流式读取 offset/length 指定 rune 片段。
// 采用 os.Open + bufio 按 rune 流式定位与切片，绝不把整个文件载入内存，
// 因此超大文件（如数十 MB 以上）也只会读取所需片段，内存占用有界。
func (t *readFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path   string  `json:"path"`
		Offset float64 `json:"offset"`
		Length float64 `json:"length"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 read_file 参数失败: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", errors.New("read_file 参数缺少 path")
	}
	if err := validateTextLen("path", path, maxToolShortText); err != nil {
		return "", err
	}
	fullPath, err := t.resolvePath(path)
	if err != nil {
		return "", err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	offset := int(args.Offset)
	if offset < 0 {
		offset = 0
	}
	length := int(args.Length)
	if length <= 0 {
		length = readFilePageSize
	}

	// 有界流式读取：仅读取 offset 之后至多 length 个 rune，并多探测一个 rune
	// 判断是否还有后续内容（用于续读提示），内存占用 ≤ length+1 个 rune。
	msg, hasMore, err := readRunesBounded(f, offset, length)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	// offset 越界或文件为空：没有任何内容可读
	if msg == "" {
		return "已全部读取", nil
	}
	if hasMore {
		msg += fmt.Sprintf("\n[文件未读完，可继续用 offset=%d 读取后续内容]", offset+length)
	}
	return msg, nil
}

// readRunesBounded 从 r 中定位第 offset 个 rune 后读取至多 length 个 rune。
// 返回读取到的文本、是否还有后续内容（是否读到文件末尾）、以及读取错误。
// 若 offset 越过文件末尾（无可读内容）返回空字符串。内存占用 ≤ length+1 个 rune。
func readRunesBounded(r io.Reader, offset, length int) (string, bool, error) {
	br := bufio.NewReader(r)
	runes := make([]rune, 0, length)
	pos := 0 // 全局 rune 位置计数（0 基），用于精确跳过 offset 之前的 rune
	for {
		rn, _, err := br.ReadRune()
		if err != nil {
			if err == io.EOF {
				return string(runes), false, nil
			}
			return "", false, err
		}
		if pos < offset {
			pos++
			continue // 尚未到达目标偏移：跳过
		}
		pos++
		runes = append(runes, rn)
		if len(runes) >= length {
			// 已读够：多探测一个 rune 判断是否还有后续内容
			_, _, perr := br.ReadRune()
			return string(runes), perr != io.EOF, nil
		}
	}
}

// NewReadFile 创建读取文件工具。
func NewReadFile(ctx *Context) tool.InvokableTool {
	return &readFileTool{fsToolBase: fsToolBase{ctx: ctx}}
}

// writeFileTool 创建/覆盖/追加写入工作目录内文件的工具。
type writeFileTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*writeFileTool)(nil)
var _ ActionTextProvider = (*writeFileTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *writeFileTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "写入文件"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "写入文件：" + TruncateRunes(p, 30)
	}
	return "写入文件"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *writeFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "write_file",
		Desc: "在工作目录内创建/覆盖/追加写入文件。内容超长时写为 multiples 个文件需自行组织，单文件写入按顺序。append=false 时覆盖已存在文件（会被审批机制检查）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要写入的文件路径，相对 ~/.jot/workspace 或为其内绝对路径",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "要写入的文件内容",
				Required: true,
			},
			"append": {
				Type:     schema.Boolean,
				Desc:     "true=追加到文件末尾，false=覆盖已存在文件（缺省 false）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行写入：边界校验 → 内容长度校验 → 覆盖场景审批检查 →
// 自动创建父目录后写入。
func (t *writeFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Append  bool   `json:"append"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 write_file 参数失败: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", errors.New("write_file 参数缺少 path")
	}
	if err := validateTextLen("path", path, maxToolShortText); err != nil {
		return "", err
	}
	if err := validateTextLen("content", args.Content, maxToolLongText); err != nil {
		return "", err
	}
	fullPath, err := t.resolvePath(path)
	if err != nil {
		return "", err
	}

	// 审批检查点：覆盖已存在文件（append=false 且文件已存在）时需取得用户批准；
	// 未被拒绝（Approver 未注入/批准）则继续执行
	if !args.Append {
		if _, statErr := os.Lstat(fullPath); statErr == nil {
			if err := t.requestApproval(ctx, "write_file", fmt.Sprintf("覆盖文件：%s", path), false); err != nil {
				return "", err
			}
		}
	}

	// 自动创建父目录（幂等）
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("创建父目录失败: %w", err)
	}

	if args.Append {
		f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return "", fmt.Errorf("打开文件追加失败: %w", err)
		}
		_, wErr := f.WriteString(args.Content)
		closeErr := f.Close()
		if wErr != nil {
			return "", fmt.Errorf("追加写入失败: %w", wErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("关闭文件失败: %w", closeErr)
		}
		return fmt.Sprintf("已追加写入文件：%s", path), nil
	}

	if err := os.WriteFile(fullPath, []byte(args.Content), 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return fmt.Sprintf("已写入文件：%s", path), nil
}

// NewWriteFile 创建写入文件工具。
func NewWriteFile(ctx *Context) tool.InvokableTool {
	return &writeFileTool{fsToolBase: fsToolBase{ctx: ctx}}
}

// listDirTool 列出工作目录内目录结构的工具。
type listDirTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*listDirTool)(nil)
var _ ActionTextProvider = (*listDirTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *listDirTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "列出目录"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "列出目录：" + TruncateRunes(p, 30)
	}
	return "列出目录"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *listDirTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_dir",
		Desc: "列出工作目录内的目录结构，支持深度控制。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要列出的目录路径，缺省为工作目录根；相对 ~/.jot/workspace 或为其内绝对路径",
				Required: false,
			},
			"depth": {
				Type:     schema.Number,
				Desc:     "递归深度（缺省 2，上限 5）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行目录列举：边界校验（根目录自身合法）→ WalkDir 递归 → 按
// "目录: 相对路径" / "文件: 相对路径" 中文化逐行输出，超长截断。
func (t *listDirTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path  string  `json:"path"`
		Depth float64 `json:"depth"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 list_dir 参数失败: %w", err)
	}
	root, err := t.wsRoot()
	if err != nil {
		return "", err
	}
	target := strings.TrimSpace(args.Path)
	if target == "" {
		target = root
	}
	fullPath, err := t.resolvePath(target)
	if err != nil {
		return "", err
	}

	depth := int(args.Depth)
	if depth <= 0 {
		depth = defaultListDepth
	}
	if depth > maxListDepth {
		depth = maxListDepth
	}

	var b strings.Builder
	sep := string(filepath.Separator)
	// 累计输出的 rune 数，逼近上限时提前中断遍历（内存有界），避免先全量构建再截断。
	cumRunes := 0
	walkErr := filepath.WalkDir(fullPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 子项不可访问时跳过不中断（如权限受限项），其余错误原样返回
			if d != nil && !d.IsDir() {
				return nil
			}
			return err
		}
		rel, relErr := filepath.Rel(fullPath, p)
		if relErr != nil {
			return nil
		}
		// 计算层级：相对路径为 "."（根）时 0 级，否则按分隔符计数 +1
		level := 0
		if rel != "." {
			level = strings.Count(rel, sep) + 1
		}
		if level > depth {
			return fs.SkipDir
		}
		typeName := "文件"
		entry := rel
		if d.IsDir() {
			typeName = "目录"
		}
		if rel == "." {
			entry = "."
		}
		// 累计行写入前先探测长度：剩余空间不足时提前终止遍历
		line := typeName + ": " + entry
		if remaining := maxListDirRunes - cumRunes; remaining < listDirHeadroomRunes {
			return errListTooLarge
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
		cumRunes += len([]rune(line)) + 1 // 计入换行符
		return nil
	})
	if walkErr != nil {
		if walkErr == errListTooLarge {
			// 结果超长提前中断遍历：仍返回已收集内容（截断），不视为错误，附加提示
			return TruncateRunes(b.String(), MaxResultLen) + "\n[目录内容过长，已提前停止列举]", nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("列出目录失败: %w", walkErr)
	}
	return TruncateRunes(b.String(), MaxResultLen), nil
}

// NewListDir 创建列出目录工具。
func NewListDir(ctx *Context) tool.InvokableTool {
	return &listDirTool{fsToolBase: fsToolBase{ctx: ctx}}
}
