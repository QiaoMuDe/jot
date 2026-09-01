# Agent 内置 HTTP 请求工具（http_request）Spec

## Why
Agent 目前联网能力只有 `read_url`（GET-only、面向网页正文提取），无法调用 REST API（POST/带 headers/带 body）。新增一个通用 HTTP 请求工具，让模型能直接调用第三方 API、调试接口、获取原始 JSON 响应。

## What Changes
- 新增内置工具 `http_request`：基于 **Go 标准库 `net/http`**（不引入任何第三方 HTTP 客户端库）
- 参数：`url`（必填 string）、`method`（可选，**仅枚举 GET/POST/PUT/DELETE 四个常用方法**，默认 GET）、`headers`（可选 object，string→string 键值对）、`body`（可选 string）
- **安全防护（复用 + 加强）**：
  - 初始 URL 复用 `validateHTTPURL`（仅 http/https）+ `isPrivateHost`（内网/本机黑名单，同包复用）
  - `CheckRedirect` 逐跳校验重定向目标（上限 10 次，同 read_url 范式）
  - **DNS 解析后 IP 校验（本次新增要求）**：`Transport.DialContext` 中对域名解析出的实际 IP 再执行内网黑名单校验，防 DNS rebinding 绕过
- 超时 15s（独立常量，同 read_url 范式）；响应体先 `io.LimitReader` 限 1MB 原始字节，再按设置 `ai_http_max_chars`（默认 5000，上限 50000）按 rune 截断
- 输出格式：状态行 + 关键响应头（Content-Type / Content-Length）+ 响应体；二进制 Content-Type（image/、octet-stream 等）只提示类型不返回正文
- 默认携带浏览器 UA（headers 中显式指定时可覆盖），日志不输出完整请求头（防 Authorization/API Key 泄漏）
- 新增设置种子键 `ai_http_max_chars`（`database/db.go` 默认值列表，无前端 UI，同 `ai_read_url_max_chars` 范式）
- 实现可选接口 `ActionTextProvider`：动作文案如 `请求 GET https://example.com/...`（URL 截断 30 字符）

## Impact
- Affected specs: 无既有 spec 受影响（`add-agent-read-url-tool` 为独立工具，本工具与其互补不合并）
- Affected code:
  - `internal/agent/tools/http_request.go`（新增，核心实现）
  - `internal/agent/registry.go`（buildTools 追加一行注册）
  - `internal/agent/tools/meta.go`（BuiltinTools 追加展示文案）
  - `internal/agent/tools/doc.go`（工具清单与构造器名同步）
  - `internal/database/db.go`（设置种子键）
- 不改动前端（工具名直接展示英文，动作文案经 ActionTextProvider 自动下发）

## ADDED Requirements

### Requirement: http_request 工具实现
`internal/agent/tools` 子包 SHALL 新增 `httpRequestTool` 结构体，实现 `tool.InvokableTool` 接口（含编译期断言 `var _ tool.InvokableTool = (*httpRequestTool)(nil)`），并提供导出构造器 `NewHTTP(setting *services.SettingService, ctx *Context) tool.InvokableTool`。依赖仅 `services.SettingService`（读取截断设置）与 `*Context`（日志），不新增 `Deps` 字段。

#### Scenario: 成功发起 GET 请求
- **WHEN** 模型调用 `http_request`，参数 `{"url": "https://api.example.com/data", "method": "GET"}`
- **THEN** 工具发起真实 HTTP GET 请求，返回形如以下结构的纯文本：
  `HTTP/1.1 200 OK` + `Content-Type: application/json` 等关键响应头 + 截断后的响应体

#### Scenario: 成功发起带 headers/body 的 POST 请求
- **WHEN** 模型调用参数 `{"url": "...", "method": "POST", "headers": {"Content-Type": "application/json"}, "body": "{\"k\":1}"}`
- **THEN** 请求按指定的 method/headers/body 发出，响应按同格式返回

#### Scenario: method 缺省
- **WHEN** 参数中省略 `method`
- **THEN** 默认使用 GET

#### Scenario: 非法 method
- **WHEN** `method` 不在枚举 GET/POST/PUT/DELETE 内（如 PATCH/HEAD/OPTIONS 或任意其他值）
- **THEN** 返回 error（经 WrapWithError 回填模型），错误文本列出合法枚举值

#### Scenario: body 字段长度校验
- **WHEN** `body` 超过 `maxToolLongText`（20000 rune）
- **THEN** 返回 error（复用 `validateTextLen`）

### Requirement: SSRF 三层防护
工具 SHALL 对每次请求执行三层安全校验，全部复用/沿用 read_url 已有范式：

1. **初始 URL 校验**：复用 `validateHTTPURL`（仅 http/https scheme）与 `isPrivateHost`（拒绝 IP 字面量指向 loopback/private/link-local/unspecified/multicast 及 localhost/.local/.internal 主机名）
2. **重定向逐跳校验**：`http.Client.CheckRedirect` 对每个重定向目标执行 `isPrivateHost` 校验，上限 10 次跳转
3. **DNS 解析后 IP 校验（防 DNS rebinding）**：自定义 `http.Transport.DialContext`——对目标主机做 DNS 解析后，逐个校验解析出的 IP（复用 `isPrivateHost` 对 IP 的判定逻辑），任一 IP 命中内网黑名单即拒绝连接并返回描述性错误

