package agent

// 本文件实现 AgentService 主模块：基于 cloudwego/eino 的 ChatModelAgent（ReAct 循环）
// 组装一轮 Agent 对话并消费事件流，通过注入的 EmitFn 实时推送流式文本与工具状态。
//
// 工具实现在 tools 子包（每文件一个工具 + 导出构造器），父包 registry.go 统一装配注册；
// 工具经 tools.WrapWithError 包装（失败回填模型不中断循环），部分失败经 tools.Context 登记，
// 由本文件在 tool_result 之后统一 DrainPartials 发射 tool_partial 事件。
//
// 事件消费要点：
//   - 纯文本流式：assistant 事件以 IsStreaming 形式出现，逐 chunk 读 MessageStream，
//     将 chunk.Content 直接 emit（只emit内容，不拼接任何前缀）。
//   - 工具调用：assistant 的流式 chunk 中 Arguments 是增量片段，需用 schema.ConcatMessages
//     合并（按 Index 合并 ToolCalls、拼接 Arguments）后再判断 ToolCalls 是否存在；
//     合并后对每个工具调用 emit tool_start。
//   - 工具返回：Role=Tool 的事件携带工具执行结果（Content 或流），emit tool_result，
//     并把调用记录汇总进 Result.ToolCalls。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"gitee.com/MM-Q/fastlog"
	"jot/internal/agent/tools"
	"jot/internal/services"
)

// maxIterations 限制 ReAct 循环最大迭代次数，防止死循环（与 agent-demo 一致）。
const maxIterations = 8

// Deps AgentService 依赖注入。
// 注意：搜索功能在现有代码中是 services 包级函数（services.SearchWeb / SearchZhihuContent /
// SearchGlobalContent），不存在 *services.SearchService 类型，故此处不注入 Search，
// 由 web_search 工具内部直接调用包级函数。
type Deps struct {
	AI      *services.AIService
	Vector  *services.VectorService
	Setting *services.SettingService
	Todo    *services.TodoService // Todo 待办服务（manage_todo 工具使用）
	Logger  *fastlog.Logger
	// GetEmbedConfig 复用 app.go 现有逻辑：读取量化连接（ai_embed_* 三键），apiKey 已解码。
	GetEmbedConfig func() (baseURL, apiKey, model string, err error)
}

// AgentService 封装 Agent 对话链路。
type AgentService struct {
	deps Deps
}

// NewAgentService 创建一个新的 AgentService 实例。
func NewAgentService(deps Deps) *AgentService {
	return &AgentService{deps: deps}
}

