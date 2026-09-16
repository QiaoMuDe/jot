package tools

// 本文件实现 grep_file 工具：在 AI 助手工作目录（~/.jot/workspace）内按内容
// 搜索匹配行，类似 grep。path 指向文件则只搜该文件，指向目录则递归搜该目录下
// 所有文件；pattern 为搜索内容（缺省纯字符串字面匹配，regex=true 时按 Go 正则
// 解释；case_insensitive=true 忽略大小写）；目录模式下 file_pattern 按通配限定
// 文件名（path.Match 匹配基名，如 *.go 匹配所有层级的 .go）。
// 输出格式对齐 read_file 的「行 N: 」前缀：单文件输出「行 N: 内容」，目录模式
// 输出「相对路径:行 N: 内容」，行号可直接作为 edit_file 行级替换的寻址坐标。
// 与 edit_file 的 find 区别：find 有空白归一化兜底（改错也能命中），grep_file
// 是定位工具，不做归一化，否则行号与原文失真。
// 纯只读不触发审批；路径经 fsToolBase 边界校验；逐行流式读取不载入全文；二进制
// 文件经 go-kit fs 检测后跳过；输出 rune 有界（仿 glob 提前中断）。与 glob 的
// 边界：glob 按路径匹配找文件，grep_file 按内容匹配找行。

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
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	kitfs "gitee.com/MM-Q/go-kit/fs"
)

// grep_file 输出与行级控制常量：
const (
	maxGrepRunes       = 20000 // 累计输出 rune 上限（逐行写入前探测，仿 glob）
	grepHeadroomRunes  = 2000  // 提前中断的剩余阈值：距上限不足此值即停止收集
	maxGrepLineRunes   = 200   // 单行内容截断上限（超长行截断后加 …）
	maxGrepContextLine = 50    // context_lines 参数上限
)

// errGrepTooLarge grep_file 内部哨兵错误：结果累计逼近上限时提前中断遍历/读取。
var errGrepTooLarge = errors.New("grep_file 结果超长，提前中断")

// errGrepBinary grep_file 哨兵错误：目标文件为二进制（单文件模式提示、目录模式静默跳过）。
var errGrepBinary = errors.New("grep_file 二进制文件跳过")

