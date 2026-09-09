# read_url 分页读取改造方案

## Summary

为 `read_url` 网页链接读取工具增加分页读取能力：新增 `offset`（起始字符位置）与 `length`（单次读取字符数）两个可选参数，按 rune 偏移切片返回正文，并在返回中携带起止位置与总字符数，使模型可以自主翻页读完长网页。**采用无缓存、无状态设计**（对齐官方 MCP Fetch 服务器的 stateless 风格）：每次调用重新抓取整页后切片。

## Current State Analysis

- [read_url.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go) 是唯一改动文件。
  - 结构体 `readURLTool`（[L38-L42](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L38-L42)）：仅 `setting` + `ctx`，本次改动不加缓存字段。
  - `ActionText`（[L49-L60](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L49-L60)）：仅解析 `url`。
  - `Info`（[L63-L75](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L63-L75)）：Schema 只有 `url`。
  - `InvokableRun`（[L80-L150](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L80-L150)）：
    1. 解析 `url` → `validateHTTPURL`；
    2. 构建 eino URL Loader 抓取，拼接多 doc（**当前在 `b.Len() >= maxChars` 时提前跳出**，[L128-L130](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L128-L130)）；
    3. 按 `ai_read_url_max_chars`（默认 10000，上限 50000）从头截断，超长追加 `（内容过长，已截断）`（[L139-L142](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L139-L142)）。
- 可复用的同包模式：[read_note_section.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_note_section.go) 已实现 `offset/length` 参数、rune 切片、`math.Trunc` 整数校验、越界报错（`offset 超出内容范围（共 N 字符，已全部读取完毕）`）、返回定位信息（`第 X-Y 字符的内容（共 N 字符）`）。同包常量 `maxSectionLen = 100000`（[L31-L32](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_note_section.go#L31-L32)）可直接复用。
- 抓取上限：共享防护客户端限制响应体 1MB（[ssrf.go#L24-L25](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/ssrf.go#L24-L25)），拼接全文后最多约 1MB，内存无忧。
- 工具注册于 [registry.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/registry.go#L186)，本次不改注册、不改设置键、不改 `NewReadURL` 签名。

## Proposed Changes

改动仅 [read_url.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go) 一个文件。

### 1. `Info`：新增 `offset` / `length` 参数

在现有 `url` 参数旁追加：

```go
"offset": {
	Type:     schema.Number,
	Desc:     "起始字符位置，从 0 开始；首次阅读可省略，续读时传上一段返回的结尾位置（必须小于内容总字符数）",
	Required: false,
},
"length": {
	Type:     schema.Number,
	Desc:     "本次读取的字符数，可选；缺省取 ai_read_url_max_chars 设置，上限 100000",
	Required: false,
},
```

`Desc` 末尾追加翻页协议说明：`长网页可分页读取：首次省略 offset 从开头读；返回含"第 X-Y 字符（共 N 字符）"，续读时以 Y 为 offset 调用；offset 超出内容范围表示已全部读完。`

### 2. `ActionText`：展示起始位置

解析 `offset`，非 0 时显示 `阅读链接 X 第 N 字符起`，offset 为 0 或缺失时保持现有文案 `阅读链接 X`。

### 3. `InvokableRun`：改为全文拼接 + offset/length 切片

- 参数解析结构体扩展为 `{URL string; Offset float64; Length float64}`。
- 拼接阶段：**删除** `if b.Len() >= maxChars { break }` 提前跳出（[L128-L130](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L128-L130)），改为拼接全部 doc 形成全文 `full`；空正文错误逻辑保持不变。
- `maxChars` 变量改为"默认单段长度"：`getIntSetting(r.setting, "ai_read_url_max_chars", 10000, 50000)`，仅当 `args.Length <= 0` 时作为 length。
- 校验（对齐 read_note_section）：
  - `args.Offset < 0` → error；`args.Offset != math.Trunc(args.Offset)` → error（需新增 `math` import）。
  - `args.Length != 0 && args.Length != math.Trunc(args.Length)` → error；`args.Length < 0` → error。
  - 显式 `length` 超过 `maxSectionLen`（100000）时截到 `maxSectionLen`。
- 切片：
  - `runes := []rune(full); total := len(runes); offset := int(args.Offset)`
  - `offset >= total` → error `offset 超出内容范围（共 %d 字符，已全部读取完毕）`（模型收到即知翻页结束）。
  - `end := offset + length`，`end > total` 时 `end = total`。
  - `section := string(runes[offset:end])`
- 返回格式（含定位信息 + 未读完提示）：

```go
msg := fmt.Sprintf("以下为链接 %s 第 %d-%d 字符的内容（共 %d 字符）：\n%s",
	target, offset+1, end, total, section)
if end < total {
	msg += fmt.Sprintf("\n（内容未完，如需继续请以 offset=%d 调用）", end)
}
```

- 删除旧的"从头截断 + `（内容过长，已截断）`"逻辑（[L139-L142](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/read_url.go#L139-L142)）。
- 日志：`Debugw` 增加 `offset`、`end`、`total` 字段（保留 url、chars 字段）。
- 文件头注释同步更新（第 3-8 行：从"按设置截断后返回"改为"支持 offset/length 分页读取"）。

### 4. 不改动的部分

`readURLTool` 结构体、`NewReadURL` 签名、`validateHTTPURL`/SSRF 防护、registry、meta.go、设置键 `ai_read_url_max_chars`。

## Assumptions & Decisions

- **无缓存（无状态）**：每次调用重新抓取整页再切片，对齐官方 MCP Fetch 设计。代价：动态页面每次抓取内容可能有微小漂移，`共 N 字符` 与切片边界以当次抓取为准，极端情况下相邻两段有极小重叠/缺失，静态文章页基本无感。
- **偏移单位为 rune（字符）**：支持中文正确切片，与 `read_note_section` 语义一致。
- **越界语义**：`offset >= total` 返回 error，经 `WrapWithError` 回填模型，模型据此停止翻页（与 `read_note_section` 一致）。
- **`length` 默认值**沿用现有设置 `ai_read_url_max_chars`（默认 10000），用户已有配置继续生效；显式传入上限 100000。
- 老调用（只传 `url`）等价于 `offset=0`，行为兼容，仅返回格式变化（由"已截断"变为"第 X-Y 字符（共 N 字符）"），新 Desc 已说明。

## Verification

1. `go build ./...`（或 `go build ./internal/agent/...`）确认编译通过。
2. `go vet ./internal/agent/tools/` 静态检查。
3. `go test ./internal/agent/...` 运行现有测试（含同包 read_note_section 相关工具测试），确认无回归。
4. 手动验证（可选）：在应用中让模型阅读一篇长网页，观察其按 offset 翻页、到末尾后停止。
