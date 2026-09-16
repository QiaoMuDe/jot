# Tasks

## 阶段一：后端工作目录与后端接口
- [ ] Task 1: 规划工作目录
  - [ ] 在 `internal/config/config.go` 新增 `DirWorkspace = "workspace"` 常量，并补充路径获取函数
  - [ ] 提供/复用「按需创建目录」helper，确保装配前 `~/.jot/workspace/` 存在
- [ ] Task 2: 实现 workspace 后端（对齐 eino `adk/filesystem` 契约）
  - [ ] 确认依赖 `github.com/cloudwego/eino/adk/filesystem`（同 eino 版本内，无新增外部依赖）
  - [ ] 实现 `workspace_backend.go`：实现 `Backend` 接口（LsInfo/Read/GrepRaw/GlobInfo/Write/Edit），
        统一路径边界校验（归一化 + workspace 前缀检查，越界返回中文错误）
  - [ ] `Write`/`Edit` 自动创建父目录；对已存在文件返回「覆盖标记」元信息供 review 判定
  - [ ] 实现 `workspace_shell.go`：实现 `Shell` 接口，cwd 强制为 workspace，命令超时 + 输出字节截断
  - [ ] 实现破坏性命令黑名单判定 `ClassifyCommandRisk(cmd)`，黑名单集中在常量表
  - [ ] 补单元测试：路径越界、相对路径、子目录、覆盖既有文件、超时、输出截断、黑名单命中/放行

## 阶段二：工具装配（按 eino 契约）
- [ ] Task 3: 生成 fs 工具（fs_tools.go）
  - [ ] 按 eino `adk/filesystem` 契约生成 `ls` / `read_file`（offset/limit）/ `write_file` / `edit_file` / `glob` 自研工具构造器
- [ ] Task 4: 生成 execute 工具（execute_tool.go）
  - [ ] 实现 `execute` 工具：shell 与 python 脚本执行，复用 workspace_shell
  - [ ] 脚本落盘可后续复用

## 阶段三：审批机制（后端）
- [ ] Task 5: 审批暂停/续跑机制
  - [ ] 在 `tools/context.go` 新增 `ApprovalWaiter` 接口与 `Context.ApprovalWaiter` 字段（复用 AskWaiter 同轮阻塞范式，支持 ctx 取消）
  - [ ] 在 `wrappedTool`/新工具执行路径接入审批：`confirm_every` 全部暂停、`review` 按 `ClassifyCommandRisk`/覆盖/修改既有文件等规则暂停、`auto` 直接执行
  - [ ] 在 `agent.go` 注入 ApprovalWaiter，装配时读取执行模式设置
  - [ ] 审批拒绝时向模型回填「用户拒绝」中文提示，不中断 ReAct 循环；审批结果记入 `Result.ToolCalls`
  - [ ] 在 `app.go` 新增前端可调 `ApproveToolCall(id, approved)` 绑定

## 阶段四：执行模式设置与持久化
- [ ] Task 6: 执行模式设置项
  - [ ] 新增执行模式设置项（confirm_every / auto / review），默认 `confirm_every`
  - [ ] 设置可持久化，并在 `Request`/装配时读取生效

## 阶段五：注册与清单
- [ ] Task 7: 注册 6 个工具到 registry 与 meta
  - [ ] 在 `registry.go` `buildTools` 注册 `ls` / `read_file` / `write_file` / `edit_file` / `glob` / `execute`
  - [ ] 在 `tools/meta.go` `BuiltinTools` 追加 6 条展示文案（顺序即展示顺序）

## 阶段六：前端
- [ ] Task 8: 前端执行模式选择器
  - [ ] 在 AI 助手模块设置区提供三模式选择表单项，切换即持久化
- [ ] Task 9: 前端审批弹层
  - [ ] 实现审批确认层（展示工具名/参数详情/风险说明/允许拒绝按钮，ESC 关闭，符合既有确认交互规范）
  - [ ] 监听审批事件，调用 `ApproveToolCall` 回传结果
  - [ ] 工具状态展示支持「待审批」挂起态

## 阶段七：验证
- [ ] Task 10: 全链路验证
  - [ ] 三模式行为验证（confirm_every 每次暂停 / auto 直通 / review 命中才暂停）
  - [ ] 越界与黑名单拒绝符合预期
  - [ ] 脚本写盘→执行→复用链路通过
  - [ ] 审批拒绝不中断循环、结果可回溯

# Task Dependencies
- Task 2 依赖 Task 1（工作目录存在）
- Task 3、Task 4 依赖 Task 2（Backend/Shell 契约）
- Task 5 依赖 Task 3、Task 4（审批作用于 fs/shell 工具）与 Task 6（执行模式）；Task 6 无依赖可并行
- Task 7 依赖 Task 3、Task 4（工具存在）
- Task 9 依赖 Task 5（审批事件/回传接口）
- Task 8 依赖 Task 6，可并行
- Task 10 依赖 Task 8、Task 9、Task 7

（Task 6 与 Task 5 前段可并行；Task 8 与 Task 9 可并行；Task 3 与 Task 4 在 Task 2 后可并行）