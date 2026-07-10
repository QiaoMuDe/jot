# Checklist

- [x] `internal/database/db.go` 中添加了默认值 `max_file_size = "1"`
- [x] `internal/services/types.go` 中 `SettingsConfig` 添加了 `MaxFileSize` 字段
- [x] `internal/services/types.go` 的 `GetAllSettings()` 包含 `MaxFileSize`
- [x] `internal/services/types.go` 的 `SaveAllSettings()` 包含 `MaxFileSize` 写入和范围校验
- [x] `app.go` 中新增 `GetMaxFileSize()` 方法
- [x] `app.go` 中 `readAIChatFiles()` 的硬编码 10MB 已替换为动态读取，错误提示动态显示 MB 数
- [x] `app.go` 中 `ImportFiles()` 的硬编码 10MB 已替换为动态读取，错误提示动态显示 MB 数
- [x] `internal/services/note_service.go` 中 `BuildNoteRefContext()` 的硬编码 `maxTotalChars` 已替换为动态读取
- [x] `frontend/index.html` 中 AI 设置区新增了「最大文件限制数」输入框（id=maxFileSize，min=1，max=100，单位 MB）
- [x] `frontend/src/main.js` 的 `loadSettings` 中读取 `max_file_size` 并填入 UI（转换为 MB）
- [x] `frontend/src/main.js` 的 `saveSettings` 中收集 UI 值并写入 `max_file_size`（转换为字节）
- [x] `frontend/src/main.js` 中添加了 `maxFileSize` 的 change 事件监听及保存提示
- [x] `frontend/wailsjs/go/models.ts` 中添加了 `max_file_size: number` 字段
- [x] 编译通过
