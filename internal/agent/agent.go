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
	"strconv"
	"strings"
	"sync"
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
	"gorm.io/gorm"
)

// DefaultMaxIterations 限制 ReAct 循环最大迭代次数，防止死循环（未配置 ai_agent_max_iterations 时的默认值，供装配与日志引用）。
const DefaultMaxIterations = 20

// MaxCachedSessions 会话级 Agent 实例缓存上限（LRU 淘汰），防止注册表无限增长。
const MaxCachedSessions = 32

// Deps AgentService 依赖注入。
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
	// MCPServerDB 外部 MCP 服务器配置的数据来源（数据库驱动）；
	// 为 nil 时跳过 MCP 工具装配。
	MCPServerDB *gorm.DB
	// MCPPool 全局 MCP 连接池（http/sse/stdio 预热复用）；为 nil 时跳过 MCP 工具装配。
	MCPPool *mcpserver.Pool
	// GetEmbedConfig 复用 app.go 现有逻辑：读取量化连接（ai_embed_* 三键），apiKey 已解码。
	GetEmbedConfig func() (baseURL, apiKey, model string, err error)
}

// AgentService 封装 Agent 对话链路。
type AgentService struct {
	deps Deps
	// 会话级 Agent 实例注册表：以 AI 会话 ID（models.AISession.ID）为键，
	// 保存每个会话的交互状态（ask_user 同轮等待通道、run 取消源）与缓存的
	// ChatModel 客户端。同一会话的连续消息复用同一实例，实现"按会话保持
	// 一个 Agent 实例"，并支撑 ask_user 反问在同轮内暂停/续答。
	mu       sync.Mutex
	sessions map[uint]*agentSession
}

// agentSession 一个 AI 会话对应的 Agent 交互实例。
// 说明：eino 的 ChatModelAgent 图/runner 与工具链按消息重建——系统提示词
// （Instruction）逐消息组装（技能/引用/意图裁剪不同），因此会话级实例持有的是
// 跨消息可复用的部分：ask_user 等待通道、run 取消源、ChatModel 客户端
// （指纹不变即复用）。MCP 连接由全局预热池（Deps.MCPPool）持有，跨会话复用，
// 不在会话级实例中管理。真正支撑"同轮续答"的是 askCh/askPending：
// ask_user 工具在 ReAct 循环内阻塞等待，AnswerAskUser 把用户答案投递到通道，
// 循环恢复继续完成原始请求（答案不落库为新用户消息）。
type agentSession struct {
	askCh      chan string        // 反问答案投递通道（容量 1，等待期间容纳一次回答）
	askPending bool               // 当前是否有反问在等待用户回答
	askMu      sync.Mutex         // 保护 askPending（AnswerAskUser 与工具侧并发访问）
	runMu      sync.Mutex         // 同一会话 run 串行化（等待中的 run 结束前不启动新 run）
	runCancel  context.CancelFunc // 当前 run 的取消源（会话释放时取消等待中的 run）
	cancelMu   sync.Mutex         // 保护 runCancel 的读写
	chatModel  *openai.ChatModel  // 缓存的 ChatModel 客户端（跨消息复用）
	chatFP     string             // ChatModel 指纹（BaseURL/APIKey/Model/深度思考）
	lastSeen   time.Time          // 最近使用时间（LRU 淘汰依据）
}

// setRunCancel 记录当前 run 的取消源（Run 内赋值，runMu 已串行化，加锁兜底并发读）。
func (sess *agentSession) setRunCancel(c context.CancelFunc) {
	sess.cancelMu.Lock()
	sess.runCancel = c
	sess.cancelMu.Unlock()
}

// cancelRun 取消当前 run（幂等：cancel 为 nil 或已取消时无副作用）。
func (sess *agentSession) cancelRun() {
	sess.cancelMu.Lock()
	c := sess.runCancel
	sess.cancelMu.Unlock()
	if c != nil {
		c()
	}
}

