# Tasks

- [x] Task 1: 后端 - 新增 `max_file_size` DB 默认值和 SettingsConfig 字段
  - 在 `internal/database/db.go` 的 `InitDefaultSettings` 中添加 `{Key: "max_file_size", Value: "1"}`
  - 在 `internal/services/types.go` 的 `SettingsConfig` 中添加 `MaxFileSize int \`json:"max_file_size"\``
  - 在 `GetAllSettings()` 中添加 `MaxFileSize: parseIntSetting(s.Get("max_file_size"), 1)`
  - 在 `SaveAllSettings()` 中添加范围校验（1MB~100MB）和写入逻辑

- [x] Task 2: 后端 - 替换文件上传/导入的硬编码 10MB 为动态读取
  - 在 `app.go` 中新增 `GetMaxFileSize()` 方法（读取 `max_file_size`，返回 int64 字节数）
  - 替换 `readAIChatFiles()` 中的 `const maxSize int64 = 10 * 1024 * 1024` 为动态读取
  - 替换 `ImportFiles()` 中的 `const maxSize int64 = 10 * 1024 * 1024` 为动态读取
  - 更新两处错误提示文本，硬编码 "10MB" 改为动态计算显示

- [x] Task 3: 后端 - 替换笔记引用总上下文硬编码 10MB 为动态读取
  - 在 `internal/services/note_service.go` 的 `BuildNoteRefContext()` 中
  - 替换 `const maxTotalChars = 10 * 1024 * 1024` 为通过 `s.settingService.Get("max_file_size")` 动态读取

- [x] Task 4: 前端 - 新增「最大文件限制数」UI 及交互逻辑
  - 在 `frontend/index.html` 的 AI 设置区「引用截断」下方新增一行
  - 在 `frontend/src/main.js` 中：loadSettings / saveSettings / change 事件
  - 在 `frontend/wailsjs/go/models.ts` 中添加 `max_file_size: number` 字段

## Task Dependencies

- Task 2 依赖 Task 1（需要 `GetMaxFileSize()` 方法就绪）
- Task 3 依赖 Task 1（需要 DB 设置键就绪）
- Task 4 可独立并行执行（仅依赖接口契约）
