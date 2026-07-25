# Tasks

- [x] Task 1: 移除 `ai_ref_max_chars` 设置项及后端绑定方法
  - [ ] 移除 `app.go` 中的 `GetAIRefMaxChars()` 和 `SetAIRefMaxChars()` 方法
  - [ ] 从 `internal/database/db.go` 移除 `ai_ref_max_chars` 默认值
  - [ ] 从 `SettingsConfig` 中移除 `AIRefMaxChars` 字段及相关读取/写入/校验
  - [ ] 更新 Wails 生成的前端绑定文件（`models.ts`, `App.js`, `App.d.ts`）
  - [ ] 验证项目可编译通过

- [x] Task 2: 移除笔记引用截断逻辑
  - [ ] `BuildNoteRefContext` 移除 `maxPerNote` 截断代码（`ai_ref_max_chars` 读取和截断逻辑）
  - [ ] `NoteRefInfo.Truncated` 始终为 `false`（或移除该字段但保持 JSON 兼容）

- [x] Task 3: 移除文件上传截断逻辑
  - [ ] `readAIChatFiles` 移除 `GetAIRefMaxChars()` 调用和内容截断代码
  - [ ] `result.Truncated` 始终为 `false`

- [x] Task 4: 移除卡片召回截断逻辑
  - [ ] `CardRecallSearch` 签名移除 `maxChars int` 参数
  - [ ] 移除截断相关代码，`RecallCard.Truncated` 始终为 `false`
  - [ ] 更新 `app.go` 中两处调用方（移除 `ai_ref_max_chars` 读取、移除 `maxChars` 参数传递）

- [x] Task 5: 新增 `ai_web_search_max_chars` 设置项
  - [ ] `internal/database/db.go` 添加默认值 `{Key: "ai_web_search_max_chars", Value: "5000"}`
  - [ ] `SettingsConfig` 新增 `AIWebSearchMaxChars` 字段，`GetAllSettings()` 中读取，`SaveAllSettings()` 中写入（含范围校验 1-50000）
  - [ ] 更新 Wails 前端绑定文件
  - [ ] 前端 `index.html` 在「对话与搜索」区域新增输入框（id: `aiWebSearchMaxChars`）
  - [ ] 前端 `main.js` 在 `loadSettings()` 中加载值，在 `saveSettings()` 中收集值
  - [ ] 修改 `app.go` 中联网搜索调用方，读取 `ai_web_search_max_chars` 作为 `searchMaxChars`

- [x] Task 6: 新增 `ai_large_file_preview_threshold` 设置项
  - [ ] `internal/database/db.go` 添加默认值 `{Key: "ai_large_file_preview_threshold", Value: "10000"}`
  - [ ] `SettingsConfig` 新增 `AILargeFilePreviewThreshold` 字段，`GetAllSettings()` 中读取，`SaveAllSettings()` 中写入（含范围校验 1-100000）
  - [ ] 更新 Wails 前端绑定文件
  - [ ] 前端 `index.html` 在「对话与搜索」区域新增输入框（id: `aiLargeFilePreviewThreshold`）
  - [ ] 前端 `main.js` 在 `loadSettings()` 中加载值，在 `saveSettings()` 中收集值

- [x] Task 7: 更新前端大文件预览逻辑
  - [ ] 修改 `main.js` 第 3620 行，从 `aiLargeFilePreviewThreshold` 读取阈值
  - [ ] 更新通知文本

- [x] Task 8: 移除前端旧设置项 UI
  - [ ] 从 `index.html` 移除"引用截断"设置项（`#aiRefMaxChars` 整个 `.ai-setting-item` 行）
  - [ ] 从 `main.js` 移除 `#aiRefMaxChars` 的 `loadSettings` / `saveSettings` / change 事件绑定

# Task Dependencies
- Task 1, 2, 3, 4 无依赖关系，可并行
- Task 5, 6 无依赖关系，可并行
- Task 7 依赖于 Task 6
- Task 8 依赖于 Task 1