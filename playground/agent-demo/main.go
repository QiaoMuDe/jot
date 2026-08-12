package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// agentInstruction 是 Agent 的系统提示词（中文）。
// 明确说明助手身份，以及什么场景下必须调用工具。
const agentInstruction = `你是一个乐于助人的中文助手。
当用户询问当前时间、日期、星期等需要时间信息的问题时，必须调用 get_current_time 工具获取答案；
当用户需要新闻、天气、最新动态等实时信息时，可以调用 web_search 工具。
请始终用中文回答用户的问题。`

func main() {
	cfg := parseConfig()
	printConfigSummary(cfg)

	ctx := context.Background()

	// 1. 构建 ChatModel（OpenAI 兼容协议，BaseURL 指向 DeepSeek/通义/Ollama 等端点）
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

	// 2. 组装 ChatModelAgent（内部是 ReAct 循环：模型决策 -> 调用工具 -> 反馈 -> 继续）
	tools := []tool.BaseTool{
		&getCurrentTimeTool{},
		&webSearchTool{},
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "jot-assistant",
		Description: "一个能调用工具回答问题的助手",
		Instruction: agentInstruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 20, // 限制 ReAct 循环最大迭代次数，防止死循环（与主项目 internal/agent.MaxIterations 保持一致）
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 ChatModelAgent 失败: %v\n", err)
		os.Exit(1)
	}

	// 3. 创建 Runner 并运行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true, // 开启流式输出
	})

	// 入口：os.Args[1] 作为问题；无参数时打印使用说明
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	question := os.Args[1]
	fmt.Printf(">>> 用户问题: %s\n\n", question)

	iter := runner.Query(ctx, question)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		handleEvent(event)
	}
	fmt.Println("\n>>> Agent 运行结束")
}

// handleEvent 消费并打印单个 Agent 事件的关键信息。
// 事件结构（以 v0.9.13 实际 API 为准）：
//
//	adk.AgentEvent{AgentName, Output *adk.AgentOutput{MessageOutput *adk.MessageVariant{...}}, Action, Err}
//	adk.MessageVariant{IsStreaming, Message *schema.Message, MessageStream *schema.StreamReader[*schema.Message], Role, ToolName}
func handleEvent(event *adk.AgentEvent) {
	if event.Err != nil {
		fmt.Printf("[错误] %v\n", event.Err)
		return
	}
	if event.Output == nil || event.Output.MessageOutput == nil {
		return
	}

	mv := event.Output.MessageOutput
	switch mv.Role {
	case schema.Assistant:
		// 模型输出事件：可能是文本，也可能是工具调用决策
		if mv.IsStreaming {
			consumeStream(mv.MessageStream, "[模型] ", true)
		} else if mv.Message != nil {
			printAssistantMessage(mv.Message)
		}
	case schema.Tool:
		// 工具执行结果事件
		if mv.IsStreaming {
			consumeStream(mv.MessageStream, "[工具] 结果: ", false)
		} else if mv.Message != nil {
			fmt.Printf("[工具] 结果: %s\n", mv.Message.Content)
		}
	}
}

// consumeStream 消费流式消息：文本块实时打印；流结束后合并检查工具调用。
// checkToolCalls 为 true 时（assistant 事件），流结束后用 schema.ConcatMessages
// 合并全部 chunk，若包含工具调用则打印"[工具] 调用 xxx，参数 {...}"。
func consumeStream(stream *schema.StreamReader[*schema.Message], prefix string, checkToolCalls bool) {
	if stream == nil {
		return
	}
	var chunks []*schema.Message
	printed := false // 前缀（如 "[模型] "）只在首个文本分片前打印一次，避免插在句子中间
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("[流错误] %v\n", err)
			return
		}
		if chunk != nil && chunk.Content != "" {
			if !printed {
				fmt.Print(prefix)
				printed = true
			}
			fmt.Print(chunk.Content) // 后续分片只打印内容
		}
		chunks = append(chunks, chunk)
	}
	fmt.Println()

	if !checkToolCalls || len(chunks) == 0 {
		return
	}
	// 合并流式 chunk（按 Index 合并 ToolCalls），还原完整消息
	full, err := schema.ConcatMessages(chunks)
	if err != nil || full == nil || len(full.ToolCalls) == 0 {
		return
	}
	for _, tc := range full.ToolCalls {
		fmt.Printf("[工具] 调用 %s，参数 %s\n", tc.Function.Name, tc.Function.Arguments)
	}
}

// printAssistantMessage 处理非流式 assistant 消息：区分普通文本与工具调用决策。
func printAssistantMessage(msg *schema.Message) {
	if msg == nil {
		return
	}
	if len(msg.ToolCalls) > 0 {
		// 模型决定调用工具
		for _, tc := range msg.ToolCalls {
			fmt.Printf("[工具] 调用 %s，参数 %s\n", tc.Function.Name, tc.Function.Arguments)
		}
		return
	}
	if msg.Content != "" {
		fmt.Printf("[模型] %s\n", msg.Content)
	}
}

// printUsage 打印使用说明。
func printUsage() {
	fmt.Println("用法: go run . \"<你的问题>\"")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go run . \"现在几点了\"")
	fmt.Println("  go run . \"帮我搜索今天的新闻\"")
	fmt.Println()
	fmt.Println("配置方式（优先级: 命令行参数 > 环境变量 > 默认值）:")
	fmt.Println("  -base-url / AGENT_DEMO_BASE_URL    API 端点，默认 https://api.deepseek.com/v1")
	fmt.Println("  -api-key  / AGENT_DEMO_API_KEY     API Key（必填）")
	fmt.Println("  -model    / AGENT_DEMO_MODEL       模型名，默认 deepseek-chat")
}
