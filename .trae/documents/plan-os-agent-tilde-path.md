# 计划：os\_agent 文件/命令工具支持 `~` 短路径解析 + 越界报错自纠提示

## Summary

os\_agent 的 11 个文件/命令工具经常第一次调用就报错，根因是模型照抄各处文案里的 `~/.jot/workspace` 记法，而路径解析层不展开 `~`（被拼成字面量嵌套目录），或模型猜错的绝对路径触发越界拒绝。本计划做两件事（用户已拍板，不注入绝对路径到上下文）：

* **方案 B（根治）**：在 agent 工具路径解析咽喉点支持 `~` 前缀展开，使 `~/.jot/workspace/foo` 形式的路径直接可用；glob 因模式不走常规解析单独处理。

* **方案 C（辅助）**：越界报错信息追加「正确路径写法」提示，加速模型第二次自纠。

纯后端 Go 改动，前端零改动。

## Current State Analysis

* 11 个工具（read\_file/write\_file/edit\_file/ls\_dir/glob/grep\_file/copy\_file/move\_file/delete\_file/mkdir\_dir/run\_command 的 cwd）全部经 [fs\_base.go L126-132](d:\资源池\下水道\Dev\本地项目\jot\internal\agent\tools\fs_base.go) `resolvePath` → `config.WorkspaceFilePath`（[config.go L62-96](d:\资源池\下水道\Dev\本地项目\jot\internal\config\config.go)）做边界+symlink 校验；`WorkspaceFilePath` 仅此一个生产调用点。

* `WorkspaceFilePath` 不展开 `~`：`~` 非绝对路径 → 拼成 `<workspace>/~/.jot/workspace/foo`，读报不存在、写悄悄建出名为 `~` 的垃圾目录。

* 模型猜错家目录绝对路径 → 越界拒绝，报错「超出工作目录，仅允许操作 \~/.jot/workspace 内的文件」仍是 `~` 记法，无自纠提示。

* 12 处参数描述（11 个工具 + run\_command cwd）统一写「相对 \~/.jot/workspace 或为其内绝对路径」；[subagent\_os.go L29](d:\资源池\下水道\Dev\本地项目\jot\internal\agent\subagent_os.go) 内层提示词同样只写 `~/.jot/workspace`。

* 测试现状：`fsToolBase.workspaceRoot` 注入临时目录（fs\_tools\_test.go 等已沿用）；`config_test.go` 只断言 `err != nil`，不断言错误文本（改文案无回归风险）；TOOLS.md 无路径约定段落，无需改。

* glob 特例：pattern 不走 resolvePath 全路径，`validateGlobPattern` 只拒绝 `/` 开头与 `X:` 盘符绝对路径，`~` 开头会被当相对模式静默匹配为空（误导）。

## Proposed Changes

### 1. 方案 B：`~` 前缀展开（fs\_base.go）

文件：`internal/agent/tools/fs_base.go`

* `fsToolBase` 新增测试注入字段 `homeDir string`（空则运行时取 `os.UserHomeDir()`），与现有 `workspaceRoot` 注入模式一致。

