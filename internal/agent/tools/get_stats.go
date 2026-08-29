package tools

// 本文件实现 get_stats 数据统计工具（只读）：模型在 ReAct 循环中调用它获取应用数据统计概览
// 或某月每日笔记数，底层复用 services.StatsService（GetDataStats / GetMonthCounts，聚合
// 笔记/标签/待办/AI 用量/数据库大小，与数据管理页面口径一致）与
// services.VectorService（GetIndexStatus），不感知父包 agent 的事件循环细节。
// 一个工具通过 action 参数区分两个动作：
//   - overview：数据统计概览（缺省），返回笔记/回收站/置顶/笔记本/标签/待办/密码记录/AI 用量/
//     数据库大小/向量索引状态等总量信息；
//   - month：某月每日笔记数（year 年份与 month 月份缺省时取当前年月）。
// 本工具只读，不修改任何数据；查看具体笔记/待办/标签/笔记本列表请用 manage_note /
// manage_todo / manage_tag / manage_notebook 工具，本工具只回答总量与趋势。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/services"
)

// getStatsTool 数据统计工具（只读）：返回应用数据概览与月度笔记统计。
type getStatsTool struct {
	stats  *services.StatsService  // 数据统计聚合（GetDataStats / GetMonthCounts）
	vector *services.VectorService // 向量索引状态（GetIndexStatus）
	ctx    *Context                // 日志输出
}

// 编译期断言：确保 getStatsTool 实现了 tool.InvokableTool。
var _ tool.InvokableTool = (*getStatsTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 按 action 参数映射动作文案，解析失败回退空串（前端回退"执行"）。
func (g *getStatsTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	switch args.Action {
	case "", "overview":
		return "获取数据统计概览"
	case "month":
		return "获取月度笔记统计"
	default:
		return "执行"
	}
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (g *getStatsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_stats",
		Desc: "获取应用数据统计与状态概览（只读，不修改任何数据）。当用户询问'总共有多少笔记/笔记本/标签/待办/密码记录、写了多少篇笔记、向量索引状态、AI 使用量（会话/消息/token/耗时）、数据库大小'等总量或概览类问题时调用。通过 action 参数区分：overview=数据统计概览（缺省，包含笔记/回收站/置顶/笔记本/标签/待办/密码记录/AI 用量/数据库大小/向量索引状态）；month=某月每日笔记数（需提供 year 年份与 month 月份，缺省为当前年月）。边界：查看具体笔记/待办/标签/笔记本列表请用对应的 manage_note / manage_todo / manage_tag / manage_notebook 工具；本工具只回答总量与趋势。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "要执行的统计动作：overview=数据统计概览（缺省）；month=月度笔记统计",
				Enum:     []string{"overview", "month"},
				Required: false,
			},
			"year": {
				Type:     schema.Number,
				Desc:     "年份，仅 action=month 时使用，缺省当前年份",
				Required: false,
			},
			"month": {
				Type:     schema.Number,
				Desc:     "月份（1-12），仅 action=month 时使用，缺省当前月份",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 用户取消检查 → 按 action 分发到概览或月度统计。
func (g *getStatsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Action string  `json:"action"`
		Year   float64 `json:"year"`
		Month  float64 `json:"month"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 get_stats 参数失败: %w", err)
	}
	args.Action = strings.TrimSpace(args.Action)

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	switch args.Action {
	case "", "overview":
		return g.statsOverview()
	case "month":
		return g.monthCounts(args.Year, args.Month)
	default:
		return "", fmt.Errorf("get_stats 参数非法 action: %s", args.Action)
	}
}

// statsOverview 数据统计概览：聚合 StatsService.GetDataStats 与 VectorService.GetIndexStatus
// 为一段中文分点的总量汇总文本（口径与数据管理页面一致）。
func (g *getStatsTool) statsOverview() (string, error) {
	stats, err := g.stats.GetDataStats()
	if err != nil {
		return "", fmt.Errorf("获取数据统计失败: %w", err)
	}
	noteCnt, chunkCnt, sizeBytes, err := g.vector.GetIndexStatus()
	if err != nil {
		return "", fmt.Errorf("获取数据统计失败: %w", err)
	}

	var b strings.Builder
	b.WriteString("数据统计概览：\n")
	fmt.Fprintf(&b, "笔记：共 %d 篇（回收站 %d，置顶 %d）\n", stats.TotalNotes, stats.TrashedNotes, stats.PinnedNotes)
	fmt.Fprintf(&b, "笔记本：%d 个\n", stats.TotalNotebooks)
	fmt.Fprintf(&b, "标签：%d 个\n", stats.TotalTags)
	fmt.Fprintf(&b, "待办：%d 项（已完成 %d）\n", stats.TotalTodos, stats.CompletedTodos)
	fmt.Fprintf(&b, "密码记录：%d 条\n", stats.TotalPasswords)
	fmt.Fprintf(&b, "AI 对话：%d 个会话，%d 条消息，累计 %d tokens\n", stats.AISessions, stats.AIMessages, stats.TotalTokens)
	fmt.Fprintf(&b, "响应耗时：平均 %.2f 秒，思考平均 %.2f 秒，最长 %.2f 秒\n", stats.AvgResponseTime, stats.AvgThinkingTime, stats.MaxResponseTime)
	fmt.Fprintf(&b, "数据库大小：%s\n", stats.DBSizeStr)
	fmt.Fprintf(&b, "向量索引：已嵌入 %d 篇笔记，%d 个片段，占用 %s", noteCnt, chunkCnt, formatSize(sizeBytes))
	return b.String(), nil
}

// monthCounts 某月每日笔记数：year/month 缺省取当前年月，校验范围后逐日列出统计结果；
// 该月无笔记时返回提示文案。
func (g *getStatsTool) monthCounts(year, month float64) (string, error) {
	y, m := int(year), int(month)
	now := time.Now()
	if y == 0 {
		y = now.Year()
	}
	if m == 0 {
		m = int(now.Month())
	}
	if m < 1 || m > 12 {
		return "", errors.New("get_stats month 参数须在 1-12 之间")
	}
	if y < 1900 || y > 2100 {
		return "", errors.New("get_stats year 参数须在 1900-2100 之间")
	}
	counts, err := g.stats.GetMonthCounts(y, m)
	if err != nil {
		return "", fmt.Errorf("获取数据统计失败: %w", err)
	}
	if len(counts) == 0 {
		return fmt.Sprintf("%04d-%02d 暂无笔记", y, m), nil
	}
	days := make([]int, 0, len(counts))
	for d := range counts {
		days = append(days, d)
	}
	sort.Ints(days)
	var b strings.Builder
	fmt.Fprintf(&b, "%04d-%02d 每日笔记数：\n", y, m)
	for _, d := range days {
		fmt.Fprintf(&b, "%d月%d日：%d 篇\n", m, d, counts[d])
	}
	return b.String(), nil
}

// formatSize 格式化字节数为人类可读大小：<1024 显示 B，<1MB 显示 KB（1 位小数），否则显示 MB（1 位小数）。
func formatSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/1048576)
	}
}

// NewGetStats 创建数据统计工具。
func NewGetStats(stats *services.StatsService, vector *services.VectorService, ctx *Context) tool.InvokableTool {
	return &getStatsTool{stats: stats, vector: vector, ctx: ctx}
}
