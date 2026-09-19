# 新增 run\_python 工具设计方案

> 状态: 待评审 | 日期: 2026-09-19 | 类型: Agent 工具新增

## 一、概述

新增 `run_python` 工具：在 `~/.jot/workspace` 工作区内执行 Python **代码字符串**或**脚本文件**，工具内部自动探测可用的解释器（`py` / `python` / `python3`，取第一个冒烟验证通过者），省去模型反复试错解释器的步骤。复用现有审批门控（恒 critical=true）、超时与有界输出机制。

**落点：os\_agent 内层白名单（12 → 13 个工具），不注册父层 buildTools，不进 meta.go。** 与现有 11 个文件工具 + transfer\_file 完全一致——它们都只存在于 os\_agent 内层。

## 二、现状分析（已探索确认）

| 项      | 现状                                                                                               | 关键文件                                                                                         |
| ------ | ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------- |
| 命令执行   | 仅 `run_command`，裸命令不支持 shell 语法；`python`/`python3`/`py` 在 highRiskTokens 黑名单 → 执行即 critical 强制审批 | [run\_command.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/run_command.go)            |
| 执行链路   | 父层模型只能通过 `os_agent` 委托工具执行命令；内层子 Agent 白名单 12 工具，复用同一会话 ChatModel 客户端                            | [subagent\_os.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/subagent_os.go#L41-L45)          |
| 白名单装配  | 子 Agent 白名单按工具名从 `toolConstructors` 注册表取构造器                                                      | [subagent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/subagent.go#L48-L63)                 |
| 文件工具基类 | `fsToolBase` 提供 `wsRoot()` / `resolvePath()` / `requestApproval()` 沙箱与审批基础                       | [fs\_base.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/fs_base.go)                    |
| 测试模式   | `run_command_test.go` 已有审批测试模式（fakeApprover：批准/拒绝/Approver 缺失 fail-fast）可参考                      | [run\_command\_test.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/run_command_test.go) |
| 前端工具列表 | `BuiltinTools()` 仅展示父层工具；内层 12 工具（read\_file 等）均不在此列，只有 `os_agent` 一条代表                          | [meta.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/meta.go#L33-L34)                   |

**关键约束确认**：内层工具不参与父层 `ai_agent_tools_disabled` 禁用过滤，因此 `run_python` 不应加入 `meta.go`（避免前端出现无法实际控制的条目）——与现有 12 个内层工具的处理一致。

## 三、方案设计

### 3.1 工具定义

* **名称**：`run_python`

* **参数**（JSON Schema）：

  * `code`：Python 代码字符串（与 `path` 二选一），上限 `maxToolLongText`（100000）

  * `path`：工作区内 `.py` 脚本路径（与 `code` 二选一），经 `resolvePath` 沙箱校验

  * `args`：传给脚本的参数数组

  * `cwd`：命令工作目录（复用 run\_command 语义，缺省 workspace 根，越界拒绝）

* **审批**：`requestApproval(ctx, "run_python", summary, true)` —— **critical 恒为 true**（Python 可执行任意代码，如 `os.remove`/`shutil.rmtree`/`urllib` 下载执行，token 黑名单对其无效；与现有解释器类命令 `python` 命中黑名单的审批语义一致）

* **超时/输出**：复用 `runCommandTimeout`（30s）+ `limitedBuffer`（256KB 有界）+ `MaxResultLen` 截断

### 3.2 解释器探测 + 进程级缓存（核心新增逻辑）

```go
// detectPython 候选解释器探测（真实探测）：每个候选 LookPath 后执行 --version 冒烟验证
var detectPython = func() (string, error) {
    candidates := []string{"python", "python3", "py"}  // Windows 与 UNIX 顺序差异在实现内按 runtime.GOOS 调整
    for _, name := range candidates {
        if path, err := exec.LookPath(name); err == nil && smokeTest(path) {
            return path, nil
        }
    }
    return "", errors.New("环境中未找到可用的 Python 解释器（已尝试 py/python/python3），请先安装 Python")
}

// 进程级缓存：只缓存成功结果；失败不缓存（用户安装解释器后下次调用自动重试）
var (
    pythonMu        sync.Mutex
    pythonResolved string
)

// resolvePython 带缓存的解释器探测：首次调用触发真实探测并缓存成功结果，后续直接命中
func resolvePython() (string, error) {
    pythonMu.Lock()
    defer pythonMu.Unlock()
    if pythonResolved != "" {
        return pythonResolved, nil
    }
    p, err := detectPython()
    if err != nil {
        return "", err // 失败不缓存
    }
    pythonResolved = p
    return p, nil
}
```

* **候选顺序**：Windows 优先 `py`（官方 launcher，最可靠）→ `python` → `python3`；UNIX 优先 `python3` → `python` → `py`

* **冒烟验证**：对每个候选执行 `<candidate> --version`（短超时约 5s），确认"能命中"≠"能运行"（规避 Windows Microsoft Store 的 python stub 弹商店问题）

* **缓存语义**：进程级（应用重启后重新探测一次，成本极低）；只缓存成功路径，失败不缓存——用户安装/修复 Python 后，下一次工具调用自动重新探测，无需重启

* **可测性**：`detectPython` 为包级变量可整体替换（避免依赖本机 Python）；测试中直接置 `pythonResolved = ""` 即可重置缓存态（同包访问私有变量）

### 3.3 执行流程

```
InvokableRun
  ├─ ctx.Err() 检查（用户取消）
  ├─ 解析参数；校验 code/path 二选一（都缺/都给 → 报错）；validateTextLen
  ├─ cwd 边界校验（复用 run_command 逻辑：缺省 wsRoot，提供则 resolvePath）
  ├─ 审批检查点：requestApproval(ctx, "run_python", 摘要, true)
  ├─ 解释器探测：resolvePython()（带缓存，首次真实探测、后续命中缓存），失败返回友好错误
  ├─ code 模式 → os.CreateTemp 写临时 .py 文件（defer 清理）；path 模式 → resolvePath + os.Stat 确认
  ├─ exec.CommandContext(runCtx, interpreter, scriptPath, args...)（cmd.Dir = cwd，统一临时文件执行，规避 -c 的引号/编码问题）
  └─ 输出规整：空输出 → "执行成功，无输出"；超长 → TruncateRunes + "[输出过长已截断]"
```

### 3.4 摘要与动作文案

* `ActionText`（实现 `ActionTextProvider`）：`执行 Python：` + 截断摘要（code 模式取首行前 30 字；path 模式取路径），复用 `TruncateRunes`

* 审批摘要：`执行 Python：<同上摘要>`

* `Info().Desc`：说明 code/path 二选一、自动探测解释器、工作区边界、每次执行需审批确认

## 四、变更清单（Proposed Changes）

| 文件                                         | 改动        | 说明                                                                                                                            |
| ------------------------------------------ | --------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `internal/agent/tools/run_python.go`       | **新建**    | `runPythonTool`（嵌入 `fsToolBase`）+ `resolvePython` 探测 + `NewRunPython(ctx)` 构造器；实现 `tool.InvokableTool` + `ActionTextProvider` |
| `internal/agent/tools/run_python_test.go`  | **新建**    | 见验证章节                                                                                                                         |
| `internal/agent/subagent.go`               | 修改        | `toolConstructors` 追加 `"run_python": tools.NewRunPython`                                                                      |
| `internal/agent/subagent_os.go`            | 修改        | `osSubAgentToolNames` 追加 `"run_python"`（12→13）；同步更新文件头注释与提示词（提示词加一句"Python 脚本/代码请用 run\_python 工具"引导内层模型选对工具）                 |
| `internal/agent/tools/meta.go`             | **不改**    | run\_python 为内层工具，不参与父层禁用，不进 BuiltinTools（与现有 12 工具一致）                                                                        |
| `internal/agent/SUBAGENTS.md` / `TOOLS.md` | 同步（实施时检查） | 若文档列了 os\_agent 白名单 12 工具，更新为 13；工具开发文档补 run\_python 条目                                                                       |
| `AGENTS.md`                                | 临时记忆更新    | 按维护规范三步法追加临时记忆条目                                                                                                              |

## 五、假设与决策

1. **放 os\_agent 内层而非父层**：执行 Python 的诉求天然在 os\_agent 域内；父层每轮工具描述 token 不增加；`run_python` 语义上属于"文件工具族 + 命令"范畴，进白名单不违反子 Agent 安全约束（run\_command 本就能跑 python，未扩权）
2. **critical 恒 true**：Python 代码在字符串层面无法用 highRiskTokens 审计，审批语义必须与解释器类命令一致；不因"封装"降级
3. **code 模式用临时文件而非** **`-c`**：规避 Windows 上多行代码/引号/编码的转义地狱（本项目核心痛点）；临时文件放系统临时目录（`os.CreateTemp`），不污染 workspace，执行后删除
4. **探测冒烟验证**：`LookPath` 命中不代表能运行（Windows Store python stub），必须 `--version` 实测
5. **不进 meta.go**：内层工具不参与父层禁用开关，加入会造成前端出现不可控条目；与现有 12 个内层工具处理一致
6. **不引入 eino-ext 沙箱组件**：已查证 eino-ext 的代码执行组件（agentkit 远程沙箱需火山引擎云凭证、local/DeepAgents 为通用 Shell 无探测逻辑、且接口体系与 jot 自研引擎不兼容），自研方案成本最低且符合本地优先定位
7. **缓存只缓存成功结果**：进程级缓存（重启后重新探测一次，成本极低）；探测失败不缓存，用户安装/修复 Python 后下一次调用自动重新探测，无需重启应用

## 六、验证

1. **单元测试**（`run_python_test.go`，复用 run\_command\_test.go 的 fakeApprover 模式）：

   * 参数校验：code/path 都缺、都提供、code 超长 → 报错

   * `path` 逃逸：`../` 等越界路径 → resolvePath 拒绝

   * 审批路径：批准（断言 critical=true）→ 继续执行；拒绝 → 返回错误文本；`ctx` nil / Approver 缺失 → fail-fast

   * 探测失败：注入 `detectPython` 返回错误 → 友好中文错误

   * **缓存机制**：① 注入带调用计数器的 fake `detectPython`，连续两次调用 `resolvePython()` → 断言真实探测只执行一次（第二次命中缓存）；② 首次探测失败 → 断言不缓存（`pythonResolved` 为空），注入成功后再次调用 → 重新探测成功；③ 重置：测试间置 `pythonResolved = ""` 隔离缓存态

   * 真实执行：注入 fake 解释器（或本机存在 python 时执行 `print("ok")`）→ 断言输出；本机无 python 时 `t.Skip`
2. **构建回归**：`go build ./...` + `go vet ./...` + `go test ./internal/...` 全绿
3. **手工验证**：`wails dev` 中让模型执行 `print("hello")` 与工作区脚本，确认审批弹窗、输出回填、工具状态条动作文案正常；确认 `py`/`python`/`python3` 缺失时给出友好错误而非多轮试错

