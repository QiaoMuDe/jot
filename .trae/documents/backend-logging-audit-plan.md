# 后端日志全面覆盖计划

## 摘要

为后端代码全面添加 fastlog 日志记录。当前只有 `app.go` 中约 8 个方法有日志，其余 110+ 个 App 方法、所有 Service 层（6 个 service 文件）和 Database 层均无日志。

## 当前状态分析

### 已有日志的方法（app.go）

`startup`、`SaveImage`、`VacuumDatabase`、`ExportDataWithDialog`、`ImportDatabaseWithDialog`、`BackupToDir`、`RestoreFromDir`、`ResetDatabase`、`SaveAllSettings`（级别变更）、`CallAIStream`（流开始+错误）

### 日志完全缺失的区域

| 区域 | 文件数 | 方法数 | 当前状态 |
|------|--------|--------|----------|
| App 层（API 边界） | `app.go` | 110+ | 仅 ~8 个有日志 |
| Service 层（业务逻辑） | 6 文件（note/tag/notebook/ai/todo/profile） | 80+ | 全部无日志 |
| Database 层（初始化） | `db.go` | 4 个导出函数 | 全部无日志 |
| App 私有方法 | `app.go` 内部方法 | 10+ | 全部无日志 |

### 日志级别使用规范

| 级别 | 适用场景 | 示例 |
|------|----------|------|
| `ERROR` | 任何错误路径 | 数据库查询失败、文件 IO 异常、参数校验不通过 |
| `WARN` | 边界情况、非错误但值得注意 | 标签已存在、操作被跳过、默认值回退 |
| `INFO` | 用户可见的操作结果 | 创建/删除/导入/导出成功、设置变更 |
| `DEBUG` | 内部追踪、读操作、入参 | 方法调用入口、配置读取、分页查询参数 |

## 变更内容

### 任务 1：app.go — 所有导出方法添加日志

**所有 CRUD/操作方法的统一模式：**

```
func (a *App) Xxx(...) (..., error) {
    a.LogSvc.Logger.Debugw("方法名", fastlog.Uint("id", id), ...)
    ...
    if err != nil {
        a.LogSvc.Logger.Errorw("方法名 失败", fastlog.Error(err), ...)
        return ..., err
    }
    // 仅对写操作（创建/删除/修改/导入/导出）
    a.LogSvc.Logger.Infow("方法名 成功", fastlog.String("result", ...))
    return ..., nil
}
```

#### 1.1 笔记 CRUD + 操作（CreateNote / UpdateNote / DeleteNote / PermanentDeleteNote / TogglePinNote / RestoreNote）

- DEBUG: 入口（记 id/title/key 参数）
- ERROR: 每个 err 分支
- INFO: 成功（仅写操作，读操作不记 INFO）

#### 1.2 笔记批量操作（BatchPinNotes / BatchDeleteNotes / BatchRestoreNotes / BatchAddTagToNotes / BatchRemoveTagFromNotes / RestoreAllNotes / EmptyTrash / MoveNoteToNotebook / BatchMoveNotesToNotebook）

- DEBUG: 入口（记参数数量）
- ERROR: 每个 err 分支
- INFO: 成功

#### 1.3 笔记查询（GetNote / GetNoteContent / GetNotes / SearchNotes / GetTrashNotes / GetNotesByTag / GetAllNoteIDs / SearchNoteIDs / GetNoteRefContext / GetNoteIDsByNotebook）

- DEBUG: 入口（分页参数、搜索关键词）
- ERROR: 每个 err 分支
- INFO: 不记（读操作无状态变更）

#### 1.4 标签 CRUD（CreateTag / UpdateTag / DeleteTag / GetAllTags / AddTagToNote / RemoveTagFromNote）

- DEBUG: 入口
- ERROR: 每个 err 分支
- INFO: 成功（Create/Update/Delete）

#### 1.5 笔记本 CRUD（CreateNotebook / RenameNotebook / DeleteNotebook / DeleteNotebookWithNotes / GetAllNotebooks / GetNotebookNoteCounts / GetTrashNotebooks / RemoveNotebookFromTrash / RestoreTrashNotebook / PermanentDeleteTrashNotebook / RestoreAllTrashNotebooks / EmptyTrashNotebooks）

- DEBUG: 入口
- ERROR: 每个 err 分支
- INFO: 成功（写操作）

#### 1.6 AI 配置+会话（GetAIConfig / SaveAIConfig / GetProfiles / CreateProfile / UpdateProfile / DeleteProfile / SwitchProfile / TestAIBaseURL / TestTavilyConnection / TestZhihuConnection / FetchAIModels / CancelAIStream / GetAISessions / TogglePinAISession / CreateAISession / DeleteAISession / RenameAISession / LoadAISessionMessages / SaveAIMessages / ClearAISessionMessages / UpdateAIMessageContent / DeleteAIMessage / DeleteAIMessagesAfter / ClearAllAISessions / SaveSessionConfig / LoadSessionConfig / GetAISearchResultLimit / SetAISearchResultLimit / GetAICardRecallLimit / SetAICardRecallLimit / SaveAIMessageAsNote / UpdateSessionContextTokens / UpdateLastUserMessageTokens / GetAIRefMaxChars / SetAIRefMaxChars / GetMaxFileSize / SelectAIChatFiles / ReadAIChatFiles）

**注意**：涉及 API Key、Secret 等敏感信息的参数使用 `fastlog.String("key", "***")` 脱敏。

- DEBUG: 入口（脱敏参数）
- ERROR: 每个 err 分支
- INFO: 写操作成功

