# 向量召回六项修复计划

## Summary

修复向量嵌入/召回链路中已核实的六个问题：

1. `vec_distance_cosine` 在 SQL 中每行计算两次
2. 向量维度不匹配时整条召回报错（无 dim 过滤）
3. LIKE 检索未转义 `%`/`_`/`\` 通配符
4. `tokenize(query)` 每次召回执行两遍
5. HybridRecall 注释与实现不符（说"并行"实为串行）
6. 嵌入模型切换后静默降级无提示 + `ValidateCardRecall` 死代码清理

全部为小改动，无 schema 变更，无前端变更（无需重新构建前端资源）。

## Current State Analysis

* **#1 距离算两次**：[vector\_service.go#L641-L651](d:\资源池\下水道\Dev\本地项目\jot\internal\services\vector_service.go) 中 `vectorSearch` 的 SQL 在 WHERE（`dist < 1.0`）和 ORDER BY 各写一次 `vec_distance_cosine(...)`。SQLite 不做公共子表达式消除，且 `dist < 1.0` 几乎不过滤行（余弦距离 ≥1 仅在完全反向时），暴力扫描计算量约 2 倍。

* **#2 维度不匹配**：同一 SQL 中未按 `dim` 过滤。若嵌入服务商在相同模型名下变更输出维度，`vec_distance_cosine` 对维度不一致的行抛错 → 整条 SQL 失败 → `recall_notes` 工具调用整体失败。`NoteVector.Dim` 列已存在且 IndexNotes 写入路径总是赋值。

* **#3 LIKE 未转义**：\[vector\_service.go#L539-L547]\(token 计数) 与 [vector\_service.go#L569-L575](主检索) 中 token 直接拼入 `LIKE ?` 模式，query 中含 `%`/`_` 的 token（如 "100%"、"C++"）被当通配符产生误命中。

* **#4 tokenize 两遍**：`KeywordRecall` 内部 `tokenize(query)`（L513），`HybridRecall` 计算 kwScore 时又 `tokenize(query)`（L709）。且后者未做 `maxRecallKeywords` 截断，与检索用的 token 集不一致（>20 token 时 kwScore 口径漂移）。

* **#5 注释不符**：L667 注释"并行执行向量检索与关键词检索"，实际为顺序执行。不做真并行：`db.go` L56 `SetMaxOpenConns(1)` 单连接池，两路 SQL 本就串行，并行化收益不值复杂度。

* **#6 模型切换静默降级 + 死代码**：

  * `ValidateCardRecall`（app.go L1548-L1580，含"当前模型无向量数据"检查）在卡片召回开关移除后已无任何调用方（Go/前端零引用），为死代码。`CardRecallCheckResult` 类型仍被 `ValidateVectorIndexConfig`/`TestVectorIndexConnection` 使用，需保留。

  * 现在 `recall_notes` 工具（[recall\_notes.go#L59-L116](d:\资源池\下水道\Dev\本地项目\jot\internal\agent\tools\recall_notes.go)）在模型无向量数据时静默只走关键词路，用户无感知。

* 测试确认：`internal/services/*_test.go` 不直接调用 `KeywordRecall`/`vectorSearch`，重构私有签名安全。

## Proposed Changes

### 1. vector\_service.go — vectorSearch 距离单次计算 + dim 过滤（#1 + #2）

将现 SQL 改为子查询形式，距离只算一次；WHERE 增加 `dim = ?`：

```go
vecSQL := "SELECT id, note_id, chunk_index, chunk_text, model FROM (" +
    "SELECT note_vectors.id, note_vectors.note_id, note_vectors.chunk_index, note_vectors.chunk_text, note_vectors.model, " +
    "vec_distance_cosine(note_vectors.embedding, vec_f32(?)) AS dist " +
    "FROM note_vectors JOIN notes ON notes.id = note_vectors.note_id AND notes.deleted_at IS NULL"
args := []interface{}{}
// notebookIDs 过滤（保持现有逻辑，追加在 JOIN 后）
...
vecSQL += " WHERE note_vectors.model = ? AND note_vectors.dim = ?" +
    ") WHERE dist < 1.0 ORDER BY dist ASC LIMIT ?"
args = append(args, string(queryVecJSON), embedClient.Model, len(embeddings[0]), limit*chunkCandidateMultiplier)
```

要点：

* 内层每行计算一次 dist 并物化，外层 WHERE/ORDER BY 复用，计算量减半

* `dim = ?` 传入 `len(embeddings[0])`，维度不一致的行被排除而非炸整条查询（顺带过滤潜在脏数据）

* 更新该处注释说明子查询意图与 dim 过滤原因

### 2. vector\_service.go — LIKE 转义（#3）

新增辅助函数：

```go
// escapeLike 转义 LIKE 模式通配符，配合 ESCAPE '\' 使用
func escapeLike(s string) string {
    r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
    return r.Replace(s)
}
```

* token 计数查询（L543）与主检索 SQL（L569-L575）中所有 `LIKE ?` 改为 `LIKE ? ESCAPE '\'`（Go 源码字面量写 `ESCAPE '\\'`），参数由 `"%"+t+"%"` 改为 `"%"+escapeLike(t)+"%"`

* `rankKwHits` 与 kwScore 用 `strings.Contains` 做精确包含匹配，不涉及通配符，不改

### 3. vector\_service.go — tokenize 复用（#4）

拆分 `KeywordRecall`：

* 导出签名不变：`KeywordRecall` 内部 `tokenize(query)` + 截断后委托给新的私有方法 `keywordRecallByTokens(ctx, tokens []string, limit int, notebookIDs []uint)`（承接现有全部检索逻辑）

* `HybridRecall` 中：将 `kwTokens := tokenize(query)` 提前到关键词检索之前，关键词路调用 `s.keywordRecallByTokens(ctx, kwTokens, limit, notebookIDs)`，kwScore 计算复用同一 `kwTokens`

* 附带修复：kwScore 口径与检索 token 集严格一致（原先 kwTokens 未截断）

### 4. vector\_service.go — 注释修正（#5）

L667 `HybridRecall` 注释改为"依次执行向量检索与关键词检索"，并注明单连接池下无需并行。

### 5. app.go — 删除死代码 ValidateCardRecall（#6a）

* 删除 `ValidateCardRecall` 方法（L1548-L1580 整段）

* 保留 `CardRecallCheckResult` 类型（`ValidateVectorIndexConfig`/`TestVectorIndexConnection` 仍在用）

* 更新 `startVectorIndex` 中 L1628 引用该方法的注释（改为"与 ValidateVectorIndexConfig 校验强度一致"或直接删除引用）

### 6. agent/tools/recall\_notes.go — 模型无向量数据提示（#6b）

`InvokableRun` 中构建 embedClient 后增加预检：

```go
// 预检当前模型向量数据：无向量时跳过 query 向量化（省一次 API 调用），降级为仅关键词检索并提示用户
vecCnt, err := r.vector.CountVectorsByModel(model)
if err != nil {
    return "", fmt.Errorf("检查向量索引数据失败: %w", err)
}
noVec := vecCnt == 0
if noVec {
    embedClient = nil // VectorRecall 对 nil client 静默跳过向量路
}
```

结果组装：

* 召回成功（result 非 nil）：`FormattedText` 末尾追加 `\n\n（提示：当前嵌入模型暂无向量索引数据，以上结果仅来自关键词检索。建议在数据管理中重建向量索引以启用语义检索。）`

* 无命中（result == nil 且 noVec）：错误信息改为 `本地笔记库中没有检索到相关内容（当前嵌入模型暂无向量索引数据，建议先在数据管理中重建向量索引）`

`CountVectorsByModel` 已存在（vector\_service.go L290），直接复用；`VectorRecall` 是该工具的唯一调用方（全库已确认），提示不会重复出现在其他路径。

## Assumptions & Decisions

* 不做真并行召回（单连接池，收益不值复杂度），只修注释

* 不做 RRF / FTS5 / 块级嵌入复用 / updated\_at 预过滤（本轮范围外）

* 不改任何导出函数签名（`KeywordRecall`/`VectorRecall` 签名不变）

* 无前端改动，无需 `npm run build` / `wails build`（wailsjs 中残留的 `ValidateCardRecall` 绑定生成物无害，下次构建自然消失）

* 子查询改写保持语义不变：候选放大倍数、`dist < 1.0` 过滤、排序方向均与原逻辑一致

## Verification

1. `go build ./...` 编译通过
2. `go vet ./internal/services/... ./internal/agent/...` 无告警
3. `go test ./internal/services/... ./internal/agent/...` 全部通过（重点：vector\_service\_test.go、vector\_service\_keyword\_test.go、chunk\_test.go）
4. 人工核对点：

   * vectorSearch 新 SQL 参数顺序：vec\_f32 → model → dim → LIMIT

   * LIKE 转义覆盖 token 计数与主检索两处

   * `ValidateCardRecall` 删除后全库无残留引用（`grep ValidateCardRecall` 仅剩 wailsjs 生成物）

