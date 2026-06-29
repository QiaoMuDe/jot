# Tasks

- [x] Task 1: 后端 — NoteService 注入 settingService，BuildNoteRefContext 从数据库读取截断配置
  - [x] SubTask 1.1: `NoteService` 结构体新增 `settingService *SettingService` 字段，`NewNoteService` 接收 `*SettingService` 参数
  - [x] SubTask 1.2: 修改 `BuildNoteRefContext`，用 `s.settingService.Get("ai_ref_max_chars")` 替代 `const maxPerNote = 4000`，无效/空值时默认 1000
  - [x] SubTask 1.3: 更新 `app.go` 中 `NewNoteService` 的调用处传入 `settingService`
- [x] Task 2: 后端 — 新增 GetAIRefMaxChars / SetAIRefMaxChars 绑定
  - [x] SubTask 2.1: `app.go` 新增 `GetAIRefMaxChars() int` 绑定，读取 `ai_ref_max_chars`，空值时返回 1000
  - [x] SubTask 2.2: `app.go` 新增 `SetAIRefMaxChars(chars int) error` 绑定，写入 `ai_ref_max_chars`，含范围校验（1-50000）
- [x] Task 3: 前端 — 设置页 UI 增加引用截断字数输入
  - [x] SubTask 3.1: `frontend/index.html` AI 助手设置区新增一行"引用截断字数"数字输入框，id 为 `aiRefMaxChars`
  - [x] SubTask 3.2: `frontend/src/main.js` 中 `loadAISettings()` 增加加载该值
  - [x] SubTask 3.3: `frontend/src/main.js` 增加输入框 `change` 事件自动保存

# Task Dependencies

- [Task 1] 无依赖
- [Task 2] 无依赖（可和 Task 1 并行）
- [Task 3] 依赖 [Task 2]（前端需要后端绑定可用）
