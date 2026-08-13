package main

// 本文件定义验证 demo 用到的全部工具（均为假实现，不发起真实网络/数据库请求）：
//   - 主 Agent 原生工具：webSearchTool（直接注册给主 Agent）
//   - 子 Agent 内部工具：getCurrentTimeTool / simulatedMCPQueryTool（只注册给子 Agent）
//
// 通过"工具只存在于子 Agent 内部"这一事实，验证子 Agent 内部工具调用
// 是否会被转发到顶层事件流、以及顶层能否用 AgentName 区分来源。

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ---------- 主 Agent 原生工具 ----------

// webSearchTool 主 Agent 直接持有的原生工具（假实现）。
type webSearchTool struct{}

var _ tool.InvokableTool = (*webSearchTool)(nil)

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

func (w *webSearchTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 web_search 参数失败: %w", err)
	}
	return fmt.Sprintf("（主 Agent 原生工具）未接入真实搜索引擎，搜索「%s」的结果为空。", args.Query), nil
}

// ---------- 子 Agent 内部工具 ----------

// getCurrentTimeTool 子 Agent 内部的时间工具（假实现，返回确定性假时间）。
type getCurrentTimeTool struct{}

var _ tool.InvokableTool = (*getCurrentTimeTool)(nil)

func (g *getCurrentTimeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_current_time",
		Desc: "获取当前日期和时间。当用户询问时间、日期或类似“现在几点”的问题时必须调用本工具。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {
				Type:     schema.String,
				Desc:     "城市名称，例如：北京、上海",
				Required: true,
			},
		}),
	}, nil
}

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
	return fmt.Sprintf("2026-08-13 15:30（北京时间，%s）", args.City), nil
}

// simulatedMCPQueryTool 模拟"外部 MCP 服务器工具"（假实现，返回确定性数据）。
// 只注册给子 Agent，用于验证 MCP 服务器场景下的事件转发。
type simulatedMCPQueryTool struct{}

var _ tool.InvokableTool = (*simulatedMCPQueryTool)(nil)

func (s *simulatedMCPQueryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "mcp_order_query",
		Desc: "查询演示订单数据。当用户询问订单、交易记录等数据时调用。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"order_id": {
				Type:     schema.String,
				Desc:     "订单号，例如 ORD-1001",
				Required: true,
			},
		}),
	}, nil
}

func (s *simulatedMCPQueryTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 mcp_order_query 参数失败: %w", err)
	}
	if args.OrderID == "" {
		args.OrderID = "ORD-未知"
	}
	return fmt.Sprintf("（模拟 MCP 服务器数据）订单 %s：状态=已发货，金额=¥129.00，下单时间=2026-08-10 09:12", args.OrderID), nil
}
