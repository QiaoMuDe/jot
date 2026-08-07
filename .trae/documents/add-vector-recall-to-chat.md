# 向量召回接入 AI 问答流（验证阶段）

## Summary

在现有 `CallAIStream` 问答逻辑流中**新增向量召回**：每次对话无条件执行向量检索（不受「卡片召回」开关控制），从 `note_vectors` 表按余弦相似度召回 TopN 命中块，注入 system message 并发射召回卡片事件给前端。**现有关键词召回（`CardRecallSearch`）保持现有逻辑不变**：仅当「卡片召回」开关启用（`cardRecallEnabled`）时执行，本次只是在其基础上多加一路向量召回。两路召回结果在后端合并（去重）后统一注入与发射，前端零改动。

## Current State Analysis

- 向量索引能力已就绪：`note_vectors` 表（[note_vector.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/models/note_vector.go)）存有 `note_id / chunk_index / chunk_text / embedding BLOB / dim / model`；`VectorService`（[vector_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go)）具备 `IndexNotes / GetIndexStatus / DeleteAllVectors`，但**无查询（召回）方法**。
- 问答流召回入口：`CallAIStream`（[app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2087-L2132)）中 `if cardRecallEnabled { CardRecallSearch(...) }` 块——`CardRecallSearch` 返回 `CardRecallResult{FormattedText, Cards}`，`FormattedText` 注入 system message，`Cards` 经 `TruncateRecallCardsPreview(200)` 后通过 `ai:recall-cards` 事件发射给前端。
- embedding client 构建模式已存在：`startVectorIndex` 中从 `ai_embed_provider/base_url/api_key/model` 四键构建 `aicli.NewClient(...)`（[app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L1536-L1555)），且 `aicli.Embed`（含 `EmbedWithProgress`）已支持 OpenAI / Ollama 双 Provider。
- 前端召回卡片展示：`ai-chat.js` 监听 `ai:recall-cards` 事件（[ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2277)），渲染 `RecallCard` 列表，无需改动。
- `RecallCard` 结构：`ID / Title / Content / FileExt / Truncated`（[recall_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/recall_service.go#L41-L48)）。
- 笔记本范围过滤：`recallNotebookIDs []uint` 由前端传入（用户选择的召回笔记本范围），`CardRecallSearch` 通过 `SearchFull(..., notebookIDs...)` 过滤。向量召回需同样支持该过滤（`note_vectors` 表无 notebook_id，需 join `notes` 表）。

## Proposed Changes

### 1. 新增 `VectorRecall` 查询方法（internal/services/vector_service.go）

在 `VectorService` 新增方法，返回与 `CardRecallSearch` 同构的 `*CardRecallResult`，使 app.go 注入/发射逻辑可复用：

```go
// VectorRecall 向量召回：query 向量化 → note_vectors 余弦距离排序 → TopN 命中块
// 返回 CardRecallResult（FormattedText + Cards）；任一前置条件不满足时返回 nil（静默跳过）
// 前置条件：embedClient 非空且 Model 非空；note_vectors 表有当前模型的向量数据
func (s *VectorService) VectorRecall(ctx context.Context, query string, limit int, embedClient *aicli.Client, notebookIDs ...uint) *CardRecallResult
```

实现要点：
- **前置检查**：`embedClient == nil || embedClient.Model == ""` → 返回 nil；先查 `note_vectors` 表是否有当前 `Model` 的向量（`SELECT COUNT(*) WHERE model = ?`），无数据 → 返回 nil。
- **query 向量化**：`embedClient.Embed(ctx, []string{query})`，取首个向量作为查询向量。
- **召回查询**：
  - 读取 `note_vectors`（`WHERE model = ?`），若 `notebookIDs` 非空则 `JOIN notes n ON n.id = note_vectors.note_id WHERE n.notebook_id IN ? AND n.deleted_at IS NULL`。
  - 对每行 `BlobToFloat32(embedding)` 与查询向量计算余弦相似度（`Float32ToBlob/BlobToFloat32` 已存在于 [chunk.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/chunk.go)），按相似度降序取 TopN。
  - 同时查询 `notes` 表取 `Title / FileExt`（按 note_id）。
- **组装结果**：`FormattedText` 复用 `CardRecallSearch` 的格式——`--- 📄 《标题》 ---\n{命中块文本}\n`；`Cards` 填充 `RecallCard{ID: noteID, Title, Content: chunkText, FileExt, Truncated: false}`（Content 用命中块文本而非整篇笔记，天然解决 token 占用问题）。
- 日志：命中数量、query、耗时，用 `s.logger.Debugw/Infow`（fastlog），与现有一致。
- 注释遵守项目约定（函数级注释，中文）。

### 2. 新增向量召回调用（app.go，不改关键词召回逻辑）

在 [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2087-L2132) 的 `if cardRecallEnabled { ... }` 关键词召回块**之后**、技能提示词注入之前，新增一段**无条件执行**的向量召回逻辑（放在 `if cardRecallEnabled` 块外面，不依赖该开关）：

- **构建 embedding client**：从 `ai_embed_provider/base_url/api_key/model` 四键构建 `aicli.NewClient(...)`（复用 `GetEmbedConfig` 或直接读 setting，与 `startVectorIndex` 一致）。
- **调用 `VectorRecall`**：
  - query 取末条 user 消息内容（与关键词召回块的取法一致，可提取公共变量复用）。
  - `limit` 读 `ai_card_recall_limit` 设置（默认 5，现有逻辑保留）。
  - 传入 `recallNotebookIDs`。
  - 返回非 nil 时：`messages = appendToSystemMessage(messages, result.FormattedText)`；`Cards` 非空时序列化。
- **结果合并**：前端 `ai:recall-cards` 事件为覆盖式（[ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2277-L2281) 整体替换 `recallCards`），因此：
  - 若关键词召回块已发射过 `ai:recall-cards`（开关启用且有命中），则将关键词 Cards 与向量 Cards **合并去重**（按 `RecallCard.ID` note_id 去重）后**重新发射一次**覆盖事件，保证两路结果都展示。
  - 若关键词召回块未发射（开关关闭或无命中），直接发射向量 Cards。
  - `recallCardsJSON`（DB 持久化全量）同样合并去重后再赋值。
- **无向量数据 / 未配置 embedding 模型**：`VectorRecall` 返回 nil → 静默跳过，不影响关键词召回与对话。

**不改动**：`if cardRecallEnabled { CardRecallSearch(...) }` 块整体保留原样（调用、注入、发射逻辑都不动）。`CardRecallSearch` 函数保留在 recall_service.go 中。

### 3. 不加开关、不改前端

- 本次不新增任何设置项、不新增前端开关。
- 前端 `ai:recall-cards` 展示逻辑复用现有实现，零改动。
- `CallAIStream` 与 `CallAIStreamRegenerate`（委托 CallAIStream）自动同时生效。

## Assumptions & Decisions

- **关键词召回保持现状**：仅当「卡片召回」开关启用时执行（`if cardRecallEnabled { CardRecallSearch(...) }` 原样保留），不因本次改动受影响。
- **向量召回无条件执行**：每次对话都执行（不依赖 `cardRecallEnabled`）；向量库空/未配置 embedding 模型时静默跳过（零副作用）。
- **两路结果合并发射**：前端 `ai:recall-cards` 为覆盖式事件，故后端将关键词 Cards 与向量 Cards 按 `note_id` 去重合并后统一发射，避免后发覆盖先发；DB 持久化 `recallCardsJSON` 同样合并去重。
- 召回卡片 `Content` 用**命中块文本**（≤500 字）而非整篇笔记，解决 token 占用问题；前端点击跳转仍可用（`RecallCard.ID` 为 note_id）。
- 向量召回按 `model` 字段过滤：只召回与当前 embedding 模型一致的向量（换模型后旧向量不会被误用，需重新量化）。
- 余弦距离计算用纯 Go 全表扫描（个人笔记量级毫秒级），不引入 sqlite-vec 扩展依赖（与现有存储方案一致）。
- 距离阈值：不设硬阈值，仅按相似度降序取 TopN（与关键词召回的 TopN 行为对齐，便于对比验证效果）。

## Verification

1. `go build ./...`、`go vet ./...` 通过。
2. 手工验证（本机 Ollama + bge-m3 已量化部分笔记）：
   - 启动 `wails dev`，进入 AI 对话，发起对话（**不开启卡片召回开关**也应有向量召回卡片）。
   - 开启卡片召回开关后对话 → 召回卡片为关键词 + 向量合并去重结果。
   - 提问与已量化笔记相关的问题 → 前端展示召回卡片（标题 + 命中块摘要），回答引用了笔记内容。
   - 提问与未量化笔记/无关内容 → 无召回卡片，回答正常，无报错。
3. 回归验证：未量化任何笔记（`note_vectors` 空）时对话正常，无召回卡片、无异常日志。
4. 日志确认：`VectorRecall` 命中日志显示召回数量与 query。
