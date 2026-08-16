package tools

// 本文件实现 read_note_section 笔记分段读取工具：manage_note 的 view 在笔记内容超过
// ai_large_file_preview_threshold 设置时截断并给出续读指引，模型据此携带 id/offset
// 调用本工具按字符偏移（rune 索引）读取笔记后续分段。底层复用 services.NoteService
// 的 GetNoteContent 读取全文后切片，不感知父包 agent 的事件循环细节。
// 参数：
//   - id：笔记编号，必填、正整数（与 manage_note view 的 id 一致）；
//   - offset：起始字符位置，必填、≥0 且小于内容总字符数；
//   - length：读取字符数，可选；缺省取 ai_large_file_preview_threshold（缺省 10000），
//     上限 100000，超出部分自动截到内容末尾；
//   - line_numbers：可选布尔，true 时输出带「行 N: 」行号前缀（1-based 全局行号，
//     与 view 的 line_numbers=true 行号连续，作为 edit 行级替换的寻址坐标）；
//     行号前缀不属于正文，复制片段用于 find 时须去掉行号。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

// maxSectionLen read_note_section 单次读取的最大字符数上限，防止模型一次请求超长内容撑爆上下文窗口。
const maxSectionLen = 100000

// readNoteSectionTool 笔记分段读取工具。
type readNoteSectionTool struct {
	note    *services.NoteService    // 笔记服务（读取全文）
	setting *services.SettingService // 读取单段缺省长度设置（ai_large_file_preview_threshold）
	ctx     *Context                 // 日志输出
}

// 编译期断言：确保 readNoteSectionTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*readNoteSectionTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 展示笔记编号与起始位置，解析失败回退通用文案。
func (r *readNoteSectionTool) ActionText(argumentsInJSON string) string {
	var args struct {
		ID     float64 `json:"id"`
		Offset float64 `json:"offset"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "读取笔记分段"
	}
	return fmt.Sprintf("读取笔记 #%d 第 %d 字符起", int(args.ID), int(args.Offset))
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (r *readNoteSectionTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_note_section",
		Desc: "按字符位置读取笔记的后续分段。当 manage_note 的 view 返回内容被截断并提示调用本工具时，用其中的 id 与 offset 参数读取后续内容；也可在需要精确读取笔记某段内容时直接调用。id 为笔记编号（与 view 一致），offset 为起始字符位置（上一段返回的结尾位置即为下一段 offset），length 可选（缺省与 view 相同的单段长度）。如需按行编辑正文，可传 line_numbers=true，输出将带「行 N: 」行号前缀（1-based 全局行号，与 view 的 line_numbers=true 连续），行号即 edit 行级替换的寻址坐标；注意行号前缀不属于正文，复制片段用于 find 时不要包含行号。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"id": {
				Type:     schema.Number,
				Desc:     "笔记编号（正整数，manage_note 列表中的 [数字] 即为 id）",
				Required: true,
			},
			"offset": {
				Type:     schema.Number,
				Desc:     "起始字符位置，从 0 开始（上一段内容结尾位置即为下一段的 offset），必须小于笔记内容总字符数",
				Required: true,
			},
			"length": {
				Type:     schema.Number,
				Desc:     "本次读取的字符数，可选；缺省取 ai_large_file_preview_threshold 设置，上限 100000",
				Required: false,
			},
			"line_numbers": {
				Type:     schema.Boolean,
				Desc:     "是否带「行 N: 」行号前缀（1-based 全局行号，与 view 的 line_numbers=true 连续），作为行级编辑的寻址坐标；缺省 false（不带行号，便于直接复制原文片段用于 find）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行分段读取：校验 id/offset → 读取全文 → 按 rune 切片 → 返回含
// 起止位置与总字符数的结果（模型据此计算下一段 offset）。line_numbers=true 时输出
// 带「行 N: 」行号前缀（1-based 全局行号，由 offset 前已有的换行数推算，保证与
// view 的 line_numbers=true 行号连续），作为 edit 行级替换的寻址坐标。
// 错误路径（参数非法 / offset 越界 / 笔记不存在）返回 error 经 WrapWithError 回填模型继续推理。
func (r *readNoteSectionTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		ID          float64 `json:"id"`
		Offset      float64 `json:"offset"`
		Length      float64 `json:"length"`
		LineNumbers bool    `json:"line_numbers"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 read_note_section 参数失败: %w", err)
	}

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if args.ID <= 0 {
		return "", errors.New("read_note_section 缺少有效的 id")
	}
	if args.Offset < 0 {
		return "", errors.New("read_note_section 缺少有效的 offset（须为 >=0 的整数）")
	}
	if args.Offset != math.Trunc(args.Offset) {
		return "", errors.New("read_note_section 的 offset 须为整数")
	}
	if args.Length != 0 && args.Length != math.Trunc(args.Length) {
		return "", errors.New("read_note_section 的 length 须为整数")
	}

	content, err := r.note.GetNoteContent(uint(args.ID))
	if err != nil {
		return "", err
	}
	runes := []rune(content)
	total := len(runes)
	offset := int(args.Offset)
	if offset >= total {
		return "", fmt.Errorf("read_note_section 的 offset 超出内容范围（共 %d 字符，已全部读取完毕）", total)
	}

	// 单段长度：模型未指定时取预览阈值设置；指定时校验上限
	length := int(args.Length)
	if length <= 0 {
		length = notePreviewThreshold(r.setting)
	}
	if length > maxSectionLen {
		length = maxSectionLen
	}
	end := offset + length
	if end > total {
		end = total
	}

	section := string(runes[offset:end])
	// 行号模式：先算全局起始行号（offset 前换行数 + 1），再逐行加前缀
	lineInfo := ""
	display := section
	if args.LineNumbers {
		startLine := 1
		for i := 0; i < offset; i++ {
			if runes[i] == '\n' {
				startLine++
			}
		}
		display = numberLines(section, startLine)
		sectionLines := splitNoteLines(section)
		lineInfo = fmt.Sprintf("（第 %d-%d 行）", startLine, startLine+len(sectionLines)-1)
	}
	if r.ctx != nil && r.ctx.Logger != nil {
		r.ctx.Logger.Debugw("Agent read_note_section 调用",
			fastlog.Int("id", int(args.ID)),
			fastlog.Int("offset", offset),
			fastlog.Int("end", end),
			fastlog.Int("total", total),
			fastlog.Bool("line_numbers", args.LineNumbers))
	}
	return fmt.Sprintf("笔记 #%d 第 %d-%d 字符的内容（共 %d 字符）%s：\n%s", uint(args.ID), offset+1, end, total, lineInfo, display), nil
}

// NewReadNoteSection 创建笔记分段读取工具。
func NewReadNoteSection(note *services.NoteService, setting *services.SettingService, ctx *Context) tool.InvokableTool {
	return &readNoteSectionTool{note: note, setting: setting, ctx: ctx}
}