// grepFileTool 按内容搜索工作目录内文件/目录的工具。
type grepFileTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*grepFileTool)(nil)
var _ ActionTextProvider = (*grepFileTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *grepFileTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "搜索文件"
	}
	if p := strings.TrimSpace(args.Pattern); p != "" {
		return "搜索文件：" + TruncateRunes(p, 30)
	}
	return "搜索文件"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *grepFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "grep_file",
		Desc: "在工作目录内按内容搜索匹配行（类似 grep）。需要定位哪份文件、第几行包含某段内容（如确认某内容是否已存在、为 edit_file 行级替换定位行号）时调用。path 指向文件则只搜该文件，指向目录则递归搜该目录下所有文件；pattern 是要搜索的内容，默认纯字符串字面匹配（精确内容，不做空白归一化），regex=true 时按 Go 正则解释；case_insensitive=true 忽略大小写；目录模式下 file_pattern 可限定只搜文件名匹配通配的文件（如 *.go，匹配所有层级）；context_lines=N 在匹配行前后附带 N 行上下文。注意：按路径匹配找文件请用 glob，本工具只按内容匹配找行；纯只读不修改文件。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要搜索的文件或目录路径，相对 ~/.jot/workspace 或为其内绝对路径；指向文件则单文件搜索，指向目录则递归搜索",
				Required: true,
			},
			"pattern": {
				Type:     schema.String,
				Desc:     "要搜索的内容：默认纯字符串字面匹配（精确内容，不做空白归一化），regex=true 时按 Go 正则解释（不支持环视断言）",
				Required: true,
			},
			"regex": {
				Type:     schema.Boolean,
				Desc:     "是否把 pattern 按 Go 正则解释，缺省 false（纯字符串字面匹配）",
				Required: false,
			},
			"case_insensitive": {
				Type:     schema.Boolean,
				Desc:     "是否忽略大小写，缺省 false",
				Required: false,
			},
			"file_pattern": {
				Type:     schema.String,
				Desc:     "仅目录模式有效：只搜索文件名匹配该通配符的文件（如 *.go 匹配所有层级的 .go 文件）",
				Required: false,
			},
			"context_lines": {
				Type:     schema.Number,
				Desc:     "匹配行前后附带 N 行上下文（0-50 的整数），缺省 0",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行搜索：参数解析与校验 → 匹配器装配（正则/大小写）→ 路径边界
// 校验 → 文件/目录分派（单文件搜索或 WalkDir 递归有界遍历）。
func (t *grepFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path            string  `json:"path"`
		Pattern         string  `json:"pattern"`
		Regex           bool    `json:"regex"`
		CaseInsensitive bool    `json:"case_insensitive"`
		FilePattern     string  `json:"file_pattern"`
		ContextLines    float64 `json:"context_lines"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 grep_file 参数失败: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", errors.New("grep_file 参数缺少 path")
	}
	if err := validateTextLen("path", path, maxToolShortText); err != nil {
		return "", err
	}
	pattern := args.Pattern
	if strings.TrimSpace(pattern) == "" {
		return "", errors.New("grep_file 参数缺少 pattern")
	}
	if err := validateTextLen("pattern", pattern, maxToolShortText); err != nil {
		return "", err
	}
	ctxLines, err := validateGrepContextLines(args.ContextLines)
	if err != nil {
		return "", err
	}
	filePattern := strings.TrimSpace(args.FilePattern)
	if filePattern != "" {
		if err := validateTextLen("file_pattern", filePattern, maxToolShortText); err != nil {
			return "", err
		}
		if err := validateGlobPattern(filePattern); err != nil {
			return "", err
		}
	}

	// 匹配器装配：正则模式编译（case_insensitive 时加 (?i) 前缀，编译失败报错带
	// Go 正则语法位置）；纯字符串模式 case_insensitive 时 pattern 统一小写一次，
	// 匹配时逐行小写后 Contains（unicode 安全）
	matcher := grepMatcher{}
	if args.Regex {
		p := pattern
		if args.CaseInsensitive {
			p = "(?i)" + p
		}
		re, err := regexp.Compile(p)
		if err != nil {
			return "", fmt.Errorf("grep_file 正则编译失败: %w", err)
		}
		matcher.re = re
	} else {
		matcher.pattern = pattern
		matcher.ci = args.CaseInsensitive
		if args.CaseInsensitive {
			matcher.pattern = strings.ToLower(pattern)
		}
	}

	fullPath, err := t.resolvePath(path)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(fullPath)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("grep_file 路径不存在或不可访问: %w", err)
	}

	em := &grepEmitter{b: &strings.Builder{}}
	if !fi.IsDir() {
		// 单文件模式：行号从 1 起，输出「行 N: 内容」
		hits, err := t.grepSingleFile(ctx, fullPath, "", &matcher, ctxLines, em)
		if err != nil {
			if errors.Is(err, errGrepBinary) {
				return "文件为二进制，已跳过搜索", nil
			}
			if errors.Is(err, errGrepTooLarge) {
				return em.b.String() + "\n[结果超长，已提前停止]", nil
			}
			return "", err
		}
		if hits == 0 {
			return "未找到匹配行", nil
		}
		return em.b.String(), nil
	}

	// 目录模式：WalkDir 递归（不跟随符号链接目录），只处理普通文件（符号链接
	// 文件天然跳过，防逃逸），子项不可访问时跳过不中断；输出带相对工作区根的路径前缀
	wsRoot, err := t.wsRoot()
	if err != nil {
		return "", err
	}
	var walkErr error
	_ = filepath.WalkDir(fullPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && !d.IsDir() {
				return nil
			}
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		if ctx.Err() != nil {
			walkErr = ctx.Err()
			return ctx.Err()
		}
		if filePattern != "" && !globMatches(filePattern, d.Name()) {
			return nil
		}
		rel, relErr := filepath.Rel(wsRoot, p)
		if relErr != nil {
			return nil
		}
		if _, err := t.grepSingleFile(ctx, p, toSlash(rel), &matcher, ctxLines, em); err != nil {
			if errors.Is(err, errGrepBinary) {
				return nil // 目录模式静默跳过二进制文件
			}
			walkErr = err
			return err
		}
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, errGrepTooLarge) {
			if em.b.Len() == 0 {
				return "未找到匹配项", nil
			}
			return em.b.String() + "\n[结果超长，已提前停止]", nil
		}
		return "", walkErr
	}
	if em.b.Len() == 0 {
		return "未找到匹配项", nil
	}
	return em.b.String(), nil
}

// NewGrepFile 创建按内容搜索文件/目录的工具。
func NewGrepFile(ctx *Context) tool.InvokableTool {
	return &grepFileTool{fsToolBase: fsToolBase{ctx: ctx}}
}

// grepMatcher 封装匹配逻辑：正则模式用 re.MatchString；纯字符串模式用 Contains
// （case_insensitive 时 pattern 已小写，匹配时逐行小写后比较）。
type grepMatcher struct {
	pattern string // 纯字符串模式的小写化搜索内容（未忽略大小写时为原样）
	re      *regexp.Regexp
	ci      bool // 纯字符串模式是否忽略大小写
}

// match 判断单行是否命中。
func (m *grepMatcher) match(line string) bool {
	if m.re != nil {
		return m.re.MatchString(line)
	}
	if m.ci {
		return strings.Contains(strings.ToLower(line), m.pattern)
	}
	return strings.Contains(line, m.pattern)
}

// grepEmitter 有界输出缓冲：逐行写入前探测剩余 rune 空间，逼近上限返回
// errGrepTooLarge 提前中断；行间以换行分隔。
type grepEmitter struct {
	b     *strings.Builder
	cum   int
	first bool
}

// emit 写入一行输出（含行间换行），剩余空间不足时返回 errGrepTooLarge。
func (e *grepEmitter) emit(line string) error {
	if remaining := maxGrepRunes - e.cum; remaining < grepHeadroomRunes {
		return errGrepTooLarge
	}
	if !e.first {
		e.b.WriteString("\n")
	}
	e.b.WriteString(line)
	e.first = true
	e.cum += utf8.RuneCountInString(line)
	return nil
}

// grepSingleFile 搜索单个文件：二进制探测 → 流式逐行匹配 → 输出命中行与上下文。
// prefix 非空时每行加「prefix:」前缀（目录模式相对工作区根的路径），空则为单文件
// 模式的「行 N: 」格式。返回命中行数；二进制文件返回 errGrepBinary（由调用方决定
// 提示或跳过）。内存占用：前置上下文环形缓冲至多 context_lines 行，不载入全文。
func (t *grepFileTool) grepSingleFile(ctx context.Context, fullPath, prefix string, m *grepMatcher, ctxLines int, em *grepEmitter) (int, error) {
	f, err := os.Open(fullPath)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, err
	}
	defer func() { _ = f.Close() }()

	// 二进制判定：go-kit fs 读取前 8000 字节检查 NUL 判为二进制（库检测后已回绕
	// 文件指针）；目录模式静默跳过、单文件模式提示
	isBin, err := kitfs.IsBinaryFile(f)
	if err != nil {
		return 0, err
	}
	if isBin {
		return 0, errGrepBinary
	}

	var (
		hits       = 0
		lineNum    = 0
		lastOut    = 0   // 已输出的最大行号（去重前置/后置上下文）
		pendingEnd = 0   // 当前匹配行的后置上下文窗口结束行号（匹配行号 + context_lines）
		ringNum    []int // 前置上下文环形缓冲：最近 context_lines 个非匹配行的行号
		ringText   []string
	)
	// emit 输出单行：截断超长行后按「行 N: 内容」（prefix 非空时加路径前缀）
	emit := func(num int, text string) error {
		line := "行 " + strconv.Itoa(num) + ": " + truncateGrepLine(text)
		if prefix != "" {
			line = prefix + ":" + line
		}
		return em.emit(line)
	}

	br := bufio.NewReader(f)
	for {
		if ctx.Err() != nil {
			return hits, ctx.Err()
		}
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return hits, err
		}
		if err == io.EOF && line == "" {
			break
		}
		lineNum++
		text := strings.TrimSuffix(line, "\n")
		text = strings.TrimSuffix(text, "\r")

		if m.match(text) {
			hits++
			// 先输出前置上下文（环形缓冲中尚未输出的行），再输出命中行
			for i, num := range ringNum {
				if num > lastOut {
					if err := emit(num, ringText[i]); err != nil {
						return hits, err
					}
					lastOut = num
				}
			}
			if err := emit(lineNum, text); err != nil {
				return hits, err
			}
			lastOut = lineNum
			pendingEnd = lineNum + ctxLines
			ringNum, ringText = nil, nil
			continue
		}
		if ctxLines <= 0 {
			continue // 无上下文模式：只输出命中行
		}
		// 非匹配行：仍在后置上下文窗口内则输出（lastOut 保证不与前置重复）；
		// 无论是否输出都进入前置环形缓冲（超出窗口的行仅作为潜在前置上下文）
		if lineNum <= pendingEnd && lineNum > lastOut {
			if err := emit(lineNum, text); err != nil {
				return hits, err
			}
			lastOut = lineNum
		}
		ringNum = append(ringNum, lineNum)
		ringText = append(ringText, text)
		if len(ringNum) > ctxLines {
			ringNum = ringNum[1:]
			ringText = ringText[1:]
		}
	}
	return hits, nil
}

// truncateGrepLine 截断超长行：内容超过 maxGrepLineRunes 时按 rune 截断并加 …，
// 避免单行巨长内容撑爆输出缓冲。
func truncateGrepLine(s string) string {
	if n := utf8.RuneCountInString(s); n > maxGrepLineRunes {
		return string([]rune(s)[:maxGrepLineRunes]) + "…"
	}
	return s
}

// validateGrepContextLines 校验 context_lines：必须为 0-maxGrepContextLine 的整数
// （缺省 0；负数/非整数/超上限均报错，避免模型传异常值）。
func validateGrepContextLines(v float64) (int, error) {
	if v < 0 {
		return 0, fmt.Errorf("grep_file 的 context_lines 必须为 0-%d 的整数，不可为负数", maxGrepContextLine)
	}
	if v != float64(int(v)) {
		return 0, fmt.Errorf("grep_file 的 context_lines 必须为 0-%d 的整数", maxGrepContextLine)
	}
	n := int(v)
	if n > maxGrepContextLine {
		return 0, fmt.Errorf("grep_file 的 context_lines 过大（上限 %d）", maxGrepContextLine)
	}
	return n, nil
}
