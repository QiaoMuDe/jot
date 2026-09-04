# 全局记忆空间（阶段1：记忆表 + manage_memory 工具 + 注入）Spec

## Why

现有三套记忆机制（会话摘要压缩、上下文 token 窗口、笔记向量召回）均为**隐式被动**：不可见、不可编辑、按会话或按检索作用域。缺少一个**跨会话、用户与 AI 双向可写的显式长期记忆层**，用户或 AI 固化的事实/偏好没有落点，也无法在提问时被主动注入。

阶段1交付"数据 + 工具 + 注入"三件套，跑通"AI 增删改查记忆 → 提问时注入"闭环。前端 CRUD 界面（阶段2）明确不在本 spec 范围内。

## What Changes

- **新增记忆表 `a_memories`**（模型 `AIMemory`）：`summary`（简短描述，用于注入）+ `content`（详情）+ 时间戳，`summary` 唯一约束去重。
- **新增 `MemoryService`**：增/删/改/查 + 列出（全量），新建时按 `summary` 去重。
- **新增 Agent 工具 `manage_memory`**：完全遵循 [internal/agent/TOOLS.md](../internal/agent/TOOLS.md) 6 步规范新增——`tools/manage_memory.go` + `registry.go` 注册 + `meta.go` 展示文案 + 两个 `doc.go` 清单。
- **记忆变更必须可见**：`manage_memory` 的 `create`/`update` 成功返回文本须含「已保存记忆：{summary}」这类可见行，供模型直接纳入回答展示；同时依赖实现 `ActionTextProvider`（新增/更新记忆动作文案）让工具状态条同步可见。
- **超长处理策略**：`summary` 超限则工具**报错**（提示模型精简后重试）；`content` 超限则自动**截断**保存（不阻塞写回）。上限常量：`MaxMemorySummaryRunes = 200`（报错），`MaxMemoryContentRunes = 2000`（截断）。
- **依赖装配**：`agent.Deps` 增加 `Memory` 字段；`App` 增加 `memoryService` 字段并在 `NewAgentService` 传入。
- **提问时注入**：在 `buildAIContextInstruction` 末尾追加【长期记忆】段落，把全部记忆的 `summary` 拼入 system prompt（Chat/Agent 两模式共用）。空则跳过。
- **前端零改动**：`manage_memory` 注册后，前端工具清单、`ai:tool-status` 状态条、历史回放明细均自动支持（TOOLS.md §8，工具名直显英文，无需维护映射）。
- **不在本 spec 范围**（后续阶段）：前端记忆 CRUD 管理界面 / 记忆面板；`category`（偏好/事实/行动项）；`source`（用户写/AI 写）；`enabled` 单条开关；选择性注入/优先级。

## Impact

- **Affected specs**：Agent 内置工具链（`manage_*` 系列）、AI 上下文注入（`buildAIContextInstruction`）、数据模型注册。
- **Affected code**：
  - `internal/models/`　新增 `ai_memory.go`
  - `internal/database/models.go`　`AllModels` 注册一行
  - `internal/services/`　新增 `memory_service.go`
  - `internal/agent/tools/manage_memory.go`　新增
  - `internal/agent/registry.go`　`buildTools` 注册一行
  - `internal/agent/tools/meta.go`　`BuiltinTools()` 一行
  - `internal/agent/tools/doc.go`、`internal/agent/doc.go`　清单同步
  - `internal/agent/agent.go`　`Deps` 增加 `Memory` 字段
  - `app.go`　`App` 字段 + `NewAgentService` 传参 + `buildAIContextInstruction` 注入

## ADDED Requirements

### Requirement: 记忆数据模型

系统 SHALL 提供一个持久化记忆实体 `AIMemory`，至少包含：

- `summary`（简短描述，供注入系统提示词）
- `content`（详情，记录具体记忆）

并通过表中唯一约束保证 `summary` 不重复。模型注册于 `database.AllModels`，随 `AutoMigrate` 自动建表。

