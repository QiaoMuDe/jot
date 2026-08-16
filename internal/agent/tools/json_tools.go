package tools

// 本文件实现 JSON 数据处理工具三件套（json_validate / json_format / json_extract），
// 移植自 eino-tools 通用工具库（tmp/eino-tools/json），供 Agent 在 ReAct 循环中
// 自助校验/美化/按路径提取 JSON：模型经常产生畸形 JSON，或需要从大 JSON 载荷中
// 取单个字段——这些工具让模型自检与修复，避免把整段数据塞进上下文。
//
// 实现说明：采用 eino components/tool/utils 的 InferTool（结构体反射生成参数 Schema，
// 与 eino-tools 原版一致，项目其它工具为手写 ParamsOneOf，二者均为合法实现）；
// 纯标准库实现，无凭证、无状态、无外部依赖。
//
// 工具：
//
//	json_validate(text)              — 校验合法性，报告错误位置（第几行第几列）
//	json_format(text, indent?)       — 美化输出合法 JSON
//	json_extract(text, path)         — 按点路径提取，如 data.items[0].title

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	stdjson "encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/tidwall/gjson"
)

const (
	maxJSONLen = 100000 // 输入上限（字节），防撑爆上下文
	maxIndent  = 8      // indent 字符串最大长度
)

// NewJSONTools 返回 JSON 数据处理工具三件套（json_validate / json_format / json_extract）。
func NewJSONTools() []tool.BaseTool {
	return []tool.BaseTool{
		mustTool(NewJSONValidateTool()),
		mustTool(NewJSONFormatTool()),
		mustTool(NewJSONExtractTool()),
	}
}

func mustTool(t tool.InvokableTool, err error) tool.BaseTool {
	if err != nil {
		panic(err) // unreachable: InferTool with static args cannot fail
	}
	return t
}

// MustJSONValidate / MustJSONFormat / MustJSONExtract 便捷构造器：
// InferTool 静态参数不会失败，失败即 panic（与 NewJSONTools 的 must 一致），
// 供 registry 单行注册使用；需自行处理错误的调用方可用 New*Tool 版本。
func MustJSONValidate() tool.InvokableTool {
	t, err := NewJSONValidateTool()
	if err != nil {
		panic(err)
	}
	return t
}

func MustJSONFormat() tool.InvokableTool {
	t, err := NewJSONFormatTool()
	if err != nil {
		panic(err)
	}
	return t
}

func MustJSONExtract() tool.InvokableTool {
	t, err := NewJSONExtractTool()
	if err != nil {
		panic(err)
	}
	return t
}

// ---------------------------------------------------------------------------
// json_validate

type validateArgs struct {
	Text string `json:"text" jsonschema_description:"要校验的 JSON 文本"`
}

// NewJSONValidateTool 构建 json_validate 工具。
func NewJSONValidateTool() (tool.InvokableTool, error) {
	return utils.InferTool[*validateArgs, string](
		"json_validate",
		"校验一段文本是否为合法 JSON，返回校验结果。合法时说明顶层类型；不合法时给出错误信息与出错位置（第几行第几列）。生成或处理 JSON 前建议先校验。",
		func(ctx context.Context, a *validateArgs) (string, error) {
			return validateRun(ctx, a)
		},
	)
}

func validateRun(ctx context.Context, a *validateArgs) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	text := strings.TrimSpace(a.Text)
	if text == "" {
		return "json_validate: text 不能为空。", nil
	}
	if len(text) > maxJSONLen {
		return fmt.Sprintf("json_validate: text 过长（%d 字节，上限 %d）", len(text), maxJSONLen), nil
	}
	var v any
	if err := stdjson.Unmarshal([]byte(text), &v); err != nil {
		var syn *stdjson.SyntaxError
		if errors.As(err, &syn) {
			line, col := offsetToLineCol(text, syn.Offset)
			return fmt.Sprintf("JSON 不合法：%s（第 %d 行第 %d 列附近）", syn.Error(), line, col), nil
		}
		return "JSON 不合法：" + err.Error(), nil
	}
	return fmt.Sprintf("JSON 合法（顶层类型：%s）。", typeName(v)), nil
}

func typeName(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	}
	return "unknown"
}

