# 新增 edit\_file 工具（精准编辑工作目录文件）

## Summary

为 AI 助手工作目录文件工具家族新增 `edit_file`：在工作目录内对**已存在**文件做**精准编辑**（片段替换 + 行级替换双模式），而非整文件覆盖。参数契约复用管理笔记工具 `manage_note` 的 edit 语义（`find/replace/count/replace_all` + `line_start/line_end`），复用其成熟匹配逻辑；为行级模式配套给 `read_file` 增加可选的行号输出能力。经 `fsToolBase` 走统一路径边界校验与审批钩子。

## Current State Analysis

* 现有文件工具家族：`read_file` / `write_file` / `ls_dir` / `glob` / `run_command`，均嵌入 `fsToolBase`（[fs\_base.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/fs_base.go)），共享 `resolvePath` 边界校验与 `requestApproval` 审批钩子

* `write_file` 的审批模式：**覆盖已存在文件**时 `requestApproval(critical=false)`（[write\_file.go#L98-L106](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/write_file.go#L98-L106)），`critical=false` 语义 = 仅 confirm\_every 模式确认，review/auto 放行

* **`manage_note.go`** **已有完整的片段替换实现可直接复用**（同包）：`indexNth`（第 N 次精确匹配）、`whitespaceFold` + `findNormalized`（空白归一化兜底）、`replaceAllFragments`（全量替换）、`splitNoteLines` / `replaceLines` / `lineEditPreview`（行级替换）、`buildNotFoundHint` / `findMostSimilar`（未命中友好提示）

* **预留物已就位**：`context.go` 已有 `maxToolFindLen = 2000`（注释"edit 片段替换 find 原文片段上限"）；前端审批面板 [ai-chat.js#L5654](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5654) 已有 `edit_file: '编辑文件'` 映射；`.trae/specs/.../spec.md` 规划了 edit\_file

* **差距**：`read_file` 当前只输出纯文本、不携带行号（分页是字符偏移），行级模式缺少行号来源，需给 `read_file` 增加可选 `line_numbers` 参数

* 注册链路（TOOLS.md §2）：新文件 → `registry.go` `buildTools` 追加注册 → `tools/doc.go` 权威清单更新 → `meta.go` `BuiltinTools` 展示文案；本工具无新服务依赖，不动 `agent/doc.go` 的 Deps

## Proposed Changes

### 1. 新增 `internal/agent/tools/edit_file.go`

**结构体**：`editFileTool` 嵌入 `fsToolBase`，实现 `tool.InvokableTool` + `ActionTextProvider`。

**参数 Schema**（`Info()`）：

| 参数            | 类型      | 必填 | 说明                                        |
| ------------- | ------- | -- | ----------------------------------------- |
| `path`        | string  | 是  | 目标文件路径，相对/绝对均限工作目录内                       |
| `find`        | string  | 条件 | 片段模式：要替换的原文片段（≤`maxToolFindLen` 2000）     |
| `replace`     | string  | 条件 | 替换后的新文本（≤`maxToolLongText` 20000，空串=删除片段） |
| `count`       | number  | 否  | 片段第几次出现，缺省 1；与 `replace_all` 互斥           |
| `replace_all` | boolean | 否  | 替换全部出现；与 `count>1` 互斥                     |
| `line_start`  | number  | 条件 | 行级模式起始行号（1-based）                         |
| `line_end`    | number  | 否  | 行级模式结束行号，缺省=line\_start                   |

**InvokableRun 流程**：

1. `ctx.Err()` 取消检查
2. 解析参数；`path` 非空 + ≤`maxToolShortText`；`find` ≤`maxToolFindLen`、`replace` ≤`maxToolLongText`
3. 模式互斥校验：`find` 非空 && `line_start>0` → 报错；两者皆空 → 报错（复用 manage\_note 的同款文案）
4. 片段专属校验：`replace_all && count>1` → 报错（互斥）
5. `resolvePath` 路径边界校验
6. `os.Lstat` 检查目标存在：**不存在 → 报错**"文件不存在，请先用 write\_file 创建"（编辑不存在文件是参数错误，无需审批）
7. **审批检查点**：`requestApproval(ctx, "edit_file", "编辑文件："+path, false)`——每次编辑都修改已存在文件内容，对齐 `write_file` 覆盖语义（critical=false，仅 confirm\_every 确认；review/auto 放行）。在读取/写回之前发起
8. **大文件防护**：`Lstat.Size()` > `maxEditFileSize`（新常量 = 1MB）→ 报错"文件过大，建议用 run\_command 编写脚本处理"（edit 必须整读整写，内存必须有界——呼应审查报告 P1-2 精神）
9. 读取文件内容 `os.ReadFile`（防护后内存有界）
10. 分派：

    * **行级模式**（`line_start>0`）：复用 `splitNoteLines` / `replaceLines`；`start > total` 时为末尾追加语义（复用 manage\_note 逻辑）；反馈含行数变化 + `lineEditPreview` 上下文预览

    * **片段模式**：精确匹配 `indexNth`（count 缺省 1）→ 未命中走 `findNormalized` 空白归一化兜底 → 均未命中返回 `buildNotFoundHint` 同款提示（文件版文案，指引"请用 read\_file 获取精确原文后重试"）；`replace_all` 用 `replaceAllFragments`
11. 反馈文本：片段模式含匹配方式（精确/空白归一化）+ 第 N 处 + 旧/新片段摘要（`truncateSnippet`，80 字符）；行级模式含行数变化 + 上下文预览
12. `os.WriteFile(fullPath, []byte(newContent), 0o644)` 写回

**ActionText**：`"编辑文件：" + TruncateRunes(path, 30)`，解析失败回退"编辑文件"。

**构造器**：`NewEditFile(ctx *Context) tool.InvokableTool`。

### 2. 修改 `internal/agent/tools/read_file.go`：支持行号输出

* `Info()` 新增可选参数 `line_numbers`（boolean，缺省 false，"为 true 时每行输出带 \[行号] 前缀，供 edit\_file 行级替换使用"）

* 扩展内部 `readRunesBounded`：顺带统计 **offset 之前**的换行数（`linesBeforeOffset`），返回给调用方（函数签名变化，同包私有函数，调用方仅 read\_file.go，测试通过 `InvokableRun` 走，无破坏）

* `line_numbers=true` 时：起始行号 = `linesBeforeOffset + 1`，对读取到的文本按行加 `[N] `  前缀后拼接；`[文件未读完…]` 续读提示作为附加行追加在行号化内容之后、不参与编号

* 缺省行为完全不变（向后兼容）

### 3. 注册与清单（3 处）

* `internal/agent/registry.go` `buildTools` 追加一行：
  `{"edit_file", tools.WrapWithError("edit_file", tools.NewEditFile(p.ctx), p.ctx)}`
  放在 `write_file` 之后

* `internal/agent/tools/meta.go` `BuiltinTools()` 追加：
  `{Name: "edit_file", Label: "精准编辑工作目录内的文件（片段替换/行级替换）"}`

* `internal/agent/tools/doc.go` 工具清单：`read_file / write_file / edit_file / ls_dir / glob / run_command`，构造器补 `NewEditFile`

### 4. 新增 `internal/agent/tools/edit_file_test.go`

遵循 fs\_tools\_test.go 的测试模式（`workspaceRoot` 注入临时目录 + `mockApprover`）：

1. 片段精确匹配替换成功（返回含旧/新片段摘要）
2. `count=2` 替换第 2 次出现
3. `replace_all` 全部替换（含归一化等价位置）
4. 空白归一化兜底（缩进/换行差异仍命中）
5. 未找到片段：报错 + 最相似提示
6. 行级：单行替换 / 区间替换 / 末尾追加 / 删除行（replace 空）
7. 模式互斥：find+line\_start 同传报错；全空报错；replace\_all+count>1 报错
8. 参数长度：find>2000、replace>20000 报错
9. 越权路径拒绝（`../` 逃逸、绝对路径越界）
10. 文件不存在报错
11. 审批钩子生效：mockApprover 收到 `edit_file`、summary 含路径、`critical=false`；ctx 非空且 Approver 缺失时报错
12. 大文件防护：>1MB 拒绝

同时扩展 fs\_tools\_test.go（或本文件内）覆盖 read\_file `line_numbers`：

* `line_numbers=true` 输出带 `[1] `  前缀

* 与 `offset/length` 分页组合时起始行号正确（如 offset 跨过 N 行后从 N+1 起编号）

* 缺省行为不变（无前缀）

## Assumptions & Decisions

* **参数契约**：采用 `find/replace/count/replace_all`（管理笔记 edit 语义）而非 eino `old_string/new_string`，理由见前文；`maxToolFindLen` 预留注释即此方向

* **行级模式的行号来源**：由 `read_file` 新增可选 `line_numbers=true` 提供（向后兼容，缺省无前缀），模型先读后编

* **审批语义**：每次编辑（= 修改已存在文件）`critical=false` 审批，与 `write_file` 覆盖已存在文件的现有语义一致（仅 confirm\_every 确认，review/auto 放行）；与 spec.md 中"edit\_file 修改既有文件属 review 关键操作"的差异沿用现有 run\_command/write\_file 已落地的 critical 判定体系

* **大文件防护**：`maxEditFileSize = 1MB` 字节上限，超出拒绝并引导 run\_command 脚本方案

* **不做撤销/回滚**：第一版不引入备份机制，保持最小实现；破坏性由审批机制兜底

* **不新增服务依赖**：不动 `Deps`，不改 `agent/doc.go` 的 Deps 说明

## Verification

1. `go build ./...`
2. `go vet ./internal/agent/...`
3. `go test ./internal/agent/tools/`（新旧用例全绿）
4. 前端无需改动（审批面板已有 `edit_file: '编辑文件'` 映射；工具名直接展示英文，符合 TOOLS.md §8）
5. 重启应用，Agent 模式下触发：write\_file 建文件 → read\_file（line\_numbers=true）→ edit\_file 片段替换 / 行级替换 → read\_file 验证变更