#### Scenario: 首次建表
- **WHEN** 应用启动执行 `AutoMigrate(database.AllModels...)`
- **THEN** 生成 `a_memories` 表，含 `summary`/`content`/时间戳字段

#### Scenario: summary 去重约束
- **WHEN** 尝试插入两个相同 `summary` 的记忆
- **THEN** 数据库唯一约束 / `MemoryService` 层拦截，不产生重复条目

### Requirement: MemoryService 记忆读写服务

系统 SHALL 提供 `MemoryService`，暴露增、删、改、查与列出能力，新建时按 `summary` 去重（已存在则返回既有条目信息，不重复写入）。

#### Scenario: 新建
- **WHEN** 调用 `Create(summary, content)`
- **THEN** 若 `summary` 已存在则不新增并返回冲突提示；否则落库一条新记忆

#### Scenario: 删除/更新/查询/列出
- **WHEN** 按 `id` 调用 `Delete` / `Update` / `Get`
- **THEN** 对应记忆被删除 / 更新 / 返回；`List` 返回全部记忆

### Requirement: Agent 工具 manage_memory

系统 SHALL 提供 Agent 工具 `manage_memory`，允许模型在 ReAct 循环中对记忆执行增/删/改/查/列出，并严格遵守 [TOOLS.md](../internal/agent/TOOLS.md) 工具规范（`WrapWithError` 包装、纯文本返回、参数校验、无事件发射、不循环依赖父包）。

**工具动作（action 参数）**：
- `create`：新增记忆（`summary` + `content`）
- `update`：按 id 更新记忆
- `delete`：按 id 删除记忆
- `get`：按 id 查询记忆详情
- `list`：列出全部记忆（含 id / summary / content 摘要）

#### Scenario: 模型新增记忆
- **WHEN** 模型调用 `manage_memory`，`action=create` 且携带 `summary`、`content`
- **THEN** 记忆写入，工具返回「已保存记忆：{summary}」等纯文本（模型可纳入回答展示）；重复 `summary` 时返回冲突说明而非报错

#### Scenario: 模型更新记忆
- **WHEN** 模型调用 `manage_memory`，`action=update`
- **THEN** 记忆更新，工具返回「已更新记忆：{summary}」等可见文本，供模型展示

#### Scenario: 记忆超长
- **WHEN** `summary` 超过 `MaxMemorySummaryRunes`（200）
- **THEN** 工具返回错误，提示模型精简后重试（不落库），经 `WrapWithError` 回填模型
- **WHEN** `content` 超过 `MaxMemoryContentRunes`（2000）
- **THEN** 自动截断到上限内保存，不报错，返回文本注明"详情过长已截断"

#### Scenario: 模型列出记忆
- **WHEN** 模型调用 `manage_memory`，`action=list`
- **THEN** 返回全部记忆的结构化文本列表（含 id 以便后续 update/delete）

#### Scenario: 参数校验失败
- **WHEN** 必填参数缺失或 `action` 非法
- **THEN** 工具返回错误，经 `WrapWithError` 回填模型继续推理，不中断 ReAct 循环

### Requirement: 提问时注入记忆

系统 SHALL 在每次 AI 提问装配系统提示词时将记忆注入：在 `buildAIContextInstruction` 末尾追加【长期记忆】段落，拼入全部记忆的 `summary`（详情 `content` 不注入）。记忆为空时跳过该段落。Chat 模式与 Agent 模式使用同一注入路径。

#### Scenario: 存在记忆时提问
- **WHEN** 用户发起提问，且记忆表非空
- **THEN** system prompt 末尾包含【长期记忆】段落，列出各记忆的 `summary`

#### Scenario: 无记忆时提问
- **WHEN** 记忆表为空
- **THEN** 不注入记忆段落，system prompt 与改造前一致

## MODIFIED Requirements

（无，本 spec 不修改既有需求，只新增。）

## REMOVED Requirements

（无。）