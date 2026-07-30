# 搜索词精炼扩展至卡片召回 Spec

## Why

当前搜索词精炼仅在启用联网搜索时触发，精炼后的关键词也只用于联网搜索。卡片召回仍使用原始用户输入做 gse 分词。实际上卡片召回也受益于精炼后的关键词——AI 提炼出的关键实体词可以弥补纯分词遗漏的语义，提升召回质量。需要将精炼逻辑扩展为：**联网搜索或卡片召回任一启用时都触发精炼，精炼后的关键词同时供给两者使用**。

## What Changes

- **修改** `app.go` 中 `CallAIStream` — 精炼触发条件从 `len(searchSources) > 0` 改为 `len(searchSources) > 0 || cardRecallEnabled`；精炼提前到搜索/卡片召回块之前执行；卡片召回使用原 query + 精炼关键词拼接后的文本
- **修改** `app.go` 中 `CallAIStreamRegenerate` — 同上
- `query_refiner.go`、`recall_service.go`、`ai-chat.js` 不作改动

## Impact

- Affected specs: 搜索 Query 精炼功能、卡片召回功能
- Affected code: `app.go`（两个流方法，每处约 20-30 行重构）

## ADDED Requirements

### Requirement: 精炼触发条件扩展

The system SHALL trigger query refinement when either web search or card recall is enabled.

#### Scenario: 仅卡片召回启用
- **WHEN** 用户发送消息且仅卡片召回开启（联网搜索关闭）
- **THEN** 系统先发出 `ai:search-status=refining` 事件，执行搜索词精炼
- **AND** 精炼完成后发出 `ai:refined-keywords` 事件携带精炼关键词
- **AND** 不发出 `ai:search-status=searching`（因为没有联网搜索）
- **AND** 卡片召回使用原始 query 与精炼关键词拼接后的文本进行检索
- **AND** 卡片召回结束后发出 `ai:search-status=done`

#### Scenario: 联网搜索 + 卡片召回同时启用
- **WHEN** 用户发送消息且两者都开启
- **THEN** 精炼步骤只执行一次
- **AND** 精炼后的关键词同时用于联网搜索的 query 和卡片召回的检索文本
- **AND** 联网搜索和卡片召回的行为与各自单独启用时一致

### Requirement: 卡片召回使用精炼关键词

The system SHALL combine refined keywords with the original user input for card recall search.

- 拼接方式：`originalQuery + " " + refinedQuery`
- 拼接后的文本传给 `services.CardRecallSearch`，由内部 gse 统一分词去重
- 精炼关键词为空时回退为原始 query

## MODIFIED Requirements

### Requirement: CallAIStream 精炼流程（源自 add-search-query-refinement）

精炼的执行时机和条件变更为：

```
if searchSources非空 或 cardRecallEnabled:
    发射 ai:search-status=refining
    result = RefineSearchQuery(userText)
    如果失败 → 发射 ai:stream-error，终止流程
    如果成功 → 发射 ai:refined-keywords
    
    if searchSources非空:
        发射 ai:search-status=searching
        用 refinedQuery 执行联网搜索
    
    if cardRecallEnabled:
        combinedQuery = userText + " " + refinedQuery
        用 combinedQuery 执行卡片召回
    
    发射 ai:search-status=done
```

### Requirement: CallAIStreamRegenerate 精炼流程

与 `CallAIStream` 一致的变更，但 `userText` 从最后一条 user message 提取而非函数参数。
