package main

// 本文件实现"主 Agent 调度子 Agent（AgentTool）"的最小验证 demo。
//
// 验证目标（用真实 LLM 调用确认 Eino v0.9.13 的事件转发行为）：
//   1. 主 Agent 通过 adk.NewAgentTool 把子 Agent 包装成工具后，顶层事件流能看到什么；
//   2. 开启 EmitInternalEvents（-emit）后，子 Agent 内部工具调用是否被转发到顶层、
//      事件上 AgentName 字段是哪个 agent 的（用于区分来源）；
//   3. 关闭时，子 Agent 内部事件是否完全不出现（只看到主 Agent 对子 Agent 的一次调度）；
//   4. 用模拟 agent.go 的 toolRecords 收集逻辑验证：子 Agent 内部工具调用是否会
//      污染主 Agent 层的工具调用记录（Result.ToolCalls）。
//
// 结构：
//   - 主 Agent "main-agent"：持有一个原生工具 web_search + 一个 AgentTool（child-agent）
//   - 子 Agent "child-agent"：持有 get_current_time / mcp_order_query 两个内部工具
//
// 建议问题：
//   - "现在几点了"           → 主 Agent 应调度 child-agent → 内部调 get_current_time
//   - "帮我查订单 ORD-1001"   → 主 Agent 应调度 child-agent → 内部调 mcp_order_query
//   - "搜索一下今天的新闻"     → 主 Agent 直接调原生 web_search（不经子 Agent）

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// mainAgentInstruction 主 Agent 系统提示词：明示"何时直接调原生工具 / 何时调度子 Agent"。
const mainAgentInstruction = `你是一个乐于助人的中文助手，负责统一调度。
你有以下工具可用：
1. web_search：联网搜索实时信息（新闻、天气等）。用户需要联网信息时直接调用它，不需要经过其他智能体。
2. child-agent：一个负责"时间查询"与"订单数据查询"的子智能体。当用户询问当前时间/日期、或需要查询订单/交易数据时，必须调用 child-agent 来处理，把你收到的任务原样转给它。
请始终用中文回答用户的问题。调用工具后，根据工具返回内容给出最终回答。`

// childAgentInstruction 子 Agent 系统提示词：只负责自己的两个内部工具。
const childAgentInstruction = `你是一个专注于"时间与订单查询"的子智能体。
规则：
1. 用户询问当前时间、日期时，必须调用 get_current_time 工具。
2. 用户查询订单、交易数据时，必须调用 mcp_order_query 工具。
3. 调用工具获取结果后，直接把结果整理成中文返回给上层，不要调用其他工具，也不要多轮询问。
你只能使用上面这两个工具，不要假设还有其他工具存在。`

func main() {
	cfg := parseConfig()
	printConfigSummary(cfg)

	ctx := context.Background()

	// 1. 构建 ChatModel（OpenAI 兼容协议，主/子 Agent 复用同一实例）
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		Timeout: 60 * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 组装子 Agent（child-agent）：只持有自己的两个内部工具
	childTools := []tool.BaseTool{
		&getCurrentTimeTool{},
		&simulatedMCPQueryTool{},
	}
	childAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "child-agent",
		Description: "处理时间查询与订单数据查询的子智能体。当用户询问当前时间/日期，或需要查询订单/交易数据时，把任务交给它。",
		Instruction: childAgentInstruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: childTools,
			},
		},
		MaxIterations: 5, // 子 Agent 内部循环上限设小，防止嵌套死循环
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建子 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 3. 用 NewAgentTool 把子 Agent 包装成工具（工具名/描述取自子 Agent 的 Name/Description）
	childAgentTool := adk.NewAgentTool(ctx, childAgent)

	// 4. 组装主 Agent（main-agent）：原生工具 web_search + AgentTool(child-agent)
	mainTools := []tool.BaseTool{
		&webSearchTool{},
		childAgentTool,
	}
	mainAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "main-agent",
		Description: "一个能调度子智能体并调用原生工具回答问题的助手。",
		Instruction: mainAgentInstruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: mainTools,
			},
			// 验证点 2：开启后子 Agent 内部事件被转发到顶层事件流
			EmitInternalEvents: cfg.Emit,
		},
		MaxIterations: 10,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建主 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 5. 创建 Runner 并运行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           mainAgent,
		EnableStreaming: true,
	})

	// 入口：os.Args[1] 作为问题；无参数时打印使用说明
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		return
	}
	question := strings.Join(args, " ")
	fmt.Printf(">>> 用户问题: %s\n\n", question)

	iter := runner.Query(ctx, question)
	consumeEvents(iter)
	fmt.Println("\n>>> Agent 运行结束")
}

