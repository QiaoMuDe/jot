package tools

// 本文件定义 tools 子包的共享上下文类型与工具包装器。
// 工具实现（recall_notes / read_url / manage_* 等）只依赖本文件声明的类型，
// 不感知父包 agent 的事件循环细节；父包通过 registry.go 统一装配与注册工具。

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"gitee.com/MM-Q/fastlog"
	"jot/internal/services"
)

// MaxResultLen 工具调用参数 / 结果摘要截断长度（用于事件与落库，避免超长）。
const MaxResultLen = 500

// maxToolLLMTimeout 工具内辅助 LLM 调用（精炼/摘要等）的超时上限：
// 与主模型调用超时（agent.go 的 60s）一致，防止模型 API 在传输层挂起时无限阻塞 ReAct 循环。
const maxToolLLMTimeout = 60 * time.Second

// 文本字段长度上限：防止模型传入超长文本浪费 token 或触发 DB 字段越界报错。
const (
	maxToolShortText = 500   // 短文本字段上限：标题/名称/关键字/搜索词/URL/问句/颜色等
	maxToolFindLen   = 2000  // edit 片段替换 find 原文片段上限
	maxToolLongText  = 20000 // 正文级字段上限：content / replace
)

// validateTextLen 校验文本字段长度（按 rune 计），超长返回描述性错误供回填模型。
func validateTextLen(field, s string, maxLen int) error {
	if n := len([]rune(s)); n > maxLen {
		return fmt.Errorf("%s 过长（%d 字符，上限 %d），请精简后重试", field, n, maxLen)
	}
	return nil
}

