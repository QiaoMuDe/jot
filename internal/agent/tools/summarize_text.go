package tools

// 本文件实现 summarize_text 长文本摘要压缩工具：模型在 ReAct 循环中对长文本
// （read_url 抓回的网页正文、read_note_section 读出的笔记全文等）调用本工具
// 压缩为要点摘要，再基于压缩结果继续处理——节省输出 token、回答更聚焦、
// 多来源整合时先各自压缩再综合。
//
// 实现复用 services.AIService.CallAI（非流式单次调用，与 refine_search_query 同款路径），
// 摘要走当前配置的 AI 模型（BaseURL/Key/Model）。失败/无变化时降级返回原文，
// 不中断 ReAct 循环；用户取消时返回取消错误终止循环。
//
// 注意边界：text 作为工具参数传入时会进入上下文，本工具不能突破模型输入上限，
// 收益是"省输出 + 聚焦 + 结构化"，而非"绕开上下文限制"。

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

// summarizeTextPrompt 摘要 system 提示词：提取核心信息、忠实原文、结构化输出。
const summarizeTextPrompt = `你是一个文本摘要专家。你的任务是将用户提供的长文本压缩为要点摘要。

规则：
- 提取核心信息：主题、关键结论、事实与数据、专有名词、行动项
- 数字、日期、人名、术语必须准确，不得编造、不得臆测
- 使用与原文相同的语言输出（除非摘要要求指定语言）
- 输出为结构化要点列表；原文有明确分节时用小节标题组织
- 只输出摘要本身，不要任何解释、开场白或结尾语`

// summarizeTextTool 长文本摘要压缩工具。
type summarizeTextTool struct {
	ai *services.AIService // 摘要走当前配置的 AI 模型（复用 CallAI 非流式调用）
}

// 编译期断言：确保 summarizeTextTool 实现了 tool.InvokableTool 与 ActionTextProvider。
var _ tool.InvokableTool = (*summarizeTextTool)(nil)
var _ ActionTextProvider = (*summarizeTextTool)(nil)

// ActionText 提供 tool_start 动作文案（实现 ActionTextProvider）：固定文案，无需解析参数。
func (s *summarizeTextTool) ActionText(_ string) string {
	return "摘要长文本"
}

// Info 返回工具元信息。
func (s *summarizeTextTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "summarize_text",
		Desc: "把长文本压缩为要点摘要。当 read_url 抓回的网页正文、read_note_section 读出的笔记全文等内容过长、或需要整合多段长内容时，先调用本工具压缩再继续处理，节省上下文并让回答更聚焦。可用 instructions 指定摘要要求（如压缩成几点、保留数字与结论、输出行动项列表）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"text": {
				Type:     schema.String,
				Desc:     "要摘要的长文本（正文内容，上限 20000 字符）",
				Required: true,
			},
			"instructions": {
				Type:     schema.String,
				Desc:     "可选的摘要要求，如：压缩成 3 点 / 保留数字与结论 / 输出行动项列表（上限 500 字符，省略由模型自行判断）",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行摘要：解析参数 → CallAI → 返回摘要。
// 用户停止时返回取消错误终止循环；摘要失败/无变化时降级返回原文，模型仍可继续。
func (s *summarizeTextTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Text         string `json:"text"`
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 summarize_text 参数失败: %w", err)
	}
	text := strings.TrimSpace(args.Text)
	if text == "" {
		return "", errors.New("summarize_text 参数缺少 text")
	}
	if err := validateTextLen("text", text, maxToolLongText); err != nil {
		return "", err
	}
	instructions := strings.TrimSpace(args.Instructions)
	if err := validateTextLen("instructions", instructions, maxToolShortText); err != nil {
		return "", err
	}
	// 用户已取消：调用前即返回取消错误，不做无效的 LLM 调用
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 组装摘要请求：system 摘要提示词 + user 原文（附摘要要求）
	messages := []services.Message{
		{Role: "system", Content: summarizeTextPrompt},
		{Role: "user", Content: text},
	}
	if instructions != "" {
		messages[1].Content = text + "\n\n【摘要要求】" + instructions
	}

	summary, err := s.ai.CallAI(ctx, messages)
	if err != nil {
		// 用户停止：返回取消错误，终止 Agent 循环
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// 摘要失败：降级返回原文，模型仍可继续
		return text, nil
	}
	summary = strings.TrimSpace(summary)
	if summary == "" || summary == text {
		return text, nil
	}
	return summary, nil
}

// NewSummarizeText 创建长文本摘要压缩工具。
func NewSummarizeText(ai *services.AIService) tool.InvokableTool {
	return &summarizeTextTool{ai: ai}
}
