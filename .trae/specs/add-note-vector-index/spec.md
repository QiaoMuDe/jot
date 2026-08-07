# 笔记向量量化索引功能 Spec

## Why

AI 助手的卡片召回目前基于 gse 关键词 LIKE 全文检索，召回粒度粗、语义相关性差，且召回后整篇笔记注入模型占用大量 token。本功能引入本地向量索引：用户可在数据管理页对笔记（全部/指定笔记本/指定笔记）进行量化处理（embedding），存储到独立的 `note_vectors` 表，后续可支撑向量召回（本 spec 仅实现索引建立与管理，召回接入 AI 对话属于后续迭代）。

## What Changes

- **ADDED** 设置页「API 连接」面板拆分为两个并列设置模块：「对话连接」与「量化连接」，各自完整的预设下拉/服务商分段/BaseURL/APIKey/模型表单
- **ADDED** settings 表新增 4 个量化连接键：`ai_embed_provider`/`ai_embed_base_url`/`ai_embed_api_key`/`ai_embed_model`，与对话连接键 `ai_*` 完全独立
- **ADDED** APIProfile 预设表**零字段改动**：预设是"连接凭据集合"，对话/量化共用同一张表，切换时通过 `SwitchProfile` 的 target 参数写入对应键组
- **ADDED** `SwitchProfile(id uint)` 改为 `SwitchProfile(target string, id uint)`：target="chat" 写入 `ai_*` 四键并清 `AISessionConfig.model_name`；target="embed" 写入 `ai_embed_*` 四键（清 `ai_embed_model`）
- **ADDED** 数据管理页新增「AI 量化索引」操作分组：量化范围选择（全部/指定笔记本/指定笔记）、实时进度展示、删除全部量化内容
- **ADDED** 数据管理页信笺统计新增索引状态：已量化笔记数、片段总数、占用空间
- **ADDED** 新 GORM 模型 `NoteVector`（`note_vectors` 表）：笔记分块向量存储，字段含 note_id/chunk_index/chunk_text/embedding BLOB/dim/model
- **ADDED** 新 `VectorService` 业务层：切块、批量 embedding、写表、进度回调、删除、统计
- **ADDED** aicli 层新增 Embedding 支持：OpenAI `CreateEmbeddings` / Ollama `Embed` 双 Provider
- **ADDED** app.go 新增绑定方法：`IndexNotes`（按范围）、`GetVectorIndexStatus`、`DeleteAllVectors`；`GetEmbedConfig`（读 `ai_embed_*` 四键）
- **ADDED** 进度事件：`runtime.EventsEmit(ctx, "vector:index-progress", ...)` 推送切块/embedding/落库进度
- **注意**：不修改现有卡片召回链路，`note_vectors` 表与现有召回功能并行存在

## Impact

- Affected specs: 无（新功能）
- Affected code:
  - `internal/models/note_vector.go`（新增）
  - `internal/services/vector_service.go`（新增）
  - `internal/aicli/client.go`、`openai.go`、`ollama.go`、`types.go`（新增 Embedding 方法）
  - `internal/services/profile_service.go`（`SwitchProfile` 增加 target 参数）
  - `internal/database/db.go`（AutoMigrate 增加 NoteVector；默认设置新增 4 键）
  - `app.go`（新增绑定方法 + VectorService 初始化 + SwitchProfile 签名变更）
  - `frontend/index.html`（API 面板拆分两个模块 + 数据管理页新分组）
  - `frontend/src/main.js`（设置读写、预设切换 target、事件监听、页面逻辑）
  - `frontend/src/js/data-management.js`（量化操作 + 进度 + 统计）
  - `frontend/src/css/components/settings-panel.css`、`data-view.css`（样式，尽量复用现有类）

## ADDED Requirements

### Requirement: 量化连接设置模块（设置页）

系统 SHALL 将设置页「API 连接」面板拆分为两个并列设置模块：「对话连接」与「量化连接」，各自包含完整的配置表单：

- **对话连接**：现有全部内容（预设下拉、服务商分段、API 地址、API Key、模型下拉+获取），读写 `ai_provider`/`ai_base_url`/`ai_api_key`/`ai_model`
- **量化连接**：与对话连接同结构的表单（预设下拉、服务商分段、API 地址、API Key、模型下拉+获取），读写 `ai_embed_provider`/`ai_embed_base_url`/`ai_embed_api_key`/`ai_embed_model`
- 两个模块用 `ai-group-header` 分隔标题（「对话连接」「量化连接」），视觉上同属「API 连接」分区
- 两个模块的预设下拉共享同一 `APIProfile` 表数据，点击切换时分别调用 `SwitchProfile("chat", id)` / `SwitchProfile("embed", id)`
- 默认值：`ai_embed_provider="ollama"`、`ai_embed_base_url="http://localhost:11434"`、`ai_embed_api_key=""`、`ai_embed_model=""`
- 修改后保存触发 `saveSettings()` 持久化，逻辑与现有保存一致

