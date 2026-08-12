# Tasks

- [x] Task 1: 修改 `manage_note.go`：提取 helper + 联动 view 截断提示
  - [x] 在 `viewNote` 旁提取包级函数 `notePreviewThreshold(setting *services.SettingService) int`（读取 `ai_large_file_preview_threshold`，解析失败或 ≤0 回退 10000），`viewNote` 改用该 helper
  - [x] `viewNote` 截断提示改为："内容共 N 字符，已显示前 X。如需继续阅读，可调用 read_note_section 工具（id={id}, offset={X}）"，N 为全文 rune 总数、X 为阈值
- [x] Task 2: 新增 `read_note_section.go` 工具实现
  - [x] 实现 `readNoteSectionTool` 结构体（依赖 `note`、`setting`、`ctx`）与编译期断言 `var _ tool.InvokableTool = (*readNoteSectionTool)(nil)`
  - [x] 实现 `ActionText(argumentsInJSON)`：解析 `id`/`offset` 返回"读取笔记 #{id} 第 {offset} 字符起"，失败回退"读取笔记分段"
  - [x] 实现 `Info()`：名称 `read_note_section`，`Desc` 说明何时调用（manage_note view 提示截断后续读），参数 `id` 必填、`offset` 必填（≥0）、`length` 可选（缺省 `ai_large_file_preview_threshold`，上限 100000）
  - [x] 实现 `InvokableRun()`：校验 id/offset → `note.GetNoteContent(id)` 读全文 → rune 切片（length 缺省取 `notePreviewThreshold`）→ 返回含元信息（起止位置 + 总字符数）的格式化文本；错误路径含 offset 越界、笔记不存在
  - [x] 实现 `NewReadNoteSection(note, setting, ctx)` 构造器
- [x] Task 3: 注册工具并同步清单
  - [x] `internal/agent/registry.go` `buildTools` 追加 `{"read_note_section", tools.WrapWithError("read_note_section", tools.NewReadNoteSection(p.deps.Note, p.deps.Setting, p.ctx), p.ctx)}`（置于 manage_note 之后）
  - [x] `internal/agent/tools/meta.go` `BuiltinTools` 追加 `{Name: "read_note_section", Label: "分段读取笔记内容"}`
  - [x] `internal/agent/tools/doc.go` 工具列表与构造器名追加 `read_note_section` / `NewReadNoteSection`
  - [x] `internal/agent/doc.go` 只读工具列表追加 `read_note_section`
- [x] Task 4: 构建验证
  - [x] `go build ./...` 通过
  - [x] `go vet ./internal/agent/...` 通过

# Task Dependencies

- [Task 2] 依赖 [Task 1]（复用 `notePreviewThreshold` helper）
- [Task 3] 依赖 [Task 2]（需先有构造器）
- [Task 4] 依赖 [Task 1][Task 2][Task 3]
