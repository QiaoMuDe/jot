package tools

// 本文件实现 manage_note 笔记库管理工具：模型在 ReAct 循环中调用它创建笔记、列出/搜索笔记、
// 查看笔记全文、更新标题/扩展名、编辑正文、置顶/取消置顶、移动笔记本、给笔记打标签或移除标签，
// 底层复用 services.NoteService（Create / CreateWithNotebook / Search / GetNoteContent / Update /
// TogglePin / MoveToNotebook）与 services.TagService（AddTagToNote / RemoveTagFromNote），
// 不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分九个动作：
//   - create：创建笔记（title / content 必填；file_ext 可选、缺省 .md；notebook_id 可选
//     指定创建到哪个笔记本，未指定时归入默认笔记本 id=1；tag_ids 可选给新笔记打标签）；
//   - list：列出/搜索笔记（keyword 按标题/内容模糊过滤；tag_ids 多标签 AND 过滤；
//     start_date / end_date 按 updated_at 日期范围过滤；sort_by 排序，缺省 updated_at；
//     page/pageSize 分页，pageSize 缺省 10、上限 50），
//     返回"共 n 条、第 x/y 页"，列表只展示当前页条目；当页未展示完时提示可翻页；
//   - view：查看笔记全文（id 必填、正整数，来自列表中的 [数字] 编号；内容超过
//     ai_large_file_preview_threshold 设置（解析失败或<=0 时缺省 10000）时截断，
//     并给出 read_note_section 工具的续读指引（id/offset 参数）；line_numbers=true
//     时输出带「行 N: 」行号前缀，作为行级编辑（edit 的 line_start/line_end）的寻址坐标）；
//   - update：更新笔记标题/扩展名（id 必填、正整数；title / file_ext 至少提供一个，
//     非空才更新对应字段，不碰正文）；
//   - edit：编辑笔记正文（id 必填、正整数；双模式互斥——find 非空为片段替换，
//     把第 count 次（缺省 1）出现的 find 片段替换为 replace（缺省空字符串即删除
//     该片段），find 优先精确匹配、因空白/换行差异未命中时自动按空白归一化匹配兜底，
//     replace_all=true 时替换全部出现（与 count 互斥）；line_start 非 0 为行级
//     替换模式，把第 line_start 行到第 line_end 行（缺省等于 line_start）的区间
//     替换为 replace，replace 为空字符串即删除该区间行，行号来自 view/read_note_section
//     的 line_numbers=true 输出；line_start 大于笔记总行数时为末尾追加语义）；
//   - pin：置顶/取消置顶笔记（id 必填、正整数，切换置顶状态）；
//   - move：移动笔记到目标笔记本（id 必填、正整数，notebook_id 必填目标笔记本）；
//   - add_tag / remove_tag：给笔记添加/移除标签（id 必填、正整数，tag_id 必填、正整数）。
// 与 recall_notes 的边界：recall_notes 用于语义召回笔记片段回答知识类问题，
// manage_note 用于结构化操作笔记库。本工具不包含删除类/批量动作（spec 明确不暴露）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/models"
	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

// noteFileExtPattern 合法笔记扩展名格式：纯字母数字 1-10 位（不含点，如 md/txt/markdown）。
var noteFileExtPattern = regexp.MustCompile(`^[a-zA-Z0-9]{1,10}$`)

// normalizeNoteFileExt 规范化笔记扩展名：trim → 去前导点 → 校验纯字母数字 1-10 位 →
// 统一补前导点；raw 为空返回空串（由调用方决定缺省或保持原值）。
func normalizeNoteFileExt(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	s = strings.TrimPrefix(s, ".")
	if !noteFileExtPattern.MatchString(s) {
		return "", fmt.Errorf("manage_note 参数非法 file_ext: %s（应为 1-10 位字母或数字，如 md/txt/markdown）", raw)
	}
	return "." + s, nil
}

// isManageNoteWriteAction 判断 action 是否为写操作（须用户确认后才执行）。
// create 视为用户明确要求的创建指令不强制确认；读操作 list/view 无需确认。
func isManageNoteWriteAction(action string) bool {
	switch action {
	case "update", "edit", "pin", "move", "add_tag", "remove_tag":
		return true
	default:
		return false
	}
}

// manageNoteActionCN 返回 action 的中文文案（供强制确认引导提示使用），未知 action 回退"操作"。
func manageNoteActionCN(action string) string {
	switch action {
	case "update":
		return "更新笔记标题/扩展名"
	case "edit":
		return "编辑笔记正文"
	case "pin":
		return "置顶或取消置顶"
	case "move":
		return "移动笔记"
	case "add_tag":
		return "添加标签"
	case "remove_tag":
		return "移除标签"
	default:
		return "操作"
	}
}

// manageNoteTool 笔记库管理工具。
type manageNoteTool struct {
	note    *services.NoteService    // 笔记服务（创建 / 搜索 / 查看 / 置顶 / 移动）
	tag     *services.TagService     // 标签服务（打标签 / 移除标签）
	setting *services.SettingService // 设置服务（读取大文件预览阈值）
	ctx     *Context                 // 日志输出
}

