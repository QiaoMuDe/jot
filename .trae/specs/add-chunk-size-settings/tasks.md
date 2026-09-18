# Tasks

- [x] Task 1: 新增设置项持久化（后端）——`ai_chunk_target_rumes` / `ai_chunk_max_rumes`
  - [x] 1.1 `internal/database/db.go` `InitDefaultSettings()` defaults 末尾追加两 key（`ai_chunk_target_rumes="600"`、`ai_chunk_max_rumes="1500"`），加中文注释说明用途
  - [x] 1.2 `internal/services/types.go` `SettingsConfig` 新增 `AIChunkTargetRunes int` / `AIChunkMaxRunes int`，json tag = key
  - [x] 1.3 `GetAllSettings()` 读取映射：`AIChunkTargetRunes: parseIntSetting(s.Get("ai_chunk_target_rumes"))`（含默认值兜底），`AIChunkMaxRunes` 同理
  - [x] 1.4 `SaveAllSettings()` 写入映射：`strconv.Itoa(...)` 两 key
  - [x] 1.5 `SaveAllSettings()` clamp 校验区：target 夹 `[1, max]`；max 夹 `[100, 10000]` 且 `max>=target`（参照 PageSize 模式）

- [x] Task 2: 设置页 UI（前端）
  - [x] 2.1 `frontend/index.html`「AI 设置」→「向量嵌入连接」分组下新增两个数字输入 `.ai-setting-item`（label「理想块大小」「最大块大小」+ `.settings-input` 文本说明 + 单位 rune）
  - [x] 2.2 `main.js` `els` 注册 `aiChunkTargetRunesInput`/`aiChunkMaxRunesInput`
  - [x] 2.3 `loadSettings()` 回显 `cfg.ai_chunk_target_rumes`/`cfg.ai_chunk_max_rumes`
  - [x] 2.4 `saveSettings()` 收集两字段（转为 int）
  - [x] 2.5 两输入 `change` 事件触发 `saveSettings()` + 通知（参照既有自保存控件）

- [x] Task 3: `ChunkContent` 区间化改造
  - [x] 3.1 `chunk.go` `ChunkContent(content, targetRunes, maxRunes int, meta ChunkMeta)` 签名改造
  - [x] 3.2 正文/空行分支落刀条件：段落聚合到 `curRunes >= targetRunes` 时在边界 flush（`targetRunes` 作落刀点）
  - [x] 3.3 flush 内硬切判定改用 `maxRunes`（`runeLen(prefix)+1+runeLen(text) > maxRunes` 才硬切），`splitWithHeading` 预算改传 `maxRunes`
  - [x] 3.4 校验 `targetRunes >= maxRunes` 时降级为 `targetRunes = maxRunes`（防御），防止 target > max 造成负预算

- [x] Task 4: `vector_service.go` 接入配置
  - [x] 4.1 移除 `chunkMaxRunes` 常量
  - [x] 4.2 `VectorService` 新增私有方法 `chunkSizes() (target, max int)`：从 `models.Setting` 表读两 key，缺失/非法回退默认 600/1500，并 clamp（`target 1..max`、`max>=100`）
  - [x] 4.3 `IndexNotes` 用 `chunkSizes()` 取值传给 `ChunkContent`
  - [x] 4.4 `classifyVectorNotes` 用同一 `chunkSizes()` 取值（与写路径共用同一份取值，保证口径一致）

- [x] Task 5: 测试与回归
  - [x] 5.1 `chunk_test.go`：新增 target/max 区间用例——大节整段（介于 target..max）整块保留、纯文本段落聚到 target 落刀、单段超 max 硬切；现有测试改签名传原值（500,500）保持旧行为兼容
  - [x] 5.2 `types_test.go`：`SaveAllSettings` clamp 断言（target>max 夹取、越界夹取）
  - [x] 5.3 全量 `go build/vet/test ./internal/services/` 通过；`go fmt` 无输出

- [x] Task 6: 文档同步
  - [x] 6.1 AGENTS.md 临时记忆追加本次改动（切块大小可配置化，含默认值与口径一致性约束）；若超 5 条按三步轮换