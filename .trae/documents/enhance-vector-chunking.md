# 向量分块增强：结构感知标题拼接 + 相邻块轻量上下文

## Summary

对 Jot 的向量量化/召回做两项增强：

1. **策略 2（结构感知切片）**：改造 `ChunkContent`，让切出的每个块自动携带"所属标题链"，标题指引进入 embedding 向量、存储文本与召回展示。
2. **策略 1 轻量替代**：`VectorRecall` 召回命中块时，按笔记维度合并，并补充命中块相邻的 1 块上下文，近似"子块检索 + 父块上下文"效果；不引入父子表、无需重新量化 schema。

改动集中两个文件：`internal/services/chunk.go`、`internal/services/vector_service.go`。

## Current State Analysis

* `ChunkContent`（[chunk.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go#L10-L55)）：按 `##`/`###` 标题行作块首 + 空行分段 + 500 rune 硬切。问题：① 硬切后的后续段无标题；② 空行分段的新块（非标题开头）丢失标题指引。

* `IndexNotes`（[vector\_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L35-L124)）：`ChunkContent(note.Content, 500)` 的输出同时用作 **embedding 输入**、**chunk\_text 存储**、**召回卡片展示**——改造 ChunkContent 后三处自动生效。

* `VectorRecall`（[vector\_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L179-L297)）：

  * L215 `q.Find(&vectors)` 已把当前模型**全部向量块载入内存**（按笔记本过滤），相邻块可直接内存查找，零额外 SQL。

  * L266-281 当前**逐块生成卡片**：同一笔记命中多个块 → 生成多张同 ID 卡片 → 被 `MergeRecallCards` 按 ID 去重后丢失内容（隐藏缺陷），按笔记合并方案可顺带修复。

## Proposed Changes

### Change 1: `internal/services/chunk.go` — 标题链拼接（策略 2）

**目标**：每个落块要么以标题开头（现状），要么在块首自动补上所属标题链（`## A` 或 `## A\n### B`）。

**改动点**：

* `ChunkContent` 内维护 `chain []string`（当前标题链）：

  * 遇到 `## X`：`chain = ["## X"]`

  * 遇到 `### Y`：移除 chain 中已有 `###` 项后追加 `"### Y"`（保留上级 `##`；chain 为空时直接 `["### Y"]`）

  * 标题行本身仍作为新块首行（现状不变）

* `flush` 内：块文本首行非标题（`startsWithHeading` 为 false）且 chain 非空时，把 `strings.Join(chain, "\n") + "\n"` 拼到块首；随后再判断超长。

* 超长硬切路径：新增 `splitWithHeading(text, maxRunes, chain)`，对 `hardSplit` 结果中**非首段**且段首非标题的段，补上 chain 前缀。

* 新增辅助函数：

  * `startsWithHeading(text string) bool`：首行是否为 `## ` /`### `  标题

  * `splitWithHeading(text string, maxRunes int, chain []string) []string`

* **边界规则不变**：仍只识别 `##`/`###`，空行分段、500 rune 上限、`#` 不参与分块，全部保持。

* 效果：embedding 输入、chunk\_text 存储、卡片展示三处同时带上标题指引（因三者同源）。

### Change 2: `internal/services/vector_service.go` — 相邻块补充 + 按笔记合并（策略 1 轻量版）

**目标**：命中块按笔记聚合为一张卡片，并补充命中块前后各 1 个相邻块，提供上下文。

**改动点**（全部在 `VectorRecall` 组装阶段，L262-L281 替换）：

1. 新增常量 `const adjacentBlocks = 1`（命中块前后各补 1 块，最多 3 块/命中点）。
2. 用已加载的 `vectors` 构建按 `note_id` 分组的块列表，每组按 `ChunkIndex` 升序排序（`sort.Slice`）。
3. 遍历 `cands`（已按相似度降序），按 `note_id` 分组收集"需要返回的 ChunkIndex 集合"：命中块 index 本身 + `[idx-adjacentBlocks, idx+adjacentBlocks]` 范围内的相邻块 index（存在才加入）。
4. 卡片按笔记维度组装（**替代逐块组装**）：遍历命中笔记（顺序按 cands 中首次出现的先后，保持相关度排序稳定），将该笔记选中块按 ChunkIndex 升序 `strings.Join(parts, "\n\n")` 成 Content，生成一张 `RecallCard`。
5. `FormattedText` 同步改为按合并后内容注入，每个卡片块前仍带 `--- 📄 《标题》 ---`。
6. **注入长度上限**：合并后单卡片 Content 超过 1200 rune 时截断（新增 `maxCardRunes = 1200` 常量，防"相邻块补充导致 token 反弹"）。
7. 保留日志与返回值结构不变。

**顺带修复**：同一笔记多命中块不再生成同 ID 多卡片（避免 MergeRecallCards 去重丢内容）。

## Assumptions & Decisions

* **标题链格式**：直接保留 Markdown 标题原文拼接（如 `## 配置说明\n正文`），不引入新格式——embedding 语义自然、展示渲染正常、与现有块风格一致。

* **相邻块数量**：前后各 1（`adjacentBlocks = 1`）。固定常量，不做设置项（避免过度设计，后续可调）。

* **卡片合并上限**：单卡片 1200 rune 截断，控制 token 增量。

* **旧数据需重新量化**：Change 1 改变 chunk\_text 内容，已量化笔记需在数据管理页重新量化才带标题指引；不做代码级迁移（量化是幂等重跑，成本低）。

* **引用功能/关键词召回**：不受影响（关键词召回已移除，引用仍独立整篇注入）。

## Verification

1. `go build ./...` 通过。
2. 逻辑验证（Go 侧，可用临时 main 或 `go test` 跑 `ChunkContent`）：

   * 含 `## A` + 空行分段的段落 → 第二块带 `## A` 前缀

   * 超长段落硬切 → 每段均带标题指引

   * `## A` + `### B` 嵌套 → `### B` 段落带 `## A\n### B` 链；`## C` 之后段落只带 `## C`
3. 手工验证（wails dev）：

   * 重新量化一篇含多级标题的笔记 → 数据管理页查看，召回卡片内容带标题

   * 提问命中某块 → 卡片内容包含该块 + 相邻块（上下文更完整）

   * 同一笔记多个命中点 → 合并为一张卡片而非多张
4. 回归：卡片召回开关开启校验、按笔记本召回、卡片面板展示不受影响。

