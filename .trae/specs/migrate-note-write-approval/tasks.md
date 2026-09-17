# Tasks

- [x] Task 1: manage_note 写操作接入审批（核心）
  - [x] 1.1 移除 `confirm` 参数：结构体解析、schema 参数定义、Info 描述中"强制确认：先 ask_user"引导段落全部删除
  - [x] 1.2 新增写操作审批：InvokableRun 在写操作分发前调用 `m.ctx.Approver.RequestApproval(ctx, "manage_note", summary, critical)`；Approver 为 nil（ctx 非空）时 fail-fast 报错；ctx 为空（裸工具）直接放行；summary 含动作 + 目标（如"编辑笔记 #5 正文"、"批量移动 5 篇笔记到笔记本 #2"）
  - [x] 1.3 critical 分级：edit 恒 true；move/add_tag/remove_tag 按 ids 长度 >1 判 true；update/pin 恒 false
  - [x] 1.4 复用/调整 `manageNoteActionCN` 供 summary 文案；删除旧引导提示返回逻辑
  - [x] 1.5 更新 Info 描述：写操作改"按当前审批模式弹出确认面板"

- [x] Task 2: manage_notebook / manage_tag / manage_todo 接入审批
  - [x] 2.1 manage_notebook：rename 分支前调 RequestApproval（critical=false，summary"重命名笔记本 #N 为 X"）；create 豁免
  - [x] 2.2 manage_tag：update 分支前调 RequestApproval（critical=false）；create 豁免
  - [x] 2.3 manage_todo：toggle/update 分支前调 RequestApproval（critical=false）；create 豁免
  - [x] 2.4 三个工具 Info 描述补充"写操作按当前审批模式弹出确认面板"说明

- [x] Task 3: 单元测试
  - [x] 3.1 manage_note 审批用例：批准放行 / 拒绝返回拒绝错误且不落库 / Approver 缺失 fail-fast / 裸工具（ctx=nil）直接放行 / create 豁免 / critical 分级断言（edit=true、批量 move=true、单条 pin=false）
  - [x] 3.2 manage_notebook/manage_tag/manage_todo 各补最小审批用例（批准放行 + create 豁免）
  - [x] 3.3 回归：清理/适配受 confirm 移除影响的旧测试用例
  - [x] 3.4 验证：`go build ./... && go vet ./... && go test ./internal/agent/...` 全绿

- [x] Task 4: 文档同步
  - [x] 4.1 `internal/agent/TOOLS.md` 更新 manage_* 工具审批说明
  - [x] 4.2 `internal/agent/tools/context.go` Approver 注释补充"笔记管理工具"纳入范围

# Task Dependencies
- Task 3 依赖 Task 1、Task 2（测试针对改造后的工具）
- Task 4 依赖 Task 1、Task 2（文档描述与实现一致）
- Task 1 与 Task 2 文件独立、可并行；共享 Approver 接入模式，建议 Task 1 先落地作为模式样板
