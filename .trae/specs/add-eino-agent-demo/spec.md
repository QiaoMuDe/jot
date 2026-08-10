# Eino Agent 循环 Demo Spec

## Why
为后续将 jot 的 AI 对话模块从"预组装流水线"升级为"单 Agent + 工具调用 + 反思循环"做技术预研。通过一个独立的最小 demo 验证 cloudwego/eino 在项目技术栈（Go 1.26）下的三个未知点：BaseURL 兼容性（DeepSeek/通义/Ollama 的 Chat Completions 端点）、tool calling、流式事件消费。

## What Changes
- 在根目录新建 `agent-demo/` 目录（**独立 go.mod**，参照 `vec-poc/` 先例，不影响主项目）
- 引入 `github.com/cloudwego/eino`（内置 adk 模块）+ `github.com/cloudwego/eino-ext/components/model/openai`
- 提供清晰的配置入口：base URL、API key、模型 ID 三要素可通过环境变量/命令行参数配置
- 实现一个可运行的 Agent 循环 demo：
  - `openai.NewChatModel` 构建模型（读配置）
  - 至少一个自定义工具（`tool.BaseTool`），让模型有工具可调
  - `adk.NewChatModelAgent` + `adk.NewRunner` 跑 ReAct 循环
  - 终端展示流式输出与工具调用过程
- demo 内附 `README.md` 说明运行方式（含配置方式与示例命令）

## Impact
- Affected specs: 无（全新独立能力，不改动现有 AI 对话流程）
- Affected code:
  - 新增 `agent-demo/`（独立模块，含 `go.mod`、`go.sum`、`main.go`、`README.md`）
  - 主项目代码（`app.go`、`internal/aicli` 等）**零改动**
- 后续影响：验证通过后，可作为接入 `app.go` 的 `CallAIStream` 的技术参考

## ADDED Requirements

### Requirement: 独立可构建的 demo 目录
系统 SHALL 提供 `agent-demo/` 目录，包含独立 go.mod，`go build` / `go run` 可直接编译运行，不依赖主项目模块。

#### Scenario: 构建成功
- **WHEN** 在 `agent-demo/` 下执行 `go build ./...`
- **THEN** 编译通过，无错误

### Requirement: 三要素配置入口
系统 SHALL 提供配置 base URL、API key、模型 ID 的明确位置，优先级为：命令行 flag > 环境变量 > 默认值。

#### Scenario: 通过环境变量配置
- **WHEN** 用户设置 `AGENT_DEMO_BASE_URL`、`AGENT_DEMO_API_KEY`、`AGENT_DEMO_MODEL` 后运行 demo
- **THEN** demo 使用这些值连接模型服务

#### Scenario: 通过命令行参数配置
- **WHEN** 用户运行 `go run . -base-url ... -api-key ... -model ...`
- **THEN** 命令行参数覆盖环境变量

### Requirement: Agent ReAct 循环可运行
系统 SHALL 使用 Eino 的 `ChatModelAgent` 实现 ReAct 循环：模型自主决定是否调用工具、执行工具、观察结果、继续循环直至给出最终回答。

#### Scenario: 成功路径
- **WHEN** 用户输入一个需要调用工具的问题（如"现在几点了"）
- **THEN** demo 展示：模型发起工具调用 → 工具执行并返回结果 → 模型基于结果给出最终回答，循环在模型不再请求工具时自然结束

#### Scenario: 无工具需求
- **WHEN** 用户输入一个普通问题（如"你好"）
- **THEN** demo 不触发工具调用，直接流式输出回答

### Requirement: 终端可见的过程展示
系统 SHALL 在终端展示 Agent 循环的关键过程（工具调用名称、参数、结果摘要、流式文本输出），便于"看效果"。

#### Scenario: 过程可见
- **WHEN** demo 运行过程中发生工具调用
- **THEN** 终端打印工具名与参数、工具返回结果、以及最终回答的流式文本

## MODIFIED Requirements
无

## REMOVED Requirements
无
