# 召回注入对话时剥离元数据前缀

## Summary

分块元数据前缀（笔记标题/分类标签/创建时间/笔记核心内容：）在检索阶段（embedding + 关键词 LIKE）已发挥价值，但召回注入对话时成为冗余：标题在卡片头重复、标签/时间对 LLM 回答无用、引导行无信息量，每条命中浪费 \~40 rune token。本方案在组装召回上下文（FormattedText + RecallCard.Content）时剥离前缀，只注入正文（标题链+内容），数据库存储的 chunk\_text 不变。

## Current State Analysis

* `formatMetaPrefix`（chunk.go:25-40）生成前缀，末尾 `笔记核心内容：` 不带尾随换行

* chunk\_text = prefix + "\n" + 正文（标题链+内容），存于 note\_vectors 表

* HybridRecall（vector\_service.go:565-622）组装 FormattedText：

  * 第 578 行 `parts = append(parts, v.ChunkText)` 直接拼接完整 chunk\_text（含前缀）

  * 第 581 行 `content := strings.Join(parts, "\n\n")`，截断到 maxCardRunes=1200

  * 第 585 行 `fmt.Fprintf(&b, "--- 📄 《%s》 ---\n%s\n\n", note.Title, content)` 注入 LLM

  * 第 589 行 `Content: content` 同一变量用于前端 RecallCard 展示

* 前端 renderRecallCards（ai-chat.js:3611）：titleRow 独立显示 card.title，content 仅作 snippet 预览（textContent 纯文本）

* 注入链路：HybridRecall → app.go:2119/2227 appendToSystemMessage → system message → LLM + estimateUserTokens 统计

## Proposed Changes

### 1. chunk.go 新增 stripMetaPrefix 辅助函数

在 `formatMetaPrefix` 函数下方新增：

```go
// stripMetaPrefix 剥离分块元数据前缀，返回正文部分（标题链+内容）
// 前缀由 formatMetaPrefix 生成，以 "笔记核心内容：" 结尾；找该标记首次出现位置取其后内容
// 找不到标记时（旧数据/无前缀）原样返回，保证向后兼容
func stripMetaPrefix(text string) string {
    const marker = "笔记核心内容：\n"
    if idx := strings.Index(text, marker); idx >= 0 {
        return text[idx+len(marker):]
    }
    return text
}
```

**为什么**：前缀由 formatMetaPrefix 固定生成，`笔记核心内容：\n` 是稳定分隔标记；前缀在前，首次出现即前缀标记，正文即使含该字符串也不影响（取第一次出现）。找不到标记时兜底返回原文，兼容旧数据（前缀功能上线前量化的 chunk\_text 无前缀）。

### 2. vector\_service.go 组装 parts 时剥离前缀

修改 vector\_service.go:576-579，对每个 v.ChunkText 调 stripMetaPrefix：

```go
var parts []string
for _, v := range byNote[noteID] {
    if hitIndexes[noteID][v.ChunkIndex] {
        parts = append(parts, stripMetaPrefix(v.ChunkText))
    }
}
```

**效果**：

* content（同时用于 LLM 注入和前端展示）只含正文（标题链+内容），无元数据前缀

* LLM 看到 `--- 📄 《标题》 ---\n## 标题链\n正文`，标题在卡片头、结构在正文，信息无损

* 前端 snippet 预览也变干净（标题已在 titleRow 显示，不再重复）

* maxCardRunes=1200 截断预算全部用于正文，有效内容容量提升 \~40 rune/条

**不改的部分**：

* 数据库 note\_vectors.chunk\_text 不变（保留前缀，供 embedding 检索 + 关键词 LIKE 检索）

* IndexNotes 量化流程不变

* KeywordRecall 不变（仍 LIKE 匹配含前缀的 chunk\_text）

* formatMetaPrefix 不变

## Assumptions & Decisions

* **剥离粒度**：逐 chunk 剥离（每个 v.ChunkText 单独 stripMetaPrefix），而非对拼接后的 content 整体剥离（多 chunk 拼接后前缀在中间，无法整体剥离）

* **RecallCard.Content 也剥离**：前端 titleRow 已独立显示标题，content 仅作预览，剥离后更干净；无需为前端单独保留前缀版本

* **向后兼容**：旧数据（无前缀的 chunk\_text）stripMetaPrefix 找不到标记返回原文，不影响

* **标记唯一性**：`笔记核心内容：\n` 在前缀里固定出现一次（formatMetaPrefix 保证），正文极少含此串；即使含，取首次出现仍取到前缀标记，安全

* **不剥离的场景**：纯向量召回（VectorRecall）也走 HybridRecall，自动受益；无其他独立注入点

## Verification

1. `go build ./...` 通过
2. `go test ./internal/services/...` 通过
3. 新增测试：`TestStripMetaPrefix` 验证三种情况——有前缀剥离正确、无标签前缀剥离正确、无前缀旧数据兜底返回原文
4. 手动验证：重新量化一篇笔记后发起对话召回，确认：

   * 前端召回卡片 snippet 不再显示"笔记标题：/分类标签：/创建时间：/笔记核心内容："前缀

   * LLM 仍能正确引用笔记标题（来自卡片头 `--- 📄 《标题》 ---`）

   * 数据管理页 token 统计中该轮 userTokens 较改前下降（前缀不再计入）

