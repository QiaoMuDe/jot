# AI 请求错误封装层 Spec

## Why

AI 模型请求失败时，原始 HTTP 状态码和错误信息（如 `401 Unauthorized`、`429 Too Many Requests`、`500 Internal Server Error`）直接暴露给用户，体验差且不友好。

## What Changes

1. **新增** `internal/aicli/errors.go` — AI 错误分类和状态码→中文提示映射层
2. **修改** `internal/aicli/openai.go` — 使用 `errors.As()` 检测 `APIError`/`RequestError`，通过错误分类层转换为友好消息
3. **修改** `internal/aicli/ollama.go` — OIlama 错误分类处理
4. **修改** `internal/aicli/client.go` — `OnError` 回调传递结构化 JSON 字符串（含分类、友好消息、原始错误）
5. **修改** `app.go` — 流式错误事件 `ai:stream-error` 传递结构化数据
6. **修改** `frontend/src/js/ai-chat.js` — 流式和非流式错误处理：解析结构化错误，展示通知

## Impact

- Affected specs: AI 对话功能、优化表达功能
- Affected code:
  - `internal/aicli/errors.go` (新增)
  - `internal/aicli/client.go`
  - `internal/aicli/openai.go`
  - `internal/aicli/ollama.go`
  - `app.go`
  - `frontend/src/js/ai-chat.js`

## ADDED Requirements

### Requirement: AI 错误分类层

系统 SHALL 在 `internal/aicli/errors.go` 中提供错误分类和中文提示映射。

#### 错误分类与中文提示映射表

| 分类 | HTTP 状态码/条件 | 中文提示 |
|------|-----------------|---------|
| `auth_error` | 401, 403 | API 密钥无效或权限不足，请检查 API 配置 |
| `rate_limit` | 429 | 请求过于频繁，已达速率限制，请稍后重试 |
| `server_error` | 500, 502, 503 | AI 服务暂时不可用，请稍后重试 |
| `quota_exceeded` | 402, 429(code=insufficient_quota) | API 额度已用尽，请检查账户余额 |
| `model_not_found` | 404 | 模型不存在或已弃用，请更换模型名称 |
| `context_length` | 400(body 含 context_length/token 相关错误) | 上下文长度超限，请缩短对话历史或笔记内容 |
| `timeout` | 网络超时 / DeadlineExceeded | 请求超时，请检查网络连接或稍后重试 |
| `invalid_request` | 400(其他) | 请求参数有误，请检查输入内容 |
| `content_filter` | 400(body 含 content_filter) | 内容被安全策略拦截，请调整输入后重试 |
| `network_error` | 网络连接失败 / DNS 解析失败 | 网络连接失败，请检查网络设置或 API 地址 |
| `unknown` | 其他未分类错误 | AI 调用出错，请稍后重试 |

#### Scenario: OpenAI 请求返回 401

- **WHEN** 后端调用 OpenAI API 时收到 HTTP 401
- **THEN** 错误分类层返回 `{ "category": "auth_error", "user_msg": "API 密钥无效或权限不足，请检查 API 配置", "raw": "error, status code: 401, ..." }`
- **AND** 通过 `OnError` 回传 JSON 字符串给 app.go

#### Scenario: OpenAI 请求返回 429

- **WHEN** 后端调用 OpenAI API 时收到 HTTP 429
- **THEN** 错误分类层返回 `{ "category": "rate_limit", "user_msg": "请求过于频繁，已达速率限制，请稍后重试", "raw": "error, status code: 429, ..." }`

#### Scenario: 请求超时

- **WHEN** 后端调用时因 `context.DeadlineExceeded` 超时
- **THEN** 错误分类层返回 `{ "category": "timeout", "user_msg": "请求超时，请检查网络连接或稍后重试", "raw": "..." }`

#### Scenario: 网络连接失败

- **WHEN** 后端调用时发生网络连接错误（DNS/连接拒绝/无法到达）
- **THEN** 错误分类层返回 `{ "category": "network_error", "user_msg": "网络连接失败，请检查网络设置或 API 地址", "raw": "..." }`

#### Scenario: Ollama 请求返回错误

- **WHEN** 后端调用 Ollama API 时收到错误
- **THEN** Ollama 的错误信息也经过同一错误分类层处理，返回对应的中文提示

### Requirement: 错误信息传递格式

系统 SHALL 使用 JSON 字符串在 `OnError` 回调中传递结构化错误信息。

格式：
```json
{
  "category": "auth_error",
  "user_msg": "API 密钥无效或权限不足，请检查 API 配置",
  "raw": "error, status code: 401, status: 401 Unauthorized, message: Incorrect API key provided"
}
```

### Requirement: 流式调用错误处理（前端）

#### `ai:stream-error` 事件处理

- **WHEN** 前端收到 `ai:stream-error` 事件
- **THEN** 解析 JSON 中的 `user_msg`
- **AND** 调用 `window.showNotification(errData.user_msg, 'error', 5000)` 展示右上角通知
- **AND** 不再在消息流底部显示 `ai-msg-error` 红色错误条（避免重复通知）

#### `CallAI` 非流式调用错误处理

- **WHEN** `CallAI`（优化表达功能）的 `catch` 捕获错误
- **THEN** 解析 JSON 中的 `user_msg`
- **AND** 调用 `window.showNotification(errData.user_msg, 'error', 5000)` 展示通知

### Requirement: 错误兼容

- **WHEN** 后端回传的 `ai:stream-error` 数据不是有效 JSON（例如旧格式纯字符串）
- **THEN** 前端兼容降级为原有行为：直接显示原始错误信息

## MODIFIED Requirements

### Requirement: OnError 回调签名

`OnError` 回调的签名保持 `func(errMsg string)` 不变，但传递的内容从纯文本改为 JSON 字符串。

## REMOVED Requirements

### Requirement: 流式调用底部的 ai-msg-error 红色错误条

**Reason**: 错误信息改为通过右上角通知展示，体验更好且不打断对话流。
**Migration**: 错误消息不再插入消息流底部，但 `addErrorMessage()` 函数保留不删除（留给其他非 API 错误场景使用）。

## 设计说明

### 为什么用 JSON 字符串而不是 struct？

由于 `OnError` 是回调函数类型 `func(errMsg string)`，且 `runtime.EventsEmit` 只能传递基础类型参数（string），因此错误信息以 JSON 字符串传递到前端后由前端解析。

### go-openai 错误类型检测

`openai.go` 中使用 `errors.As()` 检测：
- `*openai.APIError` — 获取 `HTTPStatusCode`、`Message`、`Type` 等字段
- `*openai.RequestError` — 获取 `HTTPStatusCode`、`Err`、`Body` 等字段

### 非 `go-openai` 错误

对于 `context.DeadlineExceeded`、`context.Canceled`、`net.OpError`（网络错误）等非 go-openai 错误，使用 `errors.Is()` 和 `errors.As()` 检测，映射到对应的分类。
