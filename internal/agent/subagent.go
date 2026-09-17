package agent

// 本文件定义「子 Agent」机制的通用部分：委托工具基类（delegatedAgentTool）+ 内层 Agent
// 构造工厂（newDelegatedAgentTool）+ 全工具构造器注册表（toolConstructors）+ 事件转发循环。
// 具体子 Agent 实例按「每 Agent 一个文件」分散，见 subagent_os.go（os_agent 样板）。
//
// 机制背景：父层只注册一个委托工具（如 os_agent），内层是独立 ChatModelAgent（复用同一
// 会话 *openai.ChatModel 客户端）；内层每步工具调用以现有 ai:tool-status 事件实时转发
// （写入父层同一 toolRecords 切片），前端状态条/历史明细以「委托工具 start → 内层记录 →
// 委托工具 result」为分组边界，审批语义沿用同一会话 Approver。
//
// 开发规范（新增子 Agent）：
//   - 通用机制（本文件，一般无需改动）：delegatedAgentTool 基类 + newDelegatedAgentTool 工厂 +
//     toolConstructors 构造器注册表 + 事件转发循环（InvokableRun）。
//   - 每个子 Agent 一个文件（如 subagent_os.go / subagent_note.go）：只定义差异部分——
//     系统提示词常量 + 白名单 + subAgentConfig 配置 + 一个构造器（内部调 newDelegatedAgentTool）。
//   - registry.go buildTools 注册对应委托工具：WrapWithError 包装、受 ai_agent_tools_disabled
//     过滤、chatModel 为 nil 时跳过（参考 subagent_os.go 的 buildOSSubAgent 与注册写法）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"jot/internal/agent/tools"

	"gitee.com/MM-Q/fastlog"
)

// subAgentConfig 子 Agent 的全部差异配置：一个域子 Agent 实例 = 一份配置 + 一次工厂调用。
type subAgentConfig struct {
	name          string   // 委托工具名（os_agent / note_agent …），也是内层 ChatModelAgent 名
	description   string   // 内层 ChatModelAgent Description
	instruction   string   // 内层系统提示词（角色 + 边界 + 审批说明 + 结果摘要要求）
	toolNames     []string // 内层白名单（按名从 toolConstructors 取构造器，顺序即注册顺序）
	maxIterations int      // 内层 ReAct 循环最大迭代次数（防死循环）
	actionPrefix  string   // ActionText 动作文案前缀（tool_start 展示）
	infoDesc      string   // Info().Desc（何时调用 / 做什么）
	requestDesc   string   // Info() request 参数描述
}

// toolConstructors 全工具构造器注册表：子 Agent 白名单按工具名取构造器。
// 新增工具需在此登记才能被子 Agent 白名单引用；一工具一构造器（与 tools 包规范一致）。
var toolConstructors = map[string]func(ctx *tools.Context) tool.InvokableTool{
	"read_file":   tools.NewReadFile,
	"write_file":  tools.NewWriteFile,
	"edit_file":   tools.NewEditFile,
	"ls_dir":      tools.NewLsDir,
	"glob":        tools.NewGlob,
	"grep_file":   tools.NewGrepFile,
	"copy_file":   tools.NewCopyFile,
	"move_file":   tools.NewMoveFile,
	"delete_file": tools.NewDeleteFile,
	"mkdir_dir":   tools.NewMkdirDir,
	"run_command": tools.NewRunCommand,
}

// delegatedAgentTool 通用子 Agent 委托工具：父层只见一个委托工具，内层为独立 ChatModelAgent。
// InvokableRun 时以内层 Agent 运行，并把内层每步工具调用实时转发到父层事件流
// （emitToolStart/emitToolResult 写入父层同一 toolRecords 切片）。
type delegatedAgentTool struct {
	agent      adk.Agent       // 内层 ChatModelAgent（可注入 fake 供测试）
	innerTools []tool.BaseTool // 内层白名单（已 WrapWithError 包装），供 toolByName 索引与测试断言
	ctx        *tools.Context  // 父层工具上下文：事件发射 / 调用记录 / 部分失败登记
	cfg        subAgentConfig  // 本委托工具的差异配置（Info/ActionText 参数化）
}

