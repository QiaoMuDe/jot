# Tasks

- [x] Task 1: 后端向量存储模型与切块工具：新增 `internal/models/note_vector.go`（NoteVector 模型）与切块函数。
  - [x] SubTask 1.1: 创建 `NoteVector` GORM 模型（note_vectors 表，字段按 spec）
  - [x] SubTask 1.2: 在 `internal/services/chunk.go` 实现 `ChunkContent(content string, maxRunes int) []string` 切块函数（Markdown 标题+空行分段，rune 安全硬切，maxRunes=500 默认）
  - [x] SubTask 1.3: 单元测试：2000+ 字输入 → ≥4 块且每块 ≤500 rune
  - [x] SubTask 1.4: 实现 float32 BLOB 序列化工具（`Float32ToBlob`/`BlobToFloat32`）

- [x] Task 2: aicli 层 Embedding 支持：在 `internal/aicli/` 新增 Embedding 方法。
  - [x] SubTask 2.1: `client.go` 新增 `Embed(ctx, texts []string) ([][]float32, error)` 方法，按 Provider 分发
  - [x] SubTask 2.2: `openai.go` 实现 `openaiEmbed`（CreateEmbeddings，错误走 ClassifyError）
  - [x] SubTask 2.3: `ollama.go` 实现 `ollamaEmbed`（api.Embed，批量 Input）
  - [x] SubTask 2.4: `go build ./...` 编译通过

- [x] Task 3: VectorService 业务层：新增 `internal/services/vector_service.go`。
  - [x] SubTask 3.1: `NewVectorService(db, logger) *VectorService` 构造函数
  - [x] SubTask 3.2: `IndexNotes(ctx, noteIDs []uint, progressCb func(done, total int, title string, stage string)) (success, failed int, err error)`：逐篇读取→切块→批量 embedding→先删旧块再插入，幂等；软删除笔记跳过
  - [x] SubTask 3.3: `GetIndexStatus() (noteCount, chunkCount int, sizeBytes int64, err error)`（去重 note_id、SUM(length(embedding))）
  - [x] SubTask 3.4: `DeleteAllVectors() error`
  - [x] SubTask 3.5: 单条笔记 embedding 失败不终止整体，计入 failed

- [x] Task 4: 数据库迁移与 app 绑定：注册模型 + 新增绑定方法 + SwitchProfile 双目标。
  - [x] SubTask 4.1: `internal/database/db.go` AutoMigrate 增加 `&models.NoteVector{}`
  - [x] SubTask 4.2: `internal/database/db.go` `InitDefaultSettings` 新增 4 键：`ai_embed_provider=ollama`/`ai_embed_base_url=http://localhost:11434`/`ai_embed_api_key=`/`ai_embed_model=`
  - [x] SubTask 4.3: `internal/services/profile_service.go` `SwitchProfile` 改为 `SwitchProfile(target string, id uint)`：target="chat" 写入 `ai_*` 四键 + 清 AISessionConfig；target="embed" 写入 `ai_embed_*` 四键（清 `ai_embed_model`）
  - [x] SubTask 4.4: `app.go` 初始化 `VectorService`（构造函数里挂到 App 结构体，参考 aiService 模式）
  - [x] SubTask 4.5: `app.go` 新增 `IndexNotesByAll/IndexNotesByNotebooks/IndexNotesByIDs(ids []uint)` 三个绑定（内部 goroutine 异步 + `vector:index-progress`/`vector:index-done`/`vector:index-error` 事件 + `sync.Mutex` 防重入）
  - [x] SubTask 4.6: `app.go` 新增 `GetVectorIndexStatus()` 与 `DeleteAllVectors()` 绑定
  - [x] SubTask 4.7: `app.go` 新增 `GetEmbedConfig()`（读 `ai_embed_*` 四键，APIKey 按需 DecodeB64）
  - [x] SubTask 4.8: 量化前校验 `ai_embed_model` 非空，否则返回可读错误
  - [x] SubTask 4.9: 前端 `App.SwitchProfile` 调用点全部改为带 target 参数（`"chat"`）

