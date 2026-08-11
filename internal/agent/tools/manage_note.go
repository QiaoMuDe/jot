package tools

// 本文件实现 manage_note 笔记库管理工具：模型在 ReAct 循环中调用它创建笔记、列出/搜索笔记、
// 查看笔记全文、置顶/取消置顶、移动笔记本、给笔记打标签或移除标签，底层复用
// services.NoteService（Create / CreateWithNotebook / Search / GetNoteContent / TogglePin /
// MoveToNotebook）与 services.TagService（AddTagToNote / RemoveTagFromNote），
// 不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分七个动作：
//   - create：创建笔记（title / content 必填；file_ext 可选、缺省 .md；notebook_id 可选
//     指定创建到哪个笔记本；tag_ids 可选给新笔记打标签）；
//   - list：列出/搜索笔记（keyword 按标题/内容模糊过滤；tag_ids 多标签 AND 过滤；
//     start_date / end_date 按 updated_at 日期范围过滤；sort_by 排序，缺省 updated_at；
//     page/pageSize 分页，pageSize 缺省 10、上限 50），
//     返回"共 n 条、第 x/y 页"，列表只展示当前页条目；当页未展示完时提示可翻页；
//   - view：查看笔记全文（id 必填、正整数，来自列表中的 [数字] 编号；内容超过
//     ai_large_file_preview_threshold 设置（解析失败或<=0 时缺省 10000）时截断并提示分段查看）；
//   - pin：置顶/取消置顶笔记（id 必填、正整数，切换置顶状态）；
//   - move：移动笔记到目标笔记本（id 必填、正整数，notebook_id 必填目标笔记本）；
//   - add_tag / remove_tag：给笔记添加/移除标签（id 必填、正整数，tag_id 必填、正整数）。
// 与 recall_notes 的边界：recall_notes 用于语义召回笔记片段回答知识类问题，
// manage_note 用于结构化操作笔记库。本工具不包含 update/删除类/批量动作（spec 明确不暴露）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/models"
	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

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
		Desc: "管理用户笔记库。当用户要求创建笔记、列出/搜索笔记、查看笔记全文、置顶/取消置顶、移动笔记本、给笔记打标签或移除标签时调用。与 recall_notes 的边界：recall_notes 用于语义召回笔记片段回答知识类问题，manage_note 用于结构化操作笔记库。通过 action 参数区分动作：create=创建笔记（需提供 title 标题与 content 内容，可提供 file_ext 文件后缀（缺省 .md）、notebook_id 目标笔记本、tag_ids 标签编号列表）；list=列出/搜索笔记（可用 keyword 标题/内容关键字过滤，tag_ids 多标签 AND 过滤，start_date/end_date 按更新时间范围过滤，sort_by 排序（updated_at/created_at/title，缺省 updated_at），page 页码与 pageSize 每页条数（缺省 10、上限 50）分页查看）；view=查看笔记全文（需提供 id 笔记编号，内容过长时会截断并可要求分段查看）；pin=置顶/取消置顶笔记（需提供 id 笔记编号）；move=移动笔记到目标笔记本（需提供 id 笔记编号与 notebook_id 目标笔记本）；add_tag=给笔记添加标签（需提供 id 笔记编号与 tag_id 标签编号）；remove_tag=从笔记移除标签（需提供 id 笔记编号与 tag_id 标签编号）。返回笔记列表或操作结果，列表中的编号 [数字] 可用于后续 view/pin/move/add_tag/remove_tag。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的动作：create=创建笔记；list=列出/搜索笔记；view=查看笔记全文；pin=置顶/取消置顶；move=移动笔记到目标笔记本；add_tag=给笔记添加标签；remove_tag=从笔记移除标签",
				Enum:     []string{"create", "list", "view", "pin", "move", "add_tag", "remove_tag"},
				Required: true,
			},
			"title": {
				Type:     schema.String,
				Desc:     "笔记标题，action=create 时必填",
				Required: false,
			},
			"content": {
				Type:     schema.String,
				Desc:     "笔记内容，action=create 时必填",
				Required: false,
			},
			"file_ext": {
				Type:     schema.String,
				Desc:     "笔记文件后缀，仅 action=create 时使用，缺省 .md",
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
				Desc:     "笔记编号（正整数，列表中的 [数字] 即为 id），action=view / pin / move / add_tag / remove_tag 时必填",
				Required: false,
			},
			"tag_id": {
				Type:     schema.Number,
				Desc:     "标签编号（正整数，manage_tag 列表中的 [数字] 即为 id），action=add_tag / remove_tag 时必填",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 校验 action → 按动作分发到 Create / Search / View / Pin / Move / Tag 操作。
func (m *manageNoteTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action     string    `json:"action"`
		Title      string    `json:"title"`
		Content    string    `json:"content"`
		FileExt    string    `json:"file_ext"`
		NotebookID float64   `json:"notebook_id"`
		TagIDs     []float64 `json:"tag_ids"`
		Keyword    string    `json:"keyword"`
		StartDate  string    `json:"start_date"`
		EndDate    string    `json:"end_date"`
		SortBy     string    `json:"sort_by"`
		Page       float64   `json:"page"`
		PageSize   float64   `json:"pageSize"`
		ID         float64   `json:"id"`
		TagID      float64   `json:"tag_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 manage_note 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)
	switch args.Action {
	case "create", "list", "view", "pin", "move", "add_tag", "remove_tag":
	default:
		return "", fmt.Errorf("manage_note 参数缺少/非法 action: %s", args.Action)
	}

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
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
	fileExt = strings.TrimSpace(fileExt)
	if fileExt == "" {
		fileExt = ".md"
	}

	var (
		note *models.Note
		err  error
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

// viewNote 查看笔记全文：id 必填、正整数；内容超过 ai_large_file_preview_threshold 设置
// （解析失败或<=0 时缺省 10000）时用 TruncateRunes 截断并提示分段查看。
func (m *manageNoteTool) viewNote(id float64) (string, error) {
	if id <= 0 {
		return "", errors.New("manage_note 查看笔记缺少有效的 id")
	}
	content, err := m.note.GetNoteContent(uint(id))
	if err != nil {
		return "", err
	}

	threshold := 10000
	if m.setting != nil {
		if val := m.setting.Get("ai_large_file_preview_threshold"); val != "" {
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				threshold = n
			}
		}
	}
	if len([]rune(content)) > threshold {
		content = TruncateRunes(content, threshold) + "\n\n（内容过长已截断，如需继续阅读可要求分段查看）"
	}
	return fmt.Sprintf("笔记 #%d 内容：\n%s", uint(id), content), nil
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
