# Tasks

- [x] Task 1: 将 Logger Init 移到 NewApp，放在 DB Init 之前
  - [x] 在 `NewApp()` 中，在 `InitDB` 之前调用 `LogSvc.Init()`，使用 INFO 级别
  - [x] Init 失败且 Logger 为 nil 时 `os.Exit(1)`
  - [x] DB Init 成功 + SettingService 就绪后，读 `log_level` 并 `SetLevel`
  - [x] 在 Logger 就绪后才创建各 Service，传入非 nil Logger

- [x] Task 2: 清理 startup 中的日志初始化逻辑
  - [x] 删除 `startup()` 中的 `LogSvc.Init()`、`logLevelStr` 读取、`LevelFromInt` 调用、nil 检查 + `os.Exit(1)`
  - [x] 保留 `a.ctx = ctx` 和图片目录创建等业务逻辑

- [x] Task 3: 将 startup 中所有 fmt 替换为日志方法
  - [x] 创建图片目录失败：`fmt.Printf` → `a.LogSvc.Logger.Errorw`
  - [x] 初始化默认笔记本失败：`fmt.Printf` → `a.LogSvc.Logger.Errorw`
  - [x] profile 迁移失败：`fmt.Printf` → `a.LogSvc.Logger.Errorw`
  - [x] 迁移完成通知：`fmt.Println` → `a.LogSvc.Logger.Infow`
  - [x] 移除初始化后的三条回顾性 INFO 日志

- [x] Task 4: 移除 migrateSensitiveKeys 中的 Logger nil 检查
  - [x] 两处 `if a.LogSvc.Logger != nil` 改为直接调用

# Task Dependencies

- Task 1 是 Task 2 ~ Task 4 的前置依赖
- Task 2 和 Task 3 可并行
