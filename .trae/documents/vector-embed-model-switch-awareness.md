# 嵌入模型切换感知（方案 A+B+C+D）

## Summary

让嵌入（索引）侧对"切换嵌入模型"产生完整感知，消除换模型后的三类误导：

* **A. 状态分类模型感知**：`classifyVectorNotes` 增加"当前模型"维度，旧模型的笔记归入"需重新嵌入"

* **B. 弹窗模型归属展示**：向量索引弹窗显示当前模型向量数与其他模型残留，其他模型向量存在时显示提示条

* **C. 启动索引一致性确认**：启动嵌入前检测库中是否存在其他模型向量，存在则前端弹确认框

* **D. 修改嵌入模型设置提示**：设置页保存时检测嵌入模型变更，toast 提示建议重建索引

## Current State Analysis

* `classifyVectorNotes`（vector\_service.go L322）只比切块文本，不比 `model` 字段 → 换模型后旧模型笔记显示"已最新"，统计说谎

* 三个导出入口 `GetVectorNoteOverview` / `GetUnindexedNoteIDs` / `GetStaleNoteIDs` 无模型参数

* `startVectorIndex`（app.go L1560）启动前只校验连接配置非空，不检查库中模型归属

* 向量索引弹窗（data-management.js `renderVectorIndexAllInfo` L991）只显示 未嵌入/需重新嵌入/已最新/总数/片段/占用，无模型信息

* 前端确认框模式：全局 `showConfirmDialog(message, confirmText, cancelText)`（data-management.js L723 已在本弹窗内使用，z-index 已验证可行）

* 全局通知：`window.showNotification(msg, type, duration)`

* 设置保存为批量提交：main.js `saveSettings()`（L11084）一次性 `SaveAllSettings(cfg)`，`ai_embed_model` 取自 `els.aiEmbedModelLabel`

* 测试：vector\_service\_test.go 的 `indexNoteForTest` 写入模型名固定为 `"test-embed"`，测试直接调用 `classifyVectorNotes(nilCtx())`

## Proposed Changes

### A. classifyVectorNotes 模型感知 — vector\_service.go

1. `classifyVectorNotes(ctx context.Context, currentModel string)`：

   * 存储块查询（L339 原始 SQL）增加 `nv.model` 列，构建 `modelSetByNote map[uint]map[string]bool`

   * 分类逻辑（L373 循环内）：`currentModel != "" && !modelSetByNote[note.ID][currentModel]` → 直接归入 `StaleIDs`（模型不匹配），跳过切块比对

   * `currentModel == ""` 时保持旧行为（模型维度不参与，兼容未配置场景）
2. 导出签名同步加参（调用方均在 app.go）：

   * `GetVectorNoteOverview(currentModel string)`

   * `GetUnindexedNoteIDs(currentModel string)`（未嵌入口径本就模型无关，统一签名便于传参）

   * `GetStaleNoteIDs(currentModel string)`
3. app.go 三处调用传入 `a.settingService.Get("ai_embed_model")`：

   * `GetVectorIndexOverview`（L1484）

   * `IndexNotesUnindexed`（L1415）

   * `IndexNotesStale`（L1430）
4. 测试更新（vector\_service\_test.go）：

   * 既有调用改为传 `"test-embed"`（与 indexNoteForTest 一致，既有断言不变）

   * 新增 `TestClassifyVectorNotesModelMismatch`：笔记以 `"old-model"` 索引（需给 indexNoteForTest 加 model 参数或新增辅助函数，推荐改为 `indexNoteForTest(t, db, note)` 内部保留 `"test-embed"`、另加 `indexNoteWithModelForTest(t, db, note, model)`），断言：

     * 用 `"new-model"` 分类 → 该笔记在 StaleIDs

     * 用 `"old-model"` 分类 → 该笔记在 UpToDateIDs

     * 空模型串分类 → 行为同旧逻辑（UpToDateIDs）

### B. 弹窗模型归属展示 — services + app.go + 前端

1. **vector\_service.go**：

   * 新增类型与方法（替代现有 `GetVectorModels`，一处查询两处用）：

     ```go
     type VectorModelCount struct {
         Model      string `json:"model"`
         ChunkCount int64  `json:"chunkCount"`
     }
     func (s *VectorService) GetVectorModelCounts() ([]VectorModelCount, error)
     // SELECT model, COUNT(*) FROM note_vectors GROUP BY model ORDER BY model
     ```

   * 删除 `GetVectorModels`；recall\_notes.go 预检报错处改用 `GetVectorModelCounts` 提取模型名（行为不变）
