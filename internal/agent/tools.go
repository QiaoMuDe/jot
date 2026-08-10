package agent

// 本文件实现 Agent 的两个只读工具：web_search（多源联网搜索）与 recall_notes（本地笔记向量召回）。
// 两个工具都完整实现 tool.InvokableTool（内嵌 BaseTool + InvokableRun），
// 由 Eino 的 ToolsNode 在 ReAct 循环中按需调用，参数为模型生成的 JSON 字符串。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"gitee.com/MM-Q/fastlog"
	"jot/internal/aicli"
	"jot/internal/services"
)

// 搜索源标识，与 app.go 中 CallAIStream 的多源搜索一致。
const (
	searchSourceTavily      = "tavily"       // Tavily 联网搜索
	searchSourceZhihu       = "zhihu_search" // 知乎站内搜索
	searchSourceZhihuGlobal = "zhihu_global" // 知乎全网搜索
)

// defaultSearchSources sources 为空时的默认全部来源。
var defaultSearchSources = []string{searchSourceTavily, searchSourceZhihu, searchSourceZhihuGlobal}

// webSearchTool 多源联网搜索工具。
// 内部调用 services 包级搜索函数（与 app.go 多源并发逻辑一致）：
//   - tavily       → services.SearchWeb
//   - zhihu_search → services.SearchZhihuContent
//   - zhihu_global → services.SearchGlobalContent
//
// 每次执行时通过 deps.AI.GetConfig() 读取最新的 Tavily/知乎密钥，保证配置变更即时生效。
type webSearchTool struct {
	ai        *services.AIService      // 读取 AI 配置（Tavily API Key / 知乎 Access Secret）
	setting   *services.SettingService // 读取搜索结果数 / 最大字符数设置
	logger    *fastlog.Logger
	collector *resultCollector // 收集本轮搜索来源（Run 中创建并传入）
}

