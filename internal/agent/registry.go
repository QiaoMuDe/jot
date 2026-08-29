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

// planOnlyTools 仅在 Plan 模式下注册的工具名集合（Agent 模式跳过）。
var planOnlyTools = map[string]bool{
	"update_plan": true,
}

// planExecExcluded Plan 模式执行阶段排除的工具名集合。
// create_plan 在预规划阶段已通过 generatePlan() 完成，执行阶段不再需要。
var planExecExcluded = map[string]bool{
	"create_plan": true,
}

// buildTools 统一装配 Agent 工具，并按 disabled 和 planMode 过滤。
// planMode=false 时不注册 planOnlyTools 中的工具（create_plan/update_plan）。
// 新增工具：1) 在 tools/ 子包新增工具文件与导出构造器；2) 在此追加一行注册；
// 3) 在 tools/meta.go 的 BuiltinTools 追加展示文案（名称须与注册名一致）；
// 无需改动 Run() 的事件消费逻辑。
func buildTools(p BuildParams, disabled map[string]bool, planMode bool) []tool.BaseTool {
	type namedTool struct {
		name string
		t    tool.BaseTool
	}
	// 先收集「名字 + 已包装工具」的中间结构，再按 disabled[name] 跳过被禁工具
	all := []namedTool{
		{"read_url", tools.WrapWithError("read_url", tools.NewReadURL(p.deps.Setting, p.ctx), p.ctx)},
		{"recall_notes", tools.WrapWithError("recall_notes", tools.NewRecallNotes(p.deps.Vector, p.deps.Setting, p.deps.GetEmbedConfig, p.req.RecallNotebookIDs, p.ctx), p.ctx)},
		{"get_current_time", tools.WrapWithError("get_current_time", tools.NewGetCurrentTime(), p.ctx)},
		{"json_validate", tools.WrapWithError("json_validate", tools.MustJSONValidate(), p.ctx)},
		{"json_format", tools.WrapWithError("json_format", tools.MustJSONFormat(), p.ctx)},
		{"json_extract", tools.WrapWithError("json_extract", tools.MustJSONExtract(), p.ctx)},
		{"manage_todo", tools.WrapWithError("manage_todo", tools.NewManageTodo(p.deps.Todo, p.ctx), p.ctx)},
		{"manage_notebook", tools.WrapWithError("manage_notebook", tools.NewManageNotebook(p.deps.Notebook, p.ctx), p.ctx)},
		{"manage_tag", tools.WrapWithError("manage_tag", tools.NewManageTag(p.deps.Tag, p.ctx), p.ctx)},
		{"manage_note", tools.WrapWithError("manage_note", tools.NewManageNote(p.deps.Note, p.deps.Tag, p.deps.Setting, p.ctx), p.ctx)},
		{"read_note_section", tools.WrapWithError("read_note_section", tools.NewReadNoteSection(p.deps.Note, p.deps.Setting, p.ctx), p.ctx)},
		{"get_stats", tools.WrapWithError("get_stats", tools.NewGetStats(p.deps.Stats, p.deps.Vector, p.ctx), p.ctx)},
		{"ask_user", tools.WrapWithError("ask_user", tools.NewAskUser(p.ctx), p.ctx)},
		{"create_plan", tools.WrapWithError("create_plan", tools.NewCreatePlan(p.ctx), p.ctx)},
		{"update_plan", tools.WrapWithError("update_plan", tools.NewUpdatePlan(p.ctx), p.ctx)},
	}
	filtered := make([]tool.BaseTool, 0, len(all))
	for _, n := range all {
		if disabled[n.name] {
			continue
		}
		// Agent 模式下跳过仅 Plan 模式可用的工具
		if !planMode && planOnlyTools[n.name] {
			continue
		}
		// Plan 模式执行阶段跳过预规划阶段已使用的工具（create_plan）
		if planMode && planExecExcluded[n.name] {
			continue
		}
		filtered = append(filtered, n.t)
	}
	return filtered
}
