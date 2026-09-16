package tools

// 本文件实现 edit_file 工具：精准编辑 AI 助手工作目录（~/.jot/workspace）内的
// 已存在文件，支持片段替换与行级替换两种互斥模式（参数契约与 manage_note 的
// edit 动作一致，复用其匹配逻辑）。经 fsToolBase 做路径边界校验（仅允许操作
// 工作目录内的文件，越权一律拒绝）；每次编辑都会修改已存在文件内容，经
// Context.Approver 审批钩子接管（critical=false，常规审批可取消）。
// 二进制文件（go-kit fs 检测）拒绝编辑，防止按文本 find/replace 破坏内容；
// 检测在审批之前，避免对不可编辑文件发起无谓审批。
// 编辑必须整读整写，设 maxEditFileSize 大小上限保证内存有界。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	kitfs "gitee.com/MM-Q/go-kit/fs"
)

// maxEditFileSize edit_file 可编辑文件大小上限（字节）。编辑必须整读整写，
// 超限拒绝并引导脚本方案，保证内存占用有界。
const maxEditFileSize = 1024 * 1024

// editFileTool 精准编辑工作目录内已存在文件的工具。
type editFileTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*editFileTool)(nil)
var _ ActionTextProvider = (*editFileTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *editFileTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "编辑文件"
	}
	if p := strings.TrimSpace(args.Path); p != "" {
		return "编辑文件：" + TruncateRunes(p, 30)
	}
	return "编辑文件"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *editFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "edit_file",
		Desc: "精准编辑工作目录内已存在的文件（不整文件覆盖）。两种互斥方式：①片段替换提供 find 要替换的原文片段与 replace 新文本，find 优先精确匹配（find 会自动忽略首尾空白），若因空白/换行/缩进差异未命中会自动做空白归一化匹配兜底（标点、文字仍须一致），删除片段时 replace 传空字符串，count 可指定第几次出现（缺省 1），replace_all=true 时替换全部出现（与 count 互斥）；②行级替换提供 line_start 起始行号（必填）与 line_end 结束行号（缺省等于 line_start），将该区间整行替换为 replace（空字符串即删除这些行），行号必须来自 read_file 的 line_numbers=true 输出；line_start 大于文件总行数时为末尾追加语义。只需修改几个字或一句话用片段替换，需要修改连续多行、整段重写、或无法用简短片段定位时用行级替换。文件不存在请先调用 write_file 创建。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "要编辑的文件路径，相对 ~/.jot/workspace 或为其内绝对路径",
				Required: true,
			},
			"find": {
				Type:     schema.String,
				Desc:     "片段替换的原文片段（与 line_start 互斥）；优先精确匹配，若因空白/换行/缩进差异未命中会自动做空白归一化匹配兜底",
				Required: false,
			},
			"replace": {
				Type:     schema.String,
				Desc:     "替换后的新文本，片段替换或行级替换时使用，缺省空字符串（片段替换即删除该片段；行级替换即删除该区间行）",
				Required: false,
			},
			"count": {
				Type:     schema.Number,
				Desc:     "find 片段在文件中第几次出现，仅片段替换时使用，缺省 1；与 replace_all 互斥，不可同时使用",
				Required: false,
			},
			"replace_all": {
				Type:     schema.Boolean,
				Desc:     "是否替换 find 片段的全部出现，仅片段替换时使用，缺省 false（只替换第 count 次出现）；与 count 互斥，不可同时使用",
				Required: false,
			},
			"line_start": {
				Type:     schema.Number,
				Desc:     "行级替换的起始行号（从 1 开始，行号须来自 read_file 的 line_numbers=true 输出），仅行级替换时使用（与 find 互斥）；大于文件总行数时为末尾追加语义",
				Required: false,
			},
			"line_end": {
				Type:     schema.Number,
				Desc:     "行级替换的结束行号（从 1 开始，含该行），仅行级替换时使用，缺省等于 line_start；超出文件总行数时报错",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行编辑：解析校验 → 模式判定 → 边界校验 → 存在性检查 →
// 审批检查 → 大文件防护 → 读入 → 分派（片段/行级）→ 写回。
func (t *editFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Path       string  `json:"path"`
		Find       string  `json:"find"`
		Replace    string  `json:"replace"`
		Count      float64 `json:"count"`
		ReplaceAll bool    `json:"replace_all"`
		LineStart  float64 `json:"line_start"`
		LineEnd    float64 `json:"line_end"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 edit_file 参数失败: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", errors.New("edit_file 参数缺少 path")
	}
	if err := validateTextLen("path", path, maxToolShortText); err != nil {
		return "", err
	}
	if err := validateTextLen("find", args.Find, maxToolFindLen); err != nil {
		return "", err
	}
	if err := validateTextLen("replace", args.Replace, maxToolLongText); err != nil {
		return "", err
	}
	find := strings.TrimSpace(args.Find)

	// 行级模式基础校验：line_start 必须为从 1 开始的正整数（负数无法表达任何行）
	if args.LineStart < 0 {
		return "", errors.New("edit_file 行级替换的 line_start 必须为从 1 开始的正整数，不可为负数")
	}
	// 模式判定：二选一互斥（find 片段替换 / line_start 行级替换）
	if find != "" && args.LineStart > 0 {
		return "", errors.New("edit_file 多种编辑模式不可混用：find+replace（片段替换）与 line_start（行级替换）请二选一")
	}
	if find == "" && args.LineStart <= 0 {
		return "", errors.New("edit_file 无可编辑内容：请提供 find+replace（片段替换）或 line_start（行级替换，含末尾追加）")
	}
	// 片段模式专属参数校验：count 必须为整数（缺省 0 除外）；replace_all 与 count>1 互斥
	if find != "" && args.Count != 0 && args.Count != float64(int(args.Count)) {
		return "", errors.New("edit_file 片段替换的 count 必须为整数（指定第几次出现）")
	}
	if find != "" && args.ReplaceAll && args.Count > 1 {
		return "", errors.New("edit_file 片段替换的 replace_all 与 count>1 互斥，不可同时使用（replace_all 时 count 只接受缺省 1）")
	}

	fullPath, err := t.resolvePath(path)
	if err != nil {
		return "", err
	}

	// 存在性检查：编辑对象必须是已存在文件（不存在是参数错误，无需审批）
	stat, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("edit_file 目标文件不存在，请先用 write_file 创建")
		}
		return "", fmt.Errorf("检查目标文件失败: %w", err)
	}

	// 二进制检测：二进制文件按文本 find/replace 会破坏内容（NUL 被当文本处理），
	// 直接拒绝；检测在审批之前，避免对不可编辑文件发起无谓审批
	isBin, err := kitfs.IsBinaryFilePath(fullPath)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("检查目标文件失败: %w", err)
	}
	if isBin {
		return "", errors.New("edit_file 不支持编辑二进制文件（可能损坏），请用脚本或工具处理")
	}

	// 审批检查点：每次编辑都会修改已存在文件内容，需取得用户批准；
	// 未被拒绝（Approver 未注入/批准）则继续执行
	if err := t.requestApproval(ctx, "edit_file", "编辑文件："+path, false); err != nil {
		return "", err
	}

	// 大文件防护：编辑必须整读整写，超限拒绝并引导脚本方案，保证内存有界
	if stat.Size() > maxEditFileSize {
		return "", fmt.Errorf("edit_file 目标文件过大（%d 字节，上限 %d），建议用 run_command 编写脚本处理", stat.Size(), maxEditFileSize)
	}

	current, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	content := string(current)

	var newContent, feedback string
	if args.LineStart > 0 {
		newContent, feedback, err = t.editByLines(content, args.LineStart, args.LineEnd, args.Replace)
	} else {
		newContent, feedback, err = t.editByFragment(content, find, args.Replace, args.Count, args.ReplaceAll)
	}
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return feedback, nil
}

// editByLines 行级替换：把第 lineStart 行到第 lineEnd 行替换为 replace
// （空即删除该区间）；lineStart 大于总行数时为末尾追加语义。
func (t *editFileTool) editByLines(content string, lineStart, lineEnd float64, replace string) (string, string, error) {
	start := int(lineStart)
	end := int(lineEnd)
	if end < start {
		end = start
	}
	lines := splitNoteLines(content)
	total := len(lines)
	// 末尾追加：start > total 时，在末尾追加 replace 内容；
	// replace 为空时拒绝（避免静默追加空行），追加空行请显式传换行内容
	if start > total {
		if replace == "" {
			return "", "", errors.New("edit_file 末尾追加的内容为空：追加模式下 replace 不可为空（追加空行请显式传换行）")
		}
		newContent := content
		if strings.TrimSpace(content) != "" {
			newContent += "\n\n"
		}
		newContent += replace
		return newContent, "已在文件末尾追加内容", nil
	}
	newContent, replaced, total, err := replaceLines(content, start, end, replace)
	if err != nil {
		// replaceLines 的错误文案面向笔记（含"笔记共 N 行"），且原始错误必然
		// 是行号越界，这里直接构造 edit_file 语义的错误，避免 hack 字符串
		return "", "", fmt.Errorf("edit_file 行级替换的行号越界：文件共 %d 行，请求替换第 %d-%d 行", total, start, end)
	}
	newTotal := len(splitNoteLines(newContent))
	// 构建反馈：基本信息 + 行数变化 + 替换区域上下文预览
	var fb string
	if replaced == 1 {
		fb = fmt.Sprintf("已替换第 %d 行（原 %d 行 → 现 %d 行）", start, total, newTotal)
	} else {
		fb = fmt.Sprintf("已替换第 %d-%d 行（原 %d 行 → 现 %d 行）", start, end, total, newTotal)
	}
	if preview := lineEditPreview(newContent, start, newTotal); preview != "" {
		fb += "：\n" + preview
	}
	return newContent, fb, nil
}

// editByFragment 片段替换：定位第 count 次出现的 find 片段并替换为 replace；
// 优先精确匹配，未命中时做空白归一化匹配兜底。
func (t *editFileTool) editByFragment(content, find, replace string, count float64, replaceAll bool) (string, string, error) {
	if replaceAll {
		newContent, n := replaceAllFragments(content, find, replace)
		if n == 0 {
			_, err := buildEditFileNotFoundHint(find, content)
			return "", "", err
		}
		fb := fmt.Sprintf("文件片段已全部替换（共 %d 处）", n)
		// 展示第一处 diff 摘要
		if pos := indexNth(content, find, 1); pos >= 0 {
			fb += fmt.Sprintf("：\n-旧: %q\n+新: %q", truncateSnippet(content[pos:pos+len(find)], 80), truncateSnippet(replace, 80))
		}
		return newContent, fb, nil
	}

	n := int(count)
	if n < 1 {
		n = 1
	}
	pos := indexNth(content, find, n)
	matchedLen := len(find)
	matchedKind := "精确"
	if pos < 0 {
		// 兜底：空白归一化匹配（忽略缩进/换行/连续空白差异），映射回原文偏移
		if s, e := findNormalized(content, find, n); s >= 0 {
			pos = s
			matchedLen = e - s
			matchedKind = "空白归一化"
		}
	}
	if pos < 0 {
		_, err := buildEditFileNotFoundHint(find, content)
		return "", "", err
	}
	newContent := content[:pos] + replace + content[pos+matchedLen:]
	fb := fmt.Sprintf("文件片段已替换（第 %d 处，%s匹配）", n, matchedKind)
	fb += fmt.Sprintf("：\n-旧: %q\n+新: %q", truncateSnippet(content[pos:pos+matchedLen], 80), truncateSnippet(replace, 80))
	return newContent, fb, nil
}

// buildEditFileNotFoundHint 构建"片段未找到"的错误信息，附带文件中最相似的片段提示。
func buildEditFileNotFoundHint(find, content string) (string, error) {
	similar, lineNum := findMostSimilar(content, find)
	hint := fmt.Sprintf("未在文件中找到片段「%s」（已尝试空白归一化匹配）", truncateSnippet(find, 60))
	if similar != "" {
		similar = truncateSnippet(similar, 100)
		if lineNum > 0 {
			hint += fmt.Sprintf("。文件中最接近的内容（第 %d 行附近）：\n「%s」", lineNum, similar)
		} else {
			hint += fmt.Sprintf("。文件中最接近的内容：\n「%s」", similar)
		}
	}
	hint += "\n请确认片段是否正确，或调用 read_file 获取精确原文后重试"
	return "", errors.New(hint)
}

// NewEditFile 创建精准编辑文件工具。
func NewEditFile(ctx *Context) tool.InvokableTool {
	return &editFileTool{fsToolBase: fsToolBase{ctx: ctx}}
}