// 编译期断言：确保 manageNoteTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*manageNoteTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 按 action 参数映射动作文案，解析失败回退空串（前端回退"执行"）。
func (m *manageNoteTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	switch args.Action {
	case "create":
		return "创建笔记"
	case "list":
		return "列出笔记"
	case "view":
		return "查看笔记全文"
	case "update":
		return "更新笔记标题/扩展名"
	case "edit":
		return "编辑笔记正文"
	case "pin":
		return "置顶或取消置顶笔记"
	case "move":
		return "移动笔记"
	case "add_tag":
		return "添加标签"
	case "remove_tag":
		return "移除标签"
	default:
		return "执行"
	}
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (m *manageNoteTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_note",
		Desc: "管理用户笔记库。当用户要求创建笔记、列出/搜索笔记、查看笔记全文、更新笔记标题或扩展名、编辑笔记正文、置顶/取消置顶、移动笔记本、给笔记打标签或移除标签时调用。与 recall_notes 的边界：recall_notes 用于语义召回笔记片段回答知识类问题，manage_note 用于结构化操作笔记库。通过 action 参数区分动作：create=创建笔记（需提供 title 标题与 content 内容，可提供 file_ext 文件后缀（缺省 .md）、notebook_id 目标笔记本（未指定时归入默认笔记本）、tag_ids 标签编号列表）；list=列出/搜索笔记（可用 keyword 标题/内容关键字过滤，tag_ids 多标签 AND 过滤，start_date/end_date 按更新时间范围过滤，sort_by 排序（updated_at/created_at/title，缺省 updated_at），page 页码与 pageSize 每页条数（缺省 10、上限 50）分页查看）；view=查看笔记全文（需提供 id 笔记编号；内容过长时会截断并可要求分段查看；如需按行编辑正文，请传 line_numbers=true，输出将带「行 N: 」行号前缀，行号即 edit 行级替换的寻址坐标，注意行号前缀不属于正文，复制片段用于 find 时不要包含行号）；update=更新笔记标题/扩展名（需提供 id 笔记编号与 title 新标题、file_ext 新扩展名至少其一，只改元数据不碰正文）；edit=编辑笔记正文（需提供 id 笔记编号；两种方式互斥：①片段替换提供 find 要替换的原文片段与 replace 新文本，find 优先精确匹配，若因空白/换行/缩进差异未命中会自动做空白归一化匹配兜底（标点、文字仍须一致），删除片段时 replace 传空字符串，count 可指定第几次出现（缺省 1），replace_all=true 时替换全部出现（与 count 互斥，二者不可同时使用）；②行级替换提供 line_start 起始行号（必填）与 line_end 结束行号（缺省等于 line_start），将该区间整行替换为 replace（空字符串即删除这些行），行号必须来自 view/read_note_section 的 line_numbers=true 输出；line_start 大于笔记总行数时为末尾追加语义，replace 即为追加内容；只需修改几个字或一句话用片段替换，需要修改连续多行、整段重写、或无法用简短片段定位时用行级替换）；pin=置顶/取消置顶笔记（需提供 id 笔记编号）；move=移动笔记到目标笔记本（需提供 id 笔记编号与 notebook_id 目标笔记本）；add_tag=给笔记添加标签（需提供 id 笔记编号与 tag_id 标签编号）；remove_tag=从笔记移除标签（需提供 id 笔记编号与 tag_id 标签编号）。强制确认：update / edit / pin / move / add_tag / remove_tag 均属写操作，执行前必须先向用户确认修改意图——在回复正文中说明要执行的具体操作与影响，并调用 ask_user 工具向用户提问，用户明确同意后再携带 confirm=true 调用本工具；未携带 confirm=true 时工具会拒绝执行并提示先确认（create 为用户明确要求的创建指令，无需确认）。返回笔记列表或操作结果，列表中的编号 [数字] 可用于后续 view/update/edit/pin/move/add_tag/remove_tag。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的动作：create=创建笔记；list=列出/搜索笔记；view=查看笔记全文；update=更新标题/扩展名；edit=编辑正文；pin=置顶/取消置顶；move=移动笔记到目标笔记本；add_tag=给笔记添加标签；remove_tag=从笔记移除标签",
				Enum:     []string{"create", "list", "view", "update", "edit", "pin", "move", "add_tag", "remove_tag"},
				Required: true,
			},
			"title": {
				Type:     schema.String,
				Desc:     "笔记标题，action=create 时必填；action=update 时可选（不传则不修改标题）",
				Required: false,
			},
			"content": {
				Type:     schema.String,
				Desc:     "笔记内容，action=create 时必填",
				Required: false,
			},
			"file_ext": {
				Type:     schema.String,
				Desc:     "笔记文件后缀，action=create 时缺省 .md；action=update 时可选（不传则保持原扩展名）",
				Required: false,
			},
			"find": {
				Type:     schema.String,
				Desc:     "片段替换的原文片段，仅 action=edit 片段替换时使用（与 line_start 互斥）；优先精确匹配，若因空白/换行/缩进差异未命中会自动做空白归一化匹配兜底（标点、文字仍须一致）",
				Required: false,
			},
			"replace": {
				Type:     schema.String,
				Desc:     "替换后的新文本，action=edit 片段替换或行级替换时使用，缺省空字符串（片段替换即删除该片段；行级替换即删除该区间行）",
				Required: false,
			},
			"count": {
				Type:     schema.Number,
				Desc:     "find 片段在正文中第几次出现，仅 action=edit 片段替换时使用，缺省 1；与 replace_all 互斥，不可同时使用",
				Required: false,
			},
			"replace_all": {
				Type:     schema.Boolean,
				Desc:     "是否替换 find 片段的全部出现，仅 action=edit 片段替换时使用，缺省 false（只替换第 count 次出现）；与 count 互斥，不可同时使用",
				Required: false,
			},
			"line_start": {
				Type:     schema.Number,
				Desc:     "行级替换的起始行号（从 1 开始，行号须来自 view/read_note_section 的 line_numbers=true 输出），仅 action=edit 行级替换时使用（与 find 互斥）；大于笔记总行数时为末尾追加语义",
				Required: false,
			},
			"line_end": {
				Type:     schema.Number,
				Desc:     "行级替换的结束行号（从 1 开始，含该行），仅 action=edit 行级替换时使用，缺省等于 line_start；超出笔记总行数时报错",
				Required: false,
			},
			"line_numbers": {
				Type:     schema.Boolean,
				Desc:     "是否在 view/read_note_section 输出中带「行 N: 」行号前缀（作为行级编辑的寻址坐标），缺省 false（不带行号，便于直接复制原文片段用于 find）",
				Required: false,
			},
			"notebook_id": {
				Type:     schema.Number,
				Desc:     "笔记本编号，action=create 时可选（指定创建到该笔记本，未指定时归入默认笔记本 id=1）；action=move 时必填（目标笔记本）；action=list 时可选（仅列出该笔记本下的笔记）",
				Required: false,
			},
			"tag_ids": {
				Type:     schema.Array,
				ElemInfo: &schema.ParameterInfo{Type: schema.Number},
				Desc:     "标签编号列表，action=create 时可选给新笔记打标签；action=list 时可选多标签 AND 过滤",
				Required: false,
			},
			"keyword": {
				Type:     schema.String,
				Desc:     "笔记标题/内容关键字过滤，仅 action=list 时使用，缺省不过滤",
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
			"id": {
				Type:     schema.Number,
				Desc:     "笔记编号（正整数，列表中的 [数字] 即为 id），action=view / update / edit / pin / move / add_tag / remove_tag 时必填",
				Required: false,
			},
			"tag_id": {
				Type:     schema.Number,
				Desc:     "标签编号（正整数，manage_tag 列表中的 [数字] 即为 id），action=add_tag / remove_tag 时必填",
				Required: false,
			},
			"confirm": {
				Type:     schema.Boolean,
				Desc:     "用户确认标记：update / edit / pin / move / add_tag / remove_tag 等写操作执行前必须先向用户确认修改意图，用户明确同意后传 true 才执行；缺省 false 时工具会拒绝执行并引导先确认",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 校验 action → 按动作分发到 Create / Search / View / Update / Edit / Pin / Move / Tag 操作。
func (m *manageNoteTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action      string    `json:"action"`
		Title       string    `json:"title"`
		Content     string    `json:"content"`
		FileExt     string    `json:"file_ext"`
		Find        string    `json:"find"`
		Replace     string    `json:"replace"`
		Count       float64   `json:"count"`
		ReplaceAll  bool      `json:"replace_all"`
		LineStart   float64   `json:"line_start"`
		LineEnd     float64   `json:"line_end"`
		LineNumbers bool      `json:"line_numbers"`
		NotebookID  float64   `json:"notebook_id"`
		TagIDs      []float64 `json:"tag_ids"`
		Keyword     string    `json:"keyword"`
		StartDate   string    `json:"start_date"`
		EndDate     string    `json:"end_date"`
		SortBy      string    `json:"sort_by"`
		Page        float64   `json:"page"`
		PageSize    float64   `json:"pageSize"`
		ID          float64   `json:"id"`
		TagID       float64   `json:"tag_id"`
		Confirm     bool      `json:"confirm"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 manage_note 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)
	switch args.Action {
	case "create", "list", "view", "update", "edit", "pin", "move", "add_tag", "remove_tag":
	default:
		return "", fmt.Errorf("manage_note 参数缺少/非法 action: %s", args.Action)
	}

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 写操作强制确认：update / edit / pin / move / add_tag / remove_tag 均须携带
	// confirm=true（用户已明确同意）才执行；缺省 false 时拒绝执行并引导先向用户确认。
	// 返回正常结果（nil error）而非 error，避免被包装器记为 tool_error 失败态。
	if !args.Confirm && isManageNoteWriteAction(args.Action) {
		if m.ctx != nil && m.ctx.Logger != nil {
			m.ctx.Logger.Debugw("Agent manage_note 写操作待确认",
				fastlog.String("action", args.Action))
		}
		return "该操作需要用户确认：manage_note 的 " + manageNoteActionCN(args.Action) +
			" 属于写操作，执行前必须先征得用户同意。" +
			"请在回复正文中说明要执行的具体操作与影响，并调用 ask_user 工具向用户确认；" +
			"用户明确同意后，携带 confirm=true 重新调用本工具即可执行。", nil
	}

	if m.ctx != nil && m.ctx.Logger != nil {
		m.ctx.Logger.Debugw("Agent manage_note 调用",
			fastlog.String("action", args.Action),
			fastlog.Int("id", int(args.ID)),
			fastlog.String("title", args.Title),
			fastlog.String("keyword", args.Keyword),
			fastlog.Int("page", int(args.Page)),
			fastlog.Int("pageSize", int(args.PageSize)))
	}

	switch args.Action {
	case "create":
		return m.createNote(args.Title, args.Content, args.FileExt, args.NotebookID, args.TagIDs)
	case "list":
		return m.listNotes(args.Keyword, int(args.Page), int(args.PageSize), int(args.NotebookID), args.SortBy, args.StartDate, args.EndDate, args.TagIDs)
	case "view":
		return m.viewNote(args.ID, args.LineNumbers)
	case "update":
		return m.updateNote(args.ID, args.Title, args.FileExt)
	case "edit":
		return m.editNote(args.ID, args.Find, args.Replace, args.Count, args.ReplaceAll, args.LineStart, args.LineEnd)
	case "pin":
		return m.pinNote(args.ID)
	case "move":
		return m.moveNote(args.ID, args.NotebookID)
	case "add_tag":
		return m.addTag(args.ID, args.TagID)
	case "remove_tag":
		return m.removeTag(args.ID, args.TagID)
	}
	return "", fmt.Errorf("manage_note 未知 action: %s", args.Action)
}

// createNote 创建笔记：title / content 必填（trim 后非空）；file_ext 缺省 ".md"（不强制校验格式）；
// notebook_id > 0 时创建到指定笔记本，未指定时归入默认笔记本（id=1，EnsureDefaultNotebook 保证存在，
// 与 app.go SaveAIMessageAsNote 的归入约定一致，避免新笔记落在 notebook_id=0 的"无归属"状态）；
// tag_ids 非空时逐个给新笔记打标签（单个失败则整体报错）。
func (m *manageNoteTool) createNote(title, content, fileExt string, notebookID float64, tagIDs []float64) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", errors.New("manage_note 创建笔记缺少 title")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", errors.New("manage_note 创建笔记缺少 content")
	}
	var err error
	fileExt, err = normalizeNoteFileExt(fileExt)
	if err != nil {
		return "", err
	}
	if fileExt == "" {
		fileExt = ".md"
	}

	var note *models.Note
	if notebookID > 0 {
		note, err = m.note.CreateWithNotebook(title, content, fileExt, uint(notebookID))
	} else {
		// 未指定笔记本时归入默认笔记本（id=1），与全局"默认笔记本不可删除/不可重命名"约定一致
		note, err = m.note.CreateWithNotebook(title, content, fileExt, 1)
	}
	if err != nil {
		return "", err
	}

	tagCount := 0
	for _, t := range tagIDs {
		if t <= 0 {
			continue
		}
		if err := m.tag.AddTagToNote(note.ID, uint(t)); err != nil {
			return "", fmt.Errorf("给新笔记 #%d 添加标签失败: %w", note.ID, err)
		}
		tagCount++
	}

	msg := fmt.Sprintf("已创建笔记 [%d]：%s（创建时间 %s）", note.ID, note.Title, note.CreatedAt.Format("2006-01-02 15:04"))
	if notebookID > 0 {
		msg += fmt.Sprintf("，归入笔记本 #%d", uint(notebookID))
	} else {
		msg += "，归入默认笔记本"
	}
	if tagCount > 0 {
		msg += fmt.Sprintf("· 已添加 %d 个标签", tagCount)
	}
	return msg, nil
}

// listNotes 分页列出/搜索笔记：keyword/日期空值透传；sort_by 缺省 updated_at（非法返回错误）；
// page/pageSize 分页（pageSize 缺省 10、上限 50）；tag_ids 多标签 AND 过滤（空值不过滤）；
// notebookID > 0 时限定在指定笔记本范围内搜索（NoteService.SearchByNotebook），否则全库搜索（NoteService.Search）。
// 过滤与分页都在 DB 层完成，只加载当前页条目；
// Search 返回的 note.Content 是前 200 字符预览，直接格式化，不再额外截断。
func (m *manageNoteTool) listNotes(keyword string, page, pageSize int, notebookID int, sortBy, startDate, endDate string, tagIDs []float64) (string, error) {
	keyword = strings.TrimSpace(keyword)
	sortBy = strings.TrimSpace(sortBy)
	if sortBy == "" {
		sortBy = "updated_at"
	}
	switch sortBy {
	case "updated_at", "created_at", "title":
	default:
		return "", fmt.Errorf("manage_note 参数非法 sort_by: %s", sortBy)
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

// viewNote 查看笔记全文：id 必填、正整数；内容超过 notePreviewThreshold（缺省 10000）时
// 用 TruncateRunes 截断，并返回总字符数与 read_note_section 续读指引（模型可据此
// 携带 id/offset 调用 read_note_section 读取后续分段）。
// lineNumbers 为 true 时输出带「行 N: 」行号前缀（1-based），作为 edit 行级替换的
// 寻址坐标；行号前缀不属于正文，复制片段用于 find 时须去掉行号。
func (m *manageNoteTool) viewNote(id float64, lineNumbers bool) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 查看笔记缺少有效的 id")
	}
	content, err := m.note.GetNoteContent(uint(id))
	if err != nil {
		return "", err
	}

	total := len([]rune(content))
	totalLines := len(splitNoteLines(content))
	threshold := notePreviewThreshold(m.setting)
	truncated := false
	if total > threshold {
		content = TruncateRunes(content, threshold)
		truncated = true
	}
	displayedLines := totalLines
	if lineNumbers {
		content = numberLines(content, 1)
		if truncated {
			// 从行号化后的内容提取已显示行数（最后一个「行 N:」前缀）
			displayedLines = extractLastLineNum(content)
		}
	}
	if truncated {
		content += fmt.Sprintf("\n\n（内容共 %d 字符 / %d 行，已显示前 %d 字符 / %d 行。如需继续阅读，可调用 read_note_section 工具，参数 id=%d, offset=%d；如需按行编辑，可让 read_note_section 带 line_numbers=true 获取全局行号）",
			total, totalLines, threshold, displayedLines, uint(id), threshold)
	}
	return fmt.Sprintf("笔记 #%d 内容：\n%s", uint(id), content), nil
}

// notePreviewThreshold 读取大文件预览阈值设置 ai_large_file_preview_threshold
// （解析失败或 <=0 时缺省 10000），供 view / read_note_section 共用。
func notePreviewThreshold(setting *services.SettingService) int {
	const def = 10000
	if setting == nil {
		return def
	}
	val := setting.Get("ai_large_file_preview_threshold")
	if val == "" {
		return def
	}
	if n, err := strconv.Atoi(val); err == nil && n > 0 {
		return n
	}
	return def
}

// updateNote 更新笔记标题/文件扩展名（不碰正文）：id 必填、正整数；title / file_ext
// 至少提供一个（trim 后非空），非空字段才更新、空字段保持不变（NoteService.Update 语义）。
func (m *manageNoteTool) updateNote(id float64, title, fileExt string) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 更新笔记缺少有效的 id")
	}
	title = strings.TrimSpace(title)
	fileExt, err := normalizeNoteFileExt(fileExt)
	if err != nil {
		return "", err
	}
	if title == "" && fileExt == "" {
		return "", errors.New("manage_note 更新笔记无可更新内容：请提供 title 或 file_ext")
	}
	note, err := m.note.Update(uint(id), title, "", fileExt)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("笔记 #%d：%s（扩展名 %s）已更新", note.ID, note.Title, note.FileExt), nil
}

// editNote 编辑笔记正文：id 必填、正整数。双模式互斥——
//   - 片段模式：find 非空时，定位第 count 次（缺省 1）出现的 find 片段，
//     替换为 replace（缺省空字符串即删除该片段）；find 优先精确匹配，因空白/换行差异
//     未命中时自动按空白归一化匹配兜底；replace_all=true 时替换全部出现（与 count 互斥）；
//   - 行级模式：line_start 非 0 时，把第 line_start 行到第 line_end 行（缺省等于 line_start）
//     的区间替换为 replace（空字符串即删除该区间行），行号来自 view/read_note_section
//     的 line_numbers=true 输出；line_start 大于笔记总行数时为末尾追加语义。
//
// 只需修改几个字或一句话用片段替换，需要修改连续多行、整段重写、或无法用简短片段
// 定位时用行级替换。
func (m *manageNoteTool) editNote(id float64, find, replace string, count float64, replaceAll bool, lineStart, lineEnd float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 编辑笔记缺少有效的 id")
	}
	find = strings.TrimSpace(find)
	// 模式判定：二选一互斥（find 片段替换 / line_start 行级替换）
	if find != "" && lineStart > 0 {
		return "", errors.New("manage_note 编辑笔记多种模式不可混用：find+replace（片段替换）与 line_start（行级替换）请二选一")
	}
	if find == "" && lineStart <= 0 {
		return "", errors.New("manage_note 编辑笔记无可更新内容：请提供 find+replace（片段替换）或 line_start（行级替换，含末尾追加）")
	}
	// 片段模式专属参数校验（replace_all 与 count>1 互斥），须在触达 DB 前完成
	if find != "" && replaceAll && count > 1 {
		return "", errors.New("manage_note 片段替换的 replace_all 与 count>1 互斥，不可同时使用（replace_all 时 count 只接受缺省 1）")
	}

	// 行级模式：把第 line_start 行到第 line_end 行替换为 replace（空即删除该区间）
	// line_start 大于总行数时为末尾追加语义
	if lineStart > 0 {
		start := int(lineStart)
		end := int(lineEnd)
		if end < start {
			end = start
		}
		current, err := m.note.GetNoteContent(uint(id))
		if err != nil {
			return "", err
		}
		lines := splitNoteLines(current)
		total := len(lines)
		// 末尾追加：start > total 时，在末尾追加 replace 内容
		if start > total {
			newContent := current
			if strings.TrimSpace(current) != "" {
				newContent += "\n\n"
			}
			newContent += replace
			if _, err := m.note.Update(uint(id), "", newContent, ""); err != nil {
				return "", err
			}
			return fmt.Sprintf("笔记 #%d 已在末尾追加内容", uint(id)), nil
		}
		newContent, replaced, total, err := replaceLines(current, start, end, replace)
		if err != nil {
			return "", err
		}
		if _, err := m.note.Update(uint(id), "", newContent, ""); err != nil {
			return "", err
		}
		newTotal := len(splitNoteLines(newContent))
		// 构建反馈：基本信息 + 行数变化 + 替换区域上下文预览
		var fb string
		if replaced == 1 {
			fb = fmt.Sprintf("笔记 #%d 已替换第 %d 行（原 %d 行 → 现 %d 行）", uint(id), start, total, newTotal)
		} else {
			fb = fmt.Sprintf("笔记 #%d 已替换第 %d-%d 行（原 %d 行 → 现 %d 行）", uint(id), start, end, total, newTotal)
		}
		// 附带替换区域上下文预览（前 1 行 + 替换内容 + 后 1 行，限 300 字符）
		if preview := lineEditPreview(newContent, start, newTotal); preview != "" {
			fb += "：\n" + preview
		}
		return fb, nil
	}

	// 片段模式：定位第 n 次出现的 find 片段并替换为 replace
	current, err := m.note.GetNoteContent(uint(id))
	if err != nil {
		return "", err
	}

	if replaceAll {
		newContent, n := replaceAllFragments(current, find, replace)
		if n == 0 {
			return buildNotFoundHint(uint(id), find, current)
		}
		if _, err := m.note.Update(uint(id), "", newContent, ""); err != nil {
			return "", err
		}
		fb := fmt.Sprintf("笔记 #%d 正文片段已全部替换（共 %d 处）", uint(id), n)
		// 展示第一处 diff 摘要
		if pos := indexNth(current, find, 1); pos >= 0 {
			fb += fmt.Sprintf("：\n-旧: %q\n+新: %q", truncateSnippet(current[pos:pos+len(find)], 80), truncateSnippet(replace, 80))
		}
		return fb, nil
	}

	n := int(count)
	if n < 1 {
		n = 1
	}
	pos := indexNth(current, find, n)
	matchedLen := len(find)
	matchedKind := "精确"
	if pos < 0 {
		// 兜底：空白归一化匹配（忽略缩进/换行/连续空白差异），映射回原文偏移
		if s, e := findNormalized(current, find, n); s >= 0 {
			pos = s
			matchedLen = e - s
			matchedKind = "空白归一化"
		}
	}
	if pos < 0 {
		return buildNotFoundHint(uint(id), find, current)
	}
	newContent := current[:pos] + replace + current[pos+matchedLen:]
	if _, err := m.note.Update(uint(id), "", newContent, ""); err != nil {
		return "", err
	}
	fb := fmt.Sprintf("笔记 #%d 正文片段已替换（第 %d 处，%s匹配）", uint(id), n, matchedKind)
	fb += fmt.Sprintf("：\n-旧: %q\n+新: %q", truncateSnippet(current[pos:pos+matchedLen], 80), truncateSnippet(replace, 80))
	return fb, nil
}

// indexNth 返回 s 中第 nth（从 1 开始）次出现 sub 的起始下标；不存在返回 -1。
func indexNth(s, sub string, nth int) int {
	if nth < 1 || sub == "" {
		return -1
	}
	idx := 0
	for i := 0; i < nth; i++ {
		pos := strings.Index(s[idx:], sub)
		if pos < 0 {
			return -1
		}
		idx += pos
		if i < nth-1 {
			idx += len(sub)
		}
	}
	return idx
}

// whitespaceFold 将文本中的空白折叠为单一形式用于归一化匹配：
// 把 \r\n 统一为 \n，再把任意连续空白（空格/制表符/换行）折叠为一个空格，并去掉首尾空白。
func whitespaceFold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true // 行首空白折叠掉
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
		i += size
	}
	return strings.TrimSuffix(b.String(), " ")
}

// findNormalized 在 current 中查找 find 的空白归一化匹配（第 nth 次，从 1 开始），
// 返回匹配片段在原文中的字节起止下标 [start, end)；未找到返回 (-1, -1)。
// 实现：把原文折叠为空白归一化文本（\r\n→\n、连续空白→单空格、去首尾空白），同时记录
// 折叠文本中每个 rune 对应的原文字节偏移（折叠出的空格无对应偏移，记 -1）；在折叠文本上
// 做普通字符串查找后，用偏移表把折叠偏移映射回原文偏移。
func findNormalized(current, find string, nth int) (int, int) {
	if nth < 1 {
		return -1, -1
	}
	normFind := whitespaceFold(find)
	if normFind == "" {
		return -1, -1
	}
	// 折叠原文并记录 rune→原文字节偏移映射（offsets 与 folded 等长）
	folded := make([]rune, 0, len(current))
	offsets := make([]int, 0, len(current))
	var b strings.Builder
	prevSpace := true // 行首空白折叠掉
	for i := 0; i < len(current); {
		r, size := utf8.DecodeRuneInString(current[i:])
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				folded = append(folded, ' ')
				offsets = append(offsets, -1)
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			folded = append(folded, r)
			offsets = append(offsets, i)
			prevSpace = false
		}
		i += size
	}
	norm := b.String()
	// 去掉折叠末尾的空格（原文以空白结尾时最多折叠出一个）
	if strings.HasSuffix(norm, " ") {
		norm = norm[:len(norm)-1]
		folded = folded[:len(folded)-1]
		offsets = offsets[:len(offsets)-1]
	}
	normFindRunes := utf8.RuneCountInString(normFind)

	// 在折叠文本上查找第 nth 次出现
	idx := 0
	for i := 0; i < nth; i++ {
		pos := strings.Index(norm[idx:], normFind)
		if pos < 0 {
			return -1, -1
		}
		startFold := idx + pos
		if i < nth-1 {
			idx = startFold + len(normFind)
			continue
		}
		// 折叠文本中匹配片段的起止 rune 下标
		startRune := utf8.RuneCountInString(norm[:startFold])
		endRune := startRune + normFindRunes
		if startRune >= len(folded) || endRune > len(folded) {
			return -1, -1
		}
		// 首字符必非空白（whitespaceFold 已去首尾），映射一定存在
		startOrig := offsets[startRune]
		if startOrig < 0 {
			return -1, -1
		}
		// 末字符亦非空白，取其在原文中的偏移 + 字节长度作为结束下标
		lastOrig := offsets[endRune-1]
		if lastOrig < 0 {
			return -1, -1
		}
		_, lastSize := utf8.DecodeRuneInString(current[lastOrig:])
		return startOrig, lastOrig + lastSize
	}
	return -1, -1
}

// replaceAllFragments 把 current 中全部出现（或空白归一化等价出现）的 find 替换为
// replace，返回新文本与替换次数；find 为空返回原文本与 0。
// 语义：等价于"全部替换"——先精确替换全部精确出现，再对剩余位置做空白归一化匹配
// （归一化替换会替换掉该片段对应的原文区间，映射回原文后执行）。
func replaceAllFragments(current, find, replace string) (string, int) {
	if find == "" {
		return current, 0
	}
	out := current
	count := 0
	if n := strings.Count(out, find); n > 0 {
		out = strings.ReplaceAll(out, find, replace)
		count += n
	}
	// 精确匹配已全部处理，剩余位置做空白归一化匹配逐个替换
	for {
		s, e := findNormalized(out, find, 1)
		if s < 0 {
			break
		}
		out = out[:s] + replace + out[e:]
		count++
		// 安全上限，防止替换文本自身引入新的归一化匹配导致死循环
		if count > 10000 {
			break
		}
	}
	return out, count
}

// extractLastLineNum 从行号化文本（"行 N: ..."）中提取最后一个行号。
// 用于 viewNote 截断时推算已显示行数。
func extractLastLineNum(numbered string) int {
	// 从末尾向前找最后一个「行 」前缀
	idx := strings.LastIndex(numbered, "行 ")
	if idx < 0 {
		return 0
	}
	// 提取数字部分
	numStr := ""
	for i := idx + len("行 "); i < len(numbered); i++ {
		ch := numbered[i]
		if ch >= '0' && ch <= '9' {
			numStr += string(ch)
		} else {
			break
		}
	}
	n := 0
	for _, ch := range numStr {
		n = n*10 + int(ch-'0')
	}
	return n
}

// lineEditPreview 从 newContent 中提取替换区域的上下文预览（前 1 行 + 替换区域 + 后 1 行），
// 用行号格式化，总长限 maxPreviewLen 字符。无内容时返回空串。
func lineEditPreview(newContent string, replacedLine, newTotal int) string {
	const maxPreviewLen = 300
	lines := splitNoteLines(newContent)
	if len(lines) == 0 {
		return ""
	}
	// 确定预览范围：[start, end)，1-based
	start := replacedLine - 1 // 替换区域起始（0-based）
	if start < 0 {
		start = 0
	}
	end := replacedLine // 替换区域结束（0-based，不含）
	if end > len(lines) {
		end = len(lines)
	}
	// 扩展：前 1 行 + 后 1 行
	previewStart := start - 1
	if previewStart < 0 {
		previewStart = 0
	}
	previewEnd := end + 1
	if previewEnd > len(lines) {
		previewEnd = len(lines)
	}
	previewLines := lines[previewStart:previewEnd]
	preview := numberLines(strings.Join(previewLines, "\n"), previewStart+1)
	if len([]rune(preview)) > maxPreviewLen {
		preview = string([]rune(preview)[:maxPreviewLen]) + "..."
	}
	return preview
}

// truncateSnippet 截取片段摘要，限 maxLen 字符，超出时加 ...。
func truncateSnippet(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// buildNotFoundHint 构建"片段未找到"的错误信息，附带笔记中最相似的片段提示。
func buildNotFoundHint(id uint, find, current string) (string, error) {
	similar, lineNum := findMostSimilar(current, find)
	hint := fmt.Sprintf("未在笔记 #%d 中找到片段「%s」（已尝试空白归一化匹配）", id, truncateSnippet(find, 60))
	if similar != "" {
		similar = truncateSnippet(similar, 100)
		if lineNum > 0 {
			hint += fmt.Sprintf("。笔记中最接近的内容（第 %d 行附近）：\n「%s」", lineNum, similar)
		} else {
			hint += fmt.Sprintf("。笔记中最接近的内容：\n「%s」", similar)
		}
	}
	hint += "\n请确认片段是否正确，或调用 view 获取精确原文后重试"
	return "", errors.New(hint)
}

// findMostSimilar 在 content 中用滑动窗口查找与 find 最相似的子串，
// 返回最相似片段及其大致行号（通过计数 \n 得到）。未找到时返回 ("", 0)。
func findMostSimilar(content, find string) (string, int) {
	if content == "" || find == "" {
		return "", 0
	}
	findRunes := []rune(find)
	fLen := len(findRunes)
	if fLen == 0 {
		return "", 0
	}
	// 窗口大小 = fLen ± 50%
	minWin := fLen * 50 / 100
	if minWin < 1 {
		minWin = 1
	}
	maxWin := fLen + fLen/2
	contentRunes := []rune(content)
	cLen := len(contentRunes)

	bestScore := -1.0
	bestStart := 0
	bestEnd := 0

	for winSize := minWin; winSize <= maxWin; winSize++ {
		if winSize > cLen {
			break
		}
		for i := 0; i <= cLen-winSize; i++ {
			window := contentRunes[i : i+winSize]
			score := runeOverlap(findRunes, window)
			if score > bestScore {
				bestScore = score
				bestStart = i
				bestEnd = i + winSize
			}
		}
	}

	if bestScore <= 0 {
		return "", 0
	}

	// 计算大致行号
	lineNum := 1
	for i := 0; i < bestStart; i++ {
		if contentRunes[i] == '\n' {
			lineNum++
		}
	}

	return string(contentRunes[bestStart:bestEnd]), lineNum
}

// runeOverlap 计算两个 rune 切片的字符重叠率（0.0 ~ 1.0）。
func runeOverlap(a, b []rune) float64 {
	set := make(map[rune]struct{}, len(a))
	for _, r := range a {
		set[r] = struct{}{}
	}
	common := 0
	for _, r := range b {
		if _, ok := set[r]; ok {
			common++
		}
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 0
	}
	return float64(common) / float64(maxLen)
}

// splitNoteLines 按 \n 拆分正文为行（保留每行内容；行尾 \r 属于上一行内容的一部分，
// 由 replaceLines 重建时还原）。末尾换行产生的空元素剔除（"a\nb\n" → ["a","b"]），
// 内容为空返回 nil。
func splitNoteLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// replaceLines 把正文中第 start 行到第 end 行（1-based，含端点）替换为 replace 文本
// （replace 为空字符串即删除该区间行）。返回新正文、被替换的行数、原总行数。
// 行号越界（start<1、end>总行数、start>end）返回错误；行号来自 view/read_note_section
// 的 line_numbers=true 输出。\r\n 的 \r 在逐行内容中被保留，重建时随行还原。
func replaceLines(content string, start, end int, replace string) (string, int, int, error) {
	lines := splitNoteLines(content)
	total := len(lines)
	if start < 1 || end > total || start > end {
		return "", 0, total, fmt.Errorf("manage_note 行级替换的行号越界：笔记共 %d 行，请求替换第 %d-%d 行", total, start, end)
	}
	var b strings.Builder
	// 保留区间前的行
	for i := 0; i < start-1; i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	// replace 按行展开写入（空字符串表示删除区间，不写任何行）
	if replace != "" {
		repLines := splitNoteLines(replace)
		for i, ln := range repLines {
			b.WriteString(ln)
			if i < len(repLines)-1 {
				b.WriteByte('\n')
			}
		}
	}
	// 区间后的行：若 replace 为空或区间后还有行，需在区间后补一个换行（除非 replace 已写入且
	// 区间后无行——此时正文以 replace 的末行结束）。为保持"替换行区间"语义，统一在区间与
	// 后续行之间补换行（若 replace 非空且未以换行结尾）。
	if end < total {
		if replace != "" {
			b.WriteByte('\n')
		}
		for i := end; i < total; i++ {
			b.WriteString(lines[i])
			if i < total-1 {
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), end - start + 1, total, nil
}

// numberLines 将内容按行拆分并加「行 N: 」行号前缀（1-based，startLine 为第一行的行号，
// 供 read_note_section 续读时保持全局行号），返回带行号的文本；内容为空返回空串。
// 行号前缀仅作寻址坐标，不属于正文——复制片段用于 find 时须去掉行号前缀。
func numberLines(content string, startLine int) string {
	if content == "" {
		return ""
	}
	lines := splitNoteLines(content)
	var b strings.Builder
	for i, ln := range lines {
		// 去掉行尾 \r（\r\n 换行），避免行号后出现多余控制符
		ln = strings.TrimSuffix(ln, "\r")
		fmt.Fprintf(&b, "行 %d: %s\n", startLine+i, ln)
	}
	return b.String()
}

// pinNote 置顶/取消置顶笔记：id 必填、正整数，切换置顶状态并返回结果文案。
func (m *manageNoteTool) pinNote(id float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 置顶笔记缺少有效的 id")
	}
	t, err := m.note.TogglePin(uint(id))
	if err != nil {
		return "", err
	}
	if t.Pinned {
		return fmt.Sprintf("笔记 #%d：%s 已置顶", t.ID, t.Title), nil
	}
	return fmt.Sprintf("笔记 #%d：%s 已取消置顶", t.ID, t.Title), nil
}

// moveNote 移动笔记到目标笔记本：id 必填、正整数，notebook_id 必填目标笔记本。
func (m *manageNoteTool) moveNote(id, notebookID float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 移动笔记缺少有效的 id")
	}
	if notebookID <= 0 {
		return "", errors.New("manage_note 移动笔记缺少有效的 notebook_id")
	}
	if err := m.note.MoveToNotebook(uint(id), uint(notebookID)); err != nil {
		return "", err
	}
	return fmt.Sprintf("已将笔记 #%d 移动到笔记本 #%d", uint(id), uint(notebookID)), nil
}

// addTag 给笔记添加标签：id 必填、正整数，tag_id 必填、正整数。
func (m *manageNoteTool) addTag(id, tagID float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 添加标签缺少有效的 id")
	}
	if tagID <= 0 {
		return "", errors.New("manage_note 添加标签缺少有效的 tag_id")
	}
	if err := m.tag.AddTagToNote(uint(id), uint(tagID)); err != nil {
		return "", err
	}
	return fmt.Sprintf("已为笔记 #%d 添加标签 #%d", uint(id), uint(tagID)), nil
}

// removeTag 从笔记移除标签：id 必填、正整数，tag_id 必填、正整数。
func (m *manageNoteTool) removeTag(id, tagID float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 移除标签缺少有效的 id")
	}
	if tagID <= 0 {
		return "", errors.New("manage_note 移除标签缺少有效的 tag_id")
	}
	if err := m.tag.RemoveTagFromNote(uint(id), uint(tagID)); err != nil {
		return "", err
	}
	return fmt.Sprintf("已从笔记 #%d 移除标签 #%d", uint(id), uint(tagID)), nil
}

// NewManageNote 创建笔记库管理工具。
func NewManageNote(note *services.NoteService, tag *services.TagService, setting *services.SettingService, ctx *Context) tool.InvokableTool {
	return &manageNoteTool{note: note, tag: tag, setting: setting, ctx: ctx}
}
