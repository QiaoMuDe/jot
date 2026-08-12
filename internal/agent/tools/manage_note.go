package tools

// 本文件实现 manage_note 笔记库管理工具：模型在 ReAct 循环中调用它创建笔记、列出/搜索笔记、
// 查看笔记全文、更新标题/扩展名、编辑正文、置顶/取消置顶、移动笔记本、给笔记打标签或移除标签，
// 底层复用 services.NoteService（Create / CreateWithNotebook / Search / GetNoteContent / Update /
// TogglePin / MoveToNotebook）与 services.TagService（AddTagToNote / RemoveTagFromNote），
// 不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分九个动作：
//   - create：创建笔记（title / content 必填；file_ext 可选、缺省 .md；notebook_id 可选
//     指定创建到哪个笔记本；tag_ids 可选给新笔记打标签）；
//   - list：列出/搜索笔记（keyword 按标题/内容模糊过滤；tag_ids 多标签 AND 过滤；
//     start_date / end_date 按 updated_at 日期范围过滤；sort_by 排序，缺省 updated_at；
//     page/pageSize 分页，pageSize 缺省 10、上限 50），
//     返回"共 n 条、第 x/y 页"，列表只展示当前页条目；当页未展示完时提示可翻页；
//   - view：查看笔记全文（id 必填、正整数，来自列表中的 [数字] 编号；内容超过
//     ai_large_file_preview_threshold 设置（解析失败或<=0 时缺省 10000）时截断，
//     并给出 read_note_section 工具的续读指引（id/offset 参数））；
//   - update：更新笔记标题/扩展名（id 必填、正整数；title / file_ext 至少提供一个，
//     非空才更新对应字段，不碰正文）；
//   - edit：编辑笔记正文（id 必填、正整数；三模式互斥——content 非空为整篇替换；
//     content 为空、find 非空为片段替换，把第 count 次（缺省 1）出现的 find 片段
//     替换为 replace（缺省空字符串即删除该片段），未找到报错引导重新 view 获取精确原文；
//     content 与 find 均为空、append_content 非空为追加模式，在正文末尾拼接文本，
//     无需先读全文）；
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
		Desc: "管理用户笔记库。当用户要求创建笔记、列出/搜索笔记、查看笔记全文、更新笔记标题或扩展名、编辑笔记正文、置顶/取消置顶、移动笔记本、给笔记打标签或移除标签时调用。与 recall_notes 的边界：recall_notes 用于语义召回笔记片段回答知识类问题，manage_note 用于结构化操作笔记库。通过 action 参数区分动作：create=创建笔记（需提供 title 标题与 content 内容，可提供 file_ext 文件后缀（缺省 .md）、notebook_id 目标笔记本、tag_ids 标签编号列表）；list=列出/搜索笔记（可用 keyword 标题/内容关键字过滤，tag_ids 多标签 AND 过滤，start_date/end_date 按更新时间范围过滤，sort_by 排序（updated_at/created_at/title，缺省 updated_at），page 页码与 pageSize 每页条数（缺省 10、上限 50）分页查看）；view=查看笔记全文（需提供 id 笔记编号，内容过长时会截断并可要求分段查看）；update=更新笔记标题/扩展名（需提供 id 笔记编号与 title 新标题、file_ext 新扩展名至少其一，只改元数据不碰正文）；edit=编辑笔记正文（需提供 id 笔记编号；整篇替换时提供 content 新正文；只改/删某段时提供 find 要替换的原文片段与 replace 新文本，务必从 view 结果中精确复制 find（标点空格须完全一致），删除片段时 replace 传空字符串，长笔记建议用 find+replace 避免回传全文；末尾追加时提供 append_content，无需先读全文）；pin=置顶/取消置顶笔记（需提供 id 笔记编号）；move=移动笔记到目标笔记本（需提供 id 笔记编号与 notebook_id 目标笔记本）；add_tag=给笔记添加标签（需提供 id 笔记编号与 tag_id 标签编号）；remove_tag=从笔记移除标签（需提供 id 笔记编号与 tag_id 标签编号）。强制确认：update / edit / pin / move / add_tag / remove_tag 均属写操作，执行前必须先向用户确认修改意图——在回复正文中说明要执行的具体操作与影响，并调用 ask_user 工具向用户提问，用户明确同意后再携带 confirm=true 调用本工具；未携带 confirm=true 时工具会拒绝执行并提示先确认（create 为用户明确要求的创建指令，无需确认）。返回笔记列表或操作结果，列表中的编号 [数字] 可用于后续 view/update/edit/pin/move/add_tag/remove_tag。",
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
				Desc:     "笔记内容，action=create 时必填；action=edit 时非空即整篇替换正文（与 find、append_content 互斥）",
				Required: false,
			},
			"file_ext": {
				Type:     schema.String,
				Desc:     "笔记文件后缀，action=create 时缺省 .md；action=update 时可选（不传则保持原扩展名）",
				Required: false,
			},
			"find": {
				Type:     schema.String,
				Desc:     "片段替换的原文片段，仅 action=edit 且未提供 content 时使用；务必从 view 结果中精确复制（标点空格须完全一致）",
				Required: false,
			},
			"replace": {
				Type:     schema.String,
				Desc:     "片段替换后的新文本，仅 action=edit 片段替换时使用，缺省空字符串（即删除该片段）",
				Required: false,
			},
			"count": {
				Type:     schema.Number,
				Desc:     "find 片段在正文中第几次出现，仅 action=edit 片段替换时使用，缺省 1",
				Required: false,
			},
			"append_content": {
				Type:     schema.String,
				Desc:     "追加到笔记末尾的文本，仅 action=edit 且需追加内容时使用（与 content、find 互斥），无需先获取全文",
				Required: false,
			},
			"notebook_id": {
				Type:     schema.Number,
				Desc:     "笔记本编号，action=create 时可选（创建到该笔记本）；action=move 时必填（目标笔记本）；action=list 时可选（仅列出该笔记本下的笔记）",
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
		Action        string    `json:"action"`
		Title         string    `json:"title"`
		Content       string    `json:"content"`
		FileExt       string    `json:"file_ext"`
		Find          string    `json:"find"`
		Replace       string    `json:"replace"`
		Count         float64   `json:"count"`
		AppendContent string    `json:"append_content"`
		NotebookID    float64   `json:"notebook_id"`
		TagIDs        []float64 `json:"tag_ids"`
		Keyword       string    `json:"keyword"`
		StartDate     string    `json:"start_date"`
		EndDate       string    `json:"end_date"`
		SortBy        string    `json:"sort_by"`
		Page          float64   `json:"page"`
		PageSize      float64   `json:"pageSize"`
		ID            float64   `json:"id"`
		TagID         float64   `json:"tag_id"`
		Confirm       bool      `json:"confirm"`
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
		return m.viewNote(args.ID)
	case "update":
		return m.updateNote(args.ID, args.Title, args.FileExt)
	case "edit":
		return m.editNote(args.ID, args.Content, args.Find, args.Replace, args.AppendContent, args.Count)
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
// notebook_id > 0 时指定笔记本；tag_ids 非空时逐个给新笔记打标签（单个失败则整体报错）。
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

	var (
		note *models.Note
	)
	if notebookID > 0 {
		note, err = m.note.CreateWithNotebook(title, content, fileExt, uint(notebookID))
	} else {
		note, err = m.note.Create(title, content, fileExt)
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
func (m *manageNoteTool) viewNote(id float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 查看笔记缺少有效的 id")
	}
	content, err := m.note.GetNoteContent(uint(id))
	if err != nil {
		return "", err
	}

	total := len([]rune(content))
	threshold := notePreviewThreshold(m.setting)
	if total > threshold {
		content = TruncateRunes(content, threshold) +
			fmt.Sprintf("\n\n（内容共 %d 字符，已显示前 %d。如需继续阅读，可调用 read_note_section 工具，参数 id=%d, offset=%d）",
				total, threshold, uint(id), threshold)
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

// editNote 编辑笔记正文：id 必填、正整数。三种模式互斥——
//   - 全量模式：content 非空时整篇替换正文；
//   - 片段模式：content 为空、find 非空时，定位第 count 次（缺省 1）出现的 find 片段，
//     替换为 replace（缺省空字符串即删除该片段）；未找到片段报错并引导重新 view 获取精确原文；
//   - 追加模式：content 与 find 均为空、append_content 非空时，在正文末尾拼接文本（无需先读全文）。
func (m *manageNoteTool) editNote(id float64, content, find, replace, appendContent string, count float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 编辑笔记缺少有效的 id")
	}
	content = strings.TrimSpace(content)
	find = strings.TrimSpace(find)
	appendContent = strings.TrimSpace(appendContent)
	// 模式判定：三选一互斥
	modes := 0
	if content != "" {
		modes++
	}
	if find != "" {
		modes++
	}
	if appendContent != "" {
		modes++
	}
	if modes > 1 {
		return "", errors.New("manage_note 编辑笔记多种模式不可混用：content / find+replace / append_content 请三选一")
	}
	if modes == 0 {
		return "", errors.New("manage_note 编辑笔记无可更新内容：请提供 content（整篇替换）、find+replace（片段替换）或 append_content（追加）")
	}

	// 全量模式：整篇替换正文（file_ext 传空保持原值）
	if content != "" {
		if _, err := m.note.Update(uint(id), "", content, ""); err != nil {
			return "", err
		}
		return fmt.Sprintf("笔记 #%d 正文已整篇更新", uint(id)), nil
	}

	// 追加模式：将 append_content 拼接到正文末尾（无需先读全文）
	if appendContent != "" {
		current, err := m.note.GetNoteContent(uint(id))
		if err != nil {
			return "", err
		}
		newContent := current
		if strings.TrimSpace(current) != "" {
			newContent += "\n\n"
		}
		newContent += appendContent
		if _, err := m.note.Update(uint(id), "", newContent, ""); err != nil {
			return "", err
		}
		return fmt.Sprintf("笔记 #%d 已在末尾追加内容", uint(id)), nil
	}

	// 片段模式：定位第 n 次出现的 find 片段并替换为 replace
	current, err := m.note.GetNoteContent(uint(id))
	if err != nil {
		return "", err
	}
	n := int(count)
	if n < 1 {
		n = 1
	}
	pos := indexNth(current, find, n)
	if pos < 0 {
		if n > 1 {
			return "", fmt.Errorf("未在笔记 #%d 中找到第 %d 次出现的片段，请重新调用 view 获取精确原文后重试", uint(id), n)
		}
		return "", fmt.Errorf("未在笔记 #%d 中找到该片段，请重新调用 view 获取精确原文后重试", uint(id))
	}
	newContent := current[:pos] + replace + current[pos+len(find):]
	if _, err := m.note.Update(uint(id), "", newContent, ""); err != nil {
		return "", err
	}
	return fmt.Sprintf("笔记 #%d 正文片段已替换（第 %d 处）", uint(id), n), nil
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