// 编译期断言：delegatedAgentTool 实现 tool.InvokableTool 与 tools.ActionTextProvider。
var _ tool.InvokableTool = (*delegatedAgentTool)(nil)
var _ tools.ActionTextProvider = (*delegatedAgentTool)(nil)

// newDelegatedAgentTool 通用子 Agent 构造器：
//   - chatModel 为 nil（未配置 AI）→ 记 Warn 日志并返回 nil（buildTools 过滤循环跳过该工具）。
//   - 白名单 cfg.toolNames 各从 toolConstructors 取构造器 + tools.WrapWithError 包装（与 registry.go 原写法一致）。
//   - disabled 参数当前不改变内层白名单（保留供将来扩展）。
//   - adk.NewChatModelAgent 构造失败记 Warn 日志返回 nil。
//
// 返回的 delegatedAgentTool 带 innerTools 字段供测试断言内层白名单。
func newDelegatedAgentTool(runCtx context.Context, chatModel *openai.ChatModel, innerCtx *tools.Context, disabled map[string]bool, cfg subAgentConfig) *delegatedAgentTool {
	logWarn := func(msg string, fields ...fastlog.Field) {
		if innerCtx != nil && innerCtx.Logger != nil {
			innerCtx.Logger.Warnw(msg, fields...)
		}
	}
	if chatModel == nil {
		logWarn("子 Agent 未装配：chatModel 为 nil（未配置 AI 服务）", fastlog.String("tool", cfg.name))
		return nil
	}
	// 内层白名单装配：按名取构造器 + WrapWithError 包装（失败回填模型不中断内层循环）
	innerTools := make([]tool.BaseTool, 0, len(cfg.toolNames))
	for _, name := range cfg.toolNames {
		ctor, ok := toolConstructors[name]
		if !ok {
			// 白名单登记了未注册构造器的工具名时跳过并记 Warn（TestBuildOSSubAgentInnerTools
			// 断言内层数量与白名单一致，漏配会失败，兜底发现）
			logWarn("内层白名单工具未登记构造器，已跳过", fastlog.String("tool", cfg.name), fastlog.String("toolName", name))
			continue
		}
		innerTools = append(innerTools, tools.WrapWithError(name, ctor(innerCtx), innerCtx))
	}
	// 内层 ChatModelAgent：与父层共用同一会话 ChatModel 客户端，独立 ReAct 循环
	agent, err := adk.NewChatModelAgent(runCtx, &adk.ChatModelAgentConfig{
		Name:        cfg.name,
		Description: cfg.description,
		Instruction: cfg.instruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: innerTools,
				UnknownToolsHandler: func(_ context.Context, name, _ string) (string, error) {
					return fmt.Sprintf("错误：工具 %q 不存在，请检查可用工具列表。", name), nil
				},
			},
		},
		MaxIterations: cfg.maxIterations,
	})
	if err != nil {
		logWarn("子 Agent 构造失败", fastlog.String("tool", cfg.name), fastlog.Error(err))
		return nil
	}
	return &delegatedAgentTool{agent: agent, innerTools: innerTools, ctx: innerCtx, cfg: cfg}
}

// Info 返回委托工具元信息：何时调用 / 做什么 / request 参数含义。
func (d *delegatedAgentTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: d.cfg.name,
		Desc: d.cfg.infoDesc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"request": {
				Type:     schema.String,
				Desc:     d.cfg.requestDesc,
				Required: true,
			},
		}),
	}, nil
}