* 新增 `expandTilde(p string) (string, error)` helper，仅处理**前导** `~`：

  * `p == "~"` → 展开为 home；

  * `p` 以 `~/` 或 `~\` 开头 → home + 余下部分；

  * 路径中间的 `~`（如 `notes/~foo.md`）不展开，原样返回；

  * `os.UserHomeDir()` 失败 → fail-fast 报错「无法获取用户家目录」（不做静默降级，避免 `~/...` 被当相对路径解析出迷惑结果）。

* `resolvePath` 在调用 `config.WorkspaceFilePath` 前先展开；展开结果仍走原有 Clean + 边界校验 + EvalSymlinks + os.Root 双防线，安全语义不变（`~/其它目录` 照样越界拒绝）。

* 同步更新 `resolvePath` 注释。

### 2. 方案 B：glob 模式的前缀剥离（glob.go）

文件：`internal/agent/tools/glob.go`

glob pattern 不能直接做 `~` 展开（pattern 需保持相对形式参与 `path.Match`），改为**字面前缀剥离**（不依赖真实 home）：

* pattern（已统一 `/` 分隔后）以 `~/.jot/workspace/` 开头 → 剥掉该前缀转为相对模式（如 `~/.jot/workspace/scripts/*.md` → `scripts/*.md`）；

* pattern 恰为 `~/.jot/workspace` → 视为 `*`（列出根下条目）；

* 其余以 `~` 开头的 pattern → 明确报错「glob pattern 须相对 \~/.jot/workspace，或以 \~/.jot/workspace/ 开头」；

* 剥离逻辑放在 `validateGlobPattern` 之前（或合并进其签名），模式合法性校验照常执行。

### 3. 方案 C：越界报错自纠提示（config.go）

文件：`internal/config/config.go` L93

错误文案改为：

```
超出工作目录，仅允许操作 ~/.jot/workspace 内的文件；请使用相对工作区的路径（如 notes/a.md）或 ~/.jot/workspace/ 开头的路径
```

已确认 config\_test.go 不断言该文本，无回归。

### 4. 参数描述统一更新（12 处 / 11 个文件）

统一文案（token 成本几乎不变）：「相对 \~/.jot/workspace 的路径、\~/.jot/workspace/ 开头路径或其内绝对路径均可」。

| 文件              | 位置                                      |
| --------------- | --------------------------------------- |
| read\_file.go   | path                                    |
| write\_file.go  | path                                    |
| edit\_file.go   | path                                    |
| delete\_file.go | path                                    |
| copy\_file.go   | src、dst（2 处）                            |
| move\_file.go   | src、dst（2 处）                            |
| mkdir\_dir.go   | path                                    |
| ls\_dir.go      | path                                    |
| grep\_file.go   | path                                    |
| run\_command.go | cwd（补「支持相对路径与 \~/.jot/workspace/ 开头路径」） |
| glob.go         | pattern（补 `~/.jot/workspace/*.md` 示例）   |

### 5. 内层提示词同步（subagent\_os.go）

文件：`internal/agent/subagent_os.go` L29

边界段补充一句：路径可直接写相对工作区的路径，或 `~/.jot/workspace/...` 形式（两者均可解析）。

### 6. 测试

* `internal/agent/tools/fs_tools_test.go` 新增 `TestResolvePathTildeExpansion`：注入 home=临时目录 A、root=A/.jot/workspace，经 read\_file 验证：

  * `~/.jot/workspace/sub/a.txt` 正确读取（含 `~\` 反斜杠变体）；

  * `~/other.txt`、`~` 单独 → 越界拒绝；

  * `notes/~x.txt` 中间 `~` 不展开、按相对路径处理。

* `internal/agent/tools/glob_test.go` 新增：`~/.jot/workspace/*.md` 命中根层 .md；`~/.jot/workspace` 单独视为 `*`；`~/foo/*.md` 报错。

* 回归：`config_test.go`、既有 tools 测试全量跑通。

### 7. AGENTS.md 临时记忆更新

按维护规范三步：删旧 1、顺移 2-4、追加新 5 记录本次变更（`~` 展开咽喉点、glob 字面剥离、报错提示文案）。

## Assumptions & Decisions

* **展开位置选 fsToolBase.resolvePath 而非 config.WorkspaceFilePath**：config 保持纯「拼接+校验」语义，不掺 `~` 展开；且 home 可注入测试（config 层注入会因真实 home ≠ 临时 root 而无法测试合法场景）。

* **glob 用字面前缀剥离而非真实展开**：pattern 需保持相对形式做 `path.Match`；字面匹配 `~/.jot/workspace` 与真实展开在生产环境等价（root 固定即该路径），且测试无需注入 home。

* **不做方案 A**：不向【环境信息】注入绝对路径（用户拍板）。

* **home 获取失败 fail-fast**，不静默降级。

* 纯后端改动：无需 `npm run build`；需 `wails build` 重新出二进制才在桌面应用生效。

## Verification

1. `go build ./...`、`go vet ./...` 通过。
2. `go test ./internal/config/... ./internal/agent/...` 全绿（含新增 tilde 展开与 glob 用例）。
3. `gofmt -l` 无输出。
4. 人工验收（wails build 后）：AI 助手 Agent 模式下发「在 workspace 里创建 notes/demo.md 并读取」，观察 os\_agent 第一次调用即用相对路径或 `~/.jot/workspace` 路径成功，不再出现首跳路径报错；越界请求报错文案含路径写法提示。

