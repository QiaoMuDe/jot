# Checklist

- [x] manage_note 移除 confirm 参数（schema/解析/描述），写操作改经 Context.Approver 审批门控
- [x] manage_note critical 分级正确：edit=true；批量 move/add_tag/remove_tag=true；update/pin/单条=false；create 豁免
- [x] manage_notebook rename、manage_tag update、manage_todo toggle/update 均接入审批；create 豁免
- [x] Approver 缺失（ctx 非空）fail-fast 报错，不静默放行；裸工具（ctx=nil）直接放行
- [x] 拒绝审批时返回中文拒绝错误文本、不落库、回填模型继续推理
- [x] manage_memory 未接入审批（豁免）
- [x] 工具 Info 描述已从"先 ask_user 确认"改写为"按当前审批模式弹出确认面板"
- [x] 前端零改动：审批弹窗/留痕/回放复用命令审批链路，无需前端修改
- [x] 新增/回归单元测试覆盖上述行为，`go build` / `go vet` / `go test ./internal/agent/...` 全绿
- [x] TOOLS.md 与 context.go 注释同步更新