// offsetToLineCol 把字节偏移转换为 1 起始的行/列号。
func offsetToLineCol(s string, offset int64) (line, col int) {
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(s)) {
		offset = int64(len(s))
	}
	line = 1
	col = 1
	for i := int64(0); i < offset; i++ {
		if s[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

// ---------------------------------------------------------------------------
// json_format

type formatArgs struct {
	Text string `json:"text" jsonschema_description:"要格式化的 JSON 文本"`

	Indent string `json:"indent" jsonschema_description:"可选的缩进字符串（仅空格或 tab，如两个空格或 \\t）；缺省为两个空格"`
}

// NewJSONFormatTool 构建 json_format 工具。
func NewJSONFormatTool() (tool.InvokableTool, error) {
	return utils.InferTool[*formatArgs, string](
		"json_format",
		"把合法的 JSON 文本格式化为易读的缩进形式（美化输出）；非法 JSON 返回错误提示。生成较长的 JSON 后调用，便于检查结构。",
		func(ctx context.Context, a *formatArgs) (string, error) {
			return formatRun(ctx, a)
		},
	)
}

func formatRun(ctx context.Context, a *formatArgs) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	text := strings.TrimSpace(a.Text)
	if text == "" {
		return "json_format: text 不能为空。", nil
	}
	if len(text) > maxJSONLen {
		return fmt.Sprintf("json_format: text 过长（%d 字节，上限 %d）", len(text), maxJSONLen), nil
	}
	var v any
	if err := stdjson.Unmarshal([]byte(text), &v); err != nil {
		return "json_format: JSON 不合法，请先用 json_validate 检查：" + err.Error(), nil
	}
	indent := a.Indent
	if indent == "" {
		indent = "  "
	}
	if !validIndent(indent) {
		return "json_format: indent 只能由空格或 tab 组成（长度 ≤ 8）。", nil
	}
	out, err := stdjson.MarshalIndent(v, "", indent)
	if err != nil {
		return "json_format: 格式化失败: " + err.Error(), nil
	}
	return string(out), nil
}

func validIndent(s string) bool {
	if len(s) == 0 || len(s) > maxIndent {
		return false
	}
	for _, c := range s {
		if c != ' ' && c != '\t' {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// json_extract

type extractArgs struct {
	Text string `json:"text" jsonschema_description:"要提取的 JSON 文本"`

	Path string `json:"path" jsonschema_description:"提取路径，点语法：data.items[0].title（数组下标支持 [0] 或 .0 两种写法）；可用 $ 开头（$.data.items[0].title）；顶层用 $；支持 # 通配符（如 data.items.#.title 返回全部 title）"`
}

// NewJSONExtractTool 构建 json_extract 工具。
func NewJSONExtractTool() (tool.InvokableTool, error) {
	return utils.InferTool[*extractArgs, string](
		"json_extract",
		"从 JSON 中按路径提取一个值。路径用点语法：对象字段直接写字段名，数组用下标（[0] 或 .0 均可）；例如 data.items[0].title 提取 data 下 items 数组首项的 title；用 # 通配可一次取回所有元素（如 data.items.#.title 返回全部 title 的数组）。标量值（字符串/数字/布尔）直接返回，对象/数组返回原始 JSON。路径不存在返回提示。从大 JSON 中取字段时调用，避免把整段塞进上下文。",
		func(ctx context.Context, a *extractArgs) (string, error) {
			return extractRun(ctx, a)
		},
	)
}

// bracketIndexRe 匹配 [数字] 数组下标写法（JSONPath 习惯），归一化时转为 .数字。
var bracketIndexRe = regexp.MustCompile(`\[\d+\]`)

// normalizeGJSONPath 把模型习惯的 JSONPath 写法归一化为 gjson 原生路径：
//   - 去掉 $ 前缀（$.name → name；$ → 空）
//   - [n] 数组下标 → .n（data.items[0].title → data.items.0.title；连续下标 name[0][1] → name.0.1）
//   - 去前导点（开头 [0] 转换后 → 0.…）
//
// gjson 原生能力（# 通配符等）保持原样透传。
func normalizeGJSONPath(path string) string {
	p := strings.TrimSpace(path)
	p = strings.TrimPrefix(p, "$")
	p = bracketIndexRe.ReplaceAllStringFunc(p, func(m string) string {
		return "." + m[1:len(m)-1]
	})
	return strings.TrimPrefix(p, ".")
}

func extractRun(ctx context.Context, a *extractArgs) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	text := strings.TrimSpace(a.Text)
	if text == "" {
		return "json_extract: text 不能为空。", nil
	}
	path := strings.TrimSpace(a.Path)
	if path == "" {
		return "json_extract: path 不能为空（提取顶层可传 \"$\"）。", nil
	}
	if len(text) > maxJSONLen {
		return fmt.Sprintf("json_extract: text 过长（%d 字节，上限 %d）", len(text), maxJSONLen), nil
	}
	if !stdjson.Valid([]byte(text)) {
		return "json_extract: JSON 不合法，请先用 json_validate 检查。", nil
	}

	norm := normalizeGJSONPath(path)
	// 顶层路径（$）：返回整段 JSON 原文
	if norm == "" {
		return text, nil
	}
	res := gjson.Get(text, norm)
	if !res.Exists() {
		return fmt.Sprintf("json_extract: 路径 %q 不存在。", path), nil
	}
	switch {
	case res.Type == gjson.Null:
		return "null", nil
	case res.IsObject() || res.IsArray():
		// 对象/数组返回原始 JSON（保留源文本键序）
		return res.Raw, nil
	default:
		return res.String(), nil
	}
}
