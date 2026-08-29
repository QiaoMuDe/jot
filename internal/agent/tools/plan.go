package tools

// 本文件实现 create_plan 和 update_plan 两个规划工具：
// 模型在 ReAct 循环中通过这两个工具实现显式 Planning——执行前先输出结构化计划，
// 执行中动态调整。计划状态存储在 Context.PlanState 中，跨轮次共享；
// 父包通过 GenModelInputFunc 钩子在每轮 LLM 调用前注入当前计划状态到系统提示词。
//
// create_plan：模型在开始执行前调用，输出目标和步骤列表，构造 Plan 存入 PlanState，
// 发射 ai:plan-created 事件供前端渲染计划卡片。
// update_plan：模型在执行过程中调用，标记步骤完成/跳过或新增步骤，更新 PlanState，
// 发射 ai:plan-updated 事件供前端刷新计划卡片。
//
// 两个工具均为 ctx.Emit 的允许例外（与 ask_user 并列），直接发射计划卡片事件。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// maxPlanSteps 计划步骤上限：防止模型输出过长计划浪费 token。
const maxPlanSteps = 10

// ──────────────────────────────────────────────────────────────────────────────
// create_plan
// ──────────────────────────────────────────────────────────────────────────────

// createPlanTool 规划工具：模型在开始执行前调用，输出结构化计划。
type createPlanTool struct {
	ctx *Context
}

var _ tool.InvokableTool = (*createPlanTool)(nil)
var _ ActionTextProvider = (*createPlanTool)(nil)

// ActionText 提供 tool_start 动作文案。
func (t *createPlanTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Goal string `json:"goal"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "创建执行计划"
	}
	goal := strings.TrimSpace(args.Goal)
	if goal == "" {
		return "创建执行计划"
	}
	return "创建执行计划：" + TruncateRunes(goal, 20)
}

// Info 返回工具元信息。
func (t *createPlanTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "create_plan",
		Desc: "在开始执行用户请求前，调用本工具制定执行计划。将目标拆解为可执行的步骤列表，" +
			"模型将按计划逐步调用其他工具完成任务。收到任何用户请求后都应首先调用本工具。" +
			"计划步骤数 ≤ 10，每步 description 简洁明确。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"goal": {
				Type:     schema.String,
				Desc:     "计划目标描述（简洁一句话）",
				Required: true,
			},
			"steps": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"description": {
							Type:     schema.String,
							Desc:     "步骤描述",
							Required: true,
						},
						"tool_name": {
							Type:     schema.String,
							Desc:     "预计调用的工具名（可选，不填则由模型自主决定）",
							Required: false,
						},
					},
				},
				Desc:     "步骤列表（≤ 10 步），每项含 description 和可选的 tool_name",
				Required: true,
			},
		}),
	}, nil
}

// ParseCreatePlanArgs 解析 create_plan 工具的参数 JSON，校验并构造 Plan。
// 供 createPlanTool.InvokableRun（ReAct 循环内）和 agent.generatePlan（预规划阶段）复用。
func ParseCreatePlanArgs(argumentsInJSON string) (*Plan, error) {
	var args struct {
		Goal  string `json:"goal"`
		Steps []struct {
			Description string `json:"description"`
			ToolName    string `json:"tool_name"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return nil, fmt.Errorf("解析 create_plan 参数失败: %w", err)
	}

	goal := strings.TrimSpace(args.Goal)
	if goal == "" {
		return nil, errors.New("create_plan 参数缺少 goal")
	}
	if len(args.Steps) == 0 {
		return nil, errors.New("create_plan 参数缺少 steps（至少需要 1 个步骤）")
	}
	if len(args.Steps) > maxPlanSteps {
		return nil, fmt.Errorf("create_plan 步骤过多（%d 步，上限 %d），请精简计划", len(args.Steps), maxPlanSteps)
	}

	steps := make([]PlanStep, len(args.Steps))
	for i, s := range args.Steps {
		desc := strings.TrimSpace(s.Description)
		if desc == "" {
			return nil, fmt.Errorf("create_plan 第 %d 步 description 不能为空", i+1)
		}
		steps[i] = PlanStep{
			ID:          i + 1,
			Description: desc,
			ToolName:    strings.TrimSpace(s.ToolName),
			Status:      "pending",
		}
	}

	return &Plan{
		Goal:    goal,
		Steps:   steps,
		Current: 0,
	}, nil
}

// InvokableRun 执行工具：解析参数 → 校验 → 构造 Plan → 存入 PlanState → 发射事件 → 返回确认文本。
func (t *createPlanTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	plan, err := ParseCreatePlanArgs(argumentsInJSON)
	if err != nil {
		return "", err
	}

	// 存入 PlanState（跨 ReAct 轮次共享）
	t.ctx.PlanState = plan

	// 发射 ai:plan-created 事件（前端渲染计划卡片）
	t.emitPlanCreated()

	return fmt.Sprintf("计划已创建，共 %d 步。请开始执行第 1 步：%s。", len(plan.Steps), plan.Steps[0].Description), nil
}

// emitPlanCreated 发射 ai:plan-created 事件。
func (t *createPlanTool) emitPlanCreated() {
	plan := t.ctx.PlanState
	if plan == nil {
		return
	}
	payload := map[string]any{
		"goal":  plan.Goal,
		"steps": plan.Steps,
	}
	if b, err := json.Marshal(payload); err == nil {
		t.ctx.Emit("ai:plan-created", string(b))
	}
}

