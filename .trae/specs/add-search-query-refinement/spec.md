# 搜索 Query 精炼功能 Spec

## Why

当前联网搜索直接把用户的原始输入作为 search query 传给 Tavily，没有经过任何优化。用户的提问往往是长句、口语化表达或多意图混合的（如"帮我查一下 Go 1.22 有什么新特性和 Rust 2024 edition 发布了没"），直接搜索效果差。增加一个 LLM Query 精炼环节，把用户输入提炼成精准的搜索关键词，可以显著提升搜索质量。

## What Changes

- **新增** `internal/services/query_refiner.go` — 封装调用 LLM 精炼搜索关键词的逻辑
- **修改** `app.go` 中 `CallAIStream` — 在调用 `SearchWeb` 之前，先调 query refiner 获取精炼后的关键词，再以精炼后的关键词搜索
- 精炼失败（网络错误、模型超时等）时直接返回错误，终止整个流程

## Impact

- Affected specs: 联网搜索（Tavily）功能
- Affected code: `internal/services/query_refiner.go`（新增）、`app.go`（修改）

## ADDED Requirements

### Requirement: Query 精炼服务

The system SHALL provide a query refinement service that uses the configured AI model to rewrite user input into precise search keywords.

#### Scenario: 正常精炼
- **WHEN** 用户发送消息且联网搜索开启
- **THEN** 系统先将用户原始输入发送给 AI 模型，通过 system prompt 引导其生成精炼后的搜索关键词
- **AND** 精炼结果只保留关键搜索词，去除口语化表达和无关信息
- **AND** 将精炼后的关键词用于 Tavily 搜索

#### Scenario: 多意图查询
- **WHEN** 用户输入包含多个独立问题的长句（如"查一下 A 和 B"）
- **THEN** 精炼结果应拆分为多个独立的搜索关键词（如 "A"、"B"），用空格或逗号分隔

#### Scenario: 精炼失败终止流程
- **WHEN** AI 模型调用失败（超时、网络错误、配置缺失等）或返回空结果
- **THEN** 系统向前端发射 `ai:stream-error` 事件，携带具体错误信息
- **AND** 搜索和 AI 回答步骤均不再执行

### Requirement: Prompt 设计

The system SHALL use a dedicated system prompt for query refinement.

- Prompt 指令：将用户输入提炼为搜索引擎友好的关键词，去除口语化表达，如有多个独立话题则用空格分隔
- 输出格式：纯文本，不加解释，不加引号包裹
- Prompt 固化在 `query_refiner.go` 中，无需用户配置

### Requirement: 调用时机

The system SHALL perform query refinement before the Tavily search call in `CallAIStream`.

- 精炼使用非流式调用（`AIService.CallAI`），输入输出简短，无需流式展示
- 精炼结果仅用于搜索，不注入到对话上下文中，用户无感知
- 精炼调用使用独立的 context（继承父 context 但带独立超时，如 10s）
- 精炼返回空串或错误时，发射 `ai:stream-error` 事件终止流程

## MODIFIED Requirements

### Requirement: CallAIStream 搜索流程

`CallAIStream` 中联网搜索的流程从：
```
取最后一条 user message → 直接传给 SearchWeb()
```
变更为：
```
取最后一条 user message → QueryRefiner 精炼 → 成功 → 精炼后的 query 传给 SearchWeb()
                                          → 失败 → 发射 error 事件，终止流程
```
