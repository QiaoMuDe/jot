# Checklist

- [x] `internal/agent/tools/manage_note.go` 提取包级函数 `notePreviewThreshold(setting)`，`viewNote` 改用且行为不变
- [x] `viewNote` 截断提示包含总字符数 N、已显示前 X 与 `read_note_section` 工具调用指引（id/offset 参数）
- [x] `internal/agent/tools/read_note_section.go` 存在，`readNoteSectionTool` 实现 `tool.InvokableTool` 与 `ActionTextProvider`（有编译期断言）
- [x] `read_note_section` 工具 `Info()` 名称/描述/参数（id 必填、offset 必填 ≥0、length 可选）正确，Desc 说明何时调用
- [x] `InvokableRun()` 校验 id/offset、offset 越界报错、length 缺省取 `ai_large_file_preview_threshold`（上限 100000）、按 rune 切片、返回含起止位置与总字符数的元信息
- [x] `internal/agent/registry.go` `buildTools` 注册 `read_note_section`（`WrapWithError` 包装，置于 manage_note 之后）
- [x] `internal/agent/tools/meta.go` `BuiltinTools` 包含 `read_note_section` 展示文案
- [x] `internal/agent/tools/doc.go` 与 `internal/agent/doc.go` 清单同步包含 `read_note_section` / `NewReadNoteSection`
- [x] `go build ./...` 通过
- [x] `go vet ./internal/agent/...` 通过