// consumeEvents 消费并打印顶层事件流的每个事件，同时模拟 agent.go 的 toolRecords 收集逻辑。
// 每个事件打印：AgentName（事件来源，主 Agent 还是子 Agent）/ Role / Action / ToolName。
func consumeEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) {
	var records []string // 模拟 agent.go 的 toolRecords（Result.ToolCalls 落库内容）

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			fmt.Printf("[错误事件] AgentName=%s err=%v\n", event.AgentName, event.Err)
			continue
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput

		// 事件头：来源 Agent + 角色 + Action（判空）
		head := fmt.Sprintf("[事件] AgentName=%-12s Role=%-11s", event.AgentName, mv.Role)
		if event.Action != nil {
			head += fmt.Sprintf(" Action=%+v", event.Action)
		}
		fmt.Println(head)

		switch mv.Role {
		case schema.Assistant:
			// 模型输出事件：可能是纯文本，也可能是工具调用决策
			if mv.IsStreaming {
				full := consumeAssistantStream(mv.MessageStream)
				if full == nil {
					continue
				}
				if len(full.ToolCalls) > 0 {
					// 模拟 agent.go 的 emitToolStart：按工具名记录 tool_start
					for _, tc := range full.ToolCalls {
						fmt.Printf("    └─ 工具调用决策: name=%s args=%s\n", tc.Function.Name, tc.Function.Arguments)
						records = append(records, fmt.Sprintf("[%s] tool_start %s", event.AgentName, tc.Function.Name))
					}
				} else if full.Content != "" {
					fmt.Printf("    └─ [文本] %s\n", preview(full.Content))
				}
			} else if mv.Message != nil {
				if len(mv.Message.ToolCalls) > 0 {
					for _, tc := range mv.Message.ToolCalls {
						fmt.Printf("    └─ 工具调用决策: name=%s args=%s\n", tc.Function.Name, tc.Function.Arguments)
						records = append(records, fmt.Sprintf("[%s] tool_start %s", event.AgentName, tc.Function.Name))
					}
				} else if mv.Message.Content != "" {
					fmt.Printf("    └─ [文本] %s\n", preview(mv.Message.Content))
				}
			}
		case schema.Tool:
			// 工具执行结果事件：模拟 agent.go 的 emitToolResult（toolRecords 记录 tool_result）
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
			fmt.Printf("    └─ 工具结果: tool=%s content=%s\n", name, preview(content))
			records = append(records, fmt.Sprintf("[%s] tool_result %s", event.AgentName, name))
		}
	}

	// 模拟 agent.go 汇总段：打印收集到的"工具调用链"（即 Result.ToolCalls 会落库的内容）
	fmt.Println("\n========== 模拟 agent.go 的 toolRecords 汇总（Result.ToolCalls 落库内容） ==========")
	if len(records) == 0 {
		fmt.Println("（空：本轮没有任何工具调用记录）")
	}
	for _, r := range records {
		fmt.Println("  " + r)
	}
	fmt.Println("========== 汇总结束 ==========")
}

// consumeAssistantStream 消费 assistant 流式消息：逐 chunk 实时打印文本，
// 流结束后用 ConcatMessages 合并，返回完整消息（含合并后的 ToolCalls），
// 用于判断本轮是文本还是工具调用决策。
func consumeAssistantStream(stream *schema.StreamReader[*schema.Message]) *schema.Message {
	if stream == nil {
		return nil
	}
	var chunks []*schema.Message
	fmt.Printf("    └─ [流式文本] ")
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("\n    [流错误] %v\n", err)
			return nil
		}
		if chunk != nil {
			if chunk.Content != "" {
				fmt.Print(chunk.Content)
			}
			chunks = append(chunks, chunk)
		}
	}
	fmt.Println()
	if len(chunks) == 0 {
		return nil
	}
	full, err := schema.ConcatMessages(chunks)
	if err != nil {
		fmt.Printf("    [合并失败] %v\n", err)
		return nil
	}
	return full
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

// preview 截断长文本（单行展示用），超长只保留前后片段。
func preview(s string) string {
	const max = 120
	if len(s) <= max {
		return s
	}
	return s[:60] + " …(截断)… " + s[len(s)-60:]
}

// printUsage 打印使用说明。
func printUsage() {
	fmt.Println("用法: go run . [-emit] \"<问题>\"")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go run . \"现在几点了\"")
	fmt.Println("  go run . \"帮我查订单 ORD-1001\"")
	fmt.Println("  go run . \"搜索一下今天的新闻\"")
	fmt.Println("  go run . -emit \"现在几点了\"   # 开启 EmitInternalEvents，观察子 Agent 内部事件转发")
	fmt.Println()
	fmt.Println("配置方式（优先级: 命令行参数 > 环境变量 > 默认值）:")
	fmt.Println("  -base-url / AGENT_DEMO_BASE_URL              API 端点，默认 https://api.deepseek.com/v1")
	fmt.Println("  -api-key  / AGENT_DEMO_API_KEY               API Key（必填）")
	fmt.Println("  -model    / AGENT_DEMO_MODEL                 模型名，默认 deepseek-chat")
	fmt.Println("  -emit     / AGENT_DEMO_EMIT_INTERNAL         是否开启子 Agent 内部事件转发，默认关闭")
}
