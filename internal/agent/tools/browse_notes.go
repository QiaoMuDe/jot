package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/models"
	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

// browseNotesTool 笔记库浏览/阅读工具（只读）：list 列出/搜索笔记，view 按 id 读取全文
// （含 offset/length 分段续读、line_numbers 全局行号）。写操作由 manage_note 承担。
type browseNotesTool struct {
	note    *services.NoteService    // 笔记服务（搜索 / 查看全文）
	setting *services.SettingService // 设置服务（读取大文件预览阈值）
	ctx     *Context                 // 日志输出
}

// 编译期断言：确保 browseNotesTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*browseNotesTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 按 action 参数映射动作文案，解析失败回退空串（前端回退"执行"）。
func (m *browseNotesTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	switch args.Action {
	case "list":
		return "列出笔记"
	case "view":
		return "查看笔记全文"
	default:
		return "执行"
	}
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (m *browseNotesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "browse_notes",
		Desc: "浏览/阅读用户笔记库（只读工具）。当用户要求列出/搜索笔记、查看某篇笔记全文、或为行级编辑获取行号时调用。与 recall_notes 的边界：recall_notes 用于按语义相似度召回笔记片段来回答知识类问题；本工具执行结构化读取——按关键字/标签/日期/分页列出笔记，或按 id 读取笔记全文（支持 offset/length 分段续读）。本工具只读，如需创建/更新/编辑/置顶/移动/打标签笔记，请使用 manage_note。通过 action 参数区分动作：list=列出/搜索笔记（可提供 keyword 标题/内容关键字过滤，tag_ids 多标签 AND 过滤，notebook_id 限定笔记本，start_date/end_date 按更新时间过滤，sort_by 排序 updated_at/created_at/title，page 页码与 pageSize 每页条数（缺省 10、上限 50）分页查看，返回的 [数字] id 可供 view 或 manage_note 写操作引用）；view=查看笔记全文（需提供 ids 笔记编号数组，通常传 [id]；内容过长时会截断，可通过 offset/length 参数分段续读——截断结果中会给出下段的 offset，直接再调用 view 并携带该 offset 即可继续读取；如需按行编辑正文，可传 line_numbers=true，输出将带「行 N: 」行号前缀，即 manage_note 的 edit 行级替换寻址坐标，且 offset 续读时行号与之连续，注意行号前缀不属于正文，复制片段用于 find 时不要包含行号）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的动作：list=列出/搜索笔记；view=查看笔记全文",
				Enum:     []string{"list", "view"},
				Required: true,
			},
			"ids": {
				Type:     schema.Array,
				ElemInfo: &schema.ParameterInfo{Type: schema.Number},
				Desc:     "笔记编号数组（正整数列表，来自 list 返回的 [数字]），仅 action=view 时使用，通常传 [id]",
				Required: false,
			},
			"keyword": {
				Type:     schema.String,
				Desc:     "笔记标题/内容关键字过滤，仅 action=list 时使用，缺省不过滤",
				Required: false,
			},
			"tag_ids": {
				Type:     schema.Array,
				ElemInfo: &schema.ParameterInfo{Type: schema.Number},
				Desc:     "标签编号列表，仅 action=list 时使用，多标签 AND 过滤",
				Required: false,
			},
			"notebook_id": {
				Type:     schema.Number,
				Desc:     "笔记本编号，仅 action=list 时使用，限定仅列出该笔记本下的笔记（不传则全库）",
				Required: false,
			},
			"start_date": {
				Type:     schema.String,
				Desc:     "按更新时间过滤的起始日期，格式 YYYY-MM-DD，仅 action=list 时使用，与 end_date 成对使用",
				Required: false,
			},
			"end_date": {
				Type:     schema.String,
				Desc:     "按更新时间过滤的结束日期，格式 YYYY-MM-DD，仅 action=list 时使用，与 start_date 成对使用",
				Required: false,
			},
			"sort_by": {
				Type:     schema.String,
				Desc:     "排序方式，仅 action=list 时使用：updated_at=按更新时间（缺省）；created_at=按创建时间；title=按标题",
				Enum:     []string{"updated_at", "created_at", "title"},
				Required: false,
			},
			"page": {
				Type:     schema.Number,
				Desc:     "页码，从 1 开始，仅 action=list 时使用，缺省 1",
				Required: false,
			},
			"pageSize": {
				Type:     schema.Number,
				Desc:     "每页条数，仅 action=list 时使用，缺省 10，范围 1-50",
				Required: false,
			},
			"line_numbers": {
				Type:     schema.Boolean,
				Desc:     "是否在 view 输出中带「行 N: 」行号前缀（作为 manage_note 行级编辑的寻址坐标），缺省 false（不带行号，便于直接复制原文片段用于 find）",
				Required: false,
			},
			"offset": {
				Type:     schema.Number,
				Desc:     "起始字符位置，仅 action=view 时使用，缺省 0（从开头读取首段）；笔记内容过长被截断后，可传上一段返回的结尾位置分段续读，此时不再按阈值截断而是从该位置读取",
				Required: false,
			},
			"length": {
				Type:     schema.Number,
				Desc:     "本次读取的字符数，仅 action=view 时使用，缺省取 ai_large_file_preview_threshold 设置，上限 100000，超出自动截到内容末尾",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 校验 action → 按动作分发到 list/view。
func (m *browseNotesTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action      string    `json:"action"`
		IDs         []float64 `json:"ids"`
		Keyword     string    `json:"keyword"`
		TagIDs      []float64 `json:"tag_ids"`
		NotebookID  float64   `json:"notebook_id"`
		StartDate   string    `json:"start_date"`
		EndDate     string    `json:"end_date"`
		SortBy      string    `json:"sort_by"`
		Page        float64   `json:"page"`
		PageSize    float64   `json:"pageSize"`
		LineNumbers bool      `json:"line_numbers"`
		Offset      float64   `json:"offset"`
		Length      float64   `json:"length"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 browse_notes 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)
	switch args.Action {
	case "list", "view":
	default:
		return "", fmt.Errorf("browse_notes 参数缺少/非法 action: %s", args.Action)
	}

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if m.ctx != nil && m.ctx.Logger != nil {
		m.ctx.Logger.Debugw("Agent browse_notes 调用",
			fastlog.String("action", args.Action),
			fastlog.String("keyword", args.Keyword),
			fastlog.Int("page", int(args.Page)),
			fastlog.Int("pageSize", int(args.PageSize)))
	}

	switch args.Action {
	case "list":
		return m.listNotes(args.Keyword, int(args.Page), int(args.PageSize), int(args.NotebookID), args.SortBy, args.StartDate, args.EndDate, args.TagIDs)
	case "view":
		return m.viewNote(args.IDs, args.LineNumbers, int(args.Offset), int(args.Length))
	}
	return "", fmt.Errorf("browse_notes 未知 action: %s", args.Action)
}

// listNotes 列出/搜索笔记：keyword 标题/内容关键字过滤（trim 后非空才过滤），sort_by 排序，
// page/pageSize 分页（pageSize 缺省 10、上限 50）；tag_ids 多标签 AND 过滤（空值不过滤）；
// notebookID > 0 时限定在指定笔记本范围内搜索（NoteService.SearchByNotebook），否则全库搜索（NoteService.Search）。
// 过滤与分页都在 DB 层完成，只加载当前页条目；
// Search 返回的 note.Content 是前 200 字符预览，直接格式化，不再额外截断。
func (m *browseNotesTool) listNotes(keyword string, page, pageSize int, notebookID int, sortBy, startDate, endDate string, tagIDs []float64) (string, error) {
	keyword = strings.TrimSpace(keyword)
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" {
		sortBy = "updated_at"
	}
	switch sortBy {
	case "updated_at", "created_at", "title":
	default:
		return "", fmt.Errorf("browse_notes 参数非法 sort_by: %s", sortBy)
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	var ids []uint
	for _, t := range tagIDs {
		if t > 0 {
			ids = append(ids, uint(t))
		}
	}

	var notes []models.Note
	var total int64
	var err error
	if notebookID > 0 {
		notes, total, err = m.note.SearchByNotebook(keyword, page, pageSize, uint(notebookID), sortBy, startDate, endDate, ids)
	} else {
		notes, total, err = m.note.Search(keyword, page, pageSize, sortBy, startDate, endDate, ids)
	}
	if err != nil {
		return "", err
	}
	if total == 0 {
		if keyword != "" {
			return fmt.Sprintf("没有找到包含「%s」的笔记", keyword), nil
		}
		return "没有找到匹配的笔记", nil
	}

	totalPages := (int(total) + pageSize - 1) / pageSize
	if page > totalPages {
		return fmt.Sprintf("笔记共 %d 条，共 %d 页，请求的第 %d 页超出范围，请从第 1 页开始查看", total, totalPages, page), nil
	}

	var b strings.Builder
	if keyword != "" {
		fmt.Fprintf(&b, "找到包含「%s」的笔记 %d 条，第 %d/%d 页，本页 %d 条：\n", keyword, total, page, totalPages, len(notes))
	} else {
		fmt.Fprintf(&b, "笔记共 %d 条，第 %d/%d 页，本页 %d 条：\n", total, page, totalPages, len(notes))
	}
	for i := range notes {
		n := notes[i]
		pin := ""
		if n.Pinned {
			pin = "📌"
		}
		tagNames := make([]string, 0, len(n.Tags))
		for _, t := range n.Tags {
			tagNames = append(tagNames, t.Name)
		}
		parts := []string{fmt.Sprintf("[%d] %s%s", n.ID, pin, n.Title)}
		if len(tagNames) > 0 {
			parts = append(parts, "标签 "+strings.Join(tagNames, " / "))
		}
		if n.Content != "" {
			parts = append(parts, "预览 "+n.Content)
		}
		parts = append(parts,
			"创建 "+n.CreatedAt.Format("2006-01-02 15:04"),
			"更新 "+n.UpdatedAt.Format("2006-01-02 15:04"))
		fmt.Fprintf(&b, "%s\n", strings.Join(parts, " · "))
	}
	if rest := total - int64(page*pageSize); rest > 0 {
		fmt.Fprintf(&b, "还有 %d 条未展示，如需继续查看可要求查看第 %d 页。", rest, page+1)
	}
	return b.String(), nil
}

// viewNote 查看笔记全文：ids 必填，通常只传一个元素 [id]。支持按 offset/length 分段续读：
// offset 缺省 0（从开头读取首段），offset=0 时若内容超过 notePreviewThreshold（缺省 10000），
// 仅显示前段并返回总字符数与续读指引（提示继续调用 view、携带 offset=当前结尾）；
// offset>0 时从该位置读取后续分段，length 缺省取 notePreviewThreshold、上限 maxSectionLen，
// 超出自动截到内容末尾。lineNumbers 为 true 时输出带「行 N: 」行号前缀（1-based，起始行号为
// offset 前换行数 + 1，保证分段续读行号与首段全局连续），作为 manage_note edit 行级替换的
// 寻址坐标；行号前缀不属于正文，复制片段用于 find 时须去掉行号。
func (m *browseNotesTool) viewNote(ids []float64, lineNumbers bool, offset, length int) (string, error) {
	noteIDs := resolveNoteIDs(ids)
	if len(noteIDs) == 0 {
		return "", errors.New("browse_notes 查看笔记缺少有效的 ids")
	}
	if len(noteIDs) > 1 {
		return "", errors.New("browse_notes 查看笔记只支持单条操作，请在 ids 中传入一个笔记编号")
	}
	id := noteIDs[0]
	content, err := m.note.GetNoteContent(id)
	if err != nil {
		return "", err
	}

	runes := []rune(content)
	total := len(runes)
	totalLines := len(splitNoteLines(content))
	if offset < 0 {
		return "", errors.New("browse_notes 查看笔记的 offset 须为 >=0 的整数")
	}
	// 越界校验：常规笔记 offset>=total 已全部读完；空笔记（total==0）仅放行 offset==0 的
	// 空内容查看，其余 offset 一律视作超出，避免对空切片按 offset 取址触发 panic。
	if offset >= total && (total > 0 || offset > 0) {
		return "", fmt.Errorf("browse_notes 查看笔记的 offset 超出内容范围（共 %d 字符，已全部读取完毕）", total)
	}
	// 单段长度：模型未指定时取预览阈值设置；指定时校验上限
	if length <= 0 {
		length = notePreviewThreshold(m.setting)
	}
	if length > maxSectionLen {
		length = maxSectionLen
	}
	end := offset + length
	if end > total {
		end = total
	}

	section := string(runes[offset:end])
	truncated := end < total
	display := section
	if lineNumbers {
		// 全局起始行号：offset 前换行数 + 1，保证分段续读行号连续
		startLine := 1
		for i := 0; i < offset; i++ {
			if runes[i] == '\n' {
				startLine++
			}
		}
		display = numberLines(section, startLine)
	}
	if truncated {
		// 本段实际展示的行数（按可读行拆分），用于续读提示中的行数信息
		sectionLines := len(splitNoteLines(section))
		display += fmt.Sprintf("\n\n（内容共 %d 字符 / %d 行，已显示前 %d 字符 / %d 行。如需继续阅读，可继续调用本工具的 view，offset=%d（length 保持缺省即可）；如需按行编辑，可让 view 带 line_numbers=true 获取全局行号）",
			total, totalLines, end, sectionLines, end)
	}
	return fmt.Sprintf("笔记 #%d 内容：\n%s", id, display), nil
}

// NewBrowseNotes 构造读取笔记库工具（list/view，只读）。
func NewBrowseNotes(note *services.NoteService, setting *services.SettingService, ctx *Context) tool.InvokableTool {
	return &browseNotesTool{note: note, setting: setting, ctx: ctx}
}
