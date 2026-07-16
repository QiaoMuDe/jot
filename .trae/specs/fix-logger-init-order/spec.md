# 修复日志初始化顺序与 nil 保护 — Spec

## Why

当前 `NewApp()` 将 nil Logger 传给所有 Service，`startup()` 中才调用 `LogSvc.Init()` 创建 Logger。Service 持有的是 nil 副本的指针值，调用 `s.logger.Errorw()` 会 panic。同时 `startup()` 中有多处 `fmt.Printf`/`Println` 应统一使用日志方法。

## What Changes

- **Logger Init 移到 NewApp，放在 DB Init 之前**：先以 INFO 级别初始化，确保 DB 初始化失败也能记日志
- **DB Init 后从库读取 log_level 动态调级**：使用 `LogSvc.SetLevel()` 调整到实际配置级别，不重建 Logger
- **所有 Service 在 Logger 就绪后才创建**：根除 nil Logger 问题
- **startup() 中 fmt → Logger 替换**：`startup()` 不再负责 Init Logger，只做业务初始化

## Impact

- Affected code: `app.go` — `NewApp()` 和 `startup()` 方法
- 不涉及前端、不涉及新依赖、不涉及新文件

## ADDED Requirements

### Requirement: Logger Init 在 DB Init 之前（默认 INFO）

系统 SHALL 在 `NewApp()` 中、DB 初始化之前，先以 INFO 级别初始化 Logger。

#### Scenario: 默认 INFO 初始化
- **WHEN** `NewApp()` 被调用
- **THEN** 计算 `logDir` 路径
- **AND** 调用 `LogSvc.Init(logDir, fastlog.INFO)`
- **AND** Logger 初始化后，DB 初始化失败也可用 Logger 记录日志

### Requirement: DB Init 后根据库配置动态调级

系统 SHALL 在 DB 和 SettingService 就绪后，从库中读取 `log_level` 并动态调整 Logger 级别。

#### Scenario: 动态调级
- **WHEN** `InitDB()` 成功
- **THEN** 创建 `SettingService`，读取 `log_level`
- **AND** 调用 `LogSvc.SetLevel(level)` 调整级别，不重建 Logger

### Requirement: 所有 Service 创建时 Logger 非 nil

系统 SHALL 在 Logger 初始化完成后才创建各 Service。

#### Scenario: 非 nil Logger 传入 Service
- **WHEN** 所有 Service 创建时
- **THEN** 传入的 `logSvc.Logger` 保证非 nil
- **AND** Service 内部调用 `s.logger.Errorw()` 不会 panic

### Requirement: startup 中的输出统一使用日志方法

系统 SHALL 将 `startup()` 中所有 `fmt.Printf`/`Println` 替换为 `a.LogSvc.Logger.Errorw`/`Infow`。

#### Scenario: 统一日志记录
- **WHEN** startup 中发生错误
- **THEN** 使用 `a.LogSvc.Logger.Errorw(msg, fastlog.Error(err))` 记录
- **WHEN** startup 中迁移完成
- **THEN** 使用 `a.LogSvc.Logger.Infow` 记录

### Requirement: Logger Init 失败时退出程序

系统 SHALL 在 `Init` 失败且 Logger 为 nil 时退出程序。

#### Scenario: Init 失败 → Logger 为 nil → 退出
- **WHEN** `LogSvc.Init()` 返回错误
- **THEN** 使用 `println` 输出错误到 stderr
- **WHEN** 随后 `LogSvc.Logger == nil`
- **THEN** 打印"日志实例为空，无法继续启动"
- **AND** 调用 `os.Exit(1)` 退出程序
