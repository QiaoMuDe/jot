# 向量召回改用 sqlite-vec 函数式检索

## Summary

将 `VectorRecall` 的检索逻辑从"全量加载 + Go 循环余弦计算"改为 **sqlite-vec 函数式方案**：通过 `modernc.org/sqlite/vec` 扩展注册，在 SQL 内用 `vec_distance_cosine(embedding, vec_f32(?))` 完成距离计算与 TopN 排序。**现有** **`note_vectors`** **表结构不变、已有数据直接可用、无需重新量化**。

## Current State Analysis

* 依赖：主项目 [go.mod](file:///d:/资源池/下水道/Dev/本地项目/jot/go.mod#L94) 中 `modernc.org/sqlite v1.23.1`（indirect），不含 `vec` 子包；POC 验证 v1.51.0 内置 `modernc.org/sqlite/vec`（blank import 自动注册 sqlite-vec v0.1.9，glebarez v1.11.0 组合可用）。

* [VectorRecall](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L179-L358) 当前检索段：`q.Find(&vectors)` 全量载入内存 → 逐条 `BlobToFloat32` + `cosineSimilarity`（纯 Go 循环）→ 排序 TopN。

* 引用点确认：`cosineSimilarity` 仅 VectorRecall 内部使用（可删）；`Float32ToBlob` 被 IndexNotes 写入使用（保留）；`BlobToFloat32` 被 chunk\_test.go 测试使用（保留）。

* 相邻块补充逻辑当前依赖内存中的全量 `vectors`，改 SQL 后需改为"命中笔记的块"二次查询。

## Proposed Changes

### Change 1: 升级依赖（go.mod / go.sum）

* 执行 `go get modernc.org/sqlite@v1.51.0`，将 `modernc.org/sqlite` 从 v1.23.1（indirect）升级为 v1.51.0（direct）。

* 连带更新配套间接依赖（libc/mathutil/memory 等），由 go 工具自动解析（本地 module cache 已有 POC 验证过的版本组合）。

### Change 2: 注册扩展（internal/database/db.go）

* 在 [db.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/database/db.go#L1-L13) 增加 blank import：

  ```go
  // blank import 注册 sqlite-vec 扩展（sqlite3_auto_extension，对新打开的连接自动生效）
  _ "modernc.org/sqlite/vec"
  ```

* 时机安全：package init 早于 `InitDB` 运行时打开连接。

### Change 3: 重写 VectorRecall 检索段（internal/services/vector\_service.go）

**3.1 检索 SQL（替代 L209-L245 全量加载+循环+排序）**：

```sql
SELECT id, note_id, chunk_index, chunk_text, model
FROM note_vectors
WHERE model = ?
  AND vec_distance_cosine(embedding, vec_f32(?)) < 1.0
  [JOIN notes ON notes.id = note_vectors.note_id
   AND notes.deleted_at IS NULL AND notes.notebook_id IN ?]  -- 仅笔记本过滤时
ORDER BY vec_distance_cosine(embedding, vec_f32(?)) ASC
LIMIT ?
```

* 参数绑定顺序：`model`、`queryVecJSON`、`queryVecJSON`、`limit`（JOIN 笔记本过滤时插入 notebookIDs 于 LIMIT 前）。

* `queryVecJSON` 由 `json.Marshal(queryVec)` 生成（标准 JSON 数组，`vec_f32` 直接解析）。

* `dist < 1.0` 等价原 `score > 0` 过滤（余弦距离 = 1 - 余弦相似度），保持行为一致。

* 返回行顺序即距离升序（最近优先），替代原排序逻辑。

* 新增 `encoding/json` import。

**3.2 相邻块补充数据源调整（替代依赖全量 vectors）**：

* 命中行（TopN）收集 `note_id` 集合 → 二次查询这些笔记的全部块：

  ```sql
  SELECT id, note_id, chunk_index, chunk_text, model
  FROM note_vectors WHERE model = ? AND note_id IN ?
  ```

* 内存按 note\_id 分组 + 按 chunk\_index 升序排序（现有逻辑保留），相邻块 `±adjacentBlocks` 取块逻辑不变。

* 命中笔记顺序按检索距离升序（= 原 cands 首次出现顺序语义，保持相关度排序稳定）。

**3.3 清理**：

* 删除 `cosineSimilarity` 函数（无其他引用）。

* `BlobToFloat32`/`Float32ToBlob` 保留（分别被测试/IndexNotes 使用）。

### 不动的部分

* 前置检查（embedding 模型未配置、当前模型无向量数据 → nil）

* query 向量化、笔记元信息查询、卡片组装（命中块+相邻块按 ChunkIndex 拼接、1200 rune 截断）、`adjacentBlocks`/`maxCardRunes` 常量

* 表结构、数据、前端、召回事件链路

## Assumptions & Decisions

* **升级版本**：modernc.org/sqlite v1.51.0（POC 已验证组合：glebarez/sqlite v1.11.0 + sqlite v1.51.0 + vec v0.1.9）。

* **距离阈值**：`< 1.0` 等价原 `score > 0`，保持召回行为一致。

* **vec\_f32 传参**：query 向量用 `json.Marshal` 生成 JSON 数组字符串绑定（POC 同款用法）。

* **不做 vec0 虚拟表**：函数式已满足个人量级需求，零表结构改动。

* **升级依赖风险**：modernc 1.23.1→1.51.0 大版本跳跃，但仅被 glebarez/go-sqlite 使用且组合已在 POC 验证；实施后需 go build + 回归。

## Verification

1. `go build ./...` 通过（升级依赖后全项目编译）。
2. POC 回归：`cd vec-poc && go test ./... -run TestProbeVecLoads` 确认扩展加载（v0.1.9）。
3. 临时 Go 测试（实施后删除）：在 database 层跑通 `vec_distance_cosine` SQL（用 mock 向量验证 TopN 顺序与笔记本过滤）。
4. 手工验证（wails dev）：

   * 已有向量数据无需重新量化，开启卡片召回 → 提问 → 召回卡片正常出现、内容含命中块+相邻块

   * 选定笔记本范围后召回只命中该笔记本

   * 同笔记多命中合并为一张卡片

