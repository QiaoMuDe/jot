# Ollama 始终解析 `<think>` 标签 Spec

## Why

Ollama 的流式 API 始终将思维链内容以 `<think>...</think>` 标签形式包含在响应中，而非流式 API 则自动去除。当前实现仅在 `thinkingEnabled=true` 时解析标签，`thinkingEnabled=false` 时标签原文暴露在前端界面中显示为乱码/原文。

需要在 Ollama 场景下始终解析 `<think>` 标签，当无需展示思维链时直接静默去除。

## What Changes

- **Go 后端**：Ollama 流式回调中的 `<think>` 标签解析从条件执行（仅 `thinkingEnabled=true`）改为始终执行，`thinkingEnabled=false` 时静默去除标签内容
- **前端**：无改动

## Impact

- Affected specs: AI 对话流式输出
- Affected code: `internal/services/ai_service.go`

## ADDED Requirements

### Requirement: Ollama 始终解析 `<think>` 标签

系统 SHALL 在使用 Ollama provider 时始终通过状态机解析 `<think>` 标签。

#### Scenario: 启用深度思考（Ollama）

- **WHEN** `thinkingEnabled = true` 且 provider 为 Ollama
- **THEN** 标签内内容通过 `onThinking` 推送至前端思维链折叠块

#### Scenario: 未启用深度思考（Ollama）

- **WHEN** `thinkingEnabled = false` 且 provider 为 Ollama
- **THEN** 标签内内容被静默丢弃
- **AND** `hasThinking` 保持 `false`
- **AND** `elapsedThinking` 为 0
- **AND** `fullThinking` 保持空
- **AND** `fullContent` 仅含标签外的正常回复内容

## MODIFIED Requirements

### Requirement: Ollama `<think>` 标签实时解析状态机

状态机触发条件从 `isOllama && thinkingEnabled` 改为 `isOllama`。

- 状态机内部增加判断：仅 `thinkingEnabled=true` 时才调用 `onThinking` 和记录 `hasThinking`
- `thinkingEnabled=false` 时发现 `</think>` 后，标签内内容直接丢弃，不推送、不记录

## REMOVED Requirements

无。
