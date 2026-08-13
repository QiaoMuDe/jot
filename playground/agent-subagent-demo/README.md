# 主 Agent 调度子 Agent 事件转发验证 Demo（agent-subagent-demo）

基于 [cloudwego/eino](https://github.com/cloudwego/eino)（v0.9.13）的最小验证示例，
用于回答 jot 项目架构研判中的一个关键问题：**主 Agent 通过 `adk.NewAgentTool` 调度子 Agent 时，
子 Agent 内部的事件如何被顶层事件流消费？`EmitInternalEvents` 开关起什么作用？**

> 本 demo 不修改主项目代码，仅验证行为，供后续"MCP 服务器 → 子 Agent"改造方案决策。

## 验证目标

1. **事件来源区分**：顶层 `iter.Next()` 收到的每个事件，`AgentName` 字段是主 Agent 还是子 Agent？
2. **关闭 EmitInternalEvents（默认）**：子 Agent 内部工具调用事件是否完全不出现？
   顶层只看到"主 Agent 调用 child-agent"一次调度记录？
3. **开启 EmitInternalEvents（-emit）**：子 Agent 内部工具调用（get_current_time / mcp_order_query）
   是否被转发到顶层？事件上的 AgentName / ToolName 长什么样？
4. **toolRecords 污染验证**：demo 模拟了主项目 [agent.go](../../internal/agent/agent.go#L557) 的
   `toolRecords` 收集逻辑（tool_start / tool_result 按事件直接记录），观察子 Agent 内部
   工具调用是否会混入主 Agent 层的调用记录（影响 `Result.ToolCalls` 落库）。

## 结构

```
agent-subagent-demo/
├── main.go     # 入口：子 Agent 组装 → NewAgentTool 包装 → 主 Agent 组装 → 事件消费
├── config.go   # 配置解析（flag > 环境变量 > 默认值）+ EmitInternalEvents 开关
├── tools.go    # 主 Agent 原生工具 web_search；子 Agent 内部工具 get_current_time / mcp_order_query
├── go.mod / go.sum
└── README.md
```

```
主 Agent "main-agent"（web_search + AgentTool(child-agent)）
  └── 子 Agent "child-agent"（get_current_time / mcp_order_query）
```

## 配置方式

优先级：**命令行参数 > 环境变量 > 默认值**。

| 配置项 | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- |
| API 端点 | `-base-url` | `AGENT_DEMO_BASE_URL` | `https://api.deepseek.com/v1` |
| API Key | `-api-key` | `AGENT_DEMO_API_KEY` | （无，必填） |
| 模型名 | `-model` | `AGENT_DEMO_MODEL` | `deepseek-chat` |
| 事件转发 | `-emit` | `AGENT_DEMO_EMIT_INTERNAL` | `false` |

## 运行

```powershell
# 1. 关闭事件转发（默认）：观察子 Agent 内部事件是否不可见
go run . "现在几点了"

# 2. 开启事件转发：观察子 Agent 内部工具调用事件是否被转发到顶层
go run . -emit "现在几点了"

# 3. 触发子 Agent 内部的 mcp_order_query（模拟 MCP 服务器场景）
go run . "帮我查订单 ORD-1001"

# 4. 触发主 Agent 直接调用原生工具（对比：不经子 Agent）
go run . "搜索一下今天的新闻"
```

## 预期输出形态

### 关闭 -emit（默认）

```
[事件] AgentName="main-agent" Role=Assistant
    └─ 工具调用决策: name=child-agent args={"request":"现在几点了"}
[事件] AgentName="main-agent" Role=Tool
    └─ 工具结果: tool=child-agent content=现在是 2026 年 8 月 13 日 15:30...
[事件] AgentName="main-agent" Role=Assistant
    └─ [文本] 现在是 2026 年 8 月 13 日 15:30（北京时间）。
```

**预期结论**：子 Agent 内部 get_current_time 的调用事件不出现；顶层只有主 Agent 对
child-agent 的一次 tool_start / tool_result。toolRecords 只记录 `[main-agent] tool_start child-agent`。

### 开启 -emit

**预期观察点**：子 Agent 内部事件是否以 `AgentName="child-agent"`（或其它形态）出现在顶层？
内部工具调用 `get_current_time` 的 tool_start / tool_result 是否可见？`mv.ToolName` 是
`get_current_time` 还是 `child-agent`？——**实际输出以真实运行结果为准**，这正是本 demo 要确认的。

### 关键判断

- 若开启后子 Agent 内部事件 **AgentName 可区分** → 改造 [agent.go](../../internal/agent/agent.go#L294) 时
   可按 AgentName 分流：主 Agent 的工具调用正常记录，子 Agent 内部调用单独处理（或折叠）。
- 若开启后顶层 **仍只有主 Agent 的调度事件** → EmitInternalEvents 只影响流式文本转发，
  不影响 Tool 记录，toolRecords 无污染风险，前端展示策略更简单。
- 若子 Agent 内部 Tool 事件 **混入且无法区分** → 必须修改事件消费逻辑，否则 `Result.ToolCalls`
  会被子 Agent 内部工具名污染。

## 注意事项

- 模型需支持 function calling（DeepSeek-chat、通义千问、GPT 系列均支持）。
- 演示工具为假实现（确定性返回），不发起真实网络/数据库请求；仅 LLM 调用是真实的。
- 子 Agent `MaxIterations=5`、主 Agent `MaxIterations=10`，防止嵌套死循环。
