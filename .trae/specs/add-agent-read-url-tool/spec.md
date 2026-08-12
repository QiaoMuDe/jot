# Agent 新增 read_url 网页链接读取工具 Spec

## Why

当前 AI 助手 Agent 模式无法阅读用户输入的链接：用户发消息带 URL（或要求阅读某个网页）时，模型没有工具可抓取网页内容，只能基于链接本身猜测。需要一个 `read_url` 工具，让模型在 ReAct 循环中调用它抓取并提取网页正文，从而基于真实内容回答。

## What Changes

- 新增 eino-ext 官方依赖 `github.com/cloudwego/eino-ext/components/document/loader/url`（URL Document Loader，默认 HTML 解析器提取正文）。
- 在 `internal/agent/tools/` 新增 `read_url.go`：实现 `readURLTool`（`tool.InvokableTool` + `ActionTextProvider`），参数 `url`（必填，仅放行 http/https），内部用 `url.NewLoader` + `loader.Load` 抓取网页，按 `ai_web_search_max_chars` 设置（默认 5000、上限 50000）截断后返回正文。
- 把 `web_search.go` 的 `intSetting` 方法提取为包级函数 `getIntSetting`，`read_url` 复用同一字符上限设置。
- `internal/agent/registry.go` 的 `buildTools` 追加注册 `read_url`（`WrapWithError` 包装）。
- `internal/agent/tools/meta.go` 的 `BuiltinTools` 追加展示文案。
- 同步维护权威清单：`internal/agent/tools/doc.go` 与 `internal/agent/doc.go`。
- 前端零改动：工具名直接展示英文，动作文案走既有 `ActionTextProvider` 机制。

## Impact

- Affected specs: Agent 工具体系（TOOLS.md §2 新增工具流程）
- Affected code:
  - `internal/agent/tools/read_url.go`（新增）
  - `internal/agent/tools/web_search.go`（`intSetting` 提取为包级函数）
  - `internal/agent/registry.go`（注册一行）
  - `internal/agent/tools/meta.go`（展示文案一行）
  - `internal/agent/tools/doc.go`、`internal/agent/doc.go`（清单同步）
  - `go.mod` / `go.sum`（新增依赖）

## ADDED Requirements

### Requirement: read_url 工具

系统 SHALL 提供 `read_url` 工具，供 Agent 模式模型在用户消息包含链接或要求阅读网页时调用，抓取网页并返回提取后的正文。

#### Scenario: 成功读取链接
- **WHEN** 模型调用 `read_url`，参数 `url` 为合法的 http/https 链接
- **THEN** 工具抓取网页，用默认 HTML 解析器提取正文，按 `ai_web_search_max_chars` 截断，返回 `以下为链接 {url} 的内容：\n{正文}`；超长时正文末尾追加"（内容过长，已截断）"

#### Scenario: 参数缺失或非法 URL
- **WHEN** `url` 参数缺失，或 scheme 不是 http/https（如 `file://`、`data:`）
- **THEN** 工具返回 error，经 `WrapWithError` 回填模型继续推理，不中断 ReAct 循环

#### Scenario: 抓取失败或空正文
- **WHEN** 网络请求失败、超时（15s）、或提取不到正文（动态渲染页面）
- **THEN** 工具返回描述性 error（如"读取链接失败""未能从该链接提取到正文内容"），`WrapWithError` 捕获并发射 `tool_error` 事件

#### Scenario: 用户停止
- **WHEN** 抓取期间用户取消（ctx 取消）
- **THEN** 工具返回 `ctx.Err()`，父包不误报失败，循环随 ctx 终止

#### Scenario: 工具动作文案
- **WHEN** 模型发起 `read_url` 调用，`tool_start` 事件生成
- **THEN** `action_text` 为"阅读链接 {url 截断 30 字符}"，参数解析失败时回退"阅读网页链接"

### Requirement: 复用字符上限设置

系统 SHALL 复用现有 `ai_web_search_max_chars` 设置作为 `read_url` 的输出字符上限，不新增设置项。

#### Scenario: 设置生效
- **WHEN** 用户修改 `ai_web_search_max_chars`（默认 5000、上限 50000）
- **THEN** `read_url` 输出按新值截断，非法值回退默认值

## MODIFIED Requirements

### Requirement: intSetting 工具方法提取为包级函数

原 `webSearchTool.intSetting` 方法 SHALL 提取为包级函数 `getIntSetting(setting *services.SettingService, key string, def, max int) int`，`web_search` 与 `read_url` 共同调用，行为不变（解析失败或越界回退默认值，越上限取上限）。

## REMOVED Requirements

无
