# Checklist

- [x] 4 个新工具文件（delete_item.go / copy_item.go / move_item.go / transfer_item.go）存在，旧文件已删除
- [x] 每个新工具：struct / 构造器 / Info().Name / ActionText 文案 / 审批摘要 / 错误文案 / 文件头注释均已「项」化，目录操作行为（递归、recursive 门控、覆盖审批、critical 分级）与改名前完全一致
- [x] subagent.go `toolConstructors` 与 subagent_os.go `osSubAgentToolNames` 已同步为 item 名，两处名称集合一致
- [x] subagent_os.go 提示词与文件头注释无旧工具名残留
- [x] 测试（fs_operate_test.go / transfer_item_test.go / subagent_test.go）断言与构造引用已同步
- [x] doc.go / SUBAGENTS.md / AGENTS.md 文档已同步
- [x] `golangci-lint.exe run ./...` 0 issues；`go build ./...`、`go vet` 通过；`go test ./internal/agent/...` 全绿
- [x] 全仓 grep `delete_file|copy_file|move_file|transfer_file` 无代码/装配/测试残留（AGENTS.md 改名说明除外；EVENTS.md / fs_base.go / mkdir_dir.go / ai-chat.js 残留已修复；仅 `.trae/` 历史文档与 spec 记录保留）
