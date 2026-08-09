# 关键词召回第一级修复方案（Level-1）

## 一、Summary

修复 [KeywordRecall](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L304-L347) 的三个病根，让关键词路从"纯污染"回到"不添乱"：

1. **高频词过滤**：命中块数超过阈值（总块数/10，至少 100）的 token 直接丢弃——"数据"（实测 \~93% 命中率）出局
2. **候选放大**：SQL `LIMIT` 从 5 放大到 `limit×5`（复用 `chunkCandidateMultiplier`），别让好块被截断
3. **截断前排序**：候选块按"命中 token 数"降序、同分按块 id 升序，截断到 `limit`——把打分从"截断后补救"变成"截断前排序"

**范围**：只改关键词召回第一级。不做第二级（整篇笔记检索），不改合并层/卡片层/向量路。

## 二、当前状态分析（基于实际代码）

* [KeywordRecall](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L304-L347)（L304-347）：

  * `tokens := tokenize(query)`（L305），超 `maxRecallKeywords=20`（L72）截断

  * SQL：`OR'd LIKE`（L324-331）**无 ORDER BY**，`LIMIT limit`（L332-333）——"小时数据的代码是多少"分词出 `[小时 数据 代码 多少]`，"数据"命中 7459/8000 块，`LIMIT 5` 按 rowid 取前 5 个 → 执法函/企业端介绍等无关块

  * 无任何过滤与打分

* [HybridRecall](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L460-L474)（L460-474）在关键词**返回后**才按块重算 `kwScore`（命中 token 数）——但坏块已被 `LIMIT 5` 捞回，打分救不了已丢失的候选

* [sortHybridHits](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L633-L660)（L633-660）已正确：双命中 > 仅向量 > 仅关键词，仅关键词内部按 `kwScore` 降序——**这层不用动**

* `enableKeywordRecall = true`（L292，用户已开启用于对比测试）

* 实测依据（之前对主库 `~/.jot/data/jot.db` 的统计）："数据" 7459 块（\~93%）、"2061" 84 块（\~1%）

## 三、改动方案

### 改动 1：`internal/services/vector_service.go` — 新增常量

在 `maxRecallKeywords`（L72）附近新增：

```go
// kwHighFreqDivisor / kwHighFreqMin 高频词过滤阈值：
// 命中块数超过 max(totalChunks/kwHighFreqDivisor, kwHighFreqMin) 的 token 视为无区分度高频词，检索时丢弃
// 依据实测："数据"命中 ~93% 块、"2061"命中 ~1% 块；"数据"这类词进 OR LIKE 只会刷屏
const kwHighFreqDivisor = 10
const kwHighFreqMin = 100
```

### 改动 2：`internal/services/vector_service.go` — 新增两个纯函数（可单测）

```go
// filterHighFreqTokens 剔除命中数超过阈值的高频词（无区分度），返回保留的 token 列表
// threshold = max(total/kwHighFreqDivisor, kwHighFreqMin)
func filterHighFreqTokens(tokens []string, counts []int, total int) []string {
	threshold := total / kwHighFreqDivisor
	if threshold < kwHighFreqMin {
		threshold = kwHighFreqMin
	}
	kept := make([]string, 0, len(tokens))
	for i, t := range tokens {
		if counts[i] > threshold {
			continue
		}
		kept = append(kept, t)
	}
	return kept
}

// rankKwHits 按"命中 token 数"降序排序候选块（同分按块 id 升序，保证稳定），截断到 limit
// 与 HybridRecall 中 kwScore 口径一致（都统计块文本包含的 token 数）
func rankKwHits(hits []models.NoteVector, tokens []string, limit int) []models.NoteVector {
	if limit <= 0 {
		return nil
	}
	type scored struct {
		hit   models.NoteVector
		score int
	}
	list := make([]scored, 0, len(hits))
	for _, h := range hits {
		sc := 0
		for _, t := range tokens {
			if strings.Contains(h.ChunkText, t) {
				sc++
			}
		}
		list = append(list, scored{h, sc})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score > list[j].score
		}
		return list[i].hit.ID < list[j].hit.ID
	})
	if len(list) > limit {
		list = list[:limit]
	}
	out := make([]models.NoteVector, 0, len(list))
	for _, s := range list {
		out = append(out, s.hit)
	}
	return out
}
```

### 改动 3：`internal/services/vector_service.go` — 改造 `KeywordRecall`

保留 `tokenize` 与 `maxRecallKeywords` 截断，替换从"构建 SQL"到"返回"的部分：

