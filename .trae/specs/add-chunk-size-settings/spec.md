# 向量嵌入切块大小可配置化（target/max 双参数）Spec

## Why

当前切块大小被硬编码为 `chunkMaxRunes=600`，同时充当「理想块大小」与「硬上限」，且不与 embedded 模型的 token 上限挂钩。导致两个问题：

1. **结构清晰但节内容大**（如 `## 某节` 下有 1500 字整段）：会被 600 切碎，尽管模型完全能一次编码。
2. **结构不清晰、整篇纯文本**：只能按 600 rune 硬切，切得支离破碎。

需要把「理想落刀点」与「硬上限」解耦为两个可配置参数，接入设置页。

## What Changes

- **新增两个全局设置项**（设置页「AI 设置」→「向量嵌入」分组下）：`ai_chunk_target_rumes`（理想块大小 target）、`ai_chunk_max_rumes`（硬上限 max）。**全局统一一值**，不随模型分桶。
- **`ChunkContent` 签名改造**：`ChunkContent(content, maxRunes, meta)` → `ChunkContent(content, targetRunes, maxRunes, meta)`，实现「结构/段落边界优先在 target 落刀，仅单个不可分语义单元真超 max 才硬切」。
- **`chunkMaxRunes` 常量移除**，写路径 `IndexNotes` 与状态比对 `classifyVectorNotes` 共同改为从配置读取同一份 target/max，保证口径一致。
- **返回值语义不变**：`ChunkContent` 输出块集合的不可破坏性约束（空节丢弃、补父级标题链、补表头）保持不变。

## Impact

- Affected specs：`settings-maintenance.md`（新增两个 int 设置项的标准四文件链路）
- Affected code：
  - `internal/database/db.go`（种子默认值）
  - `internal/services/types.go`（`SettingsConfig` 字段 + `GetAllSettings`/`SaveAllSettings` + clamp）
  - `internal/services/chunk.go`（`ChunkContent` 接受区间参数 + 落刀/硬切逻辑）
  - `internal/services/vector_service.go`（`chunkMaxRunes` 常量移除，改从配置读取 target/max，两调用点统一）
  - `frontend/index.html`（AI 面板「向量嵌入」分组两个数字输入）
  - `frontend/src/main.js`（`els` 注册 + `loadSettings` 回显 + `saveSettings` 收集）
  - `internal/services/chunk_test.go`、`vector_service_test.go`、`types_test.go`（测试）

## ADDED Requirements

### Requirement: 设置项持久化
系统 SHALL 提供两个全局整数设置项 `ai_chunk_target_rumes` 与 `ai_chunk_max_rumes`，经标准设置链路（settings 表 + `SettingsConfig`）持久化。

#### Scenario: 默认值
- **WHEN** 首次安装或 setting key 缺失
- **THEN** `target=600`、`max=1500`

#### Scenario: 保存时的范围限制（clamp）
- **WHEN** 用户保存设置
- **THEN** `target` 钳制在 `[1, max]`；`max` 钳制在 `[100, 10000]` 且 `max >= target`；越界自动夹取至合法区间，不拒绝保存

### Requirement: 切块使用区间参数
`ChunkContent` SHALL 接收 target 与 max 两个参数，并遵循「结构/段落边界优先在 target 落刀，超限聚合；仅单个不可分语义单元真超 max 才硬切」的切块规则。

#### Scenario: 结构清晰、大节整段
- **WHEN** 某标题节内为一大段（rune 数介于 target 与 max 之间，无空行可切）
- **THEN** 该大段**整段成为一块**，不再被切成 target 那么碎；仅当 rune 数超过 max 才硬切

#### Scenario: 纯文本、多段落聚合
- **WHEN** 无标题的纯文本含多个段落（空行分隔）
- **THEN** 段落聚合到触及 target 时在段落边界落刀；不切断段落；仅当单段本身超过 max 才硬切

#### Scenario: 写路径与状态比对口径一致
- **WHEN** `IndexNotes` 与 `classifyVectorNotes` 各自切块
- **THEN** 两者读取**同一份** target/max 配置值，避免内容未变却误判「需重新嵌入」

## REMOVED Requirements

### Requirement: 固定 `chunkMaxRunes=600` 常量
**Reason**: 常量同时负担 target 与 max 两重职责，且与模型能力脱节，是本次要解耦的对象。
**Migration**: 由两个配置项取代；存量库因种子增量插入自动获得默认值 600/1500。切块输出因 max 放宽与落刀逻辑调整可能与旧块不一致，`classifyVectorNotes` 会把这类笔记判为「需重新嵌入」——属预期的自愈路径，用户在数据管理页重新索引一次即稳定。