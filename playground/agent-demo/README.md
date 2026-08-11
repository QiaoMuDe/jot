# Eino Agent 循环 Demo（agent-demo）

基于 [cloudwego/eino](https://github.com/cloudwego/eino)（v0.9.13）的最小可运行 Agent 循环示例，
是 jot 项目对 Eino 框架的技术预研。演示了：

- `openai` 扩展包构建 OpenAI 兼容 ChatModel（可对接 DeepSeek / 通义 / Ollama 等）
- 手写实现 `tool.InvokableTool`（含 `BaseTool` 接口）自定义工具
- `adk.NewChatModelAgent` 组装 ReAct Agent（模型决策 → 调用工具 → 反馈 → 继续，最多 8 轮迭代）
- `adk.NewRunner` + `runner.Query` 消费流式事件（文本实时打印、工具调用/结果标记行）

## 文件结构

```
agent-demo/
├── main.go     # 入口：模型构建、Agent 组装、Runner 事件流消费
├── config.go   # 配置三要素解析（flag > 环境变量 > 默认值）+ 脱敏摘要
├── tools.go    # 自定义工具：get_current_time、web_search（假实现）
├── go.mod / go.sum
└── README.md
```

## 配置方式

配置三要素的优先级：**命令行参数 > 环境变量 > 默认值**。

| 配置项 | 命令行参数 | 环境变量 | 默认值 |
| --- | --- | --- | --- |
| API 端点 | `-base-url` | `AGENT_DEMO_BASE_URL` | `https://api.deepseek.com/v1` |
| API Key | `-api-key` | `AGENT_DEMO_API_KEY` | （无，必填） |
| 模型名 | `-model` | `AGENT_DEMO_MODEL` | `deepseek-chat` |

启动时会打印脱敏配置摘要（API Key 仅显示前 4 位与后 4 位）。

### 通过环境变量配置

```powershell
# PowerShell
$env:AGENT_DEMO_API_KEY = "sk-xxxxxxxx"
$env:AGENT_DEMO_BASE_URL = "https://api.deepseek.com/v1"
$env:AGENT_DEMO_MODEL = "deepseek-chat"
go run . "现在几点了"
```

```bash
# Linux / macOS
export AGENT_DEMO_API_KEY="sk-xxxxxxxx"
export AGENT_DEMO_BASE_URL="https://api.deepseek.com/v1"
export AGENT_DEMO_MODEL="deepseek-chat"
go run . "现在几点了"
```

### 通过命令行参数配置（优先级最高）

```bash
go run . -api-key "sk-xxxxxxxx" -model "deepseek-chat" "现在几点了"
```

## 运行示例

```bash
# 无参数时打印使用说明
go run .

# 询问时间（触发 get_current_time 工具）
go run . "现在几点了"

# 询问实时信息（触发 web_search 工具，演示模式返回空结果）
go run . "帮我搜索今天的新闻"
```

预期输出形态：

```
========== Agent 配置摘要 ==========
BaseURL : https://api.deepseek.com/v1
Model   : deepseek-chat
APIKey  : sk-t****cdef
====================================
>>> 用户问题: 现在几点了

[模型] （流式文本块实时输出...）
[工具] 调用 get_current_time，参数 {"city":"北京"}
[工具] 结果: 2026-08-09 14:30（北京时间，北京）
[模型] 现在是 2026 年 8 月 9 日 14:30（北京时间）。

>>> Agent 运行结束
```

## 注意事项

- **模型需支持 function calling**：ChatModelAgent 通过 `model.WithTools` 把工具定义传给模型，
  模型需要能生成结构化工具调用（DeepSeek-chat、通义千问、GPT 系列等均支持）。
- **BaseURL 必须指向 OpenAI 兼容端点**：
  - DeepSeek：`https://api.deepseek.com/v1`
  - 通义千问（DashScope 兼容模式）：`https://dashscope.aliyuncs.com/compatible-mode/v1`
  - Ollama 本地：`http://localhost:11434/v1`（模型名如 `qwen2.5:7b`，API Key 可填任意值）
- **演示工具为假实现**：`get_current_time` 返回确定性假时间，`web_search` 不发起真实网络请求，
  仅用于演示 Agent 的 ReAct 循环与事件流输出。
- **网络不可用时无法验证真实调用**：无 API Key 或网络不通时，程序会在模型调用处打印 `[错误]` 事件并正常退出。
- 事件消费基于 `AsyncIterator.Next()` 逐事件读取；流式事件中文本块经 `MessageStream.Recv()` 实时打印，
  流结束后用 `schema.ConcatMessages` 合并 ToolCalls 并打印工具调用标记行。
