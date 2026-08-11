package tools

// 本文件实现 manage_notebook 笔记本管理工具：模型在 ReAct 循环中调用它创建笔记本、
// 重命名笔记本或列出笔记本，底层复用 services.NotebookService
// （Create / Update / ListPaged / Search），不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分三个动作：
//   - create：创建笔记本（name 必填）；
//   - rename：重命名笔记本（id 必填、正整数，来自列表中的 [数字] 编号，以及 name 新名称）；
//   - list：列出笔记本（keyword 按名称关键字过滤；page 页码从 1 开始，pageSize 每页条数，
//     缺省 10、上限 50），返回"共 n 个、第 x/y 页"，列表只展示当前页条目；当页未展示完时提示可翻页。

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

// manageNotebookTool 笔记本管理工具。
type manageNotebookTool struct {
	notebook *services.NotebookService // 笔记本服务（创建 / 重命名 / 列出）
	ctx      *Context                  // 日志输出
}

// 编译期断言：确保 manageNotebookTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*manageNotebookTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 按 action 参数映射动作文案，解析失败回退空串（前端回退"执行"）。
func (m *manageNotebookTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	switch args.Action {
	case "create":
		return "创建笔记本"
	case "rename":
		return "重命名笔记本"
	case "list":
		return "列出笔记本"
	default:
		return "执行"
	}
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (m *manageNotebookTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_notebook",
		Desc: "管理笔记本。当用户要求创建笔记本、重命名笔记本或查看笔记本列表时调用。通过 action 参数区分动作：create=创建笔记本（需提供 name 笔记本名称）；rename=重命名笔记本（需提供 id 笔记本编号，列表中的 [数字] 即为 id，以及 name 新名称）；list=列出笔记本（可用 keyword 按名称关键字过滤，定位特定笔记本时优先用 keyword 而非翻页；可用 page 页码与 pageSize 每页条数分页查看，pageSize 缺省 10、上限 50）。返回笔记本列表或操作结果，列表中的编号 [数字] 可用于后续 rename。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的动作：create=创建笔记本；rename=重命名笔记本；list=列出笔记本",
				Enum:     []string{"create", "rename", "list"},
				Required: true,
			},
			"name": {
				Type:     schema.String,
				Desc:     "笔记本名称，action=create 或 rename 时必填",
				Required: false,
			},
			"id": {
				Type:     schema.Number,
				Desc:     "笔记本编号（正整数，列表中的 [数字] 即为 id），action=rename 时必填",
				Required: false,
			},
			"keyword": {
				Type:     schema.String,
				Desc:     "笔记本名称关键字过滤，仅 action=list 时使用，缺省不过滤",
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

// InvokableRun 执行工具：解析参数 → 校验 action → 按动作分发到 Create / Update / ListPaged。
func (m *manageNotebookTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action   string  `json:"action"`
		Name     string  `json:"name"`
		ID       float64 `json:"id"`
		Keyword  string  `json:"keyword"`
		Page     float64 `json:"page"`
		PageSize float64 `json:"pageSize"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 manage_notebook 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)
	switch args.Action {
	case "create", "rename", "list":
	default:
		return "", fmt.Errorf("manage_notebook 参数缺少/非法 action: %s", args.Action)
	}

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if m.ctx != nil && m.ctx.Logger != nil {
		m.ctx.Logger.Debugw("Agent manage_notebook 调用",
			fastlog.String("action", args.Action),
			fastlog.String("name", args.Name),
			fastlog.Int("id", int(args.ID)),
			fastlog.String("keyword", args.Keyword))
	}

	switch args.Action {
	case "create":
		name := strings.TrimSpace(args.Name)
		if name == "" {
			return "", errors.New("manage_notebook 创建笔记本缺少 name")
		}
		nb, err := m.notebook.Create(name)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("已创建笔记本 #%d：%s", nb.ID, nb.Name), nil
	case "rename":
		if args.ID <= 0 {
			return "", errors.New("manage_notebook 重命名笔记本缺少有效的 id")
		}
		name := strings.TrimSpace(args.Name)
		if name == "" {
			return "", errors.New("manage_notebook 重命名笔记本缺少 name")
		}
		nb, err := m.notebook.Update(uint(args.ID), name)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("已重命名笔记本 #%d 为：%s", nb.ID, nb.Name), nil
	case "list":
		return m.listNotebooks(int(args.Page), int(args.PageSize), args.Keyword)
	}
	return "", fmt.Errorf("manage_notebook 未知 action: %s", args.Action)
}

// listNotebooks 分页列出笔记本：page/pageSize 分页（pageSize 缺省 10、上限 50）；
// keyword 非空时按名称关键字过滤（搜索结果同样分页），定位特定笔记本时优先用 keyword 而非翻页。
// 过滤与分页都在 DB 层完成（NotebookService.ListPaged / Search），只加载当前页条目。
func (m *manageNotebookTool) listNotebooks(page, pageSize int, keyword string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	var (
		notebooks []models.Notebook
		total     int64
		err       error
	)
	if keyword != "" {
		notebooks, total, err = m.notebook.Search(keyword, page, pageSize)
	} else {
		notebooks, total, err = m.notebook.ListPaged(page, pageSize)
	}
	if err != nil {
		return "", err
	}
	if total == 0 {
		if keyword != "" {
			return fmt.Sprintf("没有找到包含「%s」的笔记本", keyword), nil
		}
		return "当前没有任何笔记本", nil
	}

	totalPages := (int(total) + pageSize - 1) / pageSize
	if page > totalPages {
		if keyword != "" {
			return fmt.Sprintf("找到包含「%s」的笔记本共 %d 个，共 %d 页，请求的第 %d 页超出范围，请从第 1 页开始查看",
				keyword, total, totalPages, page), nil
		}
		return fmt.Sprintf("当前共有 %d 个笔记本，共 %d 页，请求的第 %d 页超出范围，请从第 1 页开始查看",
			total, totalPages, page), nil
	}

	var b strings.Builder
	if keyword != "" {
		fmt.Fprintf(&b, "找到包含「%s」的笔记本 %d 个，第 %d/%d 页，本页 %d 条：\n", keyword, total, page, totalPages, len(notebooks))
	} else {
		fmt.Fprintf(&b, "当前笔记本列表（共 %d 个）第 %d/%d 页，本页 %d 条：\n", total, page, totalPages, len(notebooks))
	}
	for i := range notebooks {
		n := notebooks[i]
		fmt.Fprintf(&b, "[%d] %s · 创建时间 %s\n", n.ID, n.Name, n.CreatedAt.Format("2006-01-02 15:04"))
	}
	if rest := total - int64(page*pageSize); rest > 0 {
		fmt.Fprintf(&b, "还有 %d 个未展示，如需继续查看可要求查看第 %d 页。", rest, page+1)
	}
	return b.String(), nil
}

// NewManageNotebook 创建笔记本管理工具。
func NewManageNotebook(notebook *services.NotebookService, ctx *Context) tool.InvokableTool {
	return &manageNotebookTool{notebook: notebook, ctx: ctx}
}
