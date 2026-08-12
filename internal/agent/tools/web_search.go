package tools

// 本文件实现 web_search 多源联网搜索工具（迁移自父包旧 tools.go），
// 通过注入的 Context 收集结构化来源、登记部分失败提示并输出日志，
// 不感知父包 agent 的事件循环细节。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
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
	ai      *services.AIService      // 读取 AI 配置（Tavily API Key / 知乎 Access Secret）
	setting *services.SettingService // 读取搜索结果数 / 最大字符数设置
	ctx     *Context                 // 事件发射、日志、结构化收集与部分失败登记
}

// 编译期断言：确保 webSearchTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*webSearchTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// query 非空显示具体搜索词（截断防超长），否则回退通用文案。
func (w *webSearchTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	if args.Query = strings.TrimSpace(args.Query); args.Query != "" {
		return "搜索 " + TruncateRunes(args.Query, 30)
	}
	return "搜索互联网"
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (w *webSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "搜索互联网获取实时信息。当用户询问新闻、天气、最新动态、人物、事件等需要联网获取实时或外部信息的问题时调用；当本地笔记检索（recall_notes）未返回足够内容时，也应调用本工具补充信息。可根据问题性质自主选择搜索来源（sources 参数，可选）：tavily 为通用互联网搜索，适合时事、新闻、一般性问题；zhihu_search 为知乎站内搜索，适合知乎相关话题、经验观点类问题；zhihu_global 为知乎全网搜索，可在需要全网信息时补充检索。需要多个来源时可传入多个值；不传或传空则搜索全部可用来源。如果查询词是口语化表达、含义模糊或包含多个话题，请先调用 refine_search_query 工具精炼，再使用精炼后的关键词作为本工具的 query。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "搜索关键词，应简洁明确",
				Required: true,
			},
			"sources": {
				Type:     schema.Array,
				ElemInfo: &schema.ParameterInfo{Type: schema.String, Enum: []string{searchSourceTavily, searchSourceZhihu, searchSourceZhihuGlobal}},
				Desc:     "要搜索的来源列表，可选。tavily=通用互联网搜索；zhihu_search=知乎站内内容；zhihu_global=知乎全网搜索。按问题性质选择一个或多个来源；不传或传空则搜索全部可用来源",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行搜索：模型按 sources 参数选源（未传则全部）→ 探测可用来源（未配置密钥的源不发请求）→ 并发调用可用源 → 按 URL 去重并按来源分组汇总结果。
func (w *webSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query   string   `json:"query"`
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 web_search 参数失败: %w", err)
	}
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" {
		return "", errors.New("web_search 参数缺少 query")
	}

	// 来源选择：剔除非法枚举值并去重；模型未指定（空）时回退全部来源
	sources := selectSources(args.Sources)

	cfg := w.ai.GetConfig()

	// 1. 探测来源可用性：未配置密钥的源直接标记不可用，不发请求；
	//    已配置的源进入 active 并发执行
	var failedParts []string
	var active []string
	for _, src := range sources {
		switch src {
		case searchSourceTavily:
			if cfg.TavilyAPIKey == "" {
				failedParts = append(failedParts, src+"（Tavily API Key 未配置）")
			} else {
				active = append(active, src)
			}
		case searchSourceZhihu, searchSourceZhihuGlobal:
			if cfg.ZhihuAccessSecret == "" {
				failedParts = append(failedParts, src+"（知乎 Access Secret 未配置）")
			} else {
				active = append(active, src)
			}
		}
	}

	if w.ctx != nil && w.ctx.Logger != nil {
		w.ctx.Logger.Debugw("Agent web_search 调用",
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

	// 3. 合并结果：先跨来源按 URL 去重，再按来源分组格式化返回
	var merged []services.SearchSource
	seen := make(map[string]struct{})
	for i := 0; i < len(active); i++ {
		r := <-resultCh
		if r.err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			if w.ctx != nil && w.ctx.Logger != nil {
				w.ctx.Logger.Debugw("Agent web_search 单源失败，跳过",
					fastlog.String("source", r.source), fastlog.Error(r.err))
			}
			// 记录失败源：供前端"部分来源失败"提示，并让模型感知哪些源不可用
			failedParts = append(failedParts, r.source+"（"+r.err.Error()+"）")
			continue
		}
		if r.result == nil {
			continue
		}
		// 跨来源按 URL 去重（空白 URL 直接丢弃），保留首次出现顺序
		for _, s := range r.result.Sources {
			u := strings.TrimSpace(s.URL)
			if u == "" {
				continue
			}
			if _, dup := seen[u]; dup {
				continue
			}
			seen[u] = struct{}{}
			merged = append(merged, s)
		}
	}

	if len(merged) == 0 {
		// 全部来源失败（配置了但请求失败 / 无可信结果）：错误信息带上具体原因
		if len(failedParts) > 0 {
			return "", errors.New("所有搜索来源均不可用：" + strings.Join(failedParts, "、"))
		}
		return "", errors.New("搜索未返回结果（可能搜索服务未配置或无可信结果）")
	}
	// 收集去重后的结构化来源（供落库 search_sources，与问答模式格式一致）
	if w.ctx != nil && w.ctx.Collector != nil {
		w.ctx.Collector.Sources = append(w.ctx.Collector.Sources, merged...)
	}

	// 按来源分组生成格式化文本（与前端来源列表一一对应）
	text := formatDedupResults(args.Query, merged)
	// 部分来源失败：结果文本中成功内容在前、失败说明在后，
	// 避免模型把"来源失败"误读为整体失败而忽略实际可用的搜索结果
	if len(failedParts) > 0 {
		if w.ctx != nil && w.ctx.Logger != nil {
			w.ctx.Logger.Debugw("Agent web_search 部分来源失败",
				fastlog.Int("failed_sources", len(failedParts)),
				fastlog.String("failed_list", strings.Join(failedParts, "、")))
		}
		if w.ctx != nil {
			w.ctx.AddPartial(strings.Join(failedParts, "、"))
		}
		suffix := "\n\n注意：以下搜索来源执行失败：" + strings.Join(failedParts, "、") + "。以上为其余可用来源的结果，请基于这些结果回答。"
		return text + suffix, nil
	}
	return text, nil
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

// selectSources 清洗模型传入的来源列表：剔除非法枚举值、按字符串去重。
// 为空（模型未指定或全部非法）时回退默认全部来源。
func selectSources(args []string) []string {
	selected := make([]string, 0, len(defaultSearchSources))
	seen := make(map[string]struct{})
	for _, s := range args {
		s = strings.TrimSpace(s)
		switch s {
		case searchSourceTavily, searchSourceZhihu, searchSourceZhihuGlobal:
		default:
			continue // 非法枚举值直接忽略
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		selected = append(selected, s)
	}
	if len(selected) == 0 {
		return append([]string(nil), defaultSearchSources...)
	}
	return selected
}

// formatDedupResults 将去重后的来源条目按来源分组，生成统一格式化文本。
// 分组顺序固定为 tavily → zhihu_search → zhihu_global（与探测顺序一致），
// 条目序号全局连续（与"共 N 条"对应），模型可按组读取各来源结果。
func formatDedupResults(query string, sources []services.SearchSource) string {
	// 按来源分组（预置固定顺序的组）
	grouped := make(map[string][]services.SearchSource, len(defaultSearchSources))
	for _, src := range defaultSearchSources {
		grouped[src] = nil
	}
	for _, s := range sources {
		if _, ok := grouped[s.SourceLabel]; ok {
			grouped[s.SourceLabel] = append(grouped[s.SourceLabel], s)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "以下是为“%s”搜索到的相关内容（已按 URL 去重，共 %d 条）：\n\n", query, len(sources))
	idx := 1
	for _, src := range defaultSearchSources {
		items := grouped[src]
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "【%s】\n", sourceLabelName(src))
		for _, s := range items {
			fmt.Fprintf(&b, "[%d] %s\n", idx, s.Title)
			fmt.Fprintf(&b, "    来源: %s\n", s.URL)
			if s.Content != "" {
				fmt.Fprintf(&b, "    内容: %s\n", s.Content)
			}
			b.WriteString("\n")
			idx++
		}
	}
	return b.String()
}

// sourceLabelName 将来源标识转为中文展示名（与 services 层各源标题一致）。
func sourceLabelName(label string) string {
	switch label {
	case searchSourceTavily:
		return "Tavily 联网搜索"
	case searchSourceZhihu:
		return "知乎站内搜索"
	case searchSourceZhihuGlobal:
		return "知乎全网搜索"
	default:
		return label
	}
}

// NewWebSearch 创建多源联网搜索工具。
func NewWebSearch(ai *services.AIService, setting *services.SettingService, ctx *Context) tool.InvokableTool {
	return &webSearchTool{ai: ai, setting: setting, ctx: ctx}
}
