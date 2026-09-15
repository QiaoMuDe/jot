# 合并且读工具：将 read\_note\_section 并入 manage\_note 的 view

## Summary

将独立的 `read_note_section` 工具合并进 `manage_note` 的 `view` 动作，由「view 截断 + 独立翻页工具」改为「view 自带 offset/length 分段读取」。

* `view` 新增可选参数 `offset`（缺省 0 = 现有第一段行为）与 `length`（缺省取 `notePreviewThreshold`，上限 100000）。

* 删除独立工具 `read_note_section.go` 及其注册、展示文案与文档引用。

* view 的截断/续读提示改为「继续调用 view，offset=当前结尾」。

* 用户已禁用的 `read_note_section` 配置：**静默处理**，删除工具时顺带把该名字从持久化禁用列表 `ai_agent_tools_disabled` 中清除。

* view 保留「超阈值截断」路径：`offset=0` 时仍给出「已截断、可续读」提示，行为完全向后兼容。

## Current State Analysis

* 两个工具高度耦合，本质是「读长笔记」一件事：

  * [manage\_note.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/manage_note.go) `viewNote`（L528-563）：读全文，超阈值 `TruncateRunes` 截断，提示模型调 `read_note_section`（L559-560）。

  * [read\_note\_section.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/read_note_section.go)：按 `id/offset/length` 读后续分段，`length` 缺省取 `notePreviewThreshold`，`line_numbers=true` 时从 offset 推算全局起始行号。

* 共享基础设施：`GetNoteContent`、`notePreviewThreshold`（manage\_note.go L567）、`numberLines`、`splitNoteLines`、`TruncateRunes`（context.go L231）。

* `maxSectionLen = 100000` 常量定义在 read\_note\_section.go L32，但被 [read\_url.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/read_url.go) L141-142 **同包复用**，删除文件前必须先迁移该常量。