// 编译期断言：确保 webSearchTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*webSearchTool)(nil)

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (w *webSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "搜索互联网获取实时信息。当用户询问新闻、天气、最新动态、人物、事件等需要联网获取实时或外部信息的问题时调用。会自动并发检索全部可用来源（Tavily 联网搜索、知乎站内搜索、知乎全网搜索），无需也无法指定单个来源。如果查询词是口语化表达、含义模糊或包含多个话题，请先调用 refine_search_query 工具精炼，再使用精炼后的关键词作为本工具的 query。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "搜索关键词，应简洁明确",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行搜索：探测可用来源（未配置密钥的源不发请求）→ 并发调用可用源 → 汇总结果。
func (w *webSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 web_search 参数失败: %w", err)
	}
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" {
		return "", errors.New("web_search 参数缺少 query")
	}

	cfg := w.ai.GetConfig()

	// 1. 探测来源可用性：未配置密钥的源直接标记不可用，不发请求；
	//    已配置的源进入 active 并发执行（Tavily / 知乎站内 / 知乎全网固定全部参与）
	var failedParts []string
	var active []string
	for _, src := range defaultSearchSources {
		switch src {
		case searchSourceTavily:
			if cfg.TavilyAPIKey == "" {
				failedParts = append(failedParts, src+"（Tavily API Key 未配置）")
				if w.collector != nil {
					w.collector.SourceErrors = append(w.collector.SourceErrors, sourceError{Source: src, Err: "Tavily API Key 未配置"})
				}
			} else {
				active = append(active, src)
			}
		case searchSourceZhihu, searchSourceZhihuGlobal:
			if cfg.ZhihuAccessSecret == "" {
				failedParts = append(failedParts, src+"（知乎 Access Secret 未配置）")
				if w.collector != nil {
					w.collector.SourceErrors = append(w.collector.SourceErrors, sourceError{Source: src, Err: "知乎 Access Secret 未配置"})
				}
			} else {
				active = append(active, src)
			}
		}
	}

	if w.logger != nil {
		w.logger.Debugw("Agent web_search 调用",
			fastlog.String("query", args.Query),
			fastlog.String("active_sources", strings.Join(active, ",")),
			fastlog.String("unavailable_sources", strings.Join(failedParts, "、")))
	}

	// 全部来源都未配置：直接返回错误，说明具体原因
	if len(active) == 0 {
		return "", errors.New("搜索服务均未配置：" + strings.Join(failedParts, "、"))
	}

	// 搜索结果数与单条最大字符数（复用现有设置项）
	limit := w.intSetting("ai_search_result_limit", 5, 30)
	maxChars := w.intSetting("ai_web_search_max_chars", 5000, 50000)

	// 2. 并发执行可用来源（与 app.go 的搜索状态机一致），任一源失败/未配置只跳过该源
	type searchResult struct {
		source string
		result *services.SearchWebResult
		err    error
	}
	resultCh := make(chan searchResult, len(active))
	for _, src := range active {
		go func(s string) {
			var r searchResult
			r.source = s
			switch s {
			case searchSourceTavily:
				r.result, r.err = services.SearchWeb(ctx, args.Query, cfg.TavilyAPIKey, limit, maxChars)
			case searchSourceZhihu:
				r.result, r.err = services.SearchZhihuContent(ctx, args.Query, cfg.ZhihuAccessSecret, limit, maxChars)
			case searchSourceZhihuGlobal:
				r.result, r.err = services.SearchGlobalContent(ctx, args.Query, cfg.ZhihuAccessSecret, limit, maxChars)
			default:
				r.err = fmt.Errorf("未知搜索源: %s", s)
			}
			resultCh <- r
		}(src)
	}

	// 3. 收集结果：合并各源格式化文本
	var b strings.Builder
	for i := 0; i < len(active); i++ {
		r := <-resultCh
		if r.err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			if w.logger != nil {
				w.logger.Debugw("Agent web_search 单源失败，跳过",
					fastlog.String("source", r.source), fastlog.Error(r.err))
			}
			// 记录失败源：供前端"部分来源失败"提示，并让模型感知哪些源不可用
			if w.collector != nil {
				w.collector.SourceErrors = append(w.collector.SourceErrors, sourceError{Source: r.source, Err: r.err.Error()})
			}
			failedParts = append(failedParts, r.source+"（"+r.err.Error()+"）")
			continue
		}
		if r.result != nil {
			if strings.TrimSpace(r.result.FormattedText) != "" {
				b.WriteString(r.result.FormattedText)
				b.WriteString("\n")
			}
			// 收集结构化来源（供落库 search_sources，与问答模式格式一致）
			if w.collector != nil && len(r.result.Sources) > 0 {
				w.collector.Sources = append(w.collector.Sources, r.result.Sources...)
			}
		}
	}

	if b.Len() == 0 {
		// 全部来源失败（配置了但请求失败 / 无可信结果）：错误信息带上具体原因
		if len(failedParts) > 0 {
			return "", errors.New("所有搜索来源均不可用：" + strings.Join(failedParts, "、"))
		}
		return "", errors.New("搜索未返回结果（可能搜索服务未配置或无可信结果）")
	}
	// 部分来源失败：结果文本中成功内容在前、失败说明在后，
	// 避免模型把"来源失败"误读为整体失败而忽略实际可用的搜索结果
	if len(failedParts) > 0 {
		if w.logger != nil {
			w.logger.Debugw("Agent web_search 部分来源失败",
				fastlog.Int("failed_sources", len(failedParts)),
				fastlog.String("failed_list", strings.Join(failedParts, "、")))
		}
		suffix := "\n\n注意：以下搜索来源执行失败：" + strings.Join(failedParts, "、") + "。以上为其余可用来源的结果，请基于这些结果回答。"
		return b.String() + suffix, nil
	}
	return b.String(), nil
}

// intSetting 读取 int 类型设置，解析失败或越界时回退默认值。
func (w *webSearchTool) intSetting(key string, def, max int) int {
	if w.setting == nil {
		return def
	}
	val := w.setting.Get(key)
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// recallNotesTool 本地笔记向量召回工具。
// 调用 vectorService.VectorRecall（向量 + 关键词混合检索），
// embedding client 由注入的 GetEmbedConfig 工厂按需创建（与 app.go 卡片召回逻辑一致）。
// notebookIDs 在构造工具时从 Request.RecallNotebookIDs 绑定，为空时不限定笔记本。
type recallNotesTool struct {
	vector         *services.VectorService
	setting        *services.SettingService
	getEmbedConfig func() (baseURL, apiKey, model string, err error)
	notebookIDs    []uint
	logger         *fastlog.Logger
	collector      *resultCollector // 收集本轮召回卡片（Run 中创建并传入）
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
	embedClient := aicli.NewClient(aicli.Config{
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
	if r.collector != nil && len(result.Cards) > 0 {
		r.collector.Cards = append(r.collector.Cards, result.Cards...)
	}
	return result.FormattedText, nil
}

// refineSearchQueryTool 搜索词精炼工具。
// 复用 services.RefineSearchQuery（与问答模式搜索词精炼一致），由模型在 ReAct 循环中
// 自主判断是否需要精炼：当用户输入口语化、含义模糊或包含多个话题时，先调用本工具精炼，
// 再用精炼后的关键词调用 web_search 搜索。精炼失败/无变化时降级返回原词，不中断循环。
type refineSearchQueryTool struct {
	ai *services.AIService // 用于 RefineSearchQuery（精炼走当前配置的 AI 模型）
}

// 编译期断言：确保 refineSearchQueryTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*refineSearchQueryTool)(nil)

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
