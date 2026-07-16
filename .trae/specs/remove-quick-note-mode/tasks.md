# Tasks

- [x] Task 1: 移除设置页 HTML 中的快速笔记开关
  - 文件: `frontend/index.html` L383-393
  - 删除"快速笔记"整行设置（含 checkbox 和提示文字 div）
  - 注意: 上方语法高亮的提示（`.quick-note-hint`）保留不动

- [x] Task 2: 移除前端 JS 中的快速笔记相关逻辑
  - 文件: `frontend/src/main.js`
  - 删除 L384: `quickNoteToggle: $('quickNoteToggle'),`（DOM 引用）
  - 删除 L5102-5106: change 事件监听（快速笔记开关变更保存）
  - 删除 L7059-7062: `init()` 中的启动触发（自动打开全屏编辑器）
  - 删除 L7635-7636: `loadSettings()` 中的 checkbox 状态同步
  - 删除 L7827: `saveSettings()` 中的 `quick_note_enabled` 字段收集

- [x] Task 3: 移除后端 Go SettingsConfig 中的 QuickNoteEnabled 字段
  - 文件: `internal/services/types.go`
  - 删除 L63: 结构体字段 `QuickNoteEnabled bool`
  - 删除 L94: `GetAllSettings()` 中的读取 `parseBoolSetting(s.Get("quick_note_enabled"))`
  - 删除 L175: `SaveAllSettings()` 中的写入 `"quick_note_enabled": strconv.FormatBool(cfg.QuickNoteEnabled)`

- [x] Task 4: 移除数据库默认值中的 quick_note_enabled
  - 文件: `internal/database/db.go`
  - 删除 L541: `{Key: "quick_note_enabled", Value: "false"},`

- [x] Task 5: 移除 TypeScript 模型中的 quick_note_enabled 字段
  - 文件: `frontend/wailsjs/go/models.ts`
  - 删除 L563: 字段声明 `quick_note_enabled: boolean;`
  - 删除 L596: 构造函数赋值 `this.quick_note_enabled = source["quick_note_enabled"];`

# Task Dependencies

- 无依赖关系，所有任务可并行执行