* 工具暴露点：

  * [registry.go L197](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go#L197)：装配注册

  * [meta.go L28](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/meta.go#L28)：前端工具开关展示（`BuiltinTools`）

  * doc.go / AGENTS.md / internal/agent/doc.go / playground landing1 badge：文档引用

* 禁用名单持久化：设置键 `ai_agent_tools_disabled`（JSON 数组字符串，db.go L219 默认空）。读取点两处：

  * [app.go L2303-2310](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2303-L2310)：Agent 请求装配（CallAIAgentStream）

  * [app.go L2708-2713](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2708-L2713)：`GetAgentTools` 展示

* 测试：`manage_note_test.go` 覆盖 `whitespaceFold / findNormalized / replaceAllFragments / splitNoteLines / replaceLines / numberLines / indexNth / editNote 校验 / extractLastLineNum / lineEditPreview / findMostSimilar / runeOverlap / buildNotFoundHint`。无针对 `viewNote` 或 `read_note_section` 的测试，删除后者零测试影响。

* 前端工具列表源自后端 `BuiltinTools`/`GetAgentTools`，删除 meta 项后前端自动少一个开关，**无需改前端 JS**。

## Proposed Changes

### 1. manage\_note.go — view 支持 offset/length

`viewNote` 签名从 `viewNote(ids []float64, lineNumbers bool)` 改为 `viewNote(ids []float64, lineNumbers bool, offset, length int)`。

新增逻辑（对齐原 read\_note\_section 语义）：

* 读取全文后 `runes := []rune(content)`，`total := len(runes)`。

* 校验：`offset < 0` → 报错；`offset >= total` 且 `total > 0` → 报错「offset 超出内容范围（共 N 字符，已全部读取完毕）」（**空笔记** **`total==0`** **时** **`offset`** **必为 0，返回空内容，不得报错**，保持现有空笔记可读）。

* `length <= 0` → 取 `notePreviewThreshold`；超过上限 → 截为 `maxSectionLen`。

* `end := offset + length`，`end > total` 则 `end = total`。`truncated := end < total`。

* 待显示段 `segment := string(runes[offset:end])`。

  * `line_numbers=true`：`startLine := 换行数(runes[:offset]) + 1`；显示 `numberLines(segment, startLine)`。

  * `line_numbers=false`：直接 `segment`。

* 截断提示（仅 `truncated` 时追加）改为：
  `"（内容共 N 字符 / M 行，已显示前 {end} 字符。如需继续阅读，可继续调用本工具 view：offset={end}（length 保持缺省即可）；如需按行编辑，可让 view 带 line_numbers=true 获取全局行号。"`
  其中总行数 `M = len(splitNoteLines(content))`。

* 返回 `"笔记 #%d 内容：\n" + display`。

> 行为兼容性：`callView(ids, lineNumbers, 0, notePreviewThreshold)` 与现 `viewNote` 完全一致（读首段、超阈值给续读指示）。

**参数 JSonSchema（view 相关）**：在 `Info()` 的 `ParamsOneOf` 增加：

* `offset`（Number，可选）：Desc「起始字符位置，缺省 0（= 从开头读取首段）；分段续读时传上一段返回的结尾位置」。若 `offset>0` 则不再按阈值截断，而是从该位置读取。

* `length`（Number，可选）：Desc「本次读取字符数，缺省取 ai\_large\_file\_preview\_threshold（缺省 10000），上限 100000；超出自动截到内容末尾」。

`Info` 的 `view` 动作 Desc 文案：把出现 `read_note_section` 的措辞改为「view 支持 offset/length 参数分段续读」。

**调用点**：`InvokableRun`（L359-360）把 `args.Offset`、`args.Length` 解析出来并传入 `viewNote`。入参加入 decode struct（新增 `Offset float64`、`Length float64`）。

**文件内注释清理**：manage\_note.go 内多处注释提到 `read_note_section`（L15-18、L26、L169、L214、L224、L524-525、L566、L614、L1063、L1105），逐一改为「view 分段续读」措辞；`notePreviewThreshold` 注释「view / read\_note\_section 共用」改「view 共用」（read\_url 用 `maxSectionLen` 而非它）。

### 2. 迁移 maxSectionLen（必须在删除前）

把常量 `const maxSectionLen = 100000` 从 read\_note\_section.go 移到 manage\_note.go（包内顶层），保证 read\_url.go 仍可引用。目标位置建议紧邻 `notePreviewThreshold`。

### 3. 删除 read\_note\_section.go

删除文件 [internal/agent/tools/read\_note\_section.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/read_note_section.go) 及 `NewReadNoteSection`。

### 4. 注册/展示清理

* [registry.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go)：删除 L197 的 `{"read_note_section", ...}` 及 `NewReadNoteSection` 引用。

* [meta.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/meta.go)：删除 L28 的 `read_note_section` 条目；同步更新 L3 头注释里的工具计数（含 doc.go 的「共 15 个」→ 14）。

* [doc.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/doc.go)：工具清单与构造器列表删除 `read_note_section` / `NewReadNoteSection`。

* [internal/agent/doc.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/doc.go)：只读工具清单删除 `read_note_section`。

* [AGENTS.md](file:///d:/资源池/下水道/Dev/本地项目/jot/AGENTS.md)：更新 L25 工具计数与 L502/L504 提及。

* playground/[landing1](file:///d:/资源池/下水道/Dev/本地项目/jot/playground/landing1/index.html)/index.html：删除 `read_note_section` 徽标（L353）。

* 检查 [internal/agent/TOOLS.md](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/TOOLS.md) 是否存在 `read_note_section` 具体引用，有则更新。

### 5. 禁用名单静默清理

新增包级常量 `deprecatedToolReadNoteSection = "read_note_section"` 与分析用小 helper，例如：

```go
// cleanDeprecatedTools 从禁用集合中剔除已下线的工具名，返回清理后的切片与是否发生了剔除。
func cleanDeprecatedTools(disabled []string) []string {
    return slices.DeleteFunc(disabled, func(n string) bool { return n == deprecatedToolReadNoteSection })
}
```

* **GetAgentTools（app.go L2708-2713）**：解析 `ai_agent_tools_disabled` 后调用 helper 剔除；**若发生剔除，则把清洗后的 JSON 数组回写** `a.settingService.Set("ai_agent_tools_disabled", ...)`，真正从持久化清除该残留项（幂等，仅首次触发写）。

* **CallAIAgentStream（app.go L2303-2310）**：解析后同样调用 helper 剔除（仅内存过滤，不回写，避免每次请求写库）。剔除该残留项属于安全兜底——工具已删除，`buildTools` 不会命中；此步仅保证无冗余。

> 注：AlwaysOn 豁免（L2314-2320）在剔除后执行即可，`read_note_section` 非 AlwaysOn，顺序无影响。

## Assumptions & Decisions

* 采用「**offset 并入 view**」方式，不新增独立 action。

* **静默忽略**旧禁用项：删除工具时从 `ai_agent_tools_disabled` 顺带清除，不在 UI 提示、不迁移、不做兼容别名。

* view 保留「超阈值截断」路径：`offset=0` 即现有行为（读首段 + 续读提示），向后兼容。

* 空笔记（`total==0`）不受 offset 校验影响，仍可查看空内容。

* 前端无需改 JS（开关列表由后端 `BuiltinTools` 驱动），仅可选同步 playground marketing 徽标。

## Verification

1. `go build ./...`
2. `go vet ./internal/agent/...`
3. `go test ./internal/agent/...`（现有 helper 测试全绿，确认无回归）
4. 确认全仓无 `read_note_section` / `NewReadNoteSection` 残留：`在 app.go、internal/ 下搜索 read_note_section`（playground 徽标已删）。
5. 行为抽查（手动/临时日志）：

   * `view, offset=0, length=缺省` 普通笔记 → 与旧版一致。

   * 长笔记 `view, offset=0` → 返回截断 + 「offset=当前结尾」续读提示。

   * `view, offset=prev_end` → 返回后续分段，行号（line\_numbers）与首段全局连续。

   * `view, offset>=total` → 报「已全部读取完毕」。

   * 设置键 `ai_agent_tools_disabled` 预先写入 `["read_note_section"]`：打开 AI 设置面板后，DB 中该键被清为 `[]`（GetAgentTools 回写），且 Agent 请求正常。
6. 前端重建（如需看到开关列表更新）：`cd frontend && npm run build`，再 `wails build`（按项目既有约定，CSS/HTML/前端变更后需重建才生效；本次前端无逻辑改动，仅徽标页，非强制）。

