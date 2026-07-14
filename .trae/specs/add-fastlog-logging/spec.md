# 引入 fastlog 日志库 — Spec

## Why

当前后端没有统一的日志系统，错误处理和关键操作缺乏日志记录，排查问题困难。引入 fastlog 日志库，提供分级日志能力，并让用户可在前端配置日志级别。

## What Changes

- **后端新增依赖**：添加 `gitee.com/MM-Q/fastlog` 依赖
- **后端新增 LogService**：管理 fastlog Logger 实例，支持运行时动态调整日志级别
- **后端 App 结构体扩展**：新增 `Logger` 字段和 `OpenLogDir()` 方法
- **后端日志注入**：在 `app.go` 及关键服务方法中按适当级别添加日志
- **SettingsConfig 扩展**：新增 `LogLevel` 字段（int，0=DEBUG ~ 5=PANIC，默认 1=INFO）
- **前端设置页**：新增「日志设置」卡片，包含日志级别分段滑块
- **前端数据管理页**：新增「打开日志目录」按钮（与「打开数据目录」同级）
- **日志目录**：`~/.jot/logs/`，使用 fastlog 的 `Dev()` 配置（DEBUG 级别写入，禁用缓冲，带调用者信息）

## Impact

- Affected specs: 设置页、数据管理页、后端服务初始化
- Affected code:
  - `go.mod` — 新增 fastlog 依赖
  - `internal/services/types.go` — `SettingsConfig` 新增 `LogLevel`；`GetAllSettings`/`SaveAllSettings` 读写
  - `internal/services/log_service.go` — **新建**，Logger 管理
  - `internal/app.go` — 新增 `Logger` 字段、`OpenLogDir()`、startup 初始化、各方法添加日志
  - `internal/database/db.go` — `InitDefaultSettings` 新增默认值
  - `internal/services/ai_service.go` — 添加日志
  - `internal/database/db.go` — 添加日志
  - `frontend/index.html` — 设置页新增「日志设置」卡片；数据管理页新增按钮
  - `frontend/src/main.js` — `loadSettings`/`saveSettings` 读写日志级别；按钮事件绑定

## ADDED Requirements

### Requirement: 后端 Logger 初始化与生命周期管理

系统 SHALL 在应用启动时初始化 fastlog Logger，并在应用退出时正确关闭。

#### Scenario: 启动时初始化
- **WHEN** `App.startup()` 被调用
- **THEN** 系统读取 `log_level` 设置项，创建 fastlog Logger，写入 `~/.jot/logs/app.log`
- **AND** Logger 使用 `Dev()` 配置（DEBUG 级别，禁用缓冲，带 caller 信息，彩色输出）

#### Scenario: 日志级别动态变更
- **WHEN** 用户在前端修改日志级别并保存设置
- **THEN** 后端 `SaveAllSettings` 保存新级别后，调用 `Logger.SetLevel()` 立即生效
- **AND** 无需重启应用

#### Scenario: 应用退出
- **WHEN** 应用退出时
- **THEN** 正确调用 `Logger.Close()` 确保日志缓冲区落盘

### Requirement: 后端关键路径日志记录

系统 SHALL 在后端的关键操作路径添加适当级别的日志。

#### Scenario: 信息日志 (INFO)
- **WHEN** 应用启动完成、数据库连接成功、会话创建、笔记保存等正常操作
- **THEN** 使用 `logger.Infow()` 记录结构化日志

#### Scenario: 警告日志 (WARN)
- **WHEN** 连接池接近上限、配置异常降级、非关键操作失败
- **THEN** 使用 `logger.Warnw()` 记录结构化日志

#### Scenario: 错误日志 (ERROR)
- **WHEN** 关键操作失败（数据库查询失败、API 调用失败、文件读写错误）
- **THEN** 使用 `logger.Errorw()` 记录结构化日志，包含 `fastlog.Error(err)` 字段

#### Scenario: 调试日志 (DEBUG)
- **WHEN** 流式 AI 响应的状态变更、数据量较大或频繁触发的内部步骤
- **THEN** 使用 `logger.Debugw()` 记录结构化日志

### Requirement: 前端设置页 — 日志配置

系统 SHALL 在设置页提供日志级别配置 UI。

#### Scenario: 日志级别分段滑块
- **WHEN** 用户打开设置页
- **THEN** 看到「日志设置」卡片，包含 4 档分段滑块：DEBUG / INFO / WARN / ERROR
- **AND** 当前级别高亮显示，默认选中 INFO

#### Scenario: 保存日志级别
- **WHEN** 用户点击其他级别分段按钮
- **THEN** 立即调用 `saveSettings()` 保存新级别到后端
- **AND** 按钮高亮切换到新选中级别
- **AND** 通知提示「日志级别已保存」

### Requirement: 前端数据管理页 — 打开日志目录

系统 SHALL 在数据管理页提供打开日志目录的按钮。

#### Scenario: 打开日志目录
- **WHEN** 用户点击「打开日志目录」按钮
- **THEN** 调用后端 `App.OpenLogDir()` 在文件管理器中打开 `~/.jot/logs/`
- **AND** 按钮样式与「打开数据目录」一致，放置在同一操作分组中