# 移除 summarize\_text 工具

## 背景

`summarize_text` 工具让 Agent 调用另一个 LLM 来总结一段它已经在上下文中的文字，本质上是"LLM 调用 LLM 做摘要"，多余且增加一次 API 调用开销。Agent 自身已能在回复中直接总结。移除可精简工具列表、减少模型决策负担、省去不必要的 API 调用。

## 涉及文件

| 文件                                            | 操作     | 说明                                |
| --------------------------------------------- | ------ | --------------------------------- |
| `internal/agent/tools/summarize_text.go`      | **删除** | 整个工具实现                            |
| `internal/agent/tools/summarize_text_test.go` | **删除** | 整个测试文件                            |
| `internal/agent/registry.go`                  | 修改     | 移除注册行（L27）                        |
| `internal/agent/tools/meta.go`                | 修改     | 移除元信息条目（L15）                      |
| `internal/agent/tools/doc.go`                 | 修改     | 移除包注释中的列举（L4、L7）                  |
| `playground/landing/index.html`               | 修改     | 移除落地页中的 `summarize_text` 标签（L342） |
| `AGENTS.md`                                   | 修改     | 更新记忆点（L518 相关）                    |

## 不修改的文件

* `.trae/documents/` 下的历史文档（ai-session-summary.md、chat-mode-removal-report.md、mcp-tool-fine-grained-control.md）— 保留历史记录，不清理

## 验证步骤

1. `go build ./...` 编译通过
2. `go test ./internal/agent/tools/...` 测试通过
3. `go vet ./...` 无静态分析问题

