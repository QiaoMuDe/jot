# 计划：os\_agent 平台感知提示词 + ls\_dir 路径上下文

> 目标文件：
>
> * `internal/agent/subagent_os.go` —— os\_agent 内层提示词按运行平台拼接注入
>
> * `internal/agent/tools/ls_dir.go` —— 输出首行打印当前相对路径（替代 pwd 诉求）
>
> * 相应测试文件同步更新

***

## 一、Summary

针对 os\_agent 在 Windows 上「好多命令没有、执行常报错浪费工具调用次数」以及「模型不知道自己当前在哪个路径」两个痛点，落地前一轮分析中的 **方案 A（平台感知提示词）** 与 **方案 B（ls\_dir 带路径上下文）**，不做 pwd 工具。两条均为收益高、改动小的增量，不动 `run_command` 核心逻辑与通用子 Agent 机制 `subagent.go`。

***

## 二、Current State Analysis

### 2.1 平台感知缺失（方案 A）

* `osSubAgentInstruction`（[subagent\_os.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/subagent_os.go#L26-L34)）为 `const` 常量，无任何运行平台信息；模型训练数据偏 Unix，倾向调用 `ls/cat/grep/rm/find/sed/awk` 等命令，在原生 Windows 上 `exec.LookPath`（[run\_command.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/run_command.go#L226-L228)）找不到 → 报错 → 烧掉一次内层 `maxIterations`（20）。

* 装配入口为 `buildOSSubAgent`（[subagent\_os.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/subagent_os.go#L57-L60)），直接传包级 var `osAgentConfig` 给通用工厂 `newDelegatedAgentTool`（[subagent.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/subagent.go#L86-L129)，其内部 `Instruction: cfg.instruction` 第 112 行）。

* `BuildOSSubAgent` 在 `subagent_test.go` 中复用（仅断言内层白名单工具数量），故改入口拼接提示词**不影响既有测试**。

* Go 标准库 `runtime.GOOS` / `runtime.GOARCH` 可直接取运行平台，无需新增依赖。

### 2.2 ls\_dir 无路径上下文（方案 B）

* `ls_dir`（[ls\_dir.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/ls_dir.go#L77-L152)）只列单层条目名，输出不含当前所在目录的相对路径；文件工具是无状态「path 相对工作区根解析」，无 cd 状态，故真正的诉求是「让模型从 ls\_dir 输出锚定自己正列出哪个子目录」，以便正确拼接后续相对路径。

* `fsToolBase.relDisplayPath(fullPath)`（[fs\_base.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/fs_base.go#L85-L96)）已可把工作区内绝对路径转相对展示路径（根目录返回 `"."` / filepath 相对），复用它即可，无需新增解析逻辑。

* 现有测试 `TestLsDir`（[fs\_tools\_test.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/fs_tools_test.go#L214-L290)）全部用 `strings.Contains` 断言，首行追加「当前目录」**不破坏既有断言**。

* 输出有 `cumRunes` 有界累计 + 「目录内容过多提前停止」逻辑（[ls\_dir.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/ls_dir.go#L121-L150)），首行需同步计入预算。

***

## 三、Proposed Changes

### 3.1 `internal/agent/subagent_os.go` —— 平台感知提示词注入

**做法**：`buildOSSubAgent` 入口不再直接透传包级 `osAgentConfig`，而是基于 `runtime.GOOS` 构造一段平台说明，拼接到 `osSubAgentInstruction` 末尾，生成临时 `cfg` 副本传给 `newDelegatedAgentTool`。

* 新增函数级注释（遵循项目函数注释规范），返回平台说明段（Go 标准库 `runtime.GOOS`，无外部依赖）。

* 平台片段从简：Windows 分支核心只提醒「本机无 ls/cat/grep/rm 等 Linux 命令，文件类操作用内置工具，run\_command 只调用真实存在的程序/脚本」；其它平台仅标注平台名即可。

具体改动：

```go
import "runtime"

// osSubAgentPlatformNote 依据运行平台返回 os_agent 内层提示词的平台片段：
// 标注当前平台，Windows 下额外提醒缺失常用 Linux 命令、宜用内置文件工具，
// 避免 run_command 照抄不可用命令报错浪费工具调用。
func osSubAgentPlatformNote() string {
	if runtime.GOOS == "windows" {
		return "【当前平台：Windows】本机无 ls/cat/grep/rm/find/sed 等 Linux 命令,文件类操作请用内置工具(read_file/ls_dir/grep_file 等),run_command 只调用本机确实存在的程序/脚本。"
	}
	return "【当前平台:" + runtime.GOOS + "】"
}

// osAgentConfig.instruction 拼接逻辑移动到 buildOSSubAgent：
func buildOSSubAgent(runCtx context.Context, chatModel *openai.ChatModel, innerCtx *tools.Context) *delegatedAgentTool {
	cfg := osAgentConfig
	cfg.instruction = osSubAgentInstruction + "\n\n" + osSubAgentPlatformNote()
	return newDelegatedAgentTool(runCtx, chatModel, innerCtx, cfg)
}
```

> 注意：`cfg := osAgentConfig` 复制结构体副本（内含字符串切片，只读拼接 instruction，不修改包级 `osAgentConfig` 与 `osSubAgentToolNames` 共享切片），不产生别名写副作用。

### 3.2 `internal/agent/tools/ls_dir.go` —— 首行输出当前相对路径

**做法**：在 `InvokableRun` 构造输出首行 `当前目录: <相对路径>`，复用 `relDisplayPath(fullPath)`；根目录显示 `当前目录: .`（与 filepath.Rel 语义一致，明确表示工作区根）。

**改动点在构造循环前**，把首行写入 `strings.Builder` 并把其 rune 数计入 `cumRunes`，与既有「提前停止」预算叠加，保证输出有界：

```go
// 首行输出当前所在目录的相对路径，提供路径锚点（满足模型"知道自己在哪"
// 的诉求，替代 pwd）：相对工作区根展示，根目录显示 "."。
var b strings.Builder
cumRunes := 0
if rel := b2.relDisplayPath(fullPath); rel != "" {
	header := "当前目录: " + rel
	b.WriteString(header)
	cumRunes = len([]rune(header))
}
for _, d := range entries {
	if cumRunes > 0 && b.Len() > 0 {
		b.WriteString("\n")
	}
	// ... 原逻辑补：写入条目时同步累计 cumRunes（唯一条目间分隔符与行内容）
}
```

> 说明：现有循环 `if b.Len() > 0 { b.WriteString("\n") }` 依赖 `b.Len()` 判断行前分隔，会因首行先写入而仍然正确加换行；只需把写条目前补 `cumRunes += 分隔符与行 rune 数`，保证预算计数含首行与所有条目。确保「目录内容过多提前停止」分支（[ls\_dir.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/ls_dir.go#L142-L144)）返回时首行仍在输出中。

### 3.3 测试更新（跟进验证）

* `fs_tools_test.go` 的 `TestLsDir`：在既有子用例基础上，追加验证首行含 `当前目录: .`（根）与 `当前目录: sub`（path=sub）。保持原有 `strings.Contains` 断言不变。

* 可选（若期望平台片段可测）：在 `subagent_test.go` 增加「平台拼接后 Instruction 非空且含平台标记」的轻量断言（构造 `&openai.ChatModel{}` 后读 `oa.cfg.instruction` 校验 `contains`）。若实现上 `cfg` 存于 `delegatedAgentTool` 字段（已有 `cfg subAgentConfig`），可直接访问断言，无需 mock 模型。

> 观点：`delegatedAgentTool.cfg` 是已存在字段（[subagent.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/subagent.go#L69-L74)），测试可直接断言 `oa.cfg.instruction` 的平台标记，最小代价获得回归保护。

***

## 四、Assumptions & Decisions

* **新增 pwd 工具**：不做。理由：文件工具无 cd 状态，pwd 语义错位；内层每轮付全量工具描述 token，加低价值工具不划算；ls\_dir 首行路径上下文已满足「模型知道自己在哪」的诉求。

* **通用机制不动**：平台注入与 ls\_dir 改动均在实例文件/工具文件内闭环，`subagent.go` 通用工厂不改。

* **平台来源**：`runtime.GOOS`（标准库），不额外扫描命令可用性（方案 D 不做，太重）。

* **不改 run\_command 核心**：LookPath 报错平台化（前轮方案 C）本次范围外，仅通过提示词先规避无谓报错；若后续仍频繁，再单独评估方案 C。

* **`cfg`** **副本语义**：`cfg := osAgentConfig` 为结构体拷贝，仅覆盖 `instruction` 字段，不影响包级变量与白名单切片；符合项目「不改通用配置」约定。

* **显示根目录为** **`.`**：与 `filepath.Rel` 惯例一致，简洁无歧义。

***

## 五、Verification

1. `gofmt -l` 无输出（新增代码格式合规）。
2. `go build ./...` 通过（含 `internal/agent/...` 与 `internal/agent/tools/...`）。
3. `go vet ./...` 无告警。
4. `go test ./internal/agent/...` 通过（含 `TestLsDir` 新增断言与 `subagent_test.go` 平台片段断言）。
5. 前端资源无需 `npm run build`（本次纯后端逻辑改动）；桌面生效需 `wails build` 重新生成二进制，实施完成后提示用户。

