package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// 说明：
// Eino 的 tool.BaseTool 接口只要求实现 Info() 返回工具元信息；
// 要能被真正执行，还需实现 InvokableRun —— 即完整实现 tool.InvokableTool
// （内嵌 BaseTool + InvokableRun），两个工具都按此形态手写实现。

// getCurrentTimeTool 查询当前时间的演示工具。
// 演示环境不连接真实时钟，返回确定性的假时间字符串。
type getCurrentTimeTool struct{}

// 编译期断言：确保 getCurrentTimeTool 实现了 tool.InvokableTool（含 BaseTool）。
var _ tool.InvokableTool = (*getCurrentTimeTool)(nil)

// Info 返回工具元信息（名称、描述、参数 JSON Schema），供模型决策是否调用。
func (g *getCurrentTimeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_current_time",
		Desc: "获取指定城市的当前日期和时间。当用户询问时间、日期或类似“现在几点”的问题时必须调用本工具。",
		// 参数 JSON Schema：手写 map 形式（也可用 eino-contrib/jsonschema 定义）
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {
				Type:     schema.String,
				Desc:     "城市名称，例如：北京、上海、纽约",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：入参是模型生成的 JSON 字符串，返回结果是给模型的文本。
func (g *getCurrentTimeTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		City string `json:"city"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 get_current_time 参数失败: %w", err)
	}
	if args.City == "" {
		args.City = "未知城市"
	}
	// 演示环境：返回确定性假时间，便于"看效果"
	return fmt.Sprintf("2026-08-09 14:30（北京时间，%s）", args.City), nil
}

// webSearchTool 网页搜索的演示工具（假实现，不发起真实网络请求）。
type webSearchTool struct{}

// 编译期断言：确保 webSearchTool 实现了 tool.InvokableTool（含 BaseTool）。
var _ tool.InvokableTool = (*webSearchTool)(nil)

// Info 返回工具元信息。
func (w *webSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "搜索互联网获取实时信息。当用户询问新闻、天气、最新动态等需要联网信息的问题时调用。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "搜索关键词",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行工具（假实现）。
func (w *webSearchTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 web_search 参数失败: %w", err)
	}
	// 演示环境：不接入真实搜索引擎
	return fmt.Sprintf("（演示模式）未接入真实搜索引擎，搜索「%s」的结果为空。", args.Query), nil
}
