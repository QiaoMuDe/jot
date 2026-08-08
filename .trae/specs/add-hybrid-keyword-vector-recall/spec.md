# 混合关键词+向量召回 Spec

## Why
当前卡片召回仅依赖向量语义检索，对于精确关键词匹配（如专有名词、代码标识符、人名地名）的召回质量不足。将 GSE 中文分词关键词检索重新引入，与向量检索混合，取长补短，提升整体召回质量。

## What Changes
- 重新引入 `github.com/go-ego/gse` 依赖（含 EMBED 嵌入式词典），用于 query 中文分词
- 在 `vector_service.go` 新增 `KeywordRecall` 方法：对 `note_vectors.chunk_text` 字段做 GSE 分词 + LIKE 匹配
- 新增 `HybridRecall` 方法：并行执行向量检索与关键词检索，按 (note_id, chunk_index) 去重合并，双命中优先排序
- `VectorRecall` 改为内部委托 `HybridRecall`（对外签名不变，app.go 绑定无需改动）
- 前端零改动（`ai:recall-status` 事件 + 卡片展示逻辑完全复用）

## Impact
- Affected code: `go.mod` / `go.sum`（新增 gse 依赖）、`internal/services/vector_service.go`（核心改动）
- 不影响：前端、app.go 绑定层、量化流程、数据库 schema

## ADDED Requirements

### Requirement: GSE 分词器懒加载
系统 SHALL 在首次关键词检索时通过 `sync.Once` 懒加载 GSE 分词器实例，避免启动时阻塞。通过 `import _ "github.com/go-ego/gse/pkg/embed"` 使用 EMBED 嵌入式词典（编译进二进制），词典始终可用，无需外部文件、无需回退。

#### Scenario: 首次召回触发分词器加载
- **WHEN** 首次执行关键词检索
- **THEN** GSE 分词器初始化（约 100-200ms），后续调用直接复用

### Requirement: 关键词检索
系统 SHALL 提供 `KeywordRecall` 方法，对用户 query 进行 GSE 分词后，在 `note_vectors.chunk_text` 字段上执行 LIKE 匹配，返回命中的 NoteVector 记录列表。

#### Scenario: 正常关键词命中
- **GIVEN** query = "数据库设计模式"，已量化笔记含 "数据库索引设计"
- **WHEN** GSE 分词为 ["数据库", "设计", "模式"]，LIKE 匹配 chunk_text
- **THEN** 返回命中块列表，命中 token 数越多分数越高

#### Scenario: 无关键词命中
- **WHEN** 分词结果无任何 token 命中 chunk_text
- **THEN** 返回空列表，不报错

#### Scenario: query 过短
- **WHEN** query 分词后有效 token 数为 0
- **THEN** 跳过关键词检索，仅返回向量结果

### Requirement: 混合检索合并
系统 SHALL 提供 `HybridRecall` 方法，执行向量检索与关键词检索后合并结果：
1. 按 (note_id, chunk_index) 去重
2. 排序优先级：双命中（向量+关键词）> 仅向量命中 > 仅关键词命中
3. 同优先级内按向量距离升序
4. 合并后复用现有相邻块扩展 + 卡片组装逻辑

#### Scenario: 双路命中同一块
- **GIVEN** 某 chunk 同时被向量和关键词命中
- **WHEN** 合并去重
- **THEN** 该块标记为双命中，排序优先级最高

#### Scenario: 仅向量命中
- **WHEN** 向量有命中但关键词无命中
- **THEN** 正常返回向量结果（关键词路静默跳过）

#### Scenario: 仅关键词命中
- **WHEN** 向量无命中（如模型无数据）但关键词有命中
- **THEN** 返回关键词命中结果

#### Scenario: 双路均无命中
- **WHEN** 向量和关键词均无命中
- **THEN** 返回 (nil, nil) 静默跳过

### Requirement: 笔记本范围过滤
关键词检索 SHALL 支持与向量检索相同的 notebookIDs 过滤（JOIN notes 表过滤 notebook_id）。

## MODIFIED Requirements

### Requirement: VectorRecall
`VectorRecall` 方法内部改为委托 `HybridRecall`，对外签名 `(ctx, query, limit, embedClient, notebookIDs...) (*CardRecallResult, error)` 不变。当 embedClient 为 nil 或模型无数据时，仍可仅走关键词检索路。