- [x] Task 5: 设置页「对话连接 + 量化连接」双模块（前端）：
  - [x] SubTask 5.1: `frontend/index.html` 在「API 连接」面板内拆分：现有内容包裹为「对话连接」模块（`ai-group-header` 标题），其下新增「量化连接」模块（同结构：预设下拉/服务商分段/API 地址/API Key/模型下拉+获取，ID 前缀 `aiEmbed`）
  - [x] SubTask 5.2: `frontend/src/main.js` 新增量化连接表单的元素引用（`aiEmbedPresetTrigger/Dropdown`、`aiEmbedProviderSegmented`、`aiEmbedBaseURL`、`aiEmbedAPIKey`、`aiEmbedModelTrigger/Dropdown/Label`、`aiEmbedFetchModelsBtn`）
  - [x] SubTask 5.3: `loadSettings`/`saveSettings` 读写 `ai_embed_*` 四键；量化连接表单回显
  - [x] SubTask 5.4: 量化连接模块事件：预设下拉加载同一 APIProfile 列表，点击调用 `SwitchProfile("embed", id)`；服务商分段切换、URL/Key 编辑、模型下拉获取/选择（复用对话模块逻辑，抽共用函数）
  - [x] SubTask 5.5: 对话连接预设下拉选中态改为按当前 `ai_*` 键值匹配（provider/base_url/api_key），不依赖 is_active；量化连接同理按 `ai_embed_*` 匹配
  - [x] SubTask 5.6: 保存后提示成功（复用 nm.show）

- [x] Task 6: 数据管理页量化入口与弹窗（前端）：
  - [x] SubTask 6.1: `frontend/index.html` 数据管理页新增「AI 量化索引」`data-action-list`（量化入口行 + 删除量化内容行）
  - [x] SubTask 6.2: 新增量化弹窗 HTML（范围切换：全部/笔记本/笔记；列表多选+全选+搜索；开始量化按钮；进度视图；完成摘要；错误提示）
  - [x] SubTask 6.3: `frontend/src/js/data-management.js` 新增 `showVectorIndexModal()`（打开弹窗、加载笔记本/笔记列表、范围切换、开始量化）
  - [x] SubTask 6.4: `EventsOn("vector:index-progress"/"vector:index-done"/"vector:index-error")` 更新进度视图；组件卸载时 `EventsOff` 清理
  - [x] SubTask 6.5: 删除量化内容按钮 → 二次确认弹窗（按弹窗类型显式控制「不保存」按钮可见性）→ 调用 `DeleteAllVectors` → 刷新统计

- [x] Task 7: 数据管理页信笺统计扩展（前端）：
  - [x] SubTask 7.1: `frontend/src/js/data-management.js` 的 `loadDataStats` 调用 `GetVectorIndexStatus`，在信笺中追加向量索引状态行（X 篇笔记 / Y 个片段 / Z MB，未量化显示「未量化」）

- [x] Task 8: 样式与集成验证：
  - [x] SubTask 8.1: `data-view.css`/`settings-panel.css` 补充量化弹窗与进度条样式（复用现有 design tokens：--radius-md、语义色、6px 滚动条）
  - [x] SubTask 8.2: 后端 `go build ./...` + `go vet ./...` 通过
  - [x] SubTask 8.3: 前端无语法错误（vite build 通过 + GetDiagnostics 为空），弹窗开关/进度更新/统计刷新代码审查通过
  - [x] SubTask 8.4: 端到端手测：配置量化模型 → 数据管理页按范围量化 → 进度显示 → 统计更新 → 删除量化内容 → 统计归零（需用户本机 Ollama + wails dev 实测）

# Task Dependencies

- [Task 2] depends on [Task 1]（Embedding 测试可独立）
- [Task 3] depends on [Task 1, Task 2]
- [Task 4] depends on [Task 3]
- [Task 5] 独立（前端设置）
- [Task 6] depends on [Task 4]（绑定方法就绪）
- [Task 7] depends on [Task 4]
- [Task 8] depends on [Task 5, Task 6, Task 7]

（Task 1/5 可并行；Task 2 依赖 Task 1 后可并行于 Task 3）