#### Scenario: 直连内网 IP 被拒绝
- **WHEN** 模型请求 `http://127.0.0.1:8080/api` 或 `http://192.168.1.1/`
- **THEN** 请求被拒绝，返回"拒绝访问内网/本机地址"类错误

#### Scenario: 重定向到内网被拒绝
- **WHEN** 公网 URL 302 跳转到 `http://169.254.169.254/`
- **THEN** 重定向被拒绝，返回"拒绝跟随重定向到内网/本机地址"类错误

#### Scenario: 公网域名解析到内网 IP 被拒绝（DNS rebinding 防护）
- **WHEN** 某公网域名的 DNS 记录被解析为 `127.0.0.1` 或其他内网 IP
- **THEN** DialContext 校验命中黑名单，连接被拒绝，返回含解析 IP 的描述性错误

#### Scenario: 正常公网域名不受影响
- **WHEN** 请求 `https://api.github.com/` 等解析为公网 IP 的域名
- **THEN** 正常建立连接并返回响应

### Requirement: 超时与响应体限制
- 请求总超时 SHALL 为 15 秒（常量，超时返回描述性错误，避免卡住 ReAct 循环）
- 响应体 SHALL 先经 `io.LimitReader` 限制原始读取字节（1MB），再按设置键 `ai_http_max_chars`（`getIntSetting(setting, "ai_http_max_chars", 5000, 50000)`）按 rune 截断，超长时追加"（内容过长，已截断）"提示

#### Scenario: 超时
- **WHEN** 目标服务器 15 秒内未响应
- **THEN** 返回"请求超时"类错误回填模型

#### Scenario: 超大响应体
- **WHEN** 响应体超过 1MB 或截断设置
- **THEN** 返回截断后的正文并附带截断提示，不撑爆模型上下文

### Requirement: 响应输出格式
工具 SHALL 返回结构化纯文本：
- 首行：协议与状态码（如 `HTTP/1.1 200 OK`；状态码非 2xx 时仍正常返回响应体，交由模型自行判断，不作为工具失败）
- 关键响应头：Content-Type、Content-Length（存在时）
- 响应体：文本类 Content-Type（含 JSON，可原样返回）直接输出截断后正文；二进制类 Content-Type（image/、audio/、video/、application/octet-stream、application/pdf 等）不返回正文，仅提示类型与大小

#### Scenario: 4xx/5xx 响应
- **WHEN** 目标 API 返回 404 或 500
- **THEN** 工具调用本身成功（返回状态行 + 响应体），由模型依据状态码推理

#### Scenario: 二进制响应
- **WHEN** 响应 Content-Type 为 `image/png`
- **THEN** 返回类型与大小提示，不输出二进制正文

### Requirement: 请求默认头与日志脱敏
- 默认携带浏览器 UA（`browserUserAgent`，同 read_url）；`headers` 参数中显式指定 `User-Agent` 时覆盖默认值
- 请求头中不应透传会干扰协议的 `Host`（由 net/http 按.url 自动管理，headers 中显式提供的 Host 键被忽略或拒绝）
- 日志仅记录 method、URL、状态码、耗时与响应字节数，**不得**输出完整请求头（防 Authorization/API Key 落日志）

#### Scenario: 动作文案
- **WHEN** 模型发起调用，参数含 `method=GET`、`url=https://api.example.com/data`
- **THEN** `tool_start` 事件携带 action_text `请求 GET https://api.example.com/da…`（URL 按 rune 截断 30 字符）；参数解析失败或 url 为空时回退通用文案"发起 HTTP 请求"

### Requirement: 注册与清单同步
- [registry.go](../../../internal/agent/registry.go) 的 `buildTools` SHALL 追加一行 `tools.WrapWithError("http_request", tools.NewHTTP(p.deps.Setting, p.ctx), p.ctx)`，位置紧邻 `read_url`（联网工具分组）
- [meta.go](../../../internal/agent/tools/meta.go) 的 `BuiltinTools()` SHALL 追加 `{Name: "http_request", Label: "发起 HTTP 请求调用 API（GET/POST 等）"}`
- [tools/doc.go](../../../internal/agent/tools/doc.go) 的包级文档工具清单与构造器名 SHALL 同步追加（含 `http_request` 与 `NewHTTP`）
- [db.go](../../../internal/database/db.go) 设置种子列表 SHALL 追加 `{Key: "ai_http_max_chars", Value: "5000"}`（含注释说明仅后端消费、无前端 UI）

#### Scenario: 设置页可见与可禁用
- **WHEN** 用户打开设置页 Agent 工具列表
- **THEN** `http_request` 出现在列表中（英文工具名 + 中文说明），可勾选禁用（走既有 `ai_agent_tools_disabled` 黑名单机制，无需新代码）

### Requirement: 不引入第三方依赖
工具实现 SHALL 仅使用 Go 标准库（net/http、net、context、encoding/json、io、strings、time、fmt、errors）与本包既有工具函数，**不新增任何第三方 HTTP 客户端依赖**，`go.mod` 不新增 require 条目。