// ClaimAsk 实现 tools.AskWaiter：在发射 ai:ask-user 事件前原子抢占反问名额。
// 已有反问在等待（模型同条消息并行发多条 ask_user）时返回错误，
// 工具据此拒绝阻塞，避免多个等待者共抢一个通道导致整轮挂起。
func (sess *agentSession) ClaimAsk() error {
	sess.askMu.Lock()
	defer sess.askMu.Unlock()
	if sess.askPending {
		return errors.New("已有反问正在等待你的回答，请先回答当前问题，不要重复提问")
	}
	sess.askPending = true
	return nil
}

// WaitForAnswer 实现 tools.AskWaiter：阻塞等待用户对当前反问的回答。
// 工具已在 emit 前经 ClaimAsk 抢占名额，此处幂等置位兜底；
// 收到答案或 ctx 取消（停止按钮/会话释放）后清除标记。
// 答案经通道同轮返回给 ask_user 工具。
func (sess *agentSession) WaitForAnswer(ctx context.Context) (string, error) {
	sess.askMu.Lock()
	sess.askPending = true
	sess.askMu.Unlock()
	defer func() {
		sess.askMu.Lock()
		sess.askPending = false
		sess.askMu.Unlock()
	}()
	select {
	case ans := <-sess.askCh:
		return ans, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// drainAsk 排空通道中未消费的反问答案（取消竞态残留），防止污染下一轮反问。
// 场景：用户提交答案的同时取消（停止/会话释放），工具 select 可能抢到 ctx.Done()
// 而非通道，已投递的答案残留；不排空会被下一轮 ask_user 当作本轮的答案消费。
func (sess *agentSession) drainAsk() {
	for {
		select {
		case <-sess.askCh:
		default:
			return
		}
	}
}

// NewAgentService 创建一个新的 AgentService 实例。
func NewAgentService(deps Deps) *AgentService {
	return &AgentService{deps: deps, sessions: make(map[uint]*agentSession)}
}

// getOrCreateSession 取或建指定会话的 Agent 交互实例（LRU 淘汰超限缓存）。
func (s *AgentService) getOrCreateSession(sessionID uint) *agentSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[sessionID]; ok {
		sess.lastSeen = time.Now()
		return sess
	}
	if len(s.sessions) >= MaxCachedSessions {
		// 淘汰最久未使用的空闲会话（仅无等待中反问的实例可被淘汰）
		var oldestID uint
		var oldest time.Time
		for id, ss := range s.sessions {
			ss.askMu.Lock()
			pending := ss.askPending
			ss.askMu.Unlock()
			if pending {
				continue
			}
			if oldest.IsZero() || ss.lastSeen.Before(oldest) {
				oldest, oldestID = ss.lastSeen, id
			}
		}
		if oldestID != 0 {
			delete(s.sessions, oldestID)
		}
	}
	sess := &agentSession{
		askCh:    make(chan string, 1),
		lastSeen: time.Now(),
	}
	s.sessions[sessionID] = sess
	return sess
}

// AnswerAskUser 投递用户对 ask_user 反问的回答，恢复同一轮 ReAct 循环：
// 答案作为 ask_user 工具结果返回给模型，继续完成原始请求（同轮传输，
// 不落库为新用户消息、不新开一轮）。无等待中的反问时返回中文错误。
func (s *AgentService) AnswerAskUser(sessionID uint, answer string) error {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return errors.New("当前没有等待回答的问题（会话无进行中的 Agent 轮次）")
	}
	sess.askMu.Lock()
	if !sess.askPending {
		sess.askMu.Unlock()
		return errors.New("当前没有等待回答的问题")
	}
	sess.askPending = false // 投递前先清标记，防止重复投递
	sess.askMu.Unlock()
	select {
	case sess.askCh <- answer:
		return nil
	default:
		// 通道已满（极罕见：上一轮答案未消费），兜底拒绝
		return errors.New("回答投递失败，请稍后重试")
	}
}

// ReleaseSession 释放指定会话的 Agent 实例：取消等待中的 run 并删除注册表项。
// 清空/删除会话、重建服务时调用，防止等待中的反问 run 悬挂占用资源。
func (s *AgentService) ReleaseSession(sessionID uint) {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	if !ok {
		return
	}
	sess.cancelRun()
	sess.drainAsk()
}

