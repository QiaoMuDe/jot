# Tasks

- [x] Task 1: manage_todo 新增 delete / clear action
  - [x] action 分发补 `delete`(id 单条 → TodoService.Delete)与 `clear`(→ TodoService.DeleteCompleted,返回清理条数);均 critical=true,摘要注明「不可恢复」
  - [x] Info() 参数 schema 与描述补两个 action;ActionText 补「删除待办」/「清空已完成待办」
- [x] Task 2: manage_tag 新增 delete action
  - [x] action 分发补 `delete`(id → TagService.Delete),critical=true;Info schema、ActionText(「删除标签」)同步
- [x] Task 3: manage_notebook 新增 delete action
  - [x] action 分发补 `delete`(id 必填 + with_notes 可选默认 false;false→NotebookService.Delete,true→DeleteWithNotes),critical=true
  - [x] ActionText(「删除笔记本」,with_notes=true 摘要注明「连同笔记移入回收站」);Info schema 同步;id=1 时先返回「默认笔记本不可删除」再考虑审批(不弹无效审批窗,对齐「审批前先做基本参数校验」既有约定)
- [x] Task 4: manage_note 新增 delete action
  - [x] action 分发补 `delete`(ids → NoteService.Delete 软删进回收站,critical=true,支持单条/批量);恢复不提供(用户自行到回收站页面操作)
  - [x] manageNoteApprovalSummary 补 delete 分支;Info schema、ActionText(「删除笔记(移入回收站)」)同步
- [x] Task 5: 测试扩展(manage_approval_test.go)
  - [x] 每个 delete/clear:审批参数断言(critical=true)、rejectApprover 拒绝不落库(显式 `err: errors.New(...)`)、批准后真实落库生效
  - [x] manage_note:批量 delete 后 deleted_at 置位(回收站可查)
  - [x] manage_notebook:默认迁 id=1 / with_notes=true 进回收站 / id=1 删除报错且不触发审批
  - [x] manage_todo:clear 仅清已完成并返回条数、未完成不受影响
- [x] Task 6: 文档与文案同步
  - [x] meta.go:四个工具 Label 补「删除」字样(顺序不变)
  - [x] TOOLS.md:各工具 action 清单补新 action,审批分级段写明「删除类恒 critical」
  - [x] EVENTS.md:各工具审批分级表同步
- [x] Task 7: 全量验证与记忆更新
  - [x] gofmt -l internal/agent/、go build ./...、go vet ./internal/agent/...、go test ./internal/agent/... 全绿
  - [x] AGENTS.md 临时记忆按三步规则轮转追加本变更条目(注明需 wails build 出新二进制生效)

# Task Dependencies

- Task 1 → Task 2 → Task 3 → Task 4(同包顺序实施,避免编辑冲突;四者亦可由同一 agent 连续完成)
- Task 5 依赖 Task 1-4(action 定稿后写测试)
- Task 6 依赖 Task 1-4(文档与实现对齐)
- Task 7 依赖 Task 5、Task 6
