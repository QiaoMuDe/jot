package tools

// 本文件定义 tools 子包的共享上下文类型与工具包装器。
// 工具实现（web_search / recall_notes / refine_search_query）只依赖本文件声明的类型，
// 不感知父包 agent 的事件循环细节；父包通过 registry.go 统一装配与注册工具。

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"gitee.com/MM-Q/fastlog"
	"jot/internal/services"
)

// MaxResultLen 工具调用参数 / 结果摘要截断长度（用于事件与落库，避免超长）。
const MaxResultLen = 500

// EmitFn 事件回调，由调用方注入（内部封装 runtime.EventsEmit）。
// event 为事件名（如 "ai:stream-chunk" / "ai:tool-status"），data 为事件负载。
type EmitFn func(event string, data string)

// Record 工具调用记录，用于组装 Result.ToolCalls 摘要 JSON。
type Record struct {
	Action string `json:"action"`           // "tool_start" / "tool_result" / "tool_error" / "tool_partial"
	Name   string `json:"name"`             // 工具名
	Args   string `json:"args,omitempty"`   // 工具调用参数（JSON，截断后）
	Result string `json:"result,omitempty"` // 工具返回结果摘要（截断后）
}

// Collector 收集一轮 Agent 对话中工具执行产生的结构化结果：
// web_search 的来源列表与 recall_notes 的召回卡片，供父包 Run 结束时汇总进 Result。
type Collector struct {
	Sources []services.SearchSource
	Cards   []services.RecallCard
}

// Context 注入给每个工具的执行上下文：事件发射、调用记录、结构化收集器与日志。
// 部分失败等特殊事件由工具内部经 AddPartial 登记，父包在 tool_result 之后统一 DrainPartials 发射。
type Context struct {
	Emit      EmitFn
	Records   *[]Record
	Collector *Collector
	Logger    *fastlog.Logger

	partials []string // 工具登记的部分失败提示（父包 DrainPartials 消费后清空）
}

// AddPartial 登记一条部分失败提示（如 web_search 部分来源失败），
// 父包会在 tool_result 事件之后以 tool_partial 事件统一发射，保证事件顺序。
func (c *Context) AddPartial(msg string) {
	c.partials = append(c.partials, msg)
}

// DrainPartials 把工具登记的部分失败提示以 tool_partial 事件逐条发射并清空。
// 由父包在工具返回（tool_result）之后调用，name 为工具名。
func (c *Context) DrainPartials(name string) {
	if c == nil || len(c.partials) == 0 {
		return
	}
	for _, p := range c.partials {
		rec := Record{Action: "tool_partial", Name: name, Result: p}
		*c.Records = append(*c.Records, rec)
		if b, err := json.Marshal(rec); err == nil {
			c.Emit("ai:tool-status", string(b))
		}
	}
	c.partials = nil
}

// WrapWithError 包装工具：执行失败时不中断 ReAct 循环，
// 错误文本回填给模型继续推理，同时记录服务日志并发射 tool_error 事件供前端展示失败态。
func WrapWithError(name string, t tool.InvokableTool, ctx *Context) tool.InvokableTool {
	return utils.WrapInvokableToolWithErrorHandler(t, func(c context.Context, err error) string {
		// 用户取消：不误报失败，直接返回错误文本（循环会随 ctx 终止）
		if c.Err() != nil {
			return err.Error()
		}
		if ctx.Logger != nil {
			ctx.Logger.Warnw("Agent 工具执行失败",
				fastlog.String("tool", name),
				fastlog.Error(err))
		}
		rec := Record{
			Action: "tool_error",
			Name:   name,
			Result: TruncateRunes(err.Error(), MaxResultLen),
		}
		*ctx.Records = append(*ctx.Records, rec)
		if b, err := json.Marshal(rec); err == nil {
			ctx.Emit("ai:tool-status", string(b))
		}
		return "工具执行失败：" + err.Error() + "。请依据错误信息调整策略，或直接基于已有信息回答用户。"
	})
}

// TruncateRunes 按 rune 截断字符串（支持中文），超过 maxLen 时追加省略号；maxLen<=0 不截断。
func TruncateRunes(s string, maxLen int) string {
	if maxLen <= 0 || len(s) == 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
