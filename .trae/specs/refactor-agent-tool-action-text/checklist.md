# Checklist

- [x] tools/context.go 定义了 `ActionTextProvider` 可选接口（`ActionText(argumentsInJSON string) string`）
- [x] tools/context.go 的 `Record` 含 `ActionText` 字段（json tag `action_text,omitempty`）
- [x] `WrapWithError` 改造为自定义 wrapper 后，失败回填/`tool_error` 事件/用户取消行为与改造前一致（代码逻辑逐字保留）
- [x] wrapper 实现 `ActionTextProvider` 并转发给内层工具（未实现返回空串），父包可对包装后工具统一断言
- [x] agent.go `Run()` 构建 name→tool 映射，两处 `emitToolStart` 调用点传入了映射
- [x] `emitToolStart` 按工具名断言 `ActionTextProvider` 并填充 `Record.ActionText`
- [x] 8 个工具均实现 `ActionText`，文案与 spec.md 映射表逐项一致（含 manage_note 7 个动作）
- [x] 前端 `showToolStatusStart` 改为 `payload.action_text || '执行'`，工具名 switch 已删除
- [x] TOOLS.md 规范更新为"动作文案在工具实现内维护"，§2/§3/§5/§8 同步且无残留"前端 switch 维护"说明
- [x] `go build ./...` 通过
- [x] `go vet ./internal/agent/...` 通过
- [x] `npm run build` 通过
