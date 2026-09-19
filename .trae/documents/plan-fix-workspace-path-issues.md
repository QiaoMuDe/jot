# 修复工作区工具路径失败三大问题

## 摘要

依据 `tools_log.log`(克隆/探索 memsh 仓库的 400+ 工具调用记录)分析,50 次 `tool_error` 归类为三个根因,本计划逐一修复:

1. **ls\_dir 多层子目录列举在 Windows 上失败**(23 次,46%)—— 工具 bug,收益最大
2. **Unix 风格绝对路径** **`/home/user/...`** **被静默拼接进工作区**(约 18 次,36%)—— 路径校验健壮性
3. **os\_agent 提示词未禁止模型自行展开** **`~`**(诱发根因 2 的模型行为) —— 提示词指引

## 现状分析

### 根因 1:ls\_dir 使用 `fs.ReadDir(root.FS(), rel)` 列举多层目录失败

* [ls\_dir.go](internal/agent/tools/ls_dir.go#L114) 用 `fs.ReadDir(root.FS(), rel)`,其中 `rel` 来自 [openRootFor](internal/agent/tools/fs_base.go#L111) 的 `filepath.Rel(root, fullPath)`,在 Windows 上返回反斜杠分隔路径(如 `memsh-0.0.15\pkg`)。

* `os.Root.FS()` 的 io/fs 适配器在 Windows 上只接受 `/` 分隔的多层路径;反斜杠多层路径返回 `readdir xxx: invalid argument`。

* 已用实验验证:`fs.ReadDir(root.FS(), "a")` OK、`fs.ReadDir(root.FS(), "a\\b")` 报错、`fs.ReadDir(root.FS(), "a/b")` OK、`root.Open("a/b").Readdirnames` OK。

* 单层路径(如 `memsh-0.0.15`)无分隔符恰好逃过,故现有 [TestLsDir](internal/agent/tools/fs_tools_test.go#L214) 的 `{"path":"sub"}`(单层)用例全部通过,**未覆盖多层钻取**。

* 对照 [read\_file.go](internal/agent/tools/read_file.go#L114) 用 `root.Open(rel)`(os.Root 原生 API,接受原生路径),故读文件多层正常 —— 解释了"ls\_dir 钻取全失败、read\_file 正常"的现象。

### 根因 2:外来绝对路径未被识别为非法

* Windows 上 `filepath.IsAbs("/home/user/...")` 返回 false,故 [SandboxFilePath](internal/config/config.go#L67) 把它当相对路径 `filepath.Join(rootClean, p)` 拼入工作区根,产生 `C:\...\workspace\home\user\...` 幽灵路径。

* 边界校验(`EqualFold` 前缀)对此**通过**(前缀确实在 root 下),直到 `root.Open` 才报 "The system cannot find the path specified"。

* 错误信息无诊断性,模型无法识别是路径格式问题,反复重试同一模式。

* 调用方仅 3 处:[fs\_base.go](internal/agent/tools/fs_base.go#L167)、[WorkspaceFilePath](internal/config/config.go#L59)、[workspace\_service.go](internal/services/workspace_service.go#L292)(桌面标签),修改需保证三者语义一致。

### 根因 3:os\_agent 提示词未约束 `~` 展开行为

* [subagent\_os.go](internal/agent/subagent_os.go#L29-L38) 的 `osSubAgentInstruction` 边界段仅写 `~/.jot/workspace/`,模型凭 Unix 训练偏见自行展开成 `/home/user/.jot/workspace`(日志首条 os\_agent 回复即出现"即 `/home/user/.jot/workspace`")。

* 根因 1 修复后模型可用相对路径钻取成功,根因 3 是配套防回归。

## 修改方案

### 修改 1:ls\_dir 多层路径修复(核心)

**文件**:[ls\_dir.go](internal/agent/tools/ls_dir.go)

* 将 `fs.ReadDir(root.FS(), rel)` 改为 `fs.ReadDir(root.FS(), filepath.ToSlash(rel))`,统一为 `/` 分隔后经 io/fs 适配器列举(实验已验证可行)。

* 保持 `fs.ReadDir` + `[]fs.DirEntry` 结构不变,detail 大小/时间逻辑零改动。

* 新增 `path/filepath` 导入(当前 ls\_dir.go 未导入)。

**文件**:[fs\_tools\_test.go](internal/agent/tools/fs_tools_test.go)

* `TestLsDir` 新增"多层子目录钻取"用例:`{"path":"sub/deep"}` 应列出 `c.txt`,且首行锚点为 `当前目录: sub/deep` —— 回归修复 1,覆盖 Windows 反斜杠 rel 场景。

### 修改 2:SandboxFilePath 外来绝对路径友好报错

**文件**:[config.go](internal/config/config.go)

* 在 `SandboxFilePath` 的 target 解析(拼接)之前插入校验:以 `/` 或 `\` 开头但 `!filepath.IsAbs(p)` 的路径(即外来/畸形绝对路径,Windows 上 IsAbs=false 的 Unix 风格路径)直接返回友好错误,不进入拼接:

  * 错误文案:`路径格式无效(不是可识别的绝对路径);请使用相对路径(如 notes/a.md)或以 <label>/ 开头的路径`

* 跨平台安全:类 Unix 平台 `/home/...` 是合法绝对路径(`IsAbs`=true),不触发新校验,行为不变;仅拦截"看似绝对路径但本机不识别"的情形。

**文件**:[config\_test.go](internal/config/config_test.go)

* `TestWorkspaceFilePath` 的 rejected 列表追加:

  * Unix 风格绝对路径 `/home/user/.jot/workspace/x.txt`(Windows 触发新校验;Unix 上为合法绝对路径但越界,同样拒绝,断言 `err != nil` 平台无关)

  * 反斜杠开头路径 `\home\user\x.txt`(畸形路径,应拒绝)

* 断言保持与现有测试一致(仅 `err != nil`,不断言文案,避免跨平台文案耦合)。

### 修改 3:os\_agent 提示词路径指引(防回归)

**文件**:[subagent\_os.go](internal/agent/subagent_os.go)

* `osSubAgentInstruction` 边界段追加一条(插在首条路径规则之后):

  * `- 不要猜测 ~ 的展开路径(如 /home/user/.jot/workspace),家目录展开由工具自行处理;一律使用相对工作区的路径(如 memsh-0.0.15/cmd)或以 ~/.jot/workspace/ 开头的路径,不要传本机绝对路径。`

* 纯提示词改动,无逻辑变更。

## 假设与决策

* ls\_dir 修复采用 `filepath.ToSlash` 而非改 `root.Open`+`Readdir`:改动最小、实验直接验证通过、detail 逻辑零改动;两者均已实验可用,选侵入性小的。

* 根因 2 的错误文案用"路径格式无效"而非"超出边界":语义区分"格式错误"与"越界",帮助模型理解重试方向。

* 不处理日志中 2 次 `os_agent 403`(MCP 网络问题,与路径无关)与 1 次裸文件名 `limits.go`(模型上下文丢失,偶发)。

* 不新增前端/绑定改动,纯后端,`wails build` 后生效。

## 验证步骤

1. `go build ./...` 编译通过
2. `go vet ./...` 无告警
3. `go test ./internal/config/ ./internal/agent/tools/` 全绿(含新增多层 ls\_dir 用例与外来路径拒绝用例)
4. `go test ./internal/agent/tools/ -run 'TestLsDir|TestResolvePath' -v` 重点回归
5. `golangci-lint run ./...` 零告警(项目规范)
6. `wails build` 出新二进制后,手动验证:AI 对话让 os\_agent "列出 memsh-0.0.15 下的 cmd 目录结构"—— 应一次成功,不再出现路径重试链