#### 1.7 排序/分页/版本（GetSortOrder / SetSortOrder / GetPageSize / SetPageSize / GetVersion / OpenDataDir / OpenLogDir / OpenProjectURL）

- DEBUG: 入口
- ERROR: 有 err 的记 ERROR

#### 1.8 数据管理（GetDataStats / GetBackupInfo / ImportFiles / CleanupOrphanImages）

- DEBUG: 入口
- ERROR: 每个 err 分支
- INFO: 写操作成功（CleanupOrphanImages 记删除数量）

#### 1.9 导出/备份（ExportNoteAsMarkdown / ExportAISessionAsMarkdown）

- DEBUG: 入口
- ERROR: 每个 err 分支
- INFO: 成功

#### 1.10 待办事项（CreateTodo / ListTodos / ToggleTodo / DeleteTodo / UpdateTodo / ClearCompletedTodos）

- DEBUG: 入口
- ERROR: 每个 err 分支
- INFO: 写操作成功

#### 1.11 CallAIStream 强化（已有部分日志）

- INFO: 添加搜索触发、召回触发、技能注入等关键步骤的日志
- INFO: 流完成时记录总耗时、输入/输出 token 数
- DEBUG: search/recall/skill 中间结果量级

### 任务 2：app.go — 私有方法添加日志

#### 2.1 migrateSensitiveKeys（当前用 fmt.Printf）

- INFO: 迁移开始、迁移完成
- WARN: 迁移过程中遇到无法解码的旧密钥（继续保留原值，不中断）
- 将 `fmt.Printf` 替换为结构化日志

#### 2.2 exportSnapshot / importFromArchive / replaceDatabase

- DEBUG: 入口（记路径）
- ERROR: 每个 err 分支
- INFO: 快照创建完成、导入完成

#### 2.3 reconnectDB / rebuildServices

- DEBUG: 入口
- ERROR: 重连失败
- INFO: 重连成功

### 任务 3：internal/services — 各 Service 文件添加日志

**原则**：Service 层只记 ERROR 级别日志（错误路径），不重复记 App 层已记的 entry/success。

#### 3.1 note_service.go

所有方法的 `!= nil` 分支添加：
```go
// 传入 logger 实例或使用全局 logger
// log.Errorw("NoteService.Method 失败", fastlog.Error(err), ...)
```

但 Service 层没有 `LogSvc` 引用。有两种方案可选：

**方案 A（推荐）**：每个 Service 结构体新增 `log *fastlog.Logger` 字段，构造时传入。

**方案 B**：只在 app.go 层记所有日志，Service 层不记。

选定方案 A，但 Logger 字段是从外部注入的，每个 Service 结构体新增 `logger` 字段：

```go
type NoteService struct {
    db     *gorm.DB
    logger *fastlog.Logger
}
```

并修改构造方法接受 logger 参数（或后期通过 SetLogger 设置）。

#### 3.2 tag_service.go / notebook_service.go / ai_service.go / todo_service.go / profile_service.go

同上，为每个 Service 注入 logger，在错误路径添加 ERROR 日志。

#### 3.3 独立函数（search_service.go/zhihu_search_service.go/recall_service.go/query_refiner.go）

这些是自由函数，没有结构体，不适合传 logger。只在 app.go 调用方处记录。

### 任务 4：internal/database — db.go 添加日志

- INFO: `InitDB` 开始/完成（记数据库路径）
- DEBUG: `InitBuiltinPrompts` 插入记录数
- DEBUG: `InitDefaultSettings` 插入记录数
- ERROR: 迁移失败、默认数据插入失败

**问题**：`db.go` 在 `NewApp` 中被调用，此时 `LogSvc` 尚未初始化。需调整初始化顺序。

**方案**：`InitDB` 完成后立即初始化 `LogSvc`（在 `rebuildServices` 之前），这样后续的 InitBuiltinPrompts / InitDefaultSettings 就可以用 Logger 了。

## 影响范围

- `app.go` — 约 110+ 处方法体添加日志调用（增量修改，不改变现有逻辑）
- `internal/services/*.go` — 6 个文件，每个结构体新增 `logger` 字段
- `internal/services/note_service.go` — 修改构造方法或添加 SetLogger 方法
- `internal/services/tag_service.go` — 同上
- `internal/services/notebook_service.go` — 同上
- `internal/services/ai_service.go` — 同上
- `internal/services/todo_service.go` — 同上
- `internal/services/profile_service.go` — 同上
- `internal/database/db.go` — 约 4-6 处日志调用
- `app.go` — `NewApp()` / `rebuildServices` 调整初始化顺序

## 假设与决策

| 决策 | 原因 |
|------|------|
| Service 层只记 ERROR，不记 INFO/DEBUG | 避免与 App 层重复，减少噪音 |
| App 层读操作只记 DEBUG，不记 INFO | 读操作频繁且无状态变更 |
| App 层写操作记 INFO | 用户可见的操作需要可追溯 |
| App 层所有方法入口记 DEBUG | 方便追踪前端到后端的调用链路 |
| API Key/Secret 日志脱敏 | 安全考虑，不记录明文凭据 |
| Service 层注入 `logger` 字段 | 避免全局变量，保持测试友好 |
| `InitDB` 日志在 `LogSvc` 初始化之后 | 确保日志管道就绪后再产生日志 |

## 验证步骤

1. `go build ./...` — 编译验证
2. 运行应用确认启动阶段日志正常输出到 `~/.jot/logs/app.log`
3. 随机测试 3-5 个操作方法（如创建笔记、创建标签、搜索），确认日志产生
4. 引入一个错误（如非法 ID），确认 ERROR 日志输出
5. 确认 API Key、密码类参数不会明文出现在日志中
