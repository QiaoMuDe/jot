# Tasks

- [x] Task 1: 定义记忆数据模型 `AIMemory`
  - [x] SubTask 1.1: 新建 `internal/models/ai_memory.go`，字段 `summary`（唯一索引）、`content`（text）、`CreatedAt`/`UpdatedAt`
  - [x] SubTask 1.2: 在 `internal/database/models.go` 的 `AllModels` 中注册 `&models.AIMemory{}`（子表无外键，放独立位置即可）

- [x] Task 2: 实现 `MemoryService`
  - [x] SubTask 2.1: 新建 `internal/services/memory_service.go`，构造器 `NewMemoryService(db)`
  - [x] SubTask 2.2: 实现 `Create(summary, content)`：按 `summary` 查重，存在则返回冲突，否则插入
  - [x] SubTask 2.3: 实现 `Update(id, summary, content)` / `Delete(id)` / `Get(id)` / `List()`
  - [x] SubTask 2.4: 关键路径打 fastlog 日志，风格对齐既有 service

- [x] Task 3: 实现并注册 Agent 工具 `manage_memory`（遵循 internal/agent/TOOLS.md）
  - [x] SubTask 3.1: 新建 `internal/agent/tools/manage_memory.go`：结构体 + `var _ tool.InvokableTool` 断言 + `Info()` + `InvokableRun()` + `NewManageMemory(memSvc *services.MemoryService, ctx *Context)`
  - [x] SubTask 3.2: 在工具文件定义常量 `MaxMemorySummaryRunes = 200`（报错）、`MaxMemoryContentRunes = 2000`（截断）
  - [x] SubTask 3.3: `action` 支持 `create`/`update`/`delete`/`get`/`list`，参数校验完备（必填、非法 action 报错）；`summary` 超长报错、`content` 超长截断并注明"详情过长已截断"
  - [x] SubTask 3.4: `create`/`update` 成功返回含「已保存记忆/已更新记忆:{summary}」的可见文本，供模型纳入回答展示
  - [x] SubTask 3.5: 实现 `ActionTextProvider.ActionText`（create→"新增记忆" / update→"更新记忆" / delete→"删除记忆" / list→"列出记忆" 等兜底"执行"）
  - [x] SubTask 3.6: 在 `internal/agent/registry.go` `buildTools` 追加 `tools.WrapWithError("manage_memory", tools.NewManageMemory(p.deps.Memory, p.ctx), p.ctx)`
  - [x] SubTask 3.7: 在 `internal/agent/tools/meta.go` `BuiltinTools()` 追加 `{Name: "manage_memory", Label: "管理长期记忆（保存/更新/删除/列出）"}`
  - [x] SubTask 3.8: 同步更新 `internal/agent/tools/doc.go` 与 `internal/agent/doc.go` 工具清单（含构造器名）

- [x] Task 4: 依赖装配（Deps + App + NewAgentService）
  - [x] SubTask 4.1: 在 `internal/agent/agent.go` 的 `Deps` 增加 `Memory *services.MemoryService`
  - [x] SubTask 4.2: 在 `app.go` 的 `App` 增加 `memoryService` 字段并初始化 `services.NewMemoryService(a.db)`（跟随既有 service 初始化处）
  - [x] SubTask 4.3: 在 `app.go` 两处 `agent.NewAgentService(agent.Deps{...})` 传入 `Memory`

- [x] Task 5: 提问时注入记忆（buildAIContextInstruction）
  - [x] SubTask 5.1: 在 `internal/app.go` 的 `buildAIContextInstruction` 末尾（环境信息段之后）追加【长期记忆】段，通过 `a.memoryService.List()` 拼入各记忆 `summary`；为空则跳过

- [x] Task 6: 构建与验证
  - [x] SubTask 6.1: `go build ./...` 通过
  - [x] SubTask 6.2: `go vet ./internal/agent/...` 通过
  - [x] SubTask 6.3: 重启应用，Agent 模式触发一次 `manage_memory` 调用（create/list），确认写入、注入与状态条生效

# Task Dependencies

- Task 2 依赖 Task 1（模型先行）
- Task 3 依赖 Task 2（工具使用 `MemoryService`）与 Task 4 的 Deps 字段（`buildTools` 取 `p.deps.Memory`）
- Task 4 依赖 Task 2（装配 `MemoryService`）
- Task 5 依赖 Task 2（注入读取 `List()`）
- Task 6 依赖 Task 1-5