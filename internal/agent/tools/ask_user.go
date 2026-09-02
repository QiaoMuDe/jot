package tools

// 本文件实现 ask_user 反向提问工具：模型在 ReAct 循环中调用它向用户发起
// 一次结构化澄清提问（不执行业务操作），前端据此渲染问题卡片并等待用户回答。
// 父包注入 AskWaiter 时（Agent 交互模式），工具在发射事件后阻塞等待用户回答，
// ReAct 循环暂停（AI 消息不结束），答案经 AnswerAskUser 同轮投递回模型继续；
// 未注入时保持原非阻塞行为（仅返回确认文本）。
// 当用户意图不明确、缺少必要信息（如未指定搜索源/方案/范围/数量等），
// 或需要在多个选项之间让用户做选择时调用；一次可问 1-3 个问题（每题选项 2-6 个，
// 每题可独立设置单选/多选），相关联的信息应合并为一次提问。
// 本工具不执行业务，仅请求澄清，严格禁止用于闲聊或无意义的确认。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// maxAskUserOptions 候选选项上限：每题最多给出 6 项供用户选择。
const maxAskUserOptions = 6

// maxAskUserQuestions 单次提问的问题数上限：一次最多向用户确认 3 个问题。
const maxAskUserQuestions = 3

// askUserQuestion 单条问题：问句文本 + 候选选项 + 选择模式（single/multiple）。
type askUserQuestion struct {
	Question  string   `json:"question"`
	Options   []string `json:"options"`
	Selection string   `json:"selection"`
}

// askUserTool 反向提问工具：向用户发起结构化澄清提问。
type askUserTool struct {
	ctx *Context // 事件发射（ai:ask-user 问题卡片数据源）与日志
}

// 编译期断言：确保 askUserTool 实现了 tool.InvokableTool 与 ActionTextProvider。
var _ tool.InvokableTool = (*askUserTool)(nil)
var _ ActionTextProvider = (*askUserTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：
// 解析 questions 参数返回"向用户提问（共N问）：{首问}"，单条返回"向用户提问：{问题}"，
// 解析失败回退空串（前端回退"执行"）。
func (g *askUserTool) ActionText(argumentsInJSON string) string {
	qs := parseAskUserArgsLoose(argumentsInJSON)
	if len(qs) == 0 {
		return ""
	}
	if len(qs) == 1 {
		return "向用户提问：" + TruncateRunes(qs[0].Question, 30)
	}
	return fmt.Sprintf("向用户提问（共%d问）：%s", len(qs), TruncateRunes(qs[0].Question, 30))
}

// Info 返回工具元信息（名称、描述、参数 JSON Schema）。
func (g *askUserTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "ask_user",
		Desc: "向用户发起一次结构化澄清提问（不执行业务操作）。以下场景必须调用本工具，不得省略或绕过，严禁在缺少必要信息时擅自猜测后直接执行：①用户请求存在信息模糊、参数不明确、需求不具体（如未指定搜索源/方案/范围/数量/操作对象等）；②需要用户在多个选项或方案之间做选择；③需要获取用户进一步确认或补充关键信息才能继续执行（含写操作前的确认）。调用时向用户展示问题卡片并等待其回答。一次可问 1-3 个问题：需要一次性收集多项信息时合并为一次调用（如同时确认搜索源和数量）；仅当问题相互独立时才拆为多次调用。每题选项 2-6 个，可省略让用户自由输入；每题可独立设置单选或多选。严禁把猜测当作已确认事实继续操作，严禁用于闲聊或无意义的确认。调用后在回复正文中完整写出你的问题，然后停止生成等待用户回答。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"questions": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"question": {
							Type:     schema.String,
							Desc:     "要向用户提出的问题（问句形式，简洁明确）",
							Required: true,
						},
						"options": {
							Type:     schema.Array,
							ElemInfo: &schema.ParameterInfo{Type: schema.String},
							Desc:     "该题的候选选项列表（2-6 项，供用户点击选择；可省略让用户自由输入）",
							Required: false,
						},
						"selection": {
							Type:     schema.String,
							Desc:     "该题的选择模式：single=单选（用户点击某个选项即作答）；multiple=多选（用户勾选多项后确认）。缺省为 single；当需要用户在多个选项中做多选决策（如选择多篇笔记、多个标签、多个方案组合）时使用 multiple",
							Required: false,
						},
					},
				},
				Desc:     "要向用户提出的问题列表（1-3 条；相关联的信息尽量合并为一次提问，仅当问题相互独立时才拆为多条）",
				Required: true,
			},
			"reason": {
				Type:     schema.String,
				Desc:     "提问原因说明（模型视角，供调试）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 用户取消检查 → 校验并规范化 questions（1-3 条，
// 旧格式单字段兜底）→ 发射 ai:ask-user 事件（前端渲染问题卡片）→ 返回提示文本或等待答案。
func (g *askUserTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var raw struct {
		Questions []askUserQuestion `json:"questions"`
		// 旧格式兜底：模型偶发仍发单条 question/options/selection 字段
		Question  string   `json:"question"`
		Options   []string `json:"options"`
		Selection string   `json:"selection"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &raw); err != nil {
		return "", fmt.Errorf("解析 ask_user 参数失败: %w", err)
	}

	// 用户取消检查：父包事件循环随 ctx 终止，工具直接返回 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 兼容旧格式：questions 为空且旧 question 非空时包装为单条
	qs := raw.Questions
	if len(qs) == 0 && strings.TrimSpace(raw.Question) != "" {
		qs = []askUserQuestion{{Question: raw.Question, Options: raw.Options, Selection: raw.Selection}}
	}
	if len(qs) == 0 {
		return "", errors.New("ask_user 参数缺少 questions")
	}
	if len(qs) > maxAskUserQuestions {
		return "", fmt.Errorf("ask_user 最多支持 %d 个问题，请精简为不超过 %d 条", maxAskUserQuestions, maxAskUserQuestions)
	}

	// 逐条校验与规范化：问句去空/限长、选项去重限长、selection 规范化
	normalized := make([]askUserQuestion, 0, len(qs))
	for i, q := range qs {
		text := strings.TrimSpace(q.Question)
		if text == "" {
			return "", fmt.Errorf("ask_user 第 %d 条问题缺少 question", i+1)
		}
		if err := validateTextLen(fmt.Sprintf("questions[%d].question", i+1), text, maxToolShortText); err != nil {
			return "", err
		}
		// 选项长度校验（防止模型传入超长选项撑爆问题卡片）
		for _, o := range q.Options {
			if err := validateTextLen("options 元素", o, 200); err != nil {
				return "", err
			}
		}
		// 规范化 options：去重、去空、最多取 6 项；options 为空则为空切片
		sel := "single"
		if q.Selection == "multiple" {
			sel = "multiple"
		}
		normalized = append(normalized, askUserQuestion{Question: text, Options: normalizeAskUserOptions(q.Options), Selection: sel})
	}

	// 同轮传输：父包注入 AskWaiter 时，先原子抢占反问名额（ClaimAsk），
	// 再发射 ai:ask-user 事件并阻塞等待用户回答。ReAct 循环暂停（AI 消息不结束），
	// 答案经 AnswerAskUser 投递到等待通道后作为本工具结果返回给模型继续完成
	// 原始请求（不落库为新用户消息）。模型同条消息并行发出多条 ask_user 时，
	// 仅第一条抢占成功并阻塞，其余 ClaimAsk 失败返回错误回填模型，避免整轮挂起。
	// 未注入 AskWaiter（非交互场景/测试）时保持原行为：仅发射事件并返回确认文本。
	if g.ctx.AskWaiter != nil {
		if err := g.ctx.AskWaiter.ClaimAsk(); err != nil {
			return "", err
		}
		// 发射 ai:ask-user 事件：前端渲染问题卡片的数据源（抢占成功后才发射，
		// 保证面板展示的问题与真正阻塞等待的是同一条）。负载含新旧双格式：
		// questions 数组 + 旧顶层字段（取首条，兼容前端旧解析）。
		if b, err := json.Marshal(askUserEventPayload(normalized)); err == nil {
			g.ctx.Emit("ai:ask-user", string(b))
		}
		answer, err := g.ctx.AskWaiter.WaitForAnswer(ctx)
		if err != nil {
			// 用户取消/会话释放：错误文本回填模型，循环随 ctx 终止
			return "", err
		}
		return buildAskUserAnswerText(normalized, answer), nil
	}

	// 发射 ai:ask-user 事件：前端渲染问题卡片的数据源
	if b, err := json.Marshal(askUserEventPayload(normalized)); err == nil {
		g.ctx.Emit("ai:ask-user", string(b))
	}

	// 本工具不执行业务，仅请求澄清；问句正文由模型在回复正文中输出
	if len(normalized) == 1 {
		return fmt.Sprintf("我需要向你确认：%s，请从上方选项中选择或直接输入你的答案。", normalized[0].Question), nil
	}
	return fmt.Sprintf("我需要向你确认 %d 个问题，请逐题从上方选项中选择或直接输入你的答案。", len(normalized)), nil
}

// askUserEventPayload 构造 ai:ask-user 事件负载：questions 数组 + 旧顶层字段
// （取首条问题的值，兼容前端旧解析，后续可移除）。
func askUserEventPayload(qs []askUserQuestion) map[string]any {
	payload := map[string]any{"questions": qs}
	if len(qs) > 0 {
		payload["question"] = qs[0].Question
		payload["options"] = qs[0].Options
		payload["selection"] = qs[0].Selection
	}
	return payload
}

// buildAskUserAnswerText 把用户提交的答案文本映射回各问题，构造回填模型的工具结果。
// 前端约定：单问题 = 原始答案文本；多问题 = 每题一行（"答案1\n答案2..."，行内无换行）。
// 行数与问题数一致时逐题映射；不匹配时整体作为单条答案返回（防御性兜底）。
func buildAskUserAnswerText(qs []askUserQuestion, answer string) string {
	answer = strings.TrimSpace(answer)
	if len(qs) <= 1 {
		return fmt.Sprintf("用户已回答你的提问。用户的回答是：%s。请结合你的问题与用户的回答继续完成用户的原始请求，直接给出最终回答或继续调用后续工具，不要重复提问。", TruncateRunes(answer, maxToolShortText))
	}
	lines := strings.Split(answer, "\n")
	if len(lines) != len(qs) {
		// 行数不匹配：无法可靠配对，整体作为单条答案返回
		return fmt.Sprintf("用户已回答你的全部提问。用户的回答是：%s。请结合你的问题与用户的回答继续完成用户的原始请求，直接给出最终回答或继续调用后续工具，不要重复提问。", TruncateRunes(answer, maxToolShortText))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "用户已回答你的全部 %d 个提问，逐题回答如下：", len(qs))
	for i, q := range qs {
		fmt.Fprintf(&b, "\n%d. %s\n   用户回答：%s", i+1, q.Question, TruncateRunes(strings.TrimSpace(lines[i]), maxToolShortText))
	}
	b.WriteString("。请结合你的问题与用户的回答继续完成用户的原始请求，直接给出最终回答或继续调用后续工具，不要重复提问。")
	return b.String()
}

// parseAskUserArgsLoose 宽松解析参数中的问题列表（仅供 ActionText 展示文案使用，
// 不做校验）：questions 数组优先，旧单字段 question 兜底。
func parseAskUserArgsLoose(argumentsInJSON string) []askUserQuestion {
	var raw struct {
		Questions []askUserQuestion `json:"questions"`
		Question  string            `json:"question"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &raw); err != nil {
		return nil
	}
	if len(raw.Questions) > 0 {
		return raw.Questions
	}
	if q := strings.TrimSpace(raw.Question); q != "" {
		return []askUserQuestion{{Question: q}}
	}
	return nil
}

// normalizeAskUserOptions 规范化候选选项：去空、去重（保留首次出现）、最多取 maxAskUserOptions 项。
func normalizeAskUserOptions(options []string) []string {
	opts := make([]string, 0, len(options))
	seen := make(map[string]struct{}, len(options))
	for _, o := range options {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if _, ok := seen[o]; ok {
			continue
		}
		seen[o] = struct{}{}
		opts = append(opts, o)
		if len(opts) >= maxAskUserOptions {
			break
		}
	}
	return opts
}

// NewAskUser 创建反向提问工具。
func NewAskUser(ctx *Context) tool.InvokableTool {
	return &askUserTool{ctx: ctx}
}
