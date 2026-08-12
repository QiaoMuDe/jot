# Agent 工具防御加固计划

## Summary

全面审查 12 个 Agent 工具的防御性，按 P0（崩溃级）/ P1（稳健级）/ P2（加固级）修复所有发现的问题：panic recover、取消检查补齐、文本长度上限、参数格式校验、写操作确认引导与内网防护。

## Current State Analysis

### 已有防御（保持不动）

* `WrapWithError`（[context.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/context.go#L96-L120)）：失败回填模型 + tool\_error 事件 + 用户取消不误报

* `ctx.Err()` 取消检查（10/12 工具）、action 白名单、id 正整数、必填字段校验、枚举校验、分页边界（pageSize 1-50）、设置读取回退默认/上限、输出截断、[MaxIterations=20](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go#L41-L42)、disabled 黑名单

### 待修复缺口（本次全部修复）

| 级别 | 问题                                                         | 位置                                        |
| -- | ---------------------------------------------------------- | ----------------------------------------- |
| P0 | 全链路无 panic recover，工具内部 panic 直接 crash 整个应用                | context.go                                |
| P0 | web\_search 单源搜索 goroutine 无 recover，一源 panic 崩掉整个进程       | web\_search.go                            |
| P1 | recall\_notes / read\_note\_section 缺 ctx.Err() 取消检查       | recall\_notes.go / read\_note\_section.go |
| P1 | manage\_tag 的 action 未 TrimSpace（其余工具一致）                   | manage\_tag.go                            |
| P1 | 文本参数无长度上限（title/content/find/query/url 等）                  | 全部文本参数                                    |
| P1 | read\_note\_section 的 offset/length 未校验整数（浮点被 int() 截断）    | read\_note\_section.go                    |
| P2 | 写操作（edit/update/move）无用户确认引导                               | manage\_note.go / app.go                  |
| P2 | read\_url 允许内网/本机地址（SSRF 面）；manage\_note 的 file\_ext 无格式校验 | read\_url.go / manage\_note.go            |

## Proposed Changes

### P0-1：WrapWithError 加 panic recover（context.go）

重构 `wrappedTool.InvokableRun` 为命名返回值 + defer recover；把现有错误处理抽为私有方法，recover 与 error 走同一路径：

* `func (w *wrappedTool) InvokableRun(c context.Context, argumentsInJSON string, opts ...tool.Option) (ret string, retErr error)`

* defer 内 `recover()`：panic 时 `retErr = fmt.Errorf("工具内部异常（panic）：%v", r)`，并复用错误记录逻辑（含用户取消分支：`c.Err() != nil` 时仅返回错误文本不记 tool\_error）

* 提取 `func (w *wrappedTool) fail(c context.Context, err error) string`：日志 Warnw + tool\_error 记录 + 事件发射 + 返回回填文本

### P0-2：web\_search 单源 goroutine recover（web\_search.go L149-163）

goroutine 内加 `defer func() { if r := recover(); r != nil { resultCh <- searchResult{source: s, err: fmt.Errorf("搜索源 %s 内部异常: %v", s, r)} } }()`，panic 转错误进结果通道，保持原有合并逻辑不变。

### P1-1：recall\_notes 补 ctx.Err()（recall\_notes.go）

参数解析与 query 校验后加 `if ctx.Err() != nil { return "", ctx.Err() }`。

### P1-2：read\_note\_section 补 ctx.Err() + 整数校验（read\_note\_section.go）

* 签名 `InvokableRun(_ context.Context, ...)` → `InvokableRun(ctx context.Context, ...)`，开头加 `ctx.Err()` 检查

* offset/length 整数校验：`args.Offset != math.Trunc(args.Offset)`（length 同理），非法返回"须为整数"；`import "math"`

### P1-3：manage\_tag action trim（manage\_tag.go）

解析后加 `args.Action = strings.TrimSpace(args.Action)`，与其他工具一致。

### P1-4：文本长度上限（context.go helper + 各工具调用点）

context.go 新增：

```go
const (
	maxToolShortText = 500   // 短文本字段上限：标题/名称/关键字/搜索词/URL/问句/颜色等
	maxToolFindLen   = 2000  // edit 片段替换 find 原文片段上限
	maxToolLongText  = 20000 // 正文级字段上限：content / replace / append_content
)

// validateTextLen 校验文本字段长度（按 rune 计），超长返回描述性错误供回填模型。
func validateTextLen(field, s string, maxLen int) error {
	if n := len([]rune(s)); n > maxLen {
		return fmt.Errorf("%s 过长（%d 字符，上限 %d），请精简后重试", field, n, maxLen)
	}
	return nil
}
```

各工具在参数校验处调用（字段 → 上限）：

| 工具                                                  | 字段                                  | 上限                    |
| --------------------------------------------------- | ----------------------------------- | --------------------- |
| manage\_note                                        | title / keyword                     | 500 / 500             |
| manage\_note                                        | content / replace / append\_content | 20000 / 20000 / 20000 |
| manage\_note                                        | find                                | 2000                  |
| manage\_todo                                        | text / keyword                      | 500 / 500             |
| manage\_notebook                                    | name / keyword                      | 500 / 500             |
| manage\_tag                                         | name                                | 500                   |
| web\_search / refine\_search\_query / recall\_notes | query                               | 500                   |
| read\_url                                           | url                                 | 500                   |
| ask\_user                                           | question / options 元素               | 500 / 200             |

注：web\_search/recall\_notes 校验后仍 trim；长度校验放 trim 后。

### P2-1：写操作确认引导（app.go + manage\_note.go Desc）

* app.go 的【工具使用规范】段落（L2557 ask\_user 规范附近）新增【写操作确认】规范："执行破坏性或不可逆的写操作（如整篇替换笔记正文、删除笔记片段、重命名笔记本、移动笔记等）前，应先向用户确认修改意图；信息不足时用 ask\_user 澄清，不要直接执行"

* manage\_note 的 `Desc` 中 update/edit 描述补一句"修改重要内容前建议先向用户确认"

### P2-2：read\_url 内网地址防护（read\_url.go）

`validateHTTPURL` 增加内网/本机地址拒绝：

* `import "net"`；新增 `isPrivateHost(host string) bool`：

  * 去端口后 `net.ParseIP`：`IsLoopback()` / `IsPrivate()` / `IsLinkLocalUnicast()`（含 169.254.169.254）/ `IsUnspecified()` / `IsMulticast()` 任一命中 → 拒绝

  * 主机名为 `localhost` 或 `.local` / `.internal` 后缀 → 拒绝

* 域名不做 DNS 解析（避免额外网络 IO 与探测面）

### P2-3：manage\_note file\_ext 格式校验（manage\_note.go）

新增统一规范化 helper（create / update 共用）：

```go
var noteFileExtPattern = regexp.MustCompile(`^[a-zA-Z0-9]{1,10}$`)

// normalizeNoteFileExt 规范化笔记扩展名：trim → 去前导点 → 校验纯字母数字 1-10 位 → 补 "."。
func normalizeNoteFileExt(raw string) (string, error)
```

* create：空 → ".md"；非空 → normalize

* update：空 → 传空保持原值；非空 → normalize 后再传给 Update

## Assumptions & Decisions

* panic recover 只做错误回填，不恢复业务状态（工具无状态、可重试）

* 内网防护不解析域名（仅 IP 字面量与显式本机 hostname）

* 文本上限按 rune 计（中文友好）

* 全部改动在 Go 层，无前端改动、无 schema/注册改动

* 不改动现有错误信息风格（统一 "manage\_x 参数…" 前缀）

## Verification

1. `go build ./...` 通过
2. `go vet ./internal/agent/...` 通过
3. `GetDiagnostics` 检查所有改动文件无错误
4. 手动路径（可选，重启应用后）：

   * Agent 消息触发 web\_search 且某源异常 → 该源以部分失败提示跳过，进程不崩

   * Agent 传超长 content 编辑笔记 → 返回"content 过长"错误回填模型

   * read\_url 传 <http://localhost:8080> → 拒绝并回填

   * edit 传 file\_ext="md" / ".md" → 均规范化成功