// getIntSetting 读取 int 类型设置，解析失败或越界时回退默认值（越上限取上限）。
// 供 read_url 等工具复用同一设置项读取逻辑（原定义于已移除的 web_search.go）。
func getIntSetting(setting *services.SettingService, key string, def, max int) int {
	if setting == nil {
		return def
	}
	val := setting.Get(key)
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// EmitFn 事件回调，由调用方注入（内部封装 runtime.EventsEmit）。
// event 为事件名（如 "ai:stream-chunk" / "ai:tool-status"），data 为事件负载。
type EmitFn func(event string, data string)

// Record 工具调用记录，用于组装 Result.ToolCalls 摘要 JSON。
type Record struct {
	Action     string `json:"action"`                // "tool_start" / "tool_result" / "tool_error" / "tool_partial"
	Name       string `json:"name"`                  // 工具名
	CallID     string `json:"call_id,omitempty"`     // eino 原生 tool_call ID（同轮多条同名调用时前端精确配对的关键）
	Args       string `json:"args,omitempty"`        // 工具调用参数（JSON，截断后）
	Result     string `json:"result,omitempty"`      // 工具返回结果摘要（截断后）
	ActionText string `json:"action_text,omitempty"` // tool_start 的动作中文文案（由工具 ActionTextProvider 提供）
}

// ActionTextProvider 可选接口：工具实现后在 tool_start 时由父包自动采用其动作文案，
// 供前端状态条展示"调用「工具名」工具：{动作}"；未实现或返回空串则前端回退"执行"。
type ActionTextProvider interface {
	ActionText(argumentsInJSON string) string
}

// Collector 收集一轮 Agent 对话中工具执行产生的结构化结果：
// 搜索来源与 recall_notes 的召回卡片，供父包 Run 结束时汇总进 Result。
type Collector struct {
	Sources []services.SearchSource
	Cards   []services.RecallCard
}

// AskWaiter 反向提问等待器：由父包注入，供 ask_user 工具在 ReAct 循环内
// 阻塞等待用户回答（同轮传输）。为 nil 时 ask_user 保持原非阻塞行为
// （发射事件后立即返回确认文本，循环继续）。
type AskWaiter interface {
	// ClaimAsk 在发射 ai:ask-user 事件前原子抢占反问名额；
	// 已有反问在等待时返回错误（模型在同一条消息里并行发出多条 ask_user 时，
	// 拒绝多余提问，避免多重阻塞导致整轮挂起）。
	ClaimAsk() error
	// WaitForAnswer 阻塞等待用户回答，返回用户输入文本；
	// ctx 取消（停止按钮/会话释放）时返回 ctx.Err()。
	WaitForAnswer(ctx context.Context) (string, error)
}

// Context 注入给每个工具的执行上下文：事件发射、调用记录、结构化收集器与日志。
// 部分失败等特殊事件由工具内部经 AddPartial 登记，父包在 tool_result 之后统一 DrainPartials 发射。
type Context struct {
	Emit      EmitFn
	Records   *[]Record
	Collector *Collector
	Logger    *fastlog.Logger
	AskWaiter AskWaiter // 非 nil 时 ask_user 工具阻塞等待用户回答（同轮续答）

	partials []string // 工具登记的部分失败提示（父包 DrainPartials 消费后清空）
}

// AddPartial 登记一条部分失败提示（如多来源搜索部分来源失败），
// 父包会在 tool_result 事件之后以 tool_partial 事件统一发射，保证事件顺序。
func (c *Context) AddPartial(msg string) {
	c.partials = append(c.partials, msg)
}

// DrainPartials 把工具登记的部分失败提示以 tool_partial 事件逐条发射并清空。
// 由父包在工具返回（tool_result）之后调用；callID 为该次调用的 eino tool_call ID，
// 使前端能把部分失败提示精确定位到对应的调用行。
func (c *Context) DrainPartials(name, callID string) {
	if c == nil || len(c.partials) == 0 {
		return
	}
	for _, p := range c.partials {
		rec := Record{Action: "tool_partial", Name: name, CallID: callID, Result: p}
		*c.Records = append(*c.Records, rec)
		if b, err := json.Marshal(rec); err == nil {
			c.Emit("ai:tool-status", string(b))
		}
	}
	c.partials = nil
}

// wrappedTool 自定义包装器：委托内层工具执行；失败时不中断 ReAct 循环
// （错误文本回填模型继续推理 + 记 tool_error 记录 + 发射事件），
// 并实现 ActionTextProvider 转发，使父包可对包装后的工具统一断言动作文案。
type wrappedTool struct {
	name  string
	inner tool.InvokableTool
	ctx   *Context
}

var _ tool.InvokableTool = (*wrappedTool)(nil)
var _ ActionTextProvider = (*wrappedTool)(nil)

func (w *wrappedTool) Info(c context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(c)
}

func (w *wrappedTool) InvokableRun(c context.Context, argumentsInJSON string, opts ...tool.Option) (ret string, retErr error) {
	// panic 防护：工具内部 panic 转为错误回填模型（不记 tool_error 仅当用户已取消），
	// 避免单个工具异常导致整个 Agent 进程崩溃
	defer func() {
		if r := recover(); r != nil {
			if c.Err() != nil {
				ret = fmt.Sprintf("工具内部异常（panic）：%v", r)
				return
			}
			ret = w.fail(c, fmt.Errorf("工具内部异常（panic）：%v", r))
		}
	}()
	out, err := w.inner.InvokableRun(c, argumentsInJSON, opts...)
	if err != nil {
		// 用户取消：不误报失败，直接返回错误文本（循环会随 ctx 终止）
		if c.Err() != nil {
			return err.Error(), nil
		}
		return w.fail(c, err), nil
	}
	return out, nil
}

// fail 记录一次工具执行失败（日志 Warnw + tool_error 记录 + 事件发射），
// 并返回回填给模型的错误文本。供 error 路径与 panic recover 复用。
func (w *wrappedTool) fail(c context.Context, err error) string {
	if w.ctx.Logger != nil {
		w.ctx.Logger.Warnw("Agent 工具执行失败",
			fastlog.String("tool", w.name),
			fastlog.Error(err))
	}
	rec := Record{
		Action: "tool_error",
		Name:   w.name,
		// eino 工具执行 ctx 内注入当前 tool_call ID（compose.GetToolCallID），
		// 使前端能把失败精确定位到对应的调用行（同轮多条同名调用时关键）
		CallID: compose.GetToolCallID(c),
		Result: TruncateRunes(err.Error(), MaxResultLen),
	}
	*w.ctx.Records = append(*w.ctx.Records, rec)
	if b, err := json.Marshal(rec); err == nil {
		w.ctx.Emit("ai:tool-status", string(b))
	}
	return "工具执行失败：" + err.Error() + "。请依据错误信息调整策略，或直接基于已有信息回答用户。"
}

func (w *wrappedTool) ActionText(argumentsInJSON string) string {
	if p, ok := w.inner.(ActionTextProvider); ok {
		return p.ActionText(argumentsInJSON)
	}
	return ""
}

// WrapWithError 包装工具：执行失败时不中断 ReAct 循环，
// 错误文本回填给模型继续推理，同时记录服务日志并发射 tool_error 事件供前端展示失败态。
// 包装器同时实现 ActionTextProvider：内层工具实现了则转发其动作文案，否则返回空串。
func WrapWithError(name string, t tool.InvokableTool, ctx *Context) tool.InvokableTool {
	return &wrappedTool{name: name, inner: t, ctx: ctx}
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