1. **统计总块数 + 各 token 命中数**（复用现有 JOIN 过滤：软删除 + notebookID）：

   ```go
   // 基础过滤子句：软删除 + 指定笔记本（与主检索 SQL 一致）
   base := "FROM note_vectors JOIN notes ON notes.id = note_vectors.note_id AND notes.deleted_at IS NULL"
   cntArgs := []interface{}{}
   if len(notebookIDs) > 0 {
       base += " AND notes.notebook_id IN ?"
       cntArgs = append(cntArgs, notebookIDs)
   }
   // 总块数（用于相对阈值）
   var total int
   if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*)"+base, cntArgs...).Scan(&total).Error; err != nil {
       ... 返回错误
   }
   // 各 token 命中数
   counts := make([]int, len(tokens))
   for i, t := range tokens {
       if err := s.db.WithContext(ctx).
           Raw("SELECT COUNT(*)"+base+" WHERE note_vectors.chunk_text LIKE ?", append(cntArgs, "%"+t+"%")...).
           Scan(&counts[i]).Error; err != nil {
           ... 返回错误
       }
   }
   ```
2. **高频词过滤**：`tokens = filterHighFreqTokens(tokens, counts, total)`；过滤后为空 → `return nil, nil`（关键词路不贡献，向量路照常）
3. **主检索**：沿用现有 OR'd LIKE 拼接（L324-331），`LIMIT` 改为 `limit * chunkCandidateMultiplier`（L332-333），`args` 用过滤后的 tokens
4. **排序截断**：`hits = rankKwHits(hits, tokens, limit)` 后返回
5. 更新 Debugw 日志（L341-344）：追加显示过滤掉的 token 数（如 `dropped`），便于观察效果

### 不改动

* `HybridRecall` 的 kwScore 重算循环（L460-474）——与 `rankKwHits` 口径一致，保留作合并层双保险，最小 diff

* `sortHybridHits`、`selectTopNotes`、卡片组装、向量路、`app.go`、前端

* `enableKeywordRecall` 保持 `true`（用户已开启，用于直接观察修复效果）

### 改动 4：新增测试文件 `internal/services/vector_service_keyword_test.go`

仅测试两个纯函数（不依赖 DB）：

* `TestFilterHighFreqTokens`：

  * 场景 1：`tokens=[数据 小时 代码 2061]`、`counts=[7459 800 300 84]`、`total=8000` → threshold=800 → 期望 `[小时 代码 2061]`（数据被滤，小时 800 未超阈值保留）

  * 场景 2：`total=300`、`counts=[60 10]` → threshold=max(30,100)=100 → 全部保留

  * 场景 3：全超阈值 → 返回空

* `TestRankKwHits`：

  * 场景 1：多词命中排前（含 `[小时 数据]` 的块排在只含 `[数据]` 的块前）

  * 场景 2：同分按块 id 升序（稳定）

  * 场景 3：候选多于 limit 时截断到 limit

## 四、假设与决策

| 项                  | 决策                                                                                     |
| ------------------ | -------------------------------------------------------------------------------------- |
| 高频阈值               | `max(totalChunks/10, 100)`，相对自适应；依据实测（数据 93%、2061 1%）                                  |
| 候选放大               | 复用现有 `chunkCandidateMultiplier=5`，不新增倍数常量                                              |
| 打分口径               | "块文本包含的 token 数"，与 `HybridRecall.kwScore` 一致                                           |
| 计数查询               | 每个 token 一次 `COUNT(*) LIKE`（最多 20 次），SQLite 全表扫描，当前 \~8000 行可接受；token 全被过滤时提前返回，不再发主查询 |
| 第二级（整篇笔记检索）        | 不做（用户已确认：命中后整篇笔记交付模型 token 过大、需二次定位 chunk，得不偿失）                                        |
| 合并层优化（note 级覆盖聚合等） | 本计划不包含，等第一级验证后再评估                                                                      |

## 五、验证步骤

1. `go build ./...` 编译通过
2. `go test ./internal/services/ -run "TestFilterHighFreqTokens|TestRankKwHits" -v` 新测试通过；`go test ./internal/services/` 现有测试不回归
3. 手工验证（重新编译应用，`enableKeywordRecall` 已为 true）：

   * 问"小时数据的代码是多少"→ 卡片**不再出现**执法函/企业端介绍/数据库设计说明书这类"只命中'数据'"的块；回答趋向直接给出 `CN=2061`

   * 问"我说的是2061"→ 仍精准命中 note 77（关键词路对高区分度 token 的兜底不退化）

   * 观察 `KeywordRecall` 的 Debugw 日志：tokens 应不含"数据"，hits 数 ≤ 5
4. 若效果达标，再评估是否继续做"合并层 note 级覆盖聚合"（另立计划）

