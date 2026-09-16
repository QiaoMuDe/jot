package tools

// 本文件实现 read_file 工具：分页读取 AI 助手工作目录（~/.jot/workspace）内的
// 文件。经 fsToolBase 做路径边界校验（config.WorkspaceFilePath，杜绝 ../ 逃逸与
// 越界绝对路径）并经 os.Root 目录句柄打开（第二道防逃逸），采用 bufio 按 rune
// 流式定位与切片（offset/length），绝不把整个文件载入内存，超大文件内存占用也
// 始终有界。
// 二进制文件（go-kit fs 检测）直接跳过并提示，避免输出乱码浪费 token。

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	kitfs "gitee.com/MM-Q/go-kit/fs"
)

// readFilePageSize read_file 缺省读取页大小（按 rune 计），超长文件可分页续读。
const readFilePageSize = 8000

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
		Desc: "读取工作目录内的文件，支持分页；offset 指定起始字符偏移、length 指定读取字符数（默认 8000）。文件内容超长时可通过递增 offset 续读。line_numbers=true 时每行输出带「行 N: 」前缀（行号为全局行号，与分页组合时从 offset 所在行起编号），供 edit_file 行级替换定位使用；复制带前缀的内容用于 edit_file 的 find 片段时须去掉「行 N: 」前缀。",
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
			"line_numbers": {
				Type:     schema.Boolean,
				Desc:     "是否输出带「行 N: 」前缀的行号，缺省 false；为 true 时供 edit_file 行级替换定位",
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
		Path        string  `json:"path"`
		Offset      float64 `json:"offset"`
		Length      float64 `json:"length"`
		LineNumbers bool    `json:"line_numbers"`
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
	root, rel, err := t.openRootFor(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开工作目录失败: %w", err)
	}
	defer func() { _ = root.Close() }()

	f, err := root.Open(rel)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	// 二进制检测：二进制文件按文本读取只会输出乱码（浪费 token 且误导后续
	// edit_file 拿乱码片段做 find），直接跳过并提示（库检测后已回绕文件指针）
	isBin, err := kitfs.IsBinaryFile(f)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	if isBin {
		return "文件为二进制，已跳过读取", nil
	}

	offset := int(args.Offset)
	if offset < 0 {
		offset = 0
	}
	length := int(args.Length)
	if length <= 0 {
		length = readFilePageSize
	}

	// 有界流式读取：仅读取 offset 之后至多 length 个 rune，并多探测一个 rune
	// 判断是否还有后续内容（用于续读提示），内存占用 ≤ length+1 个 rune；
	// 同时统计 offset 之前的换行数，供 line_numbers 模式计算全局起始行号。
	msg, hasMore, linesBefore, err := readRunesBounded(f, offset, length)
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
	// line_numbers=true：给内容逐行加「行 N: 」前缀（起始行号 = offset 前换行数 + 1），
	// 供 edit_file 行级替换定位；行号前缀仅作寻址坐标，复制片段用于 find 时须去掉前缀
	if args.LineNumbers {
		msg = numberLines(msg, linesBefore+1)
	}
	if hasMore {
		msg += fmt.Sprintf("\n[文件未读完，可继续用 offset=%d 读取后续内容]", offset+length)
	}
	return msg, nil
}

// readRunesBounded 从 r 中定位第 offset 个 rune 后读取至多 length 个 rune。
// 返回读取到的文本、是否还有后续内容（是否读到文件末尾）、offset 之前的换行数
// （linesBefore，供行号模式计算起始行号）、以及读取错误。
// 若 offset 越过文件末尾（无可读内容）返回空字符串。内存占用 ≤ length+1 个 rune。
func readRunesBounded(r io.Reader, offset, length int) (string, bool, int, error) {
	br := bufio.NewReader(r)
	runes := make([]rune, 0, length)
	pos := 0   // 全局 rune 位置计数（0 基），用于精确跳过 offset 之前的 rune
	lines := 0 // offset 之前的换行数（rune 为 \n 时累计）
	for {
		rn, _, err := br.ReadRune()
		if err != nil {
			if err == io.EOF {
				return string(runes), false, lines, nil
			}
			return "", false, 0, err
		}
		if pos < offset {
			if rn == '\n' {
				lines++
			}
			pos++
			continue // 尚未到达目标偏移：跳过
		}
		pos++
		runes = append(runes, rn)
		if len(runes) >= length {
			// 已读够：多探测一个 rune 判断是否还有后续内容
			_, _, perr := br.ReadRune()
			return string(runes), perr != io.EOF, lines, nil
		}
	}
}

// NewReadFile 创建读取文件工具。
func NewReadFile(ctx *Context) tool.InvokableTool {
	return &readFileTool{fsToolBase: fsToolBase{ctx: ctx}}
}