2. **app.go** **`GetVectorIndexOverview`** 返回结构追加字段：

   ```go
   CurrentModel       string             `json:"currentModel"`       // 当前配置的嵌入模型
   CurrentModelChunks int64              `json:"currentModelChunks"` // 当前模型向量块数
   OtherModels        []VectorModelCount `json:"otherModels"`        // 其他模型向量分布
   ```

   用 `GetVectorModelCounts` 一次查询后按 `currentModel` 拆分
3. **data-management.js**：

   * `loadVectorIndexStatus`（L949）捕获新字段

   * `renderVectorIndexAllInfo`（L991）在信息卡片与分段滑块之间插入提示条（仅 `otherModels` 非空时）：
     `⚠ 库中存在其他模型的向量：bge3（120 块）。当前模型「qwen3」仅能检索自身向量，建议执行「嵌入全部」以重建索引。`
     `currentModelChunks === 0` 时文案追加强调当前模型尚无任何向量
4. **data-view\.css**：新增 `.vector-index-model-hint` 样式（警告色 `var(--warning-color, #d97706)` 调、`--hover-bg` 底、圆角、与卡片间距一致）

### C. 启动索引一致性确认 — app.go + data-management.js

1. **app.go** 新增 Wails 绑定方法（复用 `CardRecallCheckResult` 类型）：

   ```go
   // CheckVectorIndexModelConsistency 检查库中向量模型与当前配置是否一致（供启动嵌入前确认）
   func (a *App) CheckVectorIndexModelConsistency() CardRecallCheckResult
   ```

   * `ai_embed_model` 为空 → `{OK: true}`（配置校验由 startVectorIndex 兜底）

   * 存在其他模型向量 → `{OK: false, Message: "库中存在其他模型的向量（bge3，共 120 块）。本次嵌入将把所选笔记重建为「qwen3」的向量；未纳入本次索引的笔记仍保留旧模型向量，无法被当前模型检索到。是否继续？"}`（文案由后端拼装，含具体模型名与块数）
2. **data-management.js** **`startVectorIndex`（L1153）**：方法选择与基本校验之后、切换进度视图之前插入：

   ```js
   const check = await app.CheckVectorIndexModelConsistency();
   if (check && !check.ok) {
       const confirmed = await showConfirmDialog(check.message, '继续嵌入', '取消');
       if (!confirmed) return;
   }
   ```

   后端不可用（绑定缺失）时静默跳过，不阻塞主流程

### D. 修改嵌入模型设置提示 — main.js

`saveSettings()`（L11084）在调用 `SaveAllSettings` 前：

* `const old = await window.go.main.App.GetAllSettings()` 读取旧值（失败则静默跳过提示）

* 若 `old.ai_embed_model` 与新 `cfg.ai_embed_model` 均非空且不同 → 保存成功后：
  `window.showNotification('嵌入模型已从「old」变更为「new」，建议在数据管理中重建向量索引', 'warning', 5000)`

## Assumptions & Decisions

* A 改变 stale 语义：换模型后"仅需重新嵌入"会把全部模型不匹配笔记刷成新模型（这正是期望行为），测试断言随之更新

* 当前模型直接读 settings 键 `ai_embed_model`（不需要 apiKey 解码，不走 GetEmbedConfig）

* 单笔记多模型在应用流程内不可达（IndexNotes 先删后插），B/C 的"其他模型"统计按全表 GROUP BY 即可，无需按笔记拆分

* C 的确认框使用全局 `showConfirmDialog`（已在本弹窗 L723 验证过 z-index 层级正确）

* D 纯前端比对（saveSettings 为批量保存，后端无单键变更钩子），不改动 SaveAllSettings 后端

* 有前端改动，完成后需 `npm run build` + `wails build`（或 wails dev 调试）

## Verification

1. `go build ./...`、`go vet ./internal/services/... ./internal/agent/...`
2. `go test ./internal/services/ -count=1`（重点：新增 TestClassifyVectorNotesModelMismatch + 既有分类测试全过）
3. 手动链路（wails dev）：

   * 换嵌入模型保存 → D 的 toast 出现

   * 打开向量索引弹窗 → B 的提示条显示旧模型名与块数，"需重新嵌入"数量 = 旧模型笔记数（A 生效）

   * 点"仅需重新嵌入"→ 用新模型重刷全部不匹配笔记；完成后提示条消失、当前模型块数正常

   * 库中仍有旧模型向量时点"开始嵌入"→ C 的确认框出现，取消不启动

   * recall\_notes 预检报错（上一轮功能）仍正常：错误信息中模型名来自 GetVectorModelCounts