#### Scenario: 分别配置对话与量化连接

- **WHEN** 用户在「对话连接」选 DeepSeek 预设并选模型，在「量化连接」选 Ollama 预设并选 `bge-m3`
- **THEN** 两套配置互不影响地持久化：对话用 DeepSeek 模型，量化用本地 Ollama bge-m3

#### Scenario: 切换对话预设不影响量化配置

- **WHEN** 用户在「对话连接」切换预设（如 DeepSeek → 智谱）
- **THEN** `ai_embed_*` 四键保持不变，量化连接表单不受影响

### Requirement: SwitchProfile 双目标写入

系统 SHALL 将 `SwitchProfile(id uint)` 修改为 `SwitchProfile(target string, id uint)`：

- target="chat"：将预设的 provider/base_url/api_key 写入 `ai_provider`/`ai_base_url`/`ai_api_key`，清空 `ai_model` 与所有 `AISessionConfig.model_name`（保持现有行为）
- target="embed"：将预设的 provider/base_url/api_key 写入 `ai_embed_provider`/`ai_embed_base_url`/`ai_embed_api_key`，清空 `ai_embed_model`（不清 `AISessionConfig`）
- `is_active` 标记保持现有全局唯一逻辑（表示最近一次切换），前端预设下拉选中态**改为按当前模块的 settings 键值匹配**（provider/base_url/api_key 与预设一致即高亮），不依赖 is_active 字段
- **BREAKING**：前端所有 `App.SwitchProfile(id)` 调用改为 `App.SwitchProfile("chat", id)` 或 `App.SwitchProfile("embed", id)`

#### Scenario: 切换量化预设

- **WHEN** 用户在「量化连接」模块点击 Ollama 预设
- **THEN** `ai_embed_provider/base_url/api_key` 更新为 Ollama 预设值，`ai_embed_model` 清空，`ai_*` 对话键不受影响

#### Scenario: 未配置量化模型

- **WHEN** 用户在数据管理页发起量化操作但 `ai_embed_model` 为空
- **THEN** 弹窗提示「请先在设置中配置量化连接与量化模型」，不发起索引

### Requirement: 量化范围选择（数据管理页）

系统 SHALL 在数据管理页新增「AI 量化索引」分组，提供量化入口弹窗，支持三种范围：

- **全部笔记**：一键选择，无需额外输入
- **指定笔记本**：展示笔记本多选列表（复用 `GetAllNotebooks`）
- **指定笔记**：展示笔记多选列表（复用 `GetAllNoteIDs` + 标题查询）

弹窗内包含：范围类型切换、对应列表（带全选/搜索过滤）、「开始量化」按钮。

#### Scenario: 按笔记本量化

- **WHEN** 用户选择「指定笔记本」并勾选 2 个笔记本，点击「开始量化」
- **THEN** 仅这两个笔记本下的笔记被切块并 embedding，弹窗切换为进度视图

### Requirement: 量化进度展示

系统 SHALL 在量化过程中实时展示进度：

- 后端按「笔记级」粒度推送事件：`runtime.EventsEmit(ctx, "vector:index-progress", done, total, noteTitle, currentNoteProgress)`
- 前端 `EventsOn("vector:index-progress", ...)` 接收，进度视图显示：整体进度条（done/total 笔记）、当前笔记标题、当前笔记内部进度（切块/embedding 子阶段）
- 量化完成后推送 `"vector:index-done"`（含成功/失败统计），弹窗显示完成摘要，可关闭
- 失败时推送 `"vector:index-error"`（含错误信息），前端展示错误，量化继续处理剩余笔记或终止（单条失败不终止整体）

#### Scenario: 量化 50 篇笔记

- **WHEN** 用户对 50 篇笔记发起量化
- **THEN** 进度视图依次显示 50 个笔记的处理进度，完成后显示「成功 X 篇 / 失败 Y 篇 / 片段总数 Z」

### Requirement: 删除量化内容

系统 SHALL 在「AI 量化索引」分组提供「删除量化内容」按钮：

