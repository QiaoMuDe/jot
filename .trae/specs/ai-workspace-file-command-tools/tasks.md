# Tasks

## 阶段一：文件工具（workspace 边界 + read/write/list）
- [x] Task 1: workspace 边界校验辅助函数
  - [x] 在 `internal/config/config.go`（或工具包）实现 `WorkspacePath(root, rel)`：`filepath.Clean` + `Abs` + 前缀校验，落出返回错误
  - [x] 单测：绝对路径越界 / 嵌套合法 / 分隔符混淆（`..` 逃逸）均正确处理
- [x] Task 2: read_file 工具
  - [x] 参数 `path`、`offset`、`length`，rune 分页，越界报"已全部读取"，未读完附续读提示
  - [x] 路径经 Task 1 校验，落出拒绝
  - [x] 注册入 `registry.go`，前端元信息自动生效
  - [x] 单测：首段/续读/末段截尾/越界/越权拒绝
- [x] Task 3: write_file 工具
  - [x] 参数 `path`/`content`/`append`，自动 `MkdirAll`；覆盖已存在文件标记为破坏性（预留审批点）
  - [x] 路径经 Task 1 校验
  - [x] 注册 + 单测（新建/覆盖/越权拒绝）
- [x] Task 4: list_dir 工具
  - [x] 参数 `path`，递归深度上限，返回相对路径 + 类型
  - [x] 路径经 Task 1 校验，注册 + 单测

## 阶段二：run_command（os/exec 裸命令）
- [x] Task 5: run_command 工具
  - [x] 参数 `command`/`args`/`cwd`；强制 cwd=workspace（`cwd` 越界拒绝）
  - [x] `exec.CommandContext` 超时；`exec.LookPath` 定位命令，不存在报"环境无此命令"回填
  - [x] 输出合并截断回填（复用 `TruncateRunes`）
  - [x] **禁止 shell 语法**：工具 Desc 明确"只接受可执行文件+参数，不支持管道/重定向/shell 表达式"
  - [x] 注册 + 单测（成功/超时/LookPath 失败/cwd 越界）
- [x] Task 6: 命令黑名单定义
  - [x] 常量集（rm/del/rmdir/shutdown/reboot/mkfs/format/dd 写盘/联网拉取/sudo/systemctl 等），`review`/`auto` 模式命中须审批

## 阶段三：审批暂停/续跑机制（核心工程）
- [x] Task 7: ApprovalWaiter 泛化
  - [x] `tools` 包新增 `ApprovalWaiter` 接口（`BeginApproval` 原子抢占 + `WaitDecision` 阻塞），`Context` 增加该字段
  - [x] `agentSession` 实现 `ApprovalWaiter`（复用 ask_user 的通道/互斥/排空模式）
  - [x] 单测：抢占互斥 / 等待-投递 / 取消解锁 / 残留排空
- [x] Task 8: wrappedTool 前置审批门
  - [x] 执行前按审批模式判定是否阻塞；危险操作（写覆盖/命令黑名单）触发
  - [x] 拒绝=返回"用户已拒绝"回填模型，不中断循环；审批动作记入 `toolRecords`
  - [x] 发射 `ai:tool-approval` 事件（callID/工具名/参数摘要/原因）
- [x] Task 9: 前端确认面板
  - [x] `ApproveToolCall(sessionID, callID, approved)` 绑定方法（app.go）
  - [x] 前端渲染确认卡（允许/拒绝），复刻 `.ai-mode-tip`/ask_user 交互与 z-index；记入工具记录

## 阶段四：三模式接线 + 收尾
- [x] Task 10: 接入 approval_mode
  - [x] `ai_service` 读会话 `approval_mode`，`confirm_every`/`review`/`auto` 驱动审批门；auto 黑名单仍确认
- [x] Task 11: 文档与验收整合
  - [x] `TOOLS.md`/`EVENTS.md` 更新（工具、`ai:tool-approval` 事件、审批语义）
  - [x] `AGENTS.md` 记忆点收尾；`wails build` 前端生效验证

# Task Dependencies
- Task 1 ← Task 2,3,4（边界校验是文件工具前置）
- Task 2,3,4 可并行（均依赖 Task 1）
- Task 5 依赖 Task 1/6、Task 6；Task 6 独立可并行
- Task 7 ← Task 8,9；Task 8,9 依赖 Task 7（审批门与面板需 waiter 就绪）
- Task 10 依赖 Task 8/9 + `ai_service` 读取；Task 11 最后
- Task 6/server 与 Task 2-4 无依赖可并行