package agent

import (
	"github.com/cloudwego/eino/components/tool"

	"jot/internal/agent/tools"
)

// BuildParams 构建工具所需的装配上下文：父包 Deps、本轮 Request 与工具执行上下文。
type BuildParams struct {
	deps Deps
	req  Request
	ctx  *tools.Context
}

// buildTools 统一注册全部 Agent 工具。
// 新增工具：1) 在 tools/ 子包新增工具文件与导出构造器；2) 在此追加一行注册；
// 无需改动 Run() 的事件消费逻辑。
func buildTools(p BuildParams) []tool.BaseTool {
	return []tool.BaseTool{
		tools.WrapWithError("refine_search_query", tools.NewRefineSearchQuery(p.deps.AI), p.ctx),
		tools.WrapWithError("web_search", tools.NewWebSearch(p.deps.AI, p.deps.Setting, p.ctx), p.ctx),
		tools.WrapWithError("recall_notes", tools.NewRecallNotes(p.deps.Vector, p.deps.Setting, p.deps.GetEmbedConfig, p.req.RecallNotebookIDs, p.ctx), p.ctx),
		tools.WrapWithError("get_current_time", tools.NewGetCurrentTime(), p.ctx),
		tools.WrapWithError("manage_todo", tools.NewManageTodo(p.deps.Todo, p.ctx), p.ctx),
		tools.WrapWithError("manage_notebook", tools.NewManageNotebook(p.deps.Notebook, p.ctx), p.ctx),
		tools.WrapWithError("manage_tag", tools.NewManageTag(p.deps.Tag, p.ctx), p.ctx),
		tools.WrapWithError("manage_note", tools.NewManageNote(p.deps.Note, p.deps.Tag, p.deps.Setting, p.ctx), p.ctx),
		tools.WrapWithError("get_stats", tools.NewGetStats(p.deps.Stats, p.deps.Vector, p.ctx), p.ctx),
	}
}
