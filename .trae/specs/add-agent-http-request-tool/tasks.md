# Tasks

- [x] Task 1: 新增设置种子键 `ai_http_max_chars`
  - [x] 1.1 在 `internal/database/db.go` 的 defaults 列表追加 `{Key: "ai_http_max_chars", Value: "5000"}`，附注释说明（read_url 截断键同款范式：无前端 UI，仅初始化默认值，由 http_request 直接读取）

- [x] Task 2: 实现 `internal/agent/tools/http_request.go`
  - [x] 2.1 文件头注释：工具职责（模型调用第三方 API / 获取原始响应）、与 read_url 的分工（read_url 面向网页正文提取，本工具面向 API/原始响应）、实现要点
  - [x] 2.2 定义 `httpRequestTool` 结构体（依赖 `setting *services.SettingService` 与 `ctx *Context`）、编译期断言 `var _ tool.InvokableTool = (*httpRequestTool)(nil)`、构造器 `NewHTTP(setting, ctx) tool.InvokableTool`
  - [x] 2.3 实现 `Info()`：名称 `http_request`；`Desc` 说清何时调用（调用 REST API / 需要 POST/自定义头/原始响应；读取网页正文请用 read_url）；参数 schema：`url`（必填 String）、`method`（可选，Enum：GET/POST/PUT/DELETE，默认 GET）、`headers`（可选 Object，SubParams 用 `AdditionalProperties` 或说明为 string→string 键值对）、`body`（可选 String）
  - [x] 2.4 实现 `InvokableRun()`：解析参数 → method 合法性校验（非法返回列枚举的 error）→ `validateTextLen("body", ...)` 长度校验 → `validateHTTPURL`（复用）校验 URL → 构造 `http.Client{Timeout: 15s, CheckRedirect: 上限10次 + isPrivateHost 逐跳校验, Transport: DialContext DNS解析后逐IP执行 isPrivateHost 黑名单校验（防 DNS rebinding，拒绝时返回含解析IP的描述性错误）}` → 构造请求（默认浏览器 UA，headers 显式提供 User-Agent 时覆盖；忽略 headers 中的 Host 键；method 为 GET 时不带 body）→ 发送 → 读取响应（`io.LimitReader` 限 1MB）
  - [x] 2.5 响应输出：首行协议+状态码 → 关键响应头（Content-Type/Content-Length）→ 文本类正文按 `getIntSetting(setting, "ai_http_max_chars", 5000, 50000)` rune 截断（超长追加"（内容过长，已截断）"）；二进制类 Content-Type 只提示类型与大小；4xx/5xx 不作为工具失败
  - [x] 2.6 错误路径：参数解析/校验失败、URL 非法、请求失败、超时均返回描述性 error；`ctx.Err() != nil` 时直接返回 ctx.Err()
  - [x] 2.7 日志脱敏：仅记录 method、URL、状态码、耗时、响应字节数（`ctx.Logger.Debugw`），不输出请求头
  - [x] 2.8 实现 `ActionTextProvider`：解析 arguments JSON，返回 `请求 {METHOD} {url截断30字符}`；解析失败或 url 为空返回"发起 HTTP 请求"

- [x] Task 3: 注册与清单同步
  - [x] 3.1 `internal/agent/registry.go` 的 `buildTools` 在 `read_url` 注册行之后追加 `{"http_request", tools.WrapWithError("http_request", tools.NewHTTP(p.deps.Setting, p.ctx), p.ctx)}`
  - [x] 3.2 `internal/agent/tools/meta.go` 的 `BuiltinTools()` 在 `read_url` 条目后追加 `{Name: "http_request", Label: "发起 HTTP 请求调用 API（GET/POST/PUT/DELETE）"}`
  - [x] 3.3 `internal/agent/tools/doc.go` 包级文档：工具清单追加 `http_request`、构造器名追加 `NewHTTP`

- [x] Task 4: 单元测试（对齐 manage_note_test.go 等既有测试风格）
  - [x] 4.1 为 DNS 解析后 IP 校验逻辑编写测试：内网 IP（127.0.0.1/192.168.x/169.254.x）拒绝、公网 IP 放行；method 枚举校验（合法/非法）；ActionText 正常与解析失败回退
  - [x] 4.2 如涉及httptest 可用标准库 `net/http/httptest` 验证 GET/POST 成功路径与超长截断（监听 127.0.0.1 的本地服务器需注入跳过 DialContext 黑名单的测试开关，避免测试被 SSRF 防护拦截——实现时在工具内预留可注入的 Transport/Dialer 或将校验函数独立导出为包内函数直接测）
  - 实现说明：注入缝为 `buildClient(guardDial bool)`（测试 client 绕过拨号防护）+ `httpRequestTool.skipURLGuard`（测试跳过 validateHTTPURL 内网拒绝，零值安全，生产构造器不设置）；另补 `TestHTTPRequestURLGuard` 保证第①层防护回归覆盖

- [x] Task 5: 验证
  - [x] 5.1 `go build ./...` 通过
  - [x] 5.2 `go vet ./internal/agent/...` 通过
  - [x] 5.3 `go test ./internal/agent/...` 通过（jot/internal/agent 与 jot/internal/agent/tools 均 ok）
  - [x] 5.4 `go.mod` 确认无新增 require 条目（纯标准库）

# Task Dependencies
- Task 2 依赖 Task 1（设置键先就位；不阻塞编译，getIntSetting 有默认值兜底，可并行）
- Task 3 依赖 Task 2（构造器签名确定后才能注册）
- Task 4 依赖 Task 2
- Task 5 依赖 Task 1-4 全部完成
