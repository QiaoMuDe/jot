package tools

// 本文件实现 recall_notes 本地笔记向量召回工具（迁移自父包旧 tools.go），
// 通过注入的 Context 收集召回卡片，不感知父包 agent 的事件循环细节。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/einocli"
	"jot/internal/services"
)

// recallNotesTool 本地笔记向量召回工具。
// 调用 vectorService.VectorRecall（向量 + 关键词混合检索），
// embedding client 由注入的 GetEmbedConfig 工厂按需创建（与 app.go 卡片召回逻辑一致）。
// notebookIDs 在构造工具时从 Request.RecallNotebookIDs 绑定，为空时不限定笔记本。
type recallNotesTool struct {
	vector         *services.VectorService
	setting        *services.SettingService
	getEmbedConfig func() (baseURL, apiKey, model string, err error)
	notebookIDs    []uint
	ctx            *Context // 结构化召回卡片收集（父包 Run 中创建并传入）
}

// 编译期断言：确保 recallNotesTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*recallNotesTool)(nil)

// Info 返回工具元信息。
func (r *recallNotesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "recall_notes",
		Desc: "从用户本地笔记库中检索与问题相关的笔记片段（向量 + 关键词混合检索）。当问题涉及用户自己的笔记、文档、知识库内容时调用，优先于联网搜索。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "检索关键词或问题描述，应与用户意图高度相关",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行向量召回：解析参数 → 创建 embedding client → 调用 VectorRecall → 返回格式化文本。
func (r *recallNotesTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 recall_notes 参数失败: %w", err)
	}
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" {
		return "", errors.New("recall_notes 参数缺少 query")
	}

	// 召回条数复用 ai_card_recall_limit 设置（默认 5，≤30）
	limit := 5
	if r.setting != nil {
		if val := r.setting.Get("ai_card_recall_limit"); val != "" {
			if n, err := strconv.Atoi(val); err == nil && n > 0 && n <= 30 {
				limit = n
			}
		}
	}

	// 构建 embedding client（ai_embed_* 四键，apiKey 为 B64 存储由工厂解码）
	baseURL, apiKey, model, err := r.getEmbedConfig()
	if err != nil {
		return "", fmt.Errorf("读取量化连接配置失败: %w", err)
	}
	if baseURL == "" || apiKey == "" || model == "" {
		return "", errors.New("量化连接未配置（API 地址 / API Key / 模型），无法检索本地笔记")
	}
	embedClient := einocli.NewClient(einocli.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	})

	// VectorRecall 返回分类：成功 / 预期跳过（nil,nil）/ 意外错误（nil,err）
	result, err := r.vector.VectorRecall(ctx, args.Query, limit, embedClient, r.notebookIDs...)
	if err != nil {
		return "", fmt.Errorf("本地笔记检索失败: %w", err)
	}
	if result == nil || strings.TrimSpace(result.FormattedText) == "" {
		return "", errors.New("本地笔记库中没有检索到相关内容")
	}
	// 收集结构化召回卡片（供落库 recall_cards，与问答模式格式一致）
	if r.ctx != nil && r.ctx.Collector != nil && len(result.Cards) > 0 {
		r.ctx.Collector.Cards = append(r.ctx.Collector.Cards, result.Cards...)
	}
	return result.FormattedText, nil
}

// NewRecallNotes 创建本地笔记向量召回工具。notebookIDs 为空时不限定笔记本。
func NewRecallNotes(vector *services.VectorService, setting *services.SettingService, getEmbedConfig func() (baseURL, apiKey, model string, err error), notebookIDs []uint, ctx *Context) tool.InvokableTool {
	return &recallNotesTool{vector: vector, setting: setting, getEmbedConfig: getEmbedConfig, notebookIDs: notebookIDs, ctx: ctx}
}
