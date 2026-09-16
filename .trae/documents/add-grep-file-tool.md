# 新增 grep\_file 工具（按内容搜索工作目录内文件）

## Context

文件工具家族（`read_file` / `write_file` / `edit_file` / `ls_dir` / `glob` / `run_command`）目前缺少"**按内容过滤定位**"的工具：模型要回答"哪个文件第几行有 X"，只能 `read_file` 全读（大文件需分页续读多轮、费 token），或 `run_command` 跑系统 grep（每次触发审批打断流程，且 Windows 无原生 grep）。新增 `grep_file` 补上内容定位环节，与既有工具形成闭环：

```
grep_file 定位（哪个文件第几行有 X）
  → read_file 按行号读原文（确认上下文）
  → edit_file 行级替换（精准修改）
```

用户已确认三个设计决策：**增加该工具**；匹配语义=**字符串 + 正则双模式**；搜索范围=**单文件 + 目录递归**。

## 设计

### 工具定位

* 名称 `grep_file`，文件 `grep_file.go`，纯只读（不触发审批，同 read\_file/glob）。

* 与 `glob` 的边界在 Desc 拉开：`glob` 按**路径**匹配找文件，`grep_file` 按**内容**匹配找行；先 `glob` 定位文件、`grep_file` 定位行号、`read_file` 读原文、`edit_file` 修改。

* 与 `edit_file` 的 find 语义区别：find 有空白归一化兜底（为了"改错也能命中"）；`grep_file` **不做归一化**（定位工具，归一化会让输出行号与原文失真，误导后续读取/修改）。

### 参数（6 个）

| 参数                 | 必填     | 类型     | 说明                                                                    |
| ------------------ | ------ | ------ | --------------------------------------------------------------------- |
| `path`             | ✓      | string | 文件路径 → 单文件搜索；目录路径 → 递归搜该目录（均相对 `~/.jot/workspace`，≤500 rune）          |
| `pattern`          | ✓      | string | 搜索内容（≤500 rune，即 `maxToolShortText`）                                  |
| `regex`            | <br /> | bool   | 缺省 false：纯字符串字面匹配；true：按 Go 正则解释                                      |
| `case_insensitive` | <br /> | bool   | 缺省 false：忽略大小写（正则模式编译时加 `(?i)` 前缀）                                    |
| `file_pattern`     | <br /> | string | 仅目录模式有效：按文件名通配过滤（用 `path.Match` 匹配文件基名，如 `*.go` 匹配所有层级 .go；≤500 rune） |
| `context_lines`    | <br /> | number | 缺省 0：匹配行前后附带 N 行上下文；必须为 0-50 的整数                                      |

### 匹配与输出

* 纯字符串：`strings.Contains`；忽略大小写时 pattern 与每行先 `ToLower`（pattern 只 lower 一次）再 Contains。

* 正则：`regexp.Compile`（失败报错带 Go 正则语法位置）→ `re.MatchString(line)`；忽略大小写时编译 `(?i)`+pattern。

* 输出格式对齐 `read_file` 的「行 N: 」前缀：

  * 单文件：`行 N: 内容`

  * 目录模式：`相对路径:行 N: 内容`（相对工作区根，模型可直接用该路径调 read\_file/edit\_file）

* 超长行截断：单行内容超 `maxGrepLineRunes = 200` 时截断并加 `…`。

* 无匹配：单文件返回"未找到匹配行"；目录模式返回"未找到匹配项"。

* 输出有界：累计 rune 上限 `maxGrepRunes = 20000`、提前中断阈值 `grepHeadroomRunes = 2000`（仿 glob），超限尾部加 `[结果超长，已提前停止]` 并提前中断遍历（`errGrepTooLarge` 哨兵错误）。

### context\_lines 流式实现要点

* 前置上下文：环形缓冲保存最近 N 行（行号 + 内容）。

* 后置上下文：匹配后继续输出，直到行号超出 `匹配行号 + N`。

* 用 `lastOutputLine` 记录已输出的最大行号，避免前置/后置上下文与后续匹配重复输出。

* `context_lines=0`（默认）走无上下文简化路径。

### 边界与安全

* 路径经 `fsToolBase.resolvePath` 边界校验（杜绝 `../` 逃逸与越界绝对路径）。