- 点击后弹出二次确认（复用现有确认弹窗模式，显式控制「不保存」按钮可见性）
- 确认后调用 `DeleteAllVectors` 清空 `note_vectors` 表，完成后提示成功并刷新统计

#### Scenario: 删除全部向量

- **WHEN** 用户确认删除量化内容
- **THEN** `note_vectors` 表清空，信笺统计更新为「未量化」

### Requirement: 索引状态统计（信笺）

系统 SHALL 在数据管理页信笺统计中新增向量索引状态：

- `GetVectorIndexStatus` 返回：已量化笔记数（去重 note_id）、片段总数、BLOB 占用空间（SUM(length(embedding))）
- 前端 `loadDataStats` 展示为信笺中的一项：「向量索引：X 篇笔记 / Y 个片段（Z MB）」，未量化时显示「未量化」
- 与现有统计项（笔记数/标签/AI 会话等）同格式展示

#### Scenario: 查看统计

- **WHEN** 用户进入数据管理页
- **THEN** 信笺中包含向量索引状态行，数据与 `note_vectors` 表一致

### Requirement: 向量存储模型（NoteVector）

系统 SHALL 新增 GORM 模型：

```go
type NoteVector struct {
    ID         uint      `gorm:"primaryKey" json:"id"`
    NoteID     uint      `gorm:"index" json:"note_id"`
    ChunkIndex int       `gorm:"index" json:"chunk_index"`
    ChunkText  string    `gorm:"type:text" json:"chunk_text"`
    Embedding  []byte    `gorm:"type:blob" json:"-"`
    Dim        int       `json:"dim"`
    Model      string    `gorm:"size:128" json:"model"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

- `Embedding` 为 float32 小端字节序 BLOB（复用 vec-poc 验证的 `Float32ToBlob`/`BlobToFloat32` 思路，本仓库内实现）
- AutoMigrate 注册该模型，与现有表同库（`jot.db`）

### Requirement: VectorService 业务层

系统 SHALL 新增 `VectorService`，提供：

- `IndexNotes(ctx, noteIDs []uint, progressCb func(...)) (success, failed int, err error)`：按 note_id 列表逐篇处理——读取笔记内容 → 切块（复用 gse 分词无关的简单切块：按 Markdown 标题+空行分段，单块上限 500 rune）→ 批量 embedding → 先删该 note 旧向量再插入新块
- 幂等：以 note_id 为单位 `DELETE FROM note_vectors WHERE note_id=?` 后重新插入，重复量化不产生重复数据
- `GetIndexStatus() (noteCount, chunkCount int, sizeBytes int64, err error)`
- `DeleteAllVectors() error`
- 切块逻辑放 `internal/services/chunk.go` 或 `vector_service.go` 内（简单函数，不引入新依赖）
- 软删除笔记自动跳过：查询 note 时过滤 `deleted_at IS NOT NULL`

#### Scenario: 重复量化同一笔记

- **WHEN** 同一笔记被量化两次
- **THEN** 第二次量化后该笔记仅保留最新一批向量，无重复

### Requirement: aicli Embedding 支持

系统 SHALL 在 aicli 层新增 Embedding 能力（不破坏现有 Chat/Stream）：

```go
// EmbeddingRequest 单次 embedding 请求
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error)
```

- openai 分支：`sashabaranov/go-openai` 的 `CreateEmbeddings`（Model 用 client 的 Model 字段），失败映射 `ClassifyError`
- ollama 分支：`ollama/api` 的 `Client.Embed`（`api.EmbedRequest{Model, Input}`），返回 `Embeddings [][]float32`
- texts 为空返回空切片不调用外部 API
- 支持批量（一次性传入多个文本块）

#### Scenario: Ollama 不可用

- **WHEN** 量化时 Ollama 服务未启动（provider=ollama）
- **THEN** 返回含地址的可读错误，前端弹窗展示错误，量化终止

### Requirement: 量化任务并发安全

系统 SHALL 保证量化任务不阻塞主线程且可重入：

- `app.go` 中量化绑定方法使用 goroutine 异步执行（类似 CallAIStream 模式），立即返回「任务已启动」
- 单任务状态用全局互斥锁保护（`sync.Mutex`），防止并发发起多次量化；已在进行时拒绝新任务并提示

#### Scenario: 连续点击开始量化

- **WHEN** 用户两次点击「开始量化」
- **THEN** 第二次被拒绝，提示「量化任务正在进行中」

## MODIFIED Requirements

无。

## REMOVED Requirements

无。