// ActionText 提供 tool_start 动作文案（实现 tools.ActionTextProvider）：
// 解析 request，非空取前 40 rune 作动作文案（带 cfg.actionPrefix 前缀）；解析失败/为空返回 ""。
func (d *delegatedAgentTool) ActionText(argumentsInJSON string) string {
	var args struct {
		Request string `json:"request"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return ""
	}
	if r := strings.TrimSpace(args.Request); r != "" {
		return d.cfg.actionPrefix + tools.TruncateRunes(r, 40)
	}
	return ""
}

// InnerTools 返回内层白名单工具（已 WrapWithError 包装），供测试断言内层装配。
func (d *delegatedAgentTool) InnerTools() []tool.BaseTool {
	return d.innerTools
}

// InvokableRun 执行委托：以内层 Agent 运行一次 ReAct 循环，内层每步工具调用
// 以 ai:tool-status 事件实时转发（写入父层同一 toolRecords 切片）；内层流式正文与
// 思考链不转发（noopEmit），最终只回传内层最后一条非工具正文作为工具结果。
func (d *delegatedAgentTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 用户取消优先（父层事件循环随 ctx 终止）
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	// 解析 request：必填，为空直接报错（错误经父层 WrapWithError 回填模型）
	var args struct {
		Request string `json:"request"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 %s 参数失败: %w", d.cfg.name, err)
	}
	if strings.TrimSpace(args.Request) == "" {
		return "", fmt.Errorf("%s 参数缺少 request（要委托给子 Agent 的任务描述）", d.cfg.name)
	}
	// 按工具名索引内层白名单：emitToolStart 据此查找 ActionTextProvider 生成动作文案
	innerToolByName := make(map[string]tool.BaseTool, len(d.innerTools))
	for _, t := range d.innerTools {
		if info, err := t.Info(ctx); err == nil && info != nil {
			innerToolByName[info.Name] = t
		}
	}
	// 内层 Agent 以流式模式运行：正文/思考链消费但不转发（noopEmit）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: d.agent, EnableStreaming: true})
	iter := runner.Run(ctx, []*schema.Message{schema.UserMessage(args.Request)})

	noopEmit := func(string, string) {}
	// 跟踪本次内层发射且尚未收口的工具调用（callID→name）：内层执行中断时兜底补发
	// tool_result 收口，避免子步骤记录在父层状态条/回放中悬空为「执行中」。
	startedByCallID := map[string]string{}
	// closeStrayStarts 对尚未收口的内层工具调用补发 tool_result（仅处理带 callID 的调用；
	// callID 为空时无精确配对依据，保持现状不臆断补发）。
	closeStrayStarts := func(reason string) {
		for callID, name := range startedByCallID {
			emitToolResult(d.ctx.Emit, d.ctx.Records, name, callID, reason)
		}
	}
	var finalContent string
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
				return "", ctx.Err()
			}
			closeStrayStarts("（子 Agent 执行中止）")
			return "", fmt.Errorf("%s 子 Agent 执行失败: %w", d.cfg.name, event.Err)
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		switch mv.Role {
		case schema.Assistant:
			// 内层模型输出：只关心工具调用决策与最终正文，流式正文/思考链不转发
			if mv.IsStreaming {
				full, _, err := consumeAssistantStream(mv.MessageStream, noopEmit, false)
				if err != nil {
					if ctx.Err() != nil {
						return "", ctx.Err()
					}
					return "", fmt.Errorf("读取 %s 子 Agent 模型流失败: %w", d.cfg.name, err)
				}
				if full == nil {
					continue
				}
				if len(full.ToolCalls) > 0 {
					for _, tc := range full.ToolCalls {
						emitToolStart(d.ctx.Emit, d.ctx.Records, tc, innerToolByName)
						if tc.ID != "" {
							startedByCallID[tc.ID] = tc.Function.Name
						}
					}
				} else if full.Content != "" {
					// finalContent 语义 = spec「最后一条非工具正文」：内层若以工具调用收尾
					// （无收尾正文），此处保留上一条正文；全程无正文时由末尾兜底文案覆盖。
					finalContent = full.Content
				}
			} else if mv.Message != nil {
				if len(mv.Message.ToolCalls) > 0 {
					for _, tc := range mv.Message.ToolCalls {
						emitToolStart(d.ctx.Emit, d.ctx.Records, tc, innerToolByName)
						if tc.ID != "" {
							startedByCallID[tc.ID] = tc.Function.Name
						}
					}
				} else if mv.Message.Content != "" {
					finalContent = mv.Message.Content
				}
			}
		case schema.Tool:
			// 内层工具执行结果：实时转发 tool_result + 部分失败提示
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
			// 无条件收口（含空结果）：内层工具返回空串也发射 tool_result，
			// 保证子步骤 start 记录必有终态，不在状态条/回放中悬空为「执行中」。
			emitToolResult(d.ctx.Emit, d.ctx.Records, name, callID, content)
			if callID != "" {
				delete(startedByCallID, callID)
			}
			d.ctx.DrainPartials(name, callID)
		}
	}
	// 用户取消优先；finalContent 兜底
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if finalContent == "" {
		finalContent = "（子 Agent 未返回文本结果）"
	}
	return finalContent, nil
}