* 目录递归用 `filepath.WalkDir`（不跟随符号链接目录）；仅处理普通文件（`d.Type().IsRegular()`，符号链接文件天然跳过，防逃逸）；子项不可访问时跳过不中断（仿 glob）。

* 二进制文件跳过：读前 `grepBinaryProbeBytes = 8000` 字节含 NUL 判定；目录模式静默跳过，单文件模式返回"文件为二进制，已跳过搜索"。

* 逐行流式读取（`bufio.Reader.ReadString('\n')`，不载入全文，超长行不会撑爆内存）；遍历中检查 `ctx.Err()` 支持用户取消。

* 参数校验：path/pattern 非空、长度校验、`context_lines` 负数或非整数或 >50 报错、`file_pattern` 复用 `validateGlobPattern`（拒绝绝对路径与 `..`）、`regex=true` 时正则编译失败报错。

### 动作文案

实现 `ActionTextProvider`：`ActionText` 返回 `"搜索文件：" + TruncateRunes(pattern, 30)`（解析失败回退"搜索文件"，仿 glob 写法）。

## 实现步骤

1. **新建** **`internal/agent/tools/grep_file.go`**：`grepFileTool` 结构体（嵌入 `fsToolBase`）+ 编译期断言 + `ActionText` + `Info()` + `InvokableRun` + `NewGrepFile(ctx *Context)`；核心搜索函数拆为 `searchFile`（单文件：二进制探测 → 流式逐行匹配 → 上下文输出）与目录遍历回调（仿 glob 的 WalkDir 有界模式）；常量 `maxGrepRunes` / `grepHeadroomRunes` / `maxGrepLineRunes` / `grepBinaryProbeBytes` / `errGrepTooLarge`。
2. **注册**：`internal/agent/registry.go` 的 `buildTools` 在 `glob` 之后、`run_command` 之前追加 `tools.WrapWithError("grep_file", tools.NewGrepFile(p.ctx), p.ctx)`（文件工具家族相邻）。
3. **清单**：`internal/agent/tools/meta.go` 的 `BuiltinTools()` 追加 `{Name: "grep_file", Label: "在工作目录内按内容搜索匹配行（类似 grep）"}`（保持顺序稳定，置于 glob 后）；`internal/agent/tools/doc.go` 工具清单与构造器补 `NewGrepFile`。
4. **测试**：新建 `internal/agent/tools/grep_file_test.go`，复用同包既有 helper（`writeTestFile` / `readTestFile` / `jsonQuote`）。覆盖：

   * 单文件匹配：行号正确、`行 N: `  格式；

   * `case_insensitive`：大小写不敏感命中；

   * `regex`：正则命中 + 非法正则报错（含"正则编译失败"）；

   * 目录递归：多文件输出带相对路径前缀、`file_pattern=*.go` 过滤、子目录递归；

   * `context_lines`：前置/后置上下文输出、默认 0 无上下文；

   * 边界：无匹配文案、二进制文件跳过、超长行截断、`../` 逃逸拒绝、路径不存在报错、`context_lines` 负数/非整数/>50 报错、空文件。
5. **验证**（不更新 TOOLS.md——指南明确不维护工具清单；不涉及 EVENTS.md——只读工具无事件变更）：

   ```bash
   go build ./...
   go vet ./internal/agent/...
   go test ./internal/agent/...
   ```

## 复用的既有实现

* `fsToolBase.resolvePath` / `wsRoot`（[fs\_base.go](internal/agent/tools/fs_base.go)）：路径边界校验。

* `validateTextLen` / `TruncateRunes` / `maxToolShortText`（[context.go](internal/agent/tools/context.go)）。

* `toSlash` / `validateGlobPattern`（[glob.go](internal/agent/tools/glob.go)）：file\_pattern 校验与路径归一化。

* `writeTestFile` / `readTestFile`（[edit\_file\_test.go](internal/agent/tools/edit_file_test.go)）、`jsonQuote`（[fs\_tools\_test.go](internal/agent/tools/fs_tools_test.go)）：测试 helper（同包直接可用）。

* 参考 `read_file.go` 的「行 N: 」输出格式与 `glob.go` 的有界遍历模式。