// Run 执行一轮 Agent 对话：
//  1. 从现有 AI 配置（aiService.GetConfig）读取 BaseURL/APIKey/Model，构建 OpenAI 兼容 ChatModel；
//  2. 以 req.Instruction 作为系统提示词（Agent Instruction），历史消息转 schema.Message；
//  3. 注册 web_search / recall_notes 工具（notebook 过滤绑定 req.RecallNotebookIDs）；
//  4. runner.Run 消费事件流，流式文本与工具状态通过 emit 推送，最后汇总回答与工具摘要。
//
// ctx 支持取消：调用方传入带 cancel 的 ctx，Agent 循环随 ctx 终止，返回 ctx.Err()。
// 模型不支持 tool calling 或调用失败的报错原样返回 error，由调用方 ClassifyError。
func (s *AgentService) Run(ctx context.Context, req Request, emit EmitFn) (Result, error) {
	var result Result
	if emit == nil {
		emit = func(string, string) {}
	}

	// 1. 读取 AI 配置（复用现有 GetConfig 逻辑，含 B64 解码）
	aiCfg := s.deps.AI.GetConfig()
	if aiCfg.BaseURL == "" || aiCfg.APIKey == "" || aiCfg.Model == "" {
		return result, errors.New("请先配置 AI 服务（BaseURL / APIKey / Model）")
	}

	// 2. 构建 ChatModel（OpenAI 兼容协议，BaseURL 指向 DeepSeek/通义等兼容端点）
	chatModelCfg := &openai.ChatModelConfig{
		APIKey:  aiCfg.APIKey,
		Model:   aiCfg.Model,
		BaseURL: aiCfg.BaseURL,
		Timeout: 60 * time.Second,
	}
	// 深度思考开启时设置 reasoning_effort=high（OpenAI 标准参数，DeepSeek V4 / Qwen3 兼容端点支持）
	if req.ThinkingEnabled {
		chatModelCfg.ReasoningEffort = openai.ReasoningEffortLevelHigh
	}
	chatModel, err := openai.NewChatModel(ctx, chatModelCfg)
	if err != nil {
		return result, fmt.Errorf("创建 ChatModel 失败: %w", err)
	}

	// 3. 统一装配并注册工具（registry.go 的 buildTools，notebook 过滤在构造 recall_notes 时绑定）
	//    collector/ctx 贯穿本轮：工具经 tools.WrapWithError 包装（失败回填模型不中断循环），
	//    部分失败由工具经 ctx.AddPartial 登记，tool_result 之后统一 DrainPartials 发射
	collector := &tools.Collector{}
	var toolRecords []tools.Record
	toolCtx := &tools.Context{Emit: emit, Records: &toolRecords, Collector: collector, Logger: s.deps.Logger}
	toolList := buildTools(BuildParams{deps: s.deps, req: req, ctx: toolCtx})

	// 4. 组装 ChatModelAgent（内部是 ReAct 循环：模型决策 → 调用工具 → 反馈 → 继续）
	//    Instruction 作为 system 消息由默认 GenModelInput 放在最前
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "jot-agent",
		Description: "一个能调用联网搜索与本地笔记召回工具回答问题的助手",
		Instruction: req.Instruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolList,
			},
		},
		MaxIterations: maxIterations,
	})
	if err != nil {
		return result, fmt.Errorf("创建 ChatModelAgent 失败: %w", err)
	}

	// 5. 创建 Runner 并执行（开启流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 6. 消费事件流
	messages := buildMessages(req)
	iter := runner.Run(ctx, messages)

	var finalContent string
	var promptTokensTotal, completionTokensTotal int

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			return result, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		switch mv.Role {
		case schema.Assistant:
			// 模型输出事件：可能是纯文本，也可能是工具调用决策
			if mv.IsStreaming {
				full, roundThinking, err := consumeAssistantStream(mv.MessageStream, emit, req.ThinkingEnabled)
				if err != nil {
					if ctx.Err() != nil {
						return result, ctx.Err()
					}
					return result, fmt.Errorf("读取模型流失败: %w", err)
				}
				if full == nil {
					continue
				}
				// 该轮真实 usage（流式最后空消息携带，ConcatMessages 已合并到 ResponseMeta）
				if u := full.ResponseMeta; u != nil && u.Usage != nil {
					promptTokensTotal += u.Usage.PromptTokens
					completionTokensTotal += u.Usage.CompletionTokens
				}
				// 思考净时长：按轮累加（该轮首个 reasoning 分片到该轮最后分片），
				// 排除轮与轮之间的工具执行时间，避免思考耗时虚高
				result.ThinkingElapsed += roundThinking.Seconds()
				// 深度思考链：按开关累加所有 assistant 消息的 reasoning（与实时 ai:stream-thinking 展示一致）
				if req.ThinkingEnabled && full.ReasoningContent != "" {
					result.ReasoningContent += full.ReasoningContent
				}
				if len(full.ToolCalls) > 0 {
					// 模型决定调用工具：流式 Arguments 已按 Index 合并
					for _, tc := range full.ToolCalls {
						emitToolStart(emit, &toolRecords, tc)
					}
				} else if full.Content != "" {
					// 无工具调用的 assistant 消息即最终回答（ReAct 循环最后一条）
					finalContent = full.Content
				}
			} else if mv.Message != nil {
				// 非流式消息同样携带该轮真实 usage
				if u := mv.Message.ResponseMeta; u != nil && u.Usage != nil {
					promptTokensTotal += u.Usage.PromptTokens
					completionTokensTotal += u.Usage.CompletionTokens
				}
				if len(mv.Message.ToolCalls) > 0 {
					for _, tc := range mv.Message.ToolCalls {
						emitToolStart(emit, &toolRecords, tc)
					}
				} else if mv.Message.Content != "" {
					emit("ai:stream-chunk", mv.Message.Content)
					finalContent = mv.Message.Content
				}
			}
		case schema.Tool:
			// 工具执行结果事件
			name := mv.ToolName
			var content string
			if mv.IsStreaming {
				content = consumeToolStream(mv.MessageStream)
			} else if mv.Message != nil {
				content = mv.Message.Content
				if name == "" {
					name = mv.Message.ToolName
				}
			}
			if content != "" {
				if s.deps.Logger != nil {
					s.deps.Logger.Debugw("Agent 工具执行完成",
						fastlog.String("tool", name),
						fastlog.Int("result_len", len(content)))
				}
				emitToolResult(emit, &toolRecords, name, content)
				// 部分失败提示（如 web_search 部分来源失败）：工具内部经 ctx.AddPartial 登记，
				// 此处统一在 tool_result 之后发射 tool_partial，前端展示 ⚠️ 警告；发射后清空防重复
				toolCtx.DrainPartials(name)
			}
		}
	}

	// 用户取消：Agent 循环随 ctx 终止
	if ctx.Err() != nil {
		return result, ctx.Err()
	}

	// 7. 汇总结果：内容 + 工具收集的结构化来源/卡片 + 工具调用链 + 真实 token usage
	result.Content = finalContent
	result.PromptTokens = promptTokensTotal
	result.CompletionTokens = completionTokensTotal
	if len(toolCtx.Collector.Sources) > 0 {
		if b, err := json.Marshal(toolCtx.Collector.Sources); err == nil {
			result.SearchSources = string(b)
		}
	}
	if len(toolCtx.Collector.Cards) > 0 {
		if b, err := json.Marshal(toolCtx.Collector.Cards); err == nil {
			result.RecallCards = string(b)
		}
	}
	if len(toolRecords) > 0 {
		if b, err := json.Marshal(toolRecords); err == nil {
			result.ToolCalls = string(b)
		}
	}
	if s.deps.Logger != nil {
		s.deps.Logger.Debugw("Agent 对话完成",
			fastlog.Int("content_len", len(result.Content)),
			fastlog.Int("tool_calls", len(toolRecords)),
			fastlog.Int("max_iterations", maxIterations))
	}
	return result, nil
}

