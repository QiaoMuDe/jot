# Checklist

- [x] 五个新删除 action 全部可用:manage_note.delete、manage_notebook.delete、manage_tag.delete、manage_todo.delete、manage_todo.clear(恢复不提供,用户自行到回收站页面操作)
- [x] 所有 delete/clear 审批分级一律 critical=true(review 必确认、confirm_every 全确认、auto 放行+留痕)
- [x] 审批摘要完整:todo delete/clear 注明「不可恢复」;notebook with_notes=true 注明「连同笔记移入回收站」;note delete 注明「移入回收站」
- [x] TodoService.DeleteUnfinished / DeleteAll / NotebookService.ResetAll 未接入任何工具(tools 目录 grep 零引用)
- [x] manage_notebook id=1 删除返回「默认笔记本不可删除」且不弹审批窗(测试断言 Approver 未被调用)
- [x] manage_notebook 默认删除(with_notes=false)时笔记迁入默认笔记本 id=1,数据零丢失
- [x] 管理工具既有 action(create/list/toggle/update/rename/pin/move/add_tag/remove_tag/edit)行为无回归(全量 go test ok)
- [x] ActionText 动作文案齐备,前端工具状态行正确显示(缺省回退「执行」)
- [x] Info() schema 与描述覆盖新 action(action enum 补 delete/clear,notebook 新增 with_notes 参数)
- [x] meta.go 四个 Label 补「删除」字样,顺序未变
- [x] TOOLS.md / EVENTS.md 与实现一致
- [x] manage_approval_test.go 新增用例全绿(审批断言 + 拒绝不落库 + 真实落库 + id=1 保护 + clear 只清已完成)
- [x] gofmt / go build / go vet / go test ./internal/agent/... 全绿
- [x] AGENTS.md 临时记忆已按三步规则轮转更新
