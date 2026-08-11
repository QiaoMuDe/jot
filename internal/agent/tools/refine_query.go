package tools

// 本文件实现 refine_search_query 搜索词精炼工具（迁移自父包旧 tools.go），
// 复用 services.RefineSearchQuery，由模型在 ReAct 循环中自主判断是否需要精炼。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/services"
)

// refineSearchQueryTool 搜索词精炼工具。
// 复用 services.RefineSearchQuery（与问答模式搜索词精炼一致），由模型在 ReAct 循环中
// 自主判断是否需要精炼：当用户输入口语化、含义模糊或包含多个话题时，先调用本工具精炼，
// 再用精炼后的关键词调用 web_search 搜索。精炼失败/无变化时降级返回原词，不中断循环。
type refineSearchQueryTool struct {
	ai *services.AIService // 用于 RefineSearchQuery（精炼走当前配置的 AI 模型）
}

// 编译期断言：确保 refineSearchQueryTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*refineSearchQueryTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：固定文案，无需解析参数。
func (r *refineSearchQueryTool) ActionText(_ string) string {
	return "精炼搜索关键词"
}

// Info 返回工具元信息。
func (r *refineSearchQueryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "refine_search_query",
		Desc: "将用户口语化、模糊的搜索意图精炼为简洁搜索关键词。当 web_search 的查询词是口语化表达、含义模糊或包含多个话题时，先调用本工具精炼，再用精炼后的关键词调用 web_search 进行搜索。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "用户原始搜索意图，可为口语化句子",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行精炼：解析 query → RefineSearchQuery → 返回精炼词。
// 用户停止时返回取消错误终止循环；精炼失败/无变化时降级返回原词，让模型继续搜索。
func (r *refineSearchQueryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 refine_search_query 参数失败: %w", err)
	}
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" {
		return "", errors.New("refine_search_query 参数缺少 query")
	}

	refined, err := services.RefineSearchQuery(ctx, args.Query, r.ai)
	if err != nil {
		// 用户停止：返回取消错误，终止 Agent 循环
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// 精炼失败：降级返回原词，模型仍可继续搜索
		return args.Query, nil
	}
	refined = strings.TrimSpace(refined)
	if refined == "" || refined == args.Query {
		return args.Query, nil
	}
	return refined, nil
}

// NewRefineSearchQuery 创建搜索词精炼工具。
func NewRefineSearchQuery(ai *services.AIService) tool.InvokableTool {
	return &refineSearchQueryTool{ai: ai}
}
