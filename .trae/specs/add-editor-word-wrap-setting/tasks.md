# Tasks

- [x] Task 1: 后端 SettingsConfig 新增 EditorWordWrap 字段
  - 在 `internal/services/types.go` 的 `SettingsConfig` 结构体中追加 `EditorWordWrap bool \`json:"editor_word_wrap"\`` 字段
  - 在 `GetAllSettings()` 结构体初始化中补充 `EditorWordWrap: parseBoolSetting(s.Get("editor_word_wrap"))`
  - 在 `SaveAllSettings()` 的 sets map 中补充 `"editor_word_wrap": strconv.FormatBool(cfg.EditorWordWrap)`

- [x] Task 2: 前端设置面板编辑器卡片新增自动换行 toggle
  - 在 `frontend/index.html` 编辑器设置分区（全屏打开 toggle 之后）添加自动换行 toggle 行
  - HTML 结构与现有 toggle 一致（`font-setting-row` + `toggle-switch`），id 命名为 `editorWordWrapToggle`

- [x] Task 3: 前端 loadSettings 读取并应用自动换行设置
  - 在 `frontend/src/main.js` 的 `loadSettings()` 中添加：`if (els.editorWordWrapToggle) els.editorWordWrapToggle.checked = cfg.editor_word_wrap;`

- [x] Task 4: 前端 saveSettings 收集自动换行状态
  - 在 `frontend/src/main.js` 的 `saveSettings()` 的 cfg 对象中添加：`editor_word_wrap: els.editorWordWrapToggle?.checked || false,`

- [x] Task 5: CM6 初始化根据配置注入 lineWrapping 扩展
  - 在 `frontend/src/main.js` 的 `initCodeMirror()` 函数中：
    - 新增参数 `enableWordWrap = false`
    - 当 `enableWordWrap` 为 true 时，在 extensions 数组中添加 `EditorView.lineWrapping()`
    - import 补充 `lineWrapping`（从 `@codemirror/view` 解构导入）

- [x] Task 6: 调用 initCodeMirror 处透传自动换行配置
  - 在 `frontend/src/main.js` 的 `openEditor()` 中调用 `initCodeMirror()` 时，从 `els.editorWordWrapToggle.checked` 读取值传入第 7 个参数

- [x] Task 7: 验证与测试
  - Go 编译通过（`go build ./internal/services/`）
  - 三个文件的诊断均无错误
  - 启动应用 → 进入设置页 → 确认"自动换行" toggle 存在
  - 默认关闭 → 新建笔记 → 输入长文本 → 确认不换行（横向滚动）
  - 开启自动换行 → 保存 → 新建笔记 → 输入长文本 → 确认自动换行
  - 编辑/查看模式验证同上
  - 重启应用 → 确认设置持久化

# Task Dependencies
- Task 5 依赖 Task 1 和 Task 3（读取配置后传入 CM6）
- Task 6 依赖 Task 5（新增参数后才可透传）
- 其余任务可并行
