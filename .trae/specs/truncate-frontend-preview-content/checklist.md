# Checklist

- [x] `TruncateRecallCardsPreview` 函数正确截断 `[]RecallCard` 中每条 card 的 `Content` 到 200 字（rune 安全）
- [x] `TruncateSearchSourcesPreview` 函数正确截断 `[]SearchSource` 中每条 source 的 `Content` 到 200 字（rune 安全）
- [x] `CallAIStream` 发射 `ai:recall-cards` 事件时 payload 中的 `Content` 已被截断到 200 字
- [x] `CallAIStream` 发射 `ai:search-sources` 事件时 payload 中的 `Content` 已被截断到 200 字
- [x] `recallCardsJSON`（存 DB）保持全量内容不变
- [x] `searchSourcesJSON`（存 DB）保持全量内容不变
- [x] `LoadAISessionMessagesPaginated` 返回的消息中 `RecallCards` 和 `SearchSources` 的 `Content` 已被截断
- [x] `LoadAISessionMessages`（AI 内部调用）未受影响，保持全量
- [x] JSON 解析失败时静默跳过，不 panic 不阻塞
- [x] 项目可正常编译
- [ ] 开启卡片召回并发送消息，切换会话后召回卡片折叠面板显示正常（内容为前 200 字）
- [ ] 开启联网搜索并发送消息，切换会话后搜索来源折叠面板显示正常（内容为前 200 字）
