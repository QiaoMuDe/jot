# Tasks

- [x] Task 1: 添加 fastlog 依赖并创建 LogService
  - [x] 在 `go.mod` 中添加 `gitee.com/MM-Q/fastlog` 依赖
  - [x] 新建 `internal/services/log_service.go`：`LogService` 结构体（包装 `*fastlog.Logger`），提供 `Init(logDir, level)`、`SetLevel(level)`、`Close()` 方法
  - [x] 日志目录使用 `~/.jot/logs/`，基于 `Dev()` 配置自定义（DEBUG 级别、禁用缓冲、带 caller、彩色输出）
  - [x] 导出 `LevelFromInt(n int) fastlog.Level` 工具函数（0=DEBUG ~ 5=PANIC）

- [x] Task 2: 扩展 SettingsConfig 并初始化 Logger
  - [x] `internal/models/` 或 `internal/services/types.go`：`SettingsConfig` 新增 `LogLevel int` 字段（默认 1=INFO）
  - [x] `internal/services/types.go`：`GetAllSettings()` 读取 `log_level`，`SaveAllSettings()` 写入 `log_level`（范围校验 0-5）
  - [x] `internal/database/db.go`：`InitDefaultSettings` 新增 `log_level = "1"`（INFO）
  - [x] `app.go`：`App` 结构体新增 `LogSvc *services.LogService` 字段
  - [x] `app.go`：`startup()` 中初始化 LogService 并赋值到 `a.LogSvc`
  - [x] `app.go`：`SaveAllSettings` 保存成功后调用 `a.LogSvc.SetLevel()` 动态调整级别
  - [x] `app.go`：`rebuildServices()` 中重建 LogService
  - [x] `app.go`：`startup()` 中 defer 关闭 Logger

- [x] Task 3: 添加 App.OpenLogDir 方法
  - [x] `app.go`：参照 `OpenDataDir()` 实现 `OpenLogDir()`，打开 `~/.jot/logs/`
  - [x] 前端 wailsjs 自动生成 bindings

- [x] Task 4: 在后端关键路径添加日志
  - [x] `app.go`：`startup()` — INFO（启动完成、数据库连接）、WARN/ERROR（初始化失败）
  - [x] `app.go`：`CallAIStream()` — DEBUG（流状态变更）、ERROR（流错误）
  - [x] `app.go`：`SaveImage/SaveImageFromPath` — INFO（图片保存）、ERROR（失败）
  - [x] `app.go`：`ExportData/ImportData/BackupToDir/RestoreFromDir` — INFO（操作开始/完成）、ERROR（失败）
  - [x] `app.go`：`ResetDatabase/VacuumDatabase` — INFO（清理完成）、WARN（异常）
  - [x] `internal/services/note_service.go` — 通过 app.go 调用点日志覆盖
  - [x] `internal/services/ai_service.go` — 通过 app.go 调用点日志覆盖

- [x] Task 5: 前端 — 设置页新增日志配置卡片
  - [x] `frontend/index.html`：在「回收站清理」卡片之前新增「日志设置」卡片
  - [x] 包含 `.ai-group-header`（文件/日志 SVG 图标 + "日志设置"标题）
  - [x] 包含日志级别 4 档分段滑块（DEBUG / INFO / WARN / ERROR），`id="logLevelControl"`
  - [x] 滑块样式复用现有 `.settings-segmented` + `.seg-btn` 体系

- [x] Task 6: 前端 — 数据管理页新增「打开日志目录」按钮
  - [x] `frontend/index.html`：在「打开数据目录」按钮之后新增「打开日志目录」按钮（`id="openLogDirBtn"`）
  - [x] 复用 `data-action-row` 样式，文件夹 SVG 图标，label="打开日志目录"，desc="查看日志文件"

- [x] Task 7: 前端 — loadSettings / saveSettings / 事件绑定
  - [x] `main.js`：`els` 对象新增 `logLevelControl` 和 `openLogDirBtn` DOM 引用
  - [x] `main.js`：`loadSettings()` 中根据 `cfg.log_level` 设置对应分段按钮的 `.active` 类
  - [x] `main.js`：`saveSettings()` 中从分段按钮读取 `log_level` 值
  - [x] `main.js`：分段按钮 `click` 事件绑定，切换时调用 `saveSettings()` + 通知
  - [x] `main.js`：`openLogDirBtn` 的 `click` 事件绑定，调用 `window.go.main.App.OpenLogDir()`
  - [x] `data-management.js`：新增 `openLogDir()` 函数并暴露

# Task Dependencies

- Task 1 是 Task 2 的基础
- Task 2 是 Task 3~4 的基础
- Task 5~7 无相互依赖，可与 Task 3~4 并行
- Task 6 依赖 Task 3（后端方法就绪）
- Task 7 依赖 Task 5~6（DOM 元素就绪）