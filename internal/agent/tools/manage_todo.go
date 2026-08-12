package tools

// 本文件实现 manage_todo 待办管理工具：模型在 ReAct 循环中调用它创建待办、
// 列出待办、勾选（完成/取消完成）待办或修改待办文本，底层复用 services.TodoService
// （Create / ListPaged / Search / Toggle / Update），不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分四个动作：
//   - create：创建待办（text 必填）；
//   - list：列出待办（status 过滤，缺省 active=未完成，done=已完成，all=全部；
//     keyword 按内容关键字过滤），
//     支持分页（page 页码从 1 开始，pageSize 每页条数，缺省 10、上限 50），
//     返回"共 n 条、第 x/y 页"，列表只展示当前页条目；当页未展示完时提示可翻页；
//   - toggle：勾选待办（id 必填、正整数，来自列表中的 [数字] 编号，切换完成/未完成）；
//   - update：修改待办文本（id 必填、正整数，text 必填）。

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

// manageTodoTool 待办管理工具。
type manageTodoTool struct {
	todo *services.TodoService // 待办服务（创建 / 列出 / 勾选）
	ctx  *Context              // 日志输出
}

// 编译期断言：确保 manageTodoTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*manageTodoTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 按 action 参数映射动作文案，解析失败回退空串（前端回退"执行"）。
func (m *manageTodoTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	switch args.Action {
	case "create":
		return "创建待办"
	case "list":
		return "列出待办"
	case "toggle":
		return "更新待办状态"
	case "update":
		return "修改待办文本"
	default:
		return "执行"
	}
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (m *manageTodoTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_todo",
		Desc: "管理待办事项。当用户要求创建待办、查看待办列表、勾选（完成/取消完成）待办或修改待办文本时调用。通过 action 参数区分动作：create=创建待办（需提供 text 待办内容）；list=列出待办（可用 status 过滤：active=未完成，缺省值；done=已完成；all=全部；可用 keyword 按待办内容关键字过滤，定位特定待办时优先用 keyword 而非翻页；待办较多时可用 page 页码与 pageSize 每页条数分页查看，pageSize 缺省 10、上限 50）；toggle=勾选待办（切换完成/未完成状态，需提供 id 待办编号，列表中的 [数字] 即为 id）；update=修改待办文本（需提供 id 待办编号与 text 新内容）。返回待办列表或操作结果，列表中的编号 [数字] 可用于后续 toggle/update。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的动作：create=创建待办；list=列出待办；toggle=勾选（切换完成/未完成）待办；update=修改待办文本",
				Enum:     []string{"create", "list", "toggle", "update"},
				Required: true,
			},
			"text": {
				Type:     schema.String,
				Desc:     "待办内容，action=create / update 时必填",
				Required: false,
			},
			"status": {
				Type:     schema.String,
				Desc:     "待办过滤状态，仅 action=list 时使用：active=未完成（缺省）；done=已完成；all=全部",
				Enum:     []string{"active", "done", "all"},
				Required: false,
			},
			"keyword": {
				Type:     schema.String,
				Desc:     "待办内容关键字过滤，仅 action=list 时使用，缺省不过滤",
				Required: false,
			},
			"id": {
				Type:     schema.Number,
				Desc:     "待办编号（正整数，列表中的 [数字] 即为 id），action=toggle / update 时必填",
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
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 校验 action → 按动作分发到 Create / List / Toggle。
func (m *manageTodoTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action   string  `json:"action"`
		Text     string  `json:"text"`
		Status   string  `json:"status"`
		Keyword  string  `json:"keyword"`
		ID       float64 `json:"id"`
		Page     float64 `json:"page"`
		PageSize float64 `json:"pageSize"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 manage_todo 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)
	switch args.Action {
	case "create", "list", "toggle", "update":
	default:
		return "", fmt.Errorf("manage_todo 参数缺少/非法 action: %s", args.Action)
	}

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if m.ctx != nil && m.ctx.Logger != nil {
		m.ctx.Logger.Debugw("Agent manage_todo 调用",
			fastlog.String("action", args.Action),
			fastlog.Int("id", int(args.ID)),
			fastlog.String("text", args.Text),
			fastlog.String("status", args.Status),
			fastlog.String("keyword", args.Keyword))
	}

	switch args.Action {
	case "create":
		text := strings.TrimSpace(args.Text)
		if text == "" {
			return "", errors.New("manage_todo 创建待办缺少 text")
		}
		if err := validateTextLen("text", text, maxToolShortText); err != nil {
			return "", err
		}
		t, err := m.todo.Create(text)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("已创建待办 #%d：%s（未完成）", t.ID, t.Text), nil
	case "list":
		return m.listTodos(args.Status, int(args.Page), int(args.PageSize), args.Keyword)
	case "toggle":
		if args.ID <= 0 {
			return "", errors.New("manage_todo 勾选待办缺少有效的 id")
		}
		t, err := m.todo.Toggle(uint(args.ID))
		if err != nil {
			return "", err
		}
		if t.Done {
			return fmt.Sprintf("待办 #%d：%s 已标记为完成", t.ID, t.Text), nil
		}
		return fmt.Sprintf("待办 #%d：%s 已恢复为未完成", t.ID, t.Text), nil
	case "update":
		if args.ID <= 0 {
			return "", errors.New("manage_todo 更新待办缺少有效的 id")
		}
		text := strings.TrimSpace(args.Text)
		if text == "" {
			return "", errors.New("manage_todo 更新待办缺少 text")
		}
		if err := validateTextLen("text", text, maxToolShortText); err != nil {
			return "", err
		}
		t, err := m.todo.Update(uint(args.ID), text)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("已更新待办 #%d：%s", t.ID, t.Text), nil
	}
	return "", fmt.Errorf("manage_todo 未知 action: %s", args.Action)
}

// listTodos 分页列出待办：status 缺省 active；page/pageSize 分页（pageSize 缺省 10、上限 50）；
// keyword 非空时按内容关键字过滤（搜索结果同样分页），定位特定待办时优先用 keyword 而非翻页。
// 过滤条件下沉到 DB 层（TodoService.ListPaged / Search），只加载当前页条目；
// 统计行中的"未完成 x / 已完成 y"用全量计数（CountUnfinished / CountCompleted），避免过滤后失真；
// 有 keyword 时统计行省略全量 x/y（与搜索命中数语义冲突），只展示命中总数。
func (m *manageTodoTool) listTodos(status string, page, pageSize int, keyword string) (string, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = "active"
	}
	switch status {
	case "active", "done", "all":
	default:
		return "", fmt.Errorf("manage_todo 参数非法 status: %s", status)
	}
	keyword = strings.TrimSpace(keyword)
	if err := validateTextLen("keyword", keyword, maxToolShortText); err != nil {
		return "", err
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

	// 全量统计：未完成 / 已完成计数（统计行使用全量数据，避免过滤后失真）
	activeTotal, err := m.todo.CountUnfinished()
	if err != nil {
		return "", err
	}
	doneTotal, err := m.todo.CountCompleted()
	if err != nil {
		return "", err
	}

	// 按 status 映射过滤条件，过滤与分页都在 DB 层完成
	var done *bool
	switch status {
	case "active":
		f := false
		done = &f
	case "done":
		t := true
		done = &t
	}

	var (
		todos []models.Todo
		total int64
	)
	if keyword != "" {
		todos, total, err = m.todo.Search(keyword, done, page, pageSize)
	} else {
		todos, total, err = m.todo.ListPaged(done, page, pageSize)
	}
	if err != nil {
		return "", err
	}
	if total == 0 {
		if keyword != "" {
			return fmt.Sprintf("没有找到包含「%s」的待办", keyword), nil
		}
		switch status {
		case "done":
			return "当前没有已完成的待办", nil
		case "all":
			return "当前没有任何待办", nil
		default:
			return "当前没有待办", nil
		}
	}

	totalPages := (int(total) + pageSize - 1) / pageSize
	if page > totalPages {
		if keyword != "" {
			return fmt.Sprintf("找到包含「%s」的待办共 %d 条，共 %d 页，请求的第 %d 页超出范围，请从第 1 页开始查看",
				keyword, total, totalPages, page), nil
		}
		return fmt.Sprintf("当前待办共 %d 条（未完成 %d / 已完成 %d），共 %d 页，请求的第 %d 页超出范围，请从第 1 页开始查看",
			total, activeTotal, doneTotal, totalPages, page), nil
	}

	var b strings.Builder
	if keyword != "" {
		fmt.Fprintf(&b, "找到包含「%s」的待办 %d 条，第 %d/%d 页，本页 %d 条：\n", keyword, total, page, totalPages, len(todos))
	} else {
		switch status {
		case "active":
			fmt.Fprintf(&b, "当前待办列表（未完成 %d / 已完成 %d）第 %d/%d 页，本页 %d 条：\n", activeTotal, doneTotal, page, totalPages, len(todos))
		case "done":
			fmt.Fprintf(&b, "当前待办列表（已完成 %d / 未完成 %d）第 %d/%d 页，本页 %d 条：\n", doneTotal, activeTotal, page, totalPages, len(todos))
		default:
			fmt.Fprintf(&b, "当前待办列表（共 %d 条，未完成 %d / 已完成 %d）第 %d/%d 页，本页 %d 条：\n", total, activeTotal, doneTotal, page, totalPages, len(todos))
		}
	}
	for i := range todos {
		t := todos[i]
		state := "未完成"
		if t.Done {
			state = "已完成"
		}
		fmt.Fprintf(&b, "[%d] %s（%s）· 创建时间 %s\n", t.ID, t.Text, state, t.CreatedAt.Format("2006-01-02 15:04"))
	}
	if rest := total - int64(page*pageSize); rest > 0 {
		fmt.Fprintf(&b, "还有 %d 条未展示，如需继续查看可要求查看第 %d 页。", rest, page+1)
	}
	return b.String(), nil
}

// NewManageTodo 创建待办管理工具。
func NewManageTodo(todo *services.TodoService, ctx *Context) tool.InvokableTool {
	return &manageTodoTool{todo: todo, ctx: ctx}
}
