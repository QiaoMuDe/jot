# Checklist

## 工作目录与后端
- [ ] `~/.jot/workspace/` 目录在首次进入 AI 助手模块时自动创建
- [ ] `internal/config` 新增 `DirWorkspace` 常量且路径解析正确
- [ ] `WorkspaceBackend` 实现 eino `adk/filesystem.Backend` 接口（LsInfo/Read/Write/Edit/GlobInfo/GrepRaw）
- [ ] `WorkspaceShell` 实现 eino `adk/filesystem.Shell` 接口
- [ ] 工具路径越界被拒绝并返回中文错误
- [ ] `execute` cwd 强制为 workspace，命令超时被终止，输出超限被截断
- [ ] 破坏性命令黑名单命中即需审批，黑名单集中在常量表可扩展

## 工具装配与注册
- [ ] `ls` / `read_file` / `write_file` / `edit_file` / `glob` / `execute` 六个工具已注册（registry.go）
- [ ] 六工具出现在内置工具清单（meta.go），参数契约对齐 eino（read_file offset/limit、edit_file old/new/replace_all）
- [ ] 脚本可写盘并在后续对话 read_file / edit_file / execute 复用

## 授权模式
- [ ] 三种执行模式存在且持久化，默认 `confirm_every`
- [ ] `confirm_every` 每次新工具执行前暂停等待审批
- [ ] `auto` 直接执行不触发审批
- [ ] `review` 命中关键操作（黑名单/覆盖写/修改既有文件/下载执行/系统级）才暂停，其余自动执行

## 审批机制
- [ ] 审批通过继续执行、审批拒绝向模型回填「用户拒绝」且不中断 ReAct 循环
- [ ] 审批结果记入 `Result.ToolCalls`，前端可回溯
- [ ] 后端 `ApproveToolCall(id, approved)` 绑定存在且字段语义正确

## 前端
- [ ] 前端提供执行模式选择器，切换即持久化
- [ ] 前端审批弹层可批准/拒绝，ESC 关闭等效拒绝，展示工具名/参数/风险说明
- [ ] 前端工具状态支持「待审批」挂起态

## 测试与全链路
- [ ] 单测覆盖：路径越界、相对路径、子目录、覆盖既有文件、超时、输出截断、黑名单命中/放行
- [ ] 全链路验证通过：三模式行为、越界与黑名单拒绝、脚本写盘→执行→复用、审批拒绝不中断循环
- [ ] 项目可正常 `go build` / 前端 `npm run build` + `wails build` 无回归