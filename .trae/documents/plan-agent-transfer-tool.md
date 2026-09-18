# 计划：新增 transfer_file 工具（工作目录 ↔ 用户桌面 上下传，仅入 os_agent 白名单）

## Summary

新增 `transfer_file` 工具，在工作目录（~/.jot/workspace）与用户桌面（~/Desktop）之间复制文件/目录，单参数 `direction`（download/upload）控制方向。两端均做边界校验防逃逸，是唯一允许触碰工作目录之外的文件工具。按用户拍板：**只注册进 os_agent 内层白名单，不注册到全局 buildTools**；桌面路径 = `os.UserHomeDir()`（读 USERPROFILE/HOME 环境变量）+ `"Desktop"` 拼接，不做 OneDrive/注册表精确解析。

## Current State Analysis

- **注册表**：[subagent.go L51-63](internal/agent/subagent.go) `toolConstructors` 是子 Agent 白名单的唯一构造器来源（`func(ctx *tools.Context) tool.InvokableTool`），新工具在此登记即可被 [subagent_os.go](internal/agent/subagent_os.go) 的 `osSubAgentToolNames` 白名单引用；漏登记会被 subagent_test.go 的「内层白名单恰 11」断言兜底抓住。
- **可复用实现**（[copy_file.go](internal/agent/tools/copy_file.go) 全套）：
  - go-kit `kitfs.CopyEx`（原子：临时文件+rename，覆盖时备份失败恢复，目录递归）；
  - `fsFinalTarget(dstFull, srcFull)`：目标为已存在目录时自动追加源基名；
  - `fsToolBase.requestApproval`（[fs_base.go L172](internal/agent/tools/fs_base.go)）：ctx nil 裸工具放行 / Approver nil fail-fast / 正常审批；
  - `expandTilde`（本会话已加）：前导 `~` 展开对两端通用；
  - `checkSrcDestRelation` 为同根自递归防护，跨端复制 src/dst 恒在不同子树，**不适用、不调用**。
- **路径校验**：[config.go L62-96](internal/config/config.go) `WorkspaceFilePath` 逻辑通用（Abs+Clean+EvalSymlinks 防符号链接逃逸+EqualFold 前缀校验）但错误文案写死 workspace——需泛化出带 label 的底座供桌面端复用。
- **审批现状对齐**：copy_file 覆盖审批 critical=false、纯新增免审批；delete_file critical=true；写文件覆盖 critical=false。
- **清单文档**：tools/doc.go 为权威工具清单（需同步）；meta.go 已收敛为单条 os_agent（**无需改**）；TOOLS.md §6.1 与 SUBAGENTS.md 各有一处「11 个」计数需同步。
- **前端零改动**：设置页/AI 下拉经 `GetAgentTools()` 收敛为单条 os_agent，工具组计数自适应。

## Proposed Changes

### 1. config.go：泛化沙箱路径校验底座

- 新增 `SandboxFilePath(sandboxRoot, p, label string) (string, error)`：迁移 `WorkspaceFilePath` 全部逻辑（Abs+Clean 拼接、resolveRealTarget 双重符号链接解析、EqualFold 前缀边界校验），错误文案改为模板：`超出 <label> 边界，仅允许操作 <label> 内的文件；请使用相对路径（如 notes/a.md）或以 <label>/ 开头的路径`。
- `WorkspaceFilePath` 改为薄封装：`return SandboxFilePath(root, p, "~/.jot/workspace")`。workspace 报错措辞微变（原「超出工作目录」→「超出 ~/.jot/workspace 边界」），config_test 只断言 err 非 nil 不断言文本，无回归风险。

### 2. fs_base.go：桌面端解析支撑

- `fsToolBase` 新增测试注入字段 `desktopDir string`；新增 `deskRoot() (string, error)`：注入值优先，否则 `home + "Desktop"`（`os.UserHomeDir()` 读环境变量，复用既有 homeDir 兜底逻辑）。
- 新增 `resolvePathIn(root, p, label string) (string, error)`：`expandTilde` + `config.SandboxFilePath(root, expanded, label)`；既有 `resolvePath` 可改为调它（label 固定 workspace），行为不变。

### 3. 新文件 tools/transfer_file.go（一文件一工具）

- 结构体 `transferFileTool{ fsToolBase }`，编译期断言 `tool.InvokableTool` 与 `ActionTextProvider`。
- **Info**：
  - Name: `transfer_file`；Desc 说明：工作目录与用户桌面（~/Desktop）之间复制文件/目录；download=工作目录→桌面、upload=桌面→工作目录；path 相对源端根（也支持 `~` 开头路径）；目标端为已存在目录时自动追加源文件名；目标已存在缺省拒绝、overwrite=true 才允许覆盖（触发审批）；下载到桌面属于工作目录之外的写入，始终需要审批。
  - 参数：`direction`（String，Enum `["download","upload"]`，必填）、`path`（String，必填，注明相对源端）、`overwrite`（Boolean，可选，缺省 false）。
- **InvokableRun 流程**（对齐 copy_file 风格）：
  1. `ctx.Err()` 取消检查 → 解析参数 → direction 枚举校验（非法直接报错）→ path 必填 + `validateTextLen`；
  2. 按 direction 定 srcRoot/dstRoot（download: ws→desk；upload: desk→ws）；
  3. `resolvePathIn` 解析 src（label 按端：workspace 用 `~/.jot/workspace`，桌面用 `~/Desktop`）→ `os.Lstat` 源存在性（不存在为参数错误，不触发审批）；
  4. `resolvePathIn` 解析 dst → `fsFinalTarget`（dst 为已存在目录自动追加源基名）；
  5. 覆盖判定：`destExists && !overwrite` → 报错「目标已存在，未设置 overwrite=true 时拒绝覆盖」；
  6. **审批分级**（写在文件头注释，TOOLS.md §6.1 判定标准）：
     - **download（写桌面=工作目录之外的外部副作用）一律审批，critical=true**：纯新增与覆盖都确认——confirm_every/review 强制确认，auto 放行并留审计痕；摘要：`下载到桌面：<src 显示路径> → 桌面/<dst 相对路径>`（覆盖时附「（覆盖）」）；
     - **upload（写工作区）对齐 copy_file**：纯新增免审批；`destExists` 时 `requestApproval(..., false)`。
  7. `kitfs.CopyEx(srcFull, dstFull, args.Overwrite)` 执行（目录递归、原子性）；
  8. 返回纯文本：`已下载：<rel> → 桌面/<dstRel>` / `已上传：桌面/<rel> → <dstRel>`（相对路径展示，目录复制注明「（目录，已递归复制）」）。
- **ActionText**：download → `下载文件：` + path 截 30 rune；upload → `上传文件：` + 同；解析失败回退 `传输文件`。
- 文件头注释写明：两端 SandboxFilePath 校验使工具无法借道读写两端之外的任何位置。

### 4. subagent.go：构造器登记

`toolConstructors` 追加 `"transfer_file": tools.NewTransferFile,`（仅此处，不动 buildTools）。

### 5. subagent_os.go：白名单与提示词

- `osSubAgentToolNames` 追加 `"transfer_file"`（11→12）。
- 内层 instruction 边界段补一句：与用户桌面交换文件（下载产出/上传资料）用 transfer_file，仅能操作工作目录与桌面两端。

### 6. 测试断言与文档同步

- [subagent_test.go](internal/agent/subagent_test.go)：内层白名单「恰 11」断言 → 12（并确认事件转发既有用例不受影响）。
- [tools/doc.go](internal/agent/tools/doc.go)：工具清单追加 `transfer_file` 与构造器 `NewTransferFile`（权威清单，必须）。
- [TOOLS.md](internal/agent/TOOLS.md) §6.1「read_file 等 11 个」→ 12 个。
- [SUBAGENTS.md](internal/agent/SUBAGENTS.md)「白名单恰 11 工具」等计数 → 12。
- 父包 [doc.go](internal/agent/doc.go)：无新依赖，不涉及。

### 7. 新文件 tools/transfer_file_test.go

注入 `workspaceRoot`/`homeDir`/`desktopDir`（均 t.TempDir，桌面= home/Desktop），复用 `mockApprover`（需扩展记录 critical 与摘要，或按 manage_approval_test 的记录模式新增记录字段）与 `rejectApprover`：

1. **download 新文件**：批准放行、桌面落盘内容一致、断言审批 hits=1 且 critical=true；
2. **download 覆盖**：overwrite=false 报错不落盘；overwrite=true 批准后覆盖成功（摘要含「（覆盖）」）；
3. **download 拒绝**：rejectApprover → 返回拒绝错误、桌面无文件；
4. **download 目录**：递归落盘桌面；
5. **upload 新文件**：审批 hits=0（免审批）、工作区落盘；
6. **upload 覆盖**：overwrite=true 批准（critical=false）后覆盖；
7. **越界**：download `path="../x"` 源端拒绝；upload `path=".."` 桌面端拒绝；
8. **参数校验**：direction 非法、path 缺失报错；
9. **裸工具放行**：ctx=nil 时 download 正常执行；
10. **Approver 缺失 fail-fast**：ctx 非 nil 但 Approver=nil → 报「未配置审批机制」。

### 8. AGENTS.md 临时记忆（实施收尾时）

按维护规范三步追加新 5：transfer_file 设计要点（双端 SandboxFilePath 校验、download 一律审批 critical=true、upload 对齐 copy_file、桌面=env home+Desktop、仅入 os_agent 白名单）。

## Assumptions & Decisions

- **不注册全局**（用户拍板）：registry.go buildTools 不加行；仅 toolConstructors + osSubAgentToolNames。
- **桌面路径**（用户拍板）：`os.UserHomeDir()` + `filepath.Join(home, "Desktop")`；OneDrive 重定向等特殊场景不做精确解析（后续可扩展注册表方案）。Windows zh-CN 桌面实际目录名即为 `Desktop`，无本地化问题。
- **download 一律审批 critical=true**：按 TOOLS.md §6.1「外部副作用 = critical」判定标准；auto 模式自动放行并留 `tool_auto_approval` 审计痕，与 run_command 高危语义一致。
- **upload 对齐 copy_file**：纯新增免审批、覆盖审批 critical=false。
- **symlink 处理对齐 copy_file**：两端 SandboxFilePath 的 EvalSymlinks 防逃逸 + Lstat 存在性检查；源为 symlink 时按内容复制（既有行为），不额外拒绝。
- **checkSrcDestRelation 不调用**：跨端复制 src/dst 恒在不同子树，无自递归风险。
- 纯后端改动：无需 `npm run build`；需 `wails build` 重新出二进制生效。

## Verification

1. `go build ./...`、`go vet ./internal/agent/... ./internal/config/...` 通过。
2. `go test ./internal/config/... ./internal/agent/...` 全绿（含 transfer_file 10 组用例与白名单 12 断言）。
3. `gofmt -l` 无输出。
4. 人工验收（wails build 后）：Agent 模式下让 AI「把 workspace 里 X 文件下载到桌面」/「把桌面上 Y 上传到 workspace」——观察 os_agent 调用 transfer_file，download 弹审批、越界路径被拒、桌面出现文件；auto 模式下 download 放行且留痕。
