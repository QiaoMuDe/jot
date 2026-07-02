# Ollama 思维链 `` 标签解析 Spec

## Why

Ollama 上的推理模型（DeepSeek-R1、QwQ 等）将思维链内容直接输出在 `content` 中，包裹在 `...` 标签内。当前若开启深度思考，Ollama 因不支持 `WithStreamingReasoningFunc`/`WithThinking` 导致空回复；若不开启则思维链原文可见，无法折叠。

需要在 Ollama 的流式回调中解析 `` 标签，将思维链内容通过 `onThinking` 推送，其余内容通过 `onChunk` 推送，实现与 OpenAI 一致的思维链折叠体验。

## What Changes

- **Go 后端**：在 `CallAIStream` 的 Ollama 分支（`WithStreamingFunc`）中新增 `` 标签状态机，实时分流思维链与正式内容
- **前端**：无改动，复用现有思维链折叠渲染机制
- **AGENTS.md**：新增记忆点

## Impact

- Affected specs: AI 对话流式输出
- Affected code: `internal/services/ai_service.go`

## ADDED Requirements

### Requirement: Ollama `<think>` 标签解析

系统 SHALL 在 Ollama 流式调用中实时解析 `` 标签，将思维链和正式内容分入不同回调。

#### Scenario: 正常推理回复

- **WHEN** Ollama 流式输出包含 `...` 标签
- **THEN** 标签内的所有内容通过 `onThinking` 推送
- **AND** 标签外的内容通过 `onChunk` 推送
- **AND** `fullContent` 仅累加标签外的内容（不含思维链）
- **AND** `fullThinking` 累加标签内的内容

#### Scenario: 无 `<think>` 标签

- **WHEN** Ollama 流式输出不包含 `` 标签
- **THEN** 所有内容通过 `onChunk` 推送
- **AND** `fullThinking` 保持空

#### Scenario: 标签跨 chunk 边界

- **WHEN** `` 或 `` 被拆分到多个 chunk 中
- **THEN** 使用全文缓冲区 + `strings.Contains` 检测边界
- **AND** 在闭合标签完全到达前暂存内容不推送

#### Scenario: 思维链在前、在后、或不存在

- **WHEN** `...` 标签可能出现在内容开头、中间或结尾
- **THEN** 状态机正确处理所有位置
- **AND** 正确计算 `hasThinking` 和 `elapsedThinking`

#### Scenario: 模型仅返回无标签的两端之一（纯思考或不思考）

- **WHEN** 只有 `` 开始，没有 `` 结束（模型截断）
- **THEN** 作为容错，所有内容走 `onChunk`，不丢失回复

### Requirement: Prompt 适配

系统 SHALL 在 Ollama 使用推理模型时通过系统提示说明标签格式，减少标签异常情况。

#### Scenario: Ollama 推理模型提示

- **WHEN** Provider 为 ollama
- **THEN** 在系统消息中附加说明，引导模型使用 `...`` 格式输出思维链

## MODIFIED Requirements

无。

## REMOVED Requirements

无。
