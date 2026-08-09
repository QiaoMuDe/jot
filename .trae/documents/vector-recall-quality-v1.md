# 向量召回质量优化 V1 计划（纯向量路径）

## 一、Summary

针对"小时数据的代码是多少 → 应回答 2061，实际召回不到含 2061 的块"这一核心问题，在**纯向量召回路径**上做一版优化（关键词召回保持禁用，不动）。优化分两层：

* **索引侧（分块/嵌入）**：markdown 表格行块携带表头上下文 + 压缩表格空白，提升表格类块的嵌入质量（这是 2061 块能被"代码是多少"语义命中的前提）

* **召回侧（检索/聚合）**：放大 chunk 候选数 + 按笔记（note）打分聚合选择，替代当前"chunk 级 LIMIT 截断"，解决"第 6 名以后直接出局"和"5 个 chunk 可能全来自同一笔记导致卡片过少"两个问题

改完需**重新量化受影响的协议类笔记**后验证效果。

## 二、当前状态分析（基于实际代码）

### 召回管线

* 入口 [app.go#L2197-L2250](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go)：`VectorRecall(ctx, vectorQuery, vectorLimit, embedClient, recallNotebookIDs...)`，`vectorLimit = ai_card_recall_limit`（默认 5，≤30）

* [VectorRecall](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L664-L666) → [HybridRecall](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L413-L623)

  * `vectorSearch`（[L344-L408](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L344-L408)）：SQL `ORDER BY vec_distance_cosine(...) ASC ... LIMIT ?`，**LIMIT 直接等于 limit（默认 5）** ← 问题点 3

  * 合并后 `if len(merged) > limit { merged = merged[:limit] }`（[L503-L505](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L503-L505)）：**按 chunk 截断** ← 问题点 4

  * 命中笔记顺序按 hits 首次出现先后（[L558-L566](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L558-L566)），相邻块 ±1 扩展（[L543-L556](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L543-L556)），组装卡片（[L568-L593](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L568-L593)）

* `enableKeywordRecall = false`（[L284](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L284)），关键词路不参与

### 分块管线

* [IndexNotes](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L128-L233)：`ChunkContent(note.Content, 600, meta)` → 分批 embedding（16/批）→ 事务内删旧块插新块（幂等）

* [ChunkContent](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go#L63-L134)：按行累积、标题触发 flush、`prependChain` 补标题链、`formatMetaPrefix` 加元数据前缀；**对表格行原样保留**（PDF/PPT 转换出的"| 小时数据 | +200 空格 | 2061 |"完整进入 chunk）← 问题点 1/2

* `splitWithHeading`（[L202-L213](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go#L202-L213)）、`hardSplit`（[L221-L239](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go#L221-L239)）

### 已确认的问题（来自 DB 实测）

1. **表头语义断裂**：note 77 的表格"命令编码"表头在 chunk 84，数据行（`小时数据 2061`）在 chunk 86，中间隔着几乎空白的分隔线块 85 → 2061 块的嵌入里**没有"代码/编码"语义**，query"小时数据的代码是多少"语义上匹配不上
2. **表格空白稀释嵌入**：表格行带 100-200 个空格，降低向量质量
3. **chunk 级 LIMIT=5 太紧**：排名第 6 的块直接没有机会（08-09 实测只召回 3 张卡，2061 块排第 191 位）
4. **按 chunk 截断导致卡片多样性差**：5 个 chunk 可能全来自 1 个笔记 → 卡片只有 1 张

## 三、改动方案

### 改动 1：`internal/services/chunk.go` — 索引侧（V1 表头携带 + V2 空白压缩）

**V2 空白压缩**：新增函数 `normalizeChunkSource(content string) string`

* 按行处理；用 \`\`\` / \~\~\~ 跟踪代码围栏状态（复用 `isCodeFence`）

* 非围栏行：连续 2+ 个 ASCII 空格/制表符折叠为 1 个空格；行尾空白去除

* 围栏内行不做任何处理（保护代码块缩进）

* 在 `ChunkContent` 开头调用，替换后续处理用归一化后的文本

* 理由：表格空白占满 chunk 长度预算并稀释嵌入；归一化后嵌入质量提升、注入 token 减少

**V1 表格表头携带**：

* 新增状态 `tableHeader string`，识别规则：某行以 `|` 开头，且**下一行**是 markdown 分隔线（仅含 `| - : 空格` 字符且含 `-`）→ 该行即为表头，存入 `tableHeader`（新表头覆盖旧表头）

* 在 `flush()` 中：`prependChain` 之后、拼接 `prefix` 之前，若 `tableHeader != ""` 且当前文本包含表格数据行（以 `|` 开头的非表头非分隔线行）且文本中不含 `tableHeader` → 在文本前补一行表头

* 理由：让"命令编码/数据类型"等列名进入每个表格行块，2061 块的嵌入从此携带"编码"语义，`dist` 排名大幅前移

* 注意：表头为一行，长度可忽略，不破坏 maxRunes=600 预算（若超限仍由 `splitWithHeading` 兜底，表头随首段）

**配套**：新增单元测试（见"验证"）

### 改动 2：`internal/services/vector_service.go` — 召回侧（V3 候选放大 + V4 note 聚合）

**V3 chunk 候选放大**：

* 新增常量 `const chunkCandidateMultiplier = 5`

* `vectorSearch` 中 `LIMIT ?` 的参数从 `limit` 改为 `limit * chunkCandidateMultiplier`（默认 5→25，上限 30→150，SQL 可接受）

* 理由：先多捞候选，别让第 6 名直接出局

**V4 note 级聚合选择**：

* 新增函数 `selectTopNotes(hits []models.NoteVector, limit, maxPerNote int) []models.NoteVector`：

  * 遍历（hits 已按距离升序）：首个出现的 note 计数 +1；已收集满 `limit` 个不同 note 后跳过新 note；同一 note 最多保留 `maxPerNote` 个 chunk

* 新增常量 `const maxChunksPerNote = 4`

* `HybridRecall` 中：把 `if len(merged) > limit { merged = merged[:limit] }`（[L502-L505](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L502-L505)）替换为 `merged = selectTopNotes(merged, limit, maxChunksPerNote)`（merged 转 \[]models.NoteVector 后调用）

* 后续 `hitIndexes`/`hitOrder`/卡片组装逻辑不变（hitOrder 天然按相关度排序 → 卡片按相关度展示）

* 理由：chunk 截断会把 5 个命中压缩到 1-2 张卡；note 级选择保证"相关度优先 + 笔记多样性"，2061 所在笔记只要其最佳块在候选内就能进卡片

### 不改动

* `enableKeywordRecall` 保持 `false`；`VectorRecall` 对外签名不变；`app.go`、`models`、前端零改动

## 四、假设与决策

| 项          | 决策                                                                                | <br />                     |
| ---------- | --------------------------------------------------------------------------------- | :------------------------- |
| 表格识别       | 仅识别 markdown 管道表格（行以 \`                                                           | \` 开头、头行后跟分隔线），不处理无分隔线的伪表格 |
| 空白压缩范围     | 仅非代码围栏行；全角空格不在本次压缩范围（源数据为 ASCII 空格）                                               | <br />                     |
| 参数取值       | `chunkCandidateMultiplier=5`、`maxChunksPerNote=4`，均为可调常量                          | <br />                     |
| 关键词召回      | 保持禁用，本计划只优化向量路径                                                                   | <br />                     |
| 重新量化       | 改完分块逻辑后需在应用内对协议类笔记重新量化（77/79/92/186/88/90/85 或笔记本 9 全量），Ollama(bge-m3:latest) 需在线 | <br />                     |
| 命中证据标注（V5） | 本轮不做（关键词禁用，标注无意义）；卡片按相关度排序由 V4 天然获得                                               | <br />                     |

## 五、验证步骤

1. `go build ./...` 编译通过
2. `go test ./internal/services/` 现有 chunk 测试全部通过（测试输入无表格/无多空格，不受影响）
3. 新增单元测试：

   * `normalizeChunkSource`：多空格折叠、代码围栏内缩进保留

   * 表格表头携带：表头行 + 分隔线 + 数据行 → 数据行所在 chunk 含表头；无表头时行为不变
4. 手工验证（需重新编译应用 + 重新量化）：

   * 问"小时数据的代码是多少"→ 召回卡片必须出现含 2061 的块（note 77/92/79），回答 2061

   * 问"我说的是2061"→ 仍精准命中 note 77

   * 检查 DB `recall_cards` 确认新会话召回内容