// buildMessages 将历史消息与当前用户消息转换为 schema.Message 列表。
// Instruction 作为 Agent 的 Instruction 由默认 GenModelInput 自动放最前（system 消息），
// 此处只构造 user/assistant 历史；当前用户消息追加在末尾（若历史末条已是同内容则跳过，避免重复）。
func buildMessages(req Request) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(req.History)+1)
	for _, h := range req.History {
		switch h.Role {
		case "user":
			msgs = append(msgs, schema.UserMessage(h.Content))
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(h.Content, nil))
		}
	}

	appendUser := true
	if n := len(req.History); n > 0 {
		last := req.History[n-1]
		if last.Role == "user" && last.Content == req.UserText {
			appendUser = false
		}
	}
	if appendUser && req.UserText != "" {
		msgs = append(msgs, schema.UserMessage(req.UserText))
	}
	return msgs
}

// consumeAssistantStream 消费 assistant 流式消息：
// 每个非空文本 chunk 直接 emit（不拼接前缀）；深度思考开启时流式 reasoning chunk
// 通过 ai:stream-thinking 事件实时推送；流结束后用 schema.ConcatMessages 合并全部 chunk
// （按 Index 合并 ToolCalls、Arguments 与 ReasoningContent 增量拼接），返回完整消息供调用方判断。
// 同时返回本消息的思考净时长：首个 reasoning 分片到本消息最后分片（排除跨轮的工具体执行时间）。
func consumeAssistantStream(stream *schema.StreamReader[*schema.Message], emit EmitFn, thinkingEnabled bool) (*schema.Message, time.Duration, error) {
	if stream == nil {
		return nil, 0, nil
	}
	var chunks []*schema.Message
	var thinkingStart time.Time
	var lastChunkAt time.Time
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		if chunk != nil {
			if chunk.ReasoningContent != "" && thinkingStart.IsZero() {
				thinkingStart = time.Now()
			}
			lastChunkAt = time.Now()
			if chunk.Content != "" {
				emit("ai:stream-chunk", chunk.Content)
			}
			if thinkingEnabled && chunk.ReasoningContent != "" {
				emit("ai:stream-thinking", chunk.ReasoningContent)
			}
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) == 0 {
		return nil, 0, nil
	}
	full, err := schema.ConcatMessages(chunks)
	if err != nil {
		return nil, 0, err
	}
	roundThinking := time.Duration(0)
	if !thinkingStart.IsZero() && !lastChunkAt.IsZero() {
		roundThinking = lastChunkAt.Sub(thinkingStart)
	}
	return full, roundThinking, nil
}

// consumeToolStream 消费工具结果流式消息，返回合并后的完整文本。
func consumeToolStream(stream *schema.StreamReader[*schema.Message]) string {
	if stream == nil {
		return ""
	}
	var chunks []*schema.Message
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) == 0 {
		return ""
	}
	full, err := schema.ConcatMessages(chunks)
	if err != nil || full == nil {
		return ""
	}
	return full.Content
}

// emitToolStart 推送工具调用开始事件，并记录到工具调用摘要。
func emitToolStart(emit EmitFn, records *[]tools.Record, tc schema.ToolCall) {
	rec := tools.Record{
		Action: "tool_start",
		Name:   tc.Function.Name,
		Args:   tools.TruncateRunes(tc.Function.Arguments, tools.MaxResultLen),
	}
	*records = append(*records, rec)
	b, _ := json.Marshal(rec)
	emit("ai:tool-status", string(b))
}

// emitToolResult 推送工具返回事件，并记录到工具调用摘要。
// 若该工具刚失败（已发射 tool_error），则跳过结果事件，保持失败态不被 ✓ 覆盖。
func emitToolResult(emit EmitFn, records *[]tools.Record, name, result string) {
	for i := len(*records) - 1; i >= 0; i-- {
		if (*records)[i].Name != name {
			continue
		}
		if (*records)[i].Action == "tool_error" {
			return
		}
		break
	}
	rec := tools.Record{
		Action: "tool_result",
		Name:   name,
		Result: tools.TruncateRunes(result, tools.MaxResultLen),
	}
	*records = append(*records, rec)
	b, _ := json.Marshal(rec)
	emit("ai:tool-status", string(b))
}