// NewCreatePlan 创建规划工具。
func NewCreatePlan(ctx *Context) tool.InvokableTool {
	return &createPlanTool{ctx: ctx}
}

// ──────────────────────────────────────────────────────────────────────────────
// update_plan
// ──────────────────────────────────────────────────────────────────────────────

// updatePlanTool 规划工具：模型在执行过程中调整计划。
type updatePlanTool struct {
	ctx *Context
}

var _ tool.InvokableTool = (*updatePlanTool)(nil)
var _ ActionTextProvider = (*updatePlanTool)(nil)

// ActionText 提供 tool_start 动作文案。
func (t *updatePlanTool) ActionText(argumentsInJSON string) string {
	var args struct {
		StepID *int   `json:"step_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "更新计划"
	}
	if args.StepID != nil {
		return fmt.Sprintf("更新计划：步骤 %d → %s", *args.StepID, args.Status)
	}
	return "更新计划：新增步骤"
}

// Info 返回工具元信息。
func (t *updatePlanTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_plan",
		Desc: "在执行计划过程中调整计划状态。可标记步骤为 in_progress/done/skipped，" +
			"或在 step_id 为 null 时新增步骤。每完成或跳过一个步骤后应调用本工具更新计划。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"step_id": {
				Type:     schema.Number,
				Desc:     "要更新的步骤编号（1-based）；为 null 时表示新增步骤",
				Required: false,
			},
			"status": {
				Type:     schema.String,
				Desc:     "更新后的状态",
				Enum:     []string{"in_progress", "done", "skipped"},
				Required: true,
			},
			"result": {
				Type:     schema.String,
				Desc:     "步骤执行结果摘要（可选，默认空串）",
				Required: false,
			},
			"new_step": {
				Type: schema.Object,
				SubParams: map[string]*schema.ParameterInfo{
					"description": {
						Type:     schema.String,
						Desc:     "新步骤描述",
						Required: true,
					},
					"tool_name": {
						Type:     schema.String,
						Desc:     "预计调用的工具名（可选）",
						Required: false,
					},
				},
				Desc:     "新增步骤（step_id 为 null 时必填）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：更新计划状态 → 发射事件 → 返回确认文本。
func (t *updatePlanTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	plan := t.ctx.PlanState
	if plan == nil {
		return "", errors.New("尚未创建计划，请先调用 create_plan")
	}

	var args struct {
		StepID  *int   `json:"step_id"`
		Status  string `json:"status"`
		Result  string `json:"result"`
		NewStep *struct {
			Description string `json:"description"`
			ToolName    string `json:"tool_name"`
		} `json:"new_step"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 update_plan 参数失败: %w", err)
	}

	// 校验 status
	switch args.Status {
	case "in_progress", "done", "skipped":
	default:
		return "", fmt.Errorf("update_plan status 非法：%q，合法值为 in_progress / done / skipped", args.Status)
	}

	// 新增步骤
	if args.StepID == nil {
		if args.NewStep == nil {
			return "", errors.New("update_plan step_id 为 null 时必须提供 new_step")
		}
		desc := strings.TrimSpace(args.NewStep.Description)
		if desc == "" {
			return "", errors.New("update_plan new_step.description 不能为空")
		}
		newStep := PlanStep{
			ID:          len(plan.Steps) + 1,
			Description: desc,
			ToolName:    strings.TrimSpace(args.NewStep.ToolName),
			Status:      args.Status,
		}
		plan.Steps = append(plan.Steps, newStep)

		t.emitPlanUpdated(nil, args.Status, "")
		return fmt.Sprintf("已新增步骤 %d：%s。当前共 %d 步。", newStep.ID, newStep.Description, len(plan.Steps)), nil
	}

	// 更新已有步骤
	stepID := *args.StepID
	if stepID < 1 || stepID > len(plan.Steps) {
		return "", fmt.Errorf("update_plan step_id %d 超出范围（1-%d）", stepID, len(plan.Steps))
	}

	step := &plan.Steps[stepID-1]
	step.Status = args.Status
	step.Result = strings.TrimSpace(args.Result)

	// 模型主动调用了 update_plan，不再需要催促提醒
	t.ctx.SkippedPlanUpdate = false

	// 如果标记当前步骤为 done，推进 Current 指针
	if args.Status == "done" && stepID == plan.Current+1 {
		plan.Current = stepID
	}

	t.emitPlanUpdated(&stepID, args.Status, step.Result)

	statusCN := map[string]string{"done": "完成", "skipped": "跳过", "in_progress": "进行中"}
	return fmt.Sprintf("步骤 %d 已%s。当前进度：%d/%d。", stepID, statusCN[args.Status], plan.Current+1, len(plan.Steps)), nil
}

// emitPlanUpdated 发射 ai:plan-updated 事件，携带变更的步骤信息和完整步骤快照。
func (t *updatePlanTool) emitPlanUpdated(stepID *int, status, result string) {
	plan := t.ctx.PlanState
	if plan == nil {
		return
	}
	payload := map[string]any{
		"step_id": stepID,
		"status":  status,
		"result":  result,
		"steps":   plan.Steps,
	}
	if b, err := json.Marshal(payload); err == nil {
		t.ctx.Emit("ai:plan-updated", string(b))
	}
}

// NewUpdatePlan 创建计划更新工具。
func NewUpdatePlan(ctx *Context) tool.InvokableTool {
	return &updatePlanTool{ctx: ctx}
}
