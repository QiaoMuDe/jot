# Checklist

- [x] `GetAIRefMaxChars()` 和 `SetAIRefMaxChars()` 已从 `app.go` 中移除
- [x] `ai_ref_max_chars` 已从 `db.go` 默认值列表中移除
- [x] `SettingsConfig.AIRefMaxChars` 字段已移除，`GetAllSettings()` / `SaveAllSettings()` 不再读写
- [x] `models.ts` 中 `ai_ref_max_chars` 字段已移除
- [x] `App.js` / `App.d.ts` 中 `GetAIRefMaxChars` / `SetAIRefMaxChars` 已移除
- [x] 项目可编译通过（`go build ./internal/...` 成功，`go build ./...` 因 `frontend/dist` 不存在而失败，是 Wails 前端资源嵌入的预置条件，与本次重构无关）

- [x] `BuildNoteRefContext` 不再读取 `ai_ref_max_chars`，笔记内容全量注入
- [x] `readAIChatFiles` 不再调用 `GetAIRefMaxChars()`，文件内容全量注入
- [x] `CardRecallSearch` 签名移除 `maxChars` 参数，内容全量注入
- [x] `app.go` 中两处卡片召回调用方不再读取 `ai_ref_max_chars`，不再传递 `maxChars`

- [x] `ai_web_search_max_chars` 在 `db.go` 中有默认值 5000
- [x] `SettingsConfig` 包含 `AIWebSearchMaxChars` 字段，读写/校验正确
- [x] 前端 `index.html` 有 `#aiWebSearchMaxChars` 输入框
- [x] 前端 `loadSettings()` 加载 `ai_web_search_max_chars` 值，`saveSettings()` 收集
- [x] `app.go` 联网搜索调用方使用 `ai_web_search_max_chars` 作为截断阈值

- [x] `ai_large_file_preview_threshold` 在 `db.go` 中有默认值 10000
- [x] `SettingsConfig` 包含 `AILargeFilePreviewThreshold` 字段，读写/校验正确
- [x] 前端 `index.html` 有 `#aiLargeFilePreviewThreshold` 输入框
- [x] 前端 `loadSettings()` 加载 `ai_large_file_preview_threshold` 值，`saveSettings()` 收集

- [x] 前端大文件预览逻辑从 `#aiLargeFilePreviewThreshold` 读取阈值，通知文本正确

- [x] 前端旧"引用截断"设置项（`#aiRefMaxChars`）UI 已移除
- [x] 前端 `main.js` 中 `#aiRefMaxChars` 的 change 事件绑定已移除
- [x] 前端 `saveSettings()` 不再收集 `ai_ref_max_chars` 字段