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
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"jot/internal/agent/tools"
	"jot/internal/mcpserver"
	"jot/internal/services"

	"gitee.com/MM-Q/fastlog"
)

// MaxIterations 限制 ReAct 循环最大迭代次数，防止死循环（统一默认值，供装配与日志引用）。
const MaxIterations = 20

// Deps AgentService 依赖注入。
// 注意：搜索功能在现有代码中是 services 包级函数（services.SearchWeb / SearchZhihuContent /
// SearchGlobalContent），不存在 *services.SearchService 类型，故此处不注入 Search，
// 由 web_search 工具内部直接调用包级函数。
type Deps struct {
	AI       *services.AIService
	Vector   *services.VectorService
	Setting  *services.SettingService
	Todo     *services.TodoService     // Todo 待办服务（manage_todo 工具使用）
	Notebook *services.NotebookService // Notebook 笔记本服务（manage_notebook 工具使用）
	Tag      *services.TagService      // Tag 标签服务（manage_tag 工具使用）
	Note     *services.NoteService     // Note 笔记服务（manage_note 工具使用）
	Stats    *services.StatsService    // Stats 数据统计聚合服务（get_stats 工具使用）
	Logger   *fastlog.Logger
	// MCPServerConfigPath 外部 MCP 服务器配置文件路径（测试阶段，配置文件驱动）；
	// 为空时回退 mcpserver.LoadDefault（读取 ~/.jot/mcp/mcp-servers.json）。
	MCPServerConfigPath string
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

	// 追加外部 MCP 服务器工具（测试阶段，配置文件驱动）：读取 ~/.jot/mcp/mcp-servers.json，
	// 对每个 enabled 服务器连接并发现工具，改名后并入 toolList（须在 toolByName 索引构建之前）；
	// 配置缺失 / 单服务器失败仅记录日志跳过，不中断内置工具与整体 Agent 运行
	mcpPath := s.deps.MCPServerConfigPath
	var mcpCfg *mcpserver.Config
	if mcpPath == "" {
		// 默认读取用户家目录 ~/.jot/mcp/mcp-servers.json（可由 Deps.MCPServerConfigPath 覆盖）
		mcpCfg, err = mcpserver.LoadDefault()
	} else {
		mcpCfg, err = mcpserver.Load(mcpPath)
	}
	if err != nil {
		// 默认路径下失败：家目录路径解析失败属系统级异常，直接报错退出；
		// 路径可解析而配置不可用（缺失/损坏）时回填路径供日志，跳过 MCP 装配不阻断对话
		if mcpPath == "" {
			p, pErr := mcpserver.DefaultConfigPath()
			if pErr != nil {
				return result, fmt.Errorf("解析 MCP 默认配置路径失败: %w", pErr)
			}
			mcpPath = p
		}
		if s.deps.Logger != nil {
			s.deps.Logger.Debugw("MCP 服务器配置不可用，跳过 MCP 工具装配",
				fastlog.String("path", mcpPath),
				fastlog.Error(err))
		}
	} else {
		// 单条服务器校验失败时该条被跳过，其余合法条目正常装配；逐条输出告警便于定位
		if s.deps.Logger != nil {
			for _, loadErr := range mcpCfg.LoadErrors {
				s.deps.Logger.Warnw("MCP 服务器配置校验失败，该服务器已跳过",
					fastlog.String("path", mcpPath),
					fastlog.Error(loadErr))
			}
		}
		enabledServers := mcpCfg.EnabledServers()
		if len(enabledServers) == 0 && s.deps.Logger != nil {
			s.deps.Logger.Debugw("MCP 配置无启用的服务器，跳过 MCP 工具装配",
				fastlog.String("path", mcpPath))
		}

		// 会话统一收集，待本轮 Run 结束后统一关闭（避免 for 循环内 defer 可读性隐患，
		// 且覆盖其后所有 return 路径）；关闭失败记 Warn 便于发现连接清理异常
		var mcpSessions []*mcpserver.Session
		defer func() {
			for _, sess := range mcpSessions {
				if err := sess.Close(); err != nil && s.deps.Logger != nil {
					s.deps.Logger.Warnw("MCP 会话关闭失败",
						fastlog.String("server", sess.ServerName),
						fastlog.Error(err))
				}
			}
		}()

		for _, server := range enabledServers {
			connStart := time.Now()
			sess, err := mcpserver.OpenSession(ctx, server)
			if err != nil {
				if s.deps.Logger != nil {
					s.deps.Logger.Warnw("MCP 服务器连接失败，跳过该服务器",
						fastlog.String("server", server.Name),
						fastlog.Int("duration_ms", int(time.Since(connStart).Milliseconds())),
						fastlog.Error(err))
				}
				continue
			}
			mcpSessions = append(mcpSessions, sess)
			if sess.Skipped > 0 && s.deps.Logger != nil {
				s.deps.Logger.Warnw("部分 MCP 工具因 Info 解析失败被跳过",
					fastlog.String("server", server.Name),
					fastlog.Int("skipped", sess.Skipped))
			}
			var toolNames []string
			for _, t := range sess.Tools {
				invokable, ok := t.(tool.InvokableTool)
				if !ok {
					if s.deps.Logger != nil {
						s.deps.Logger.Warnw("MCP 工具不支持执行，已跳过",
							fastlog.String("server", server.Name))
					}
					continue
				}
				// 取改名后的工具名（mcp_{服务器名}_{工具名}），供 WrapWithError 日志与调用记录使用
				mcpToolName := server.Name
				if info, err := t.Info(ctx); err == nil && info != nil {
					mcpToolName = info.Name
				}
				toolNames = append(toolNames, mcpToolName)
				toolList = append(toolList, tools.WrapWithError(mcpToolName, invokable, toolCtx))
			}
			// 上线日志：记录本服务器装配完成的 MCP 工具（改名后名称）与连接耗时，
			// 便于排查工具是否生效及定位慢服务器
			if s.deps.Logger != nil {
				s.deps.Logger.Infow("MCP 服务器工具已上线",
					fastlog.String("server", server.Name),
					fastlog.Int("count", len(toolNames)),
					fastlog.String("tools", strings.Join(toolNames, ", ")),
					fastlog.Int("duration_ms", int(time.Since(connStart).Milliseconds())))
			}
		}
	}

	// 按工具名索引已装配的工具：emitToolStart 时据此查找 ActionTextProvider，
	// 为 tool_start 事件生成动作文案（action_text）随事件下发前端
	toolByName := make(map[string]tool.BaseTool, len(toolList))
	for _, t := range toolList {
		if info, err := t.Info(ctx); err == nil && info != nil {
			toolByName[info.Name] = t
		}
	}

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
		MaxIterations: MaxIterations,
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
	// ask_user 反向提问兜底：该轮正文（问句）在最终回答为空时作为 finalContent，
	// 保证落库内容为问句、历史回放可读
	var pendingQuestion string

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
						emitToolStart(emit, &toolRecords, tc, toolByName)
					}
					// ask_user 反向提问：该轮正文（问句）登记为兜底，供最终回答为空时落库；
					// 正文为空（模型未遵守"正文写问句"约束）时退而取工具参数里的 question
					for _, tc := range full.ToolCalls {
						if tc.Function.Name != "ask_user" {
							continue
						}
						if full.Content != "" {
							pendingQuestion = full.Content
						} else if q := askUserQuestionFromArgs(tc.Function.Arguments); q != "" {
							pendingQuestion = q
						}
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
						emitToolStart(emit, &toolRecords, tc, toolByName)
					}
					// ask_user 反向提问：该轮正文（问句）登记为兜底，供最终回答为空时落库；
					// 正文为空（模型未遵守"正文写问句"约束）时退而取工具参数里的 question
					for _, tc := range mv.Message.ToolCalls {
						if tc.Function.Name != "ask_user" {
							continue
						}
						if mv.Message.Content != "" {
							pendingQuestion = mv.Message.Content
						} else if q := askUserQuestionFromArgs(tc.Function.Arguments); q != "" {
							pendingQuestion = q
						}
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

	// ask_user 反向提问兜底：最终回答为空时用该轮正文问句作为 finalContent，
	// 保证落库内容为问句、历史回放可读
	if finalContent == "" && pendingQuestion != "" {
		finalContent = pendingQuestion
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
			fastlog.Int("max_iterations", MaxIterations))
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
// toolByName 供按工具名查找 ActionTextProvider 生成动作文案，随事件下发前端。
func emitToolStart(emit EmitFn, records *[]tools.Record, tc schema.ToolCall, toolByName map[string]tool.BaseTool) {
	rec := tools.Record{
		Action: "tool_start",
		Name:   tc.Function.Name,
		Args:   tools.TruncateRunes(tc.Function.Arguments, tools.MaxResultLen),
	}
	if t, ok := toolByName[tc.Function.Name]; ok {
		if p, ok := t.(tools.ActionTextProvider); ok {
			rec.ActionText = p.ActionText(tc.Function.Arguments)
		}
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

// askUserQuestionFromArgs 从 ask_user 工具调用参数中提取 question（问句兜底用）：
// 模型未遵守"正文写问句"约束（正文为空）时，以参数里的 question 作为落库兜底，
// 保证历史回放时 assistant 消息仍为可读的问句。解析失败返回空串。
func askUserQuestionFromArgs(argumentsJSON string) string {
	if argumentsJSON == "" {
		return ""
	}
	var args struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return ""
	}
	return strings.TrimSpace(args.Question)
}
