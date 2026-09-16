package tools

// 本文件实现 glob 工具：在 AI 助手工作目录（~/.jot/workspace）内按通配符模式
// 查找匹配的文件/目录路径，返回平坦命中列表（相对工作目录），用于精确定位
// 特定后缀或名字的文件，弥补 ls_dir 只能单层穷举的不足。
// 与 read_file/write_file/ls_dir 同源：经 fsToolBase 复用 path 边界校验
// （config.WorkspaceFilePath，杜绝符号链接逃逸），纯只读不触发审批，
// 输出采用 rune 上限有界缓冲（遍历中途逼近即提前中断）。
//
// 场景定位：ls_dir 回答"工作区某层有什么"，glob 回答"哪些路径匹配某
// 模式"。匹配只针对单层结构（* 不跨目录层级，目标目录用字面前缀如 scripts/*.md），
// 不递归（无 **）；需跨多层查找时先用 ls_dir 定位目录再逐层 glob。了解整体
// 结构用 ls_dir，定位文件用 glob，二者在 Desc 中拉开以免模型误选。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// maxGlobRunes glob 累计输出 rune 上限：遍历中途达到该值（剩余不足
// globHeadroomRunes）即提前中断，避免先全量构建再截断造成的内存浪费。
const maxGlobRunes = 20000

// globHeadroomRunes glob 提前中断的剩余阈值：距上限不足此数量即停止收集。
const globHeadroomRunes = 2000

// globMeta 通配模式字符集，用于判断路径段是否为字面（非通配）段。
const globMeta = "*?["

// errGlobTooLarge glob 内部哨兵错误：用于使 WalkDir 提前中断遍历。
var errGlobTooLarge = errors.New("glob 结果超长，提前中断遍历")

// globTool 按通配符模式查找工作目录内路径的工具。
type globTool struct {
	fsToolBase
}

var _ tool.InvokableTool = (*globTool)(nil)
var _ ActionTextProvider = (*globTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）。
func (t *globTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "查找文件"
	}
	if p := strings.TrimSpace(args.Pattern); p != "" {
		return "查找文件：" + TruncateRunes(p, 30)
	}
	return "查找文件"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (t *globTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "glob",
		Desc: "按通配符模式查找工作目录内某层匹配的文件/目录路径，返回平坦命中列表，用于定位特定后缀或名字的文件（脚本、配置、产出物）。支持 *（任意字符，不跨目录层级）、?（单字符）、[abc]（字符类）；模式可用字面前缀限定目录，如 scripts/*.md 只查 scripts 层。不含递归 **，跨多层查找请先用 ls_dir 定位目录再逐层 glob。了解整体目录结构请用 ls_dir，不要用 glob。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {
				Type:     schema.String,
				Desc:     "通配符模式，相对 ~/.jot/workspace；如 *.md、scripts/*.md；不允许绝对路径或包含 .. 逃逸段",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行查找：参数校验 → 推导搜索根并做边界校验 → WalkDir 有界遍历
// → 对工作目录相对路径做 path.Match glob 匹配 → 返回平坦命中列表。
func (t *globTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var args struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 glob 参数失败: %w", err)
	}
	pattern := toSlash(strings.TrimSpace(args.Pattern))
	if pattern == "" {
		return "", errors.New("glob 参数缺少 pattern")
	}
	if err := validateTextLen("pattern", pattern, maxToolShortText); err != nil {
		return "", err
	}
	if err := validateGlobPattern(pattern); err != nil {
		return "", err
	}

	root, err := t.wsRoot()
	if err != nil {
		return "", err
	}

	// 由 pattern 的字面静态前缀推导搜索根（纯遍历优化）：如 scripts/*.md → scripts。
	// 前缀经 resolvePath 边界校验，保证搜索始终落在工作目录内。
	searchRoot := root
	if prefix := globLiteralPrefix(pattern); prefix != "" {
		searchRoot, err = t.resolvePath(prefix)
		if err != nil {
			return "", err
		}
		// 字面前缀对应目录不存在 → 无任何可匹配项，友好返回而非报错。
		if _, statErr := os.Stat(searchRoot); statErr != nil {
			return "未找到匹配项", nil
		}
	}

	var b strings.Builder
	cumRunes := 0
	walkErr := filepath.WalkDir(searchRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 子项不可访问时跳过不中断（如权限受限项），其余错误原样返回
			if d != nil && !d.IsDir() {
				return nil
			}
			return err
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		relSlash := toSlash(rel)
		if relSlash == "." { // 搜索根即工作目录根，根自身不参与匹配
			return nil
		}
		if !globMatches(pattern, relSlash) {
			return nil
		}
		// 累计行写入前先探测长度：剩余空间不足时提前终止遍历
		line := relSlash
		if remaining := maxGlobRunes - cumRunes; remaining < globHeadroomRunes {
			return errGlobTooLarge
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
		cumRunes += utf8.RuneCountInString(line)
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errGlobTooLarge) {
		return "", walkErr
	}
	if b.Len() == 0 {
		return "未找到匹配项", nil
	}
	if errors.Is(walkErr, errGlobTooLarge) {
		return b.String() + "\n[结果超长，已提前停止列举]", nil
	}
	return b.String(), nil
}

// NewGlob 创建按通配符模式查找文件路径的工具。
func NewGlob(ctx *Context) tool.InvokableTool {
	return &globTool{fsToolBase: fsToolBase{ctx: ctx}}
}

// toSlash 统一路径分隔符为正斜杠，便于模式匹配与跨平台测试。
func toSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// validateGlobPattern 校验 glob 模式：拒绝绝对路径与 .. 逃逸段，确保模式始终
// 相对工作目录解析（配合字面前缀的 resolvePath 边界校验）。
func validateGlobPattern(pattern string) error {
	if strings.HasPrefix(pattern, "/") || (len(pattern) >= 2 && pattern[1] == ':') {
		return errors.New("glob pattern 须为相对 ~/.jot/workspace 的路径，不能是绝对路径")
	}
	for _, seg := range strings.Split(pattern, "/") {
		if seg == ".." {
			return errors.New("glob pattern 不允许包含 .. 逃逸段")
		}
	}
	return nil
}

// globHasMeta 判断路径段是否含通配符（决定其是否为字面段）。
func globHasMeta(seg string) bool {
	return strings.ContainsAny(seg, globMeta)
}

// globLiteralPrefix 提取模式中首个通配符之前、由字面路径段组成的前缀（正斜杠
// 分隔），用作遍历搜索根。模式以通配符开头时返回空串（搜索根=工作目录根）。
func globLiteralPrefix(pattern string) string {
	segs := strings.Split(pattern, "/")
	var lit []string
	for _, seg := range segs {
		if globHasMeta(seg) {
			break
		}
		lit = append(lit, seg)
	}
	return strings.Join(lit, "/")
}

// globMatches 用标准库 path.Match 判断相对路径（正斜杠分隔）是否匹配模式。
// path.Match 使 * 不跨 "/"，即只在单层内匹配：如 scripts/*.md 命中 scripts 下
// 的 .md，而 *.md 不会命中子目录里的文件。匹配失败或模式非法均视为不命中。
func globMatches(pattern, rel string) bool {
	ok, err := path.Match(pattern, rel)
	return err == nil && ok
}