// ReleaseAll 释放全部会话的 Agent 实例（清空所有 AI 会话/工厂重置时调用）：
// 取消所有等待中的 run 并清空注册表，防止暂停中的反问 goroutine 泄漏。
func (s *AgentService) ReleaseAll() {
	s.mu.Lock()
	sessions := s.sessions
	s.sessions = make(map[uint]*agentSession)
	s.mu.Unlock()
	for _, sess := range sessions {
		sess.cancelRun()
		sess.drainAsk()
	}
}

// Run 执行一轮 Agent 对话：
//  1. 从现有 AI 配置（aiService.GetConfig）读取 BaseURL/APIKey/Model，构建 OpenAI 兼容 ChatModel；
//  2. 以 req.Instruction 作为系统提示词（Agent Instruction），历史消息转 schema.Message；
//  3. 注册 recall_notes 等内置工具（notebook 过滤绑定 req.RecallNotebookIDs）；
//  4. runner.Run 消费事件流，流式文本与工具状态通过 emit 推送，最后汇总回答与工具摘要。
//
// ctx 支持取消：调用方传入带 cancel 的 ctx（停止按钮），Agent 循环随 ctx 终止，返回 ctx.Err()。
// 会话释放（ReleaseSession）会独立取消由 ctx 派生的 runCtx，同样以取消语义终止。
// 模型不支持 tool calling 或调用失败的报错原样返回 error，由调用方 ClassifyError。
// ask_user 反问：本方法在工具注入的 AskWaiter（会话实例）上阻塞等待用户回答，
// 答案经 AnswerAskUser 投递后同一轮 ReAct 循环继续（AI 消息不结束，同轮续答）。
func (s *AgentService) Run(ctx context.Context, req Request, emit EmitFn) (Result, error) {
	var result Result
	if emit == nil {
		emit = func(string, string) {}
	}

	// 读取配置的最大迭代次数（默认 20），防止 ReAct 循环死循环
	maxIterations := DefaultMaxIterations
	if s.deps.Setting != nil {
		if n, err := strconv.Atoi(s.deps.Setting.Get("ai_agent_max_iterations")); err == nil && n > 0 {
			maxIterations = n
		}
	}

	// 深度研究技能：临时提升迭代次数至200（若当前设置小于200）
	for _, skillID := range req.SkillIDs {
		if skillID == "skill_deep_research" {
			if maxIterations < 200 {
				maxIterations = 200
			}
			break
		}
	}

	// 0. 会话级 Agent 实例：按会话 ID 取/建交互实例（ask_user 同轮等待通道、
	//     run 取消源、ChatModel 客户端缓存），同一会话连续消息复用；
	//     runMu 串行化同一会话的 run（等待中的 run 结束前不启动新 run）。
	//     本 run 使用从 ctx 派生的 runCtx，ReleaseSession 可独立取消它。
	sess := s.getOrCreateSession(req.SessionID)
	sess.runMu.Lock()
	defer sess.runMu.Unlock()
	runCtx, runCancel := context.WithCancel(ctx)
	sess.setRunCancel(runCancel)
	defer func() {
		// 清理会话级状态：清除反问等待标记、排空未消费的答案（取消竞态残留）、
		// 取消本 run 的取消源（幂等）
		sess.askMu.Lock()
		sess.askPending = false
		sess.askMu.Unlock()
		sess.drainAsk()
		sess.setRunCancel(nil)
		runCancel()
	}()

	// 1. 读取 AI 配置（复用现有 GetConfig 逻辑，含 B64 解码）
	aiCfg := s.deps.AI.GetConfig()
	if aiCfg.BaseURL == "" || aiCfg.APIKey == "" || aiCfg.Model == "" {
		return result, errors.New("请先配置 AI 服务（BaseURL / APIKey / Model）")
	}

	// 2. 构建/复用 ChatModel（OpenAI 兼容协议，BaseURL 指向 DeepSeek/通义等兼容端点）。
	//    会话级缓存：BaseURL/APIKey/Model/深度思考 指纹不变时复用同一实例。
	chatFP := fmt.Sprintf("%s|%s|%s|%v", aiCfg.BaseURL, aiCfg.APIKey, aiCfg.Model, req.ThinkingEnabled)
	if sess.chatModel == nil || sess.chatFP != chatFP {
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
		chatModel, err := openai.NewChatModel(runCtx, chatModelCfg)
		if err != nil {
			return result, fmt.Errorf("创建 ChatModel 失败: %w", err)
		}
		sess.chatModel = chatModel
		sess.chatFP = chatFP
	}
	chatModel := sess.chatModel

	// 3. 统一装配并注册工具（registry.go 的 buildTools，notebook 过滤在构造 recall_notes 时绑定）
	//    collector/ctx 贯穿本轮：工具经 tools.WrapWithError 包装（失败回填模型不中断循环），
	//    部分失败由工具经 ctx.AddPartial 登记，tool_result 之后统一 DrainPartials 发射；
	//    AskWaiter 注入会话实例：ask_user 工具调用时阻塞等待用户回答（同轮续答）。
	collector := &tools.Collector{}
	var toolRecords []tools.Record
	toolCtx := &tools.Context{Emit: emit, Records: &toolRecords, Collector: collector, Logger: s.deps.Logger, AskWaiter: sess}
	// 禁用工具集合转 map（黑名单语义：默认空 = 全部注册，被禁工具模型不可见）
	disabledTools := make(map[string]bool, len(req.DisabledTools))
	for _, name := range req.DisabledTools {
		disabledTools[name] = true
	}
	toolList := buildTools(BuildParams{deps: s.deps, req: req, ctx: toolCtx}, disabledTools)

	// 追加外部 MCP 服务器工具（数据库驱动 + 全局预热池）：
	// 从数据库读取 MCP 服务器配置，全部传输（http/sse/stdio）统一优先复用预热池连接
	// （Pool.Session 零网络开销），池中未命中（预热失败/从未预热）时现场连接一次并缓存
	// 入池（WarmupOne 兜底）；连接由池持有常驻，跨会话跨消息复用。
	// 查询失败 / 空库仅记录日志跳过，不中断内置工具与整体 Agent 运行
	if s.deps.MCPServerDB != nil && s.deps.MCPPool != nil {
		mcpCfg, err := mcpserver.LoadFromDB(s.deps.MCPServerDB)
		if err != nil {
			// 查询失败（数据库异常）仅记录日志，跳过 MCP 装配不阻断对话
			if s.deps.Logger != nil {
				s.deps.Logger.Debugw("MCP 服务器配置读取失败，跳过 MCP 工具装配",
					fastlog.Error(err))
			}
		} else if len(mcpCfg.Servers) == 0 {
			// 空库：无任何 MCP 服务器记录，跳过装配
			if s.deps.Logger != nil {
				s.deps.Logger.Debugw("无启用的 MCP 服务器，跳过 MCP 工具装配")
			}
		} else {
			// 单条服务器校验失败时该条被跳过，其余合法条目正常装配；逐条输出告警便于定位
			if s.deps.Logger != nil {
				for _, loadErr := range mcpCfg.LoadErrors {
					s.deps.Logger.Warnw("MCP 服务器配置校验失败，该服务器已跳过",
						fastlog.Error(loadErr))
				}
			}
			enabledServers := mcpCfg.EnabledServers()
			if len(enabledServers) == 0 && s.deps.Logger != nil {
				s.deps.Logger.Debugw("MCP 配置无启用的服务器，跳过 MCP 工具装配")
			}

			// 并行取/建会话，串行处理结果（保持工具顺序与日志顺序稳定）：
			// 未命中池时现场建连（WarmupOne），goroutine 并发最多 3 台
			// （stdio 为本地子进程，限制并发拉起进程数）；
			// 每台内部已有 10s 连接 + 10s 工具发现超时兜底，goroutine 不会永久挂起。
			type mcpResult struct {
				server   mcpserver.Server
				sess     *mcpserver.Session
				err      error
				duration time.Duration
			}
			results := make([]mcpResult, len(enabledServers))
			sem := make(chan struct{}, 3)
			var wg sync.WaitGroup
			for i, server := range enabledServers {
				wg.Add(1)
				go func(i int, server mcpserver.Server) {
					defer wg.Done()
					sem <- struct{}{} // 获取并发槽位
					defer func() { <-sem }()
					connStart := time.Now()
					// 优先复用预热池；未命中则现场连接并入池（兜底）
					mcpSess := s.deps.MCPPool.Session(server.Name)
					var err error
					if mcpSess == nil {
						mcpSess, err = s.deps.MCPPool.WarmupOne(runCtx, server)
					}
					results[i] = mcpResult{server: server, sess: mcpSess, err: err, duration: time.Since(connStart)}
				}(i, server)
			}
			wg.Wait()

			// 按索引顺序串行处理结果：日志输出与工具装配顺序和串行实现完全一致
			for i := range results {
				r := results[i]
				if r.err != nil {
					if s.deps.Logger != nil {
						s.deps.Logger.Warnw("MCP 服务器连接失败，跳过该服务器",
							fastlog.String("server", r.server.Name),
							fastlog.Int("duration_ms", int(r.duration.Milliseconds())),
							fastlog.Error(r.err))
					}
					continue
				}
				// 连接由池持有常驻，本轮不关闭
				if r.sess.Skipped > 0 && s.deps.Logger != nil {
					s.deps.Logger.Warnw("部分 MCP 工具因 Info 解析失败被跳过",
						fastlog.String("server", r.server.Name),
						fastlog.Int("skipped", r.sess.Skipped))
				}
				var toolNames []string
				for _, t := range r.sess.Tools {
					invokable, ok := t.(tool.InvokableTool)
					if !ok {
						if s.deps.Logger != nil {
							s.deps.Logger.Warnw("MCP 工具不支持执行，已跳过",
								fastlog.String("server", r.server.Name))
						}
						continue
					}
					// 取改名后的工具名（mcp_{服务器名}_{工具名}），供 WrapWithError 日志与调用记录使用
					mcpToolName := r.server.Name
					if info, err := t.Info(runCtx); err == nil && info != nil {
						mcpToolName = info.Name
					}
					// 检查是否在禁用名单中：被禁工具跳过注册，模型不可见也不可调用
					if disabledTools[mcpToolName] {
						continue
					}
					toolNames = append(toolNames, mcpToolName)
					toolList = append(toolList, tools.WrapWithError(mcpToolName, invokable, toolCtx))
				}
				// 上线日志：记录本服务器装配完成的 MCP 工具（改名后名称）与取/建会话耗时，
				// 便于排查工具是否生效及定位慢服务器（池复用场景耗时接近 0）
				if s.deps.Logger != nil {
					s.deps.Logger.Infow("MCP 服务器工具已上线",
						fastlog.String("server", r.server.Name),
						fastlog.Int("count", len(toolNames)),
						fastlog.String("tools", strings.Join(toolNames, ", ")),
						fastlog.Int("duration_ms", int(r.duration.Milliseconds())))
				}
			}
		}
	}

	// 按工具名索引已装配的工具：emitToolStart 时据此查找 ActionTextProvider，
	// 为 tool_start 事件生成动作文案（action_text）随事件下发前端
	toolByName := make(map[string]tool.BaseTool, len(toolList))
	for _, t := range toolList {
		if info, err := t.Info(runCtx); err == nil && info != nil {
			toolByName[info.Name] = t
		}
	}

	// 4. 组装 ChatModelAgent（内部是 ReAct 循环：模型决策 → 调用工具 → 反馈 → 继续）
	//    Instruction 作为 system 消息由默认 GenModelInput 放在最前
	agent, err := adk.NewChatModelAgent(runCtx, &adk.ChatModelAgentConfig{
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
	runner := adk.NewRunner(runCtx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 6. 消费事件流
	messages := buildMessages(req)
	iter := runner.Run(runCtx, messages)

	var finalContent string
	var promptTokensTotal, completionTokensTotal int
	// ask_user 反向提问兜底：该轮正文（问句）在最终回答为空时作为 finalContent，
	// 保证落库内容为问句、历史回放可读
	var pendingQuestion string
	// 本轮所有 assistant 消息的流式正文累计（与前端气泡所见一致）：
	// ask_user 同轮续答时，最终回答只是末条 assistant 消息，问句落在中间轮，
	// 仅存 finalContent 会丢失问句；故反问轮用累计正文落库（问句 + 续答全文）
	var streamedContent string
	var askedUser bool

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			if runCtx.Err() != nil {
				return result, runCtx.Err()
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
					if runCtx.Err() != nil {
						return result, runCtx.Err()
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
				// 流式正文累计（与前端气泡所见一致），供 ask_user 同轮续答时整轮落库
				streamedContent += full.Content
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
						askedUser = true
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
				// 流式正文累计（与前端气泡所见一致），供 ask_user 同轮续答时整轮落库
				streamedContent += mv.Message.Content
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
						askedUser = true
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
			var callID string
			if mv.IsStreaming {
				content, callID = consumeToolStream(mv.MessageStream)
			} else if mv.Message != nil {
				content = mv.Message.Content
				callID = mv.Message.ToolCallID
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
				emitToolResult(emit, &toolRecords, name, callID, content)
				// 部分失败提示（如多来源搜索部分来源失败）：工具内部经 ctx.AddPartial 登记，
				// 此处统一在 tool_result 之后发射 tool_partial，前端展示 ⚠️ 警告；发射后清空防重复
				toolCtx.DrainPartials(name, callID)
			}
		}
	}

	// 用户取消：Agent 循环随 runCtx 终止（父 ctx 取消或 ReleaseSession 独立取消）
	if runCtx.Err() != nil {
		return result, runCtx.Err()
	}

	// ask_user 反向提问兜底：最终回答为空时用该轮正文问句作为 finalContent，
	// 保证落库内容为问句、历史回放可读
	if finalContent == "" && pendingQuestion != "" {
		finalContent = pendingQuestion
	}
	// ask_user 同轮续答：问句在中间轮、最终回答在末轮，仅存 finalContent 会丢失问句；
	// 用本轮全部流式正文（问句 + 续答）落库，与前端同一气泡展示一致
	if askedUser && streamedContent != "" {
		finalContent = streamedContent
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

// consumeToolStream 消费工具结果流式消息，返回合并后的完整文本与 tool_call ID
// （ID 取自合并消息的 ToolCallID，供前端按调用精确定位）。
func consumeToolStream(stream *schema.StreamReader[*schema.Message]) (string, string) {
	if stream == nil {
		return "", ""
	}
	var chunks []*schema.Message
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", ""
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) == 0 {
		return "", ""
	}
	full, err := schema.ConcatMessages(chunks)
	if err != nil || full == nil {
		return "", ""
	}
	return full.Content, full.ToolCallID
}

// emitToolStart 推送工具调用开始事件，并记录到工具调用摘要。
// toolByName 供按工具名查找 ActionTextProvider 生成动作文案，随事件下发前端。
func emitToolStart(emit EmitFn, records *[]tools.Record, tc schema.ToolCall, toolByName map[string]tool.BaseTool) {
	rec := tools.Record{
		Action: "tool_start",
		Name:   tc.Function.Name,
		CallID: tc.ID, // eino 原生 tool_call ID，供前端按调用精确定位（同轮多条同名调用）
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
// callID 为该次调用的 eino tool_call ID（来自 Tool 结果事件的 ToolCallID），
// 与 emitToolStart 记录的 CallID 配对，保证同轮多条同名调用前端精确归属。
// 若该次调用刚失败（已发射 tool_error），则跳过结果事件，保持失败态不被 ✓ 覆盖。
func emitToolResult(emit EmitFn, records *[]tools.Record, name, callID, result string) {
	// 反向查找与本次调用同 CallID 的记录（CallID 为空时退化为按工具名匹配，兼容旧数据）
	matched := false
	skip := false
	for i := len(*records) - 1; i >= 0; i-- {
		rec := &(*records)[i]
		if callID != "" {
			if rec.CallID != callID {
				continue
			}
			matched = true
		} else {
			if rec.Name != name {
				continue
			}
			matched = true
		}
		if rec.Action == "tool_error" {
			skip = true
		}
		break
	}
	if matched && skip {
		return
	}
	rec := tools.Record{
		Action: "tool_result",
		Name:   name,
		CallID: callID,
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
