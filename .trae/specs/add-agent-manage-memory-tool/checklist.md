# Checklist — 全局记忆空间（阶段1）

## 数据模型
- [x] `internal/models/ai_memory.go` 存在，含 `summary`（唯一索引）、`content`（text）、`CreatedAt`/`UpdatedAt`，字段带 json 标签，风格对齐既有模型
- [x] `internal/database/models.go` 的 `AllModels` 注册了 `&models.AIMemory{}`

## MemoryService
- [x] `internal/services/memory_service.go` 存在，构造器 `NewMemoryService(db)` 签名正确
- [x] `Create` 会按 `summary` 去重：重复时返回冲突信息而非插入重复条目
- [x] `Update`/`Delete`/`Get`/`List` 均已实现，`List` 返回全量记忆（含 id 供后续 update/delete）

## Agent 工具 manage_memory
- [x] `internal/agent/tools/manage_memory.go` 存在，含 `var _ tool.InvokableTool` 断言
- [x] `Info().Desc` 说明"何时调用/参数含义"，`ParamsOneOf` 覆盖必要的 `action`/`summary`/`content`/`id` 参数
- [x] `action` 支持 create/update/delete/get/list；必填缺失与非法 action 均返回 error
- [x] `summary` 超 `MaxMemorySummaryRunes`(200) 返回 error 且不落库；`content` 超 `MaxMemoryContentRunes`(2000) 截断保存并注明"详情过长已截断"
- [x] `create`/`update` 成功返回含「已保存记忆/已更新记忆:{summary}」的可见文本（模型可纳入回答展示）
- [x] 工具实现 `ActionTextProvider`，create/update/delete/list 有中文动作文案，兜底"执行"
- [x] `internal/agent/registry.go` `buildTools` 已用 `WrapWithError` 注册 `manage_memory`，并从 `p.deps.Memory` 取依赖
- [x] `internal/agent/tools/meta.go` `BuiltinTools()` 已追加 `manage_memory` 展示项（名称与注册名一致）
- [x] `internal/agent/tools/doc.go` 与 `internal/agent/doc.go` 清单已同步（含构造器名）

## 依赖装配
- [x] `agent.Deps` 增加 `Memory *services.MemoryService` 字段
- [x] `app.go` 的 `App` 增加 `memoryService` 字段并初始化
- [x] `app.go` 所有 `agent.NewAgentService(...)` 调用处均传入 `Memory`

## 记忆注入
- [x] `buildAIContextInstruction` 末尾追加【长期记忆】段落，拼入全量记忆的 `summary`；记忆为空时跳过
- [x] 注入仅含 `summary`，不注入 `content` 详情

## 构建与行为验证
- [x] `go build ./...` 通过
- [x] `go vet ./internal/agent/...` 通过
- [x] 应用启动后 `a_memories` 表随 AutoMigrate 生成
- [x] Agent 模式触发 `manage_memory`（create/list）成功，记忆写入；再次提问时 system prompt 出现【长期记忆】段
- [x] 前端工具状态条/历史明细自动展示 `manage_memory`，无前端改动导致的问题