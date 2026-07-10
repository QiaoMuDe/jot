# 优化测试连接按钮的通知提示

## 当前问题

三个测试连接按钮的通知提示信息太笼统，用户无法根据提示判断具体问题：

| 按钮 | 当前成功提示 | 当前失败提示 | 问题 |
|------|-------------|-------------|------|
| AI 地址测试 | "API 地址连接成功" | "API 地址连接失败，请检查地址" / "连接失败: " + e | 后端返回 HTTP 401/403 时(ok=false, err=nil)，提示仍说"请检查地址"，但实际可能是 Key 不对 |
| Tavily 测试 | "连接成功！" | "连接失败: " + (e.message \|\| e) | 忽略 `ok` 返回值；成功提示无上下文 |
| 知乎测试 | "知乎连接测试成功" | "知乎连接测试失败" / "连接失败: " + (e.message \|\| e) | 失败时只报"连接失败"，无具体原因 |

## 改动方案

### 改动 1：后端 — AI 测试中非 2xx 状态码返回具体错误

**文件**: `internal/services/ai_service.go`

**现在的问题**：`testOpenAIConnection()` 和 `testOllamaConnection()` 在状态码不为 200-299 时返回 `(false, nil)`，前端只知道"失败"但完全不知道原因。

**修改**：将 `return resp.StatusCode >= 200 && resp.StatusCode < 300, nil` 改为带错误信息的返回：

```go
// 修改前
return resp.StatusCode >= 200 && resp.StatusCode < 300, nil

// 修改后
if resp.StatusCode >= 200 && resp.StatusCode < 300 {
    return true, nil
}
return false, fmt.Errorf("服务器返回状态码 %d", resp.StatusCode)
```

这样前端 `catch` 分支就能捕获到具体 HTTP 状态码。

### 改动 2：前端 — AI 地址测试提示优化

**文件**: `frontend/src/main.js` 第 1857-1882 行

**修改内容**：
- 成功时提示包含具体 provider 名称：`"${provider} 服务连接成功"`
- 失败时（ok=false 但无 error）：提示更具体，包含 provider 信息
- 异常时：直接展示后端返回的具体错误原因

```javascript
// 修改前
if (ok) {
    nm.show('API 地址连接成功', 'success');
} else {
    nm.show('API 地址连接失败，请检查地址', 'warning');
}

// 修改后
if (ok) {
    nm.show(`${provider} 服务连接成功`, 'success');
} else {
    nm.show(`${provider} 服务连接失败，请检查地址和 Key 是否正确`, 'warning');
}
```

### 改动 3：前端 — Tavily 测试连接提示优化

**文件**: `frontend/src/main.js` 第 2062-2081 行

**修改内容**：
- 增加对 `ok` 返回值的检查（当前代码忽略了它）
- 成功时提示：`"Tavily API Key 验证成功"`
- 失败时提示后端返回的具体错误

```javascript
// 修改前
try {
    await window.go.main.App.TestTavilyConnection(key);
    nm.show('连接成功！', 'success');
} catch (e) {
    nm.show('连接失败: ' + (e.message || e), 'error');
}

// 修改后
try {
    const ok = await window.go.main.App.TestTavilyConnection(key);
    if (ok) {
        nm.show('Tavily API Key 验证成功，搜索服务可用', 'success');
    } else {
        nm.show('Tavily 连接失败，请检查 API Key 是否正确', 'warning');
    }
} catch (e) {
    nm.show('Tavily 连接失败: ' + (e.message || e), 'error');
}
```

### 改动 4：前端 — 知乎测试连接提示优化

**文件**: `frontend/src/main.js` 第 2146-2169 行

**修改内容**：
- 成功时提示：`"知乎 Access Secret 验证成功"`
- 失败时展示后端具体错误信息（包括 "请检查 Access Secret 是否正确" 等后端已返回的错误）

```javascript
// 修改前
if (ok) {
    nm.show('知乎连接测试成功', 'success');
} else {
    nm.show('知乎连接测试失败', 'error');
}
// catch 分支: nm.show('连接失败: ' + (e.message || e), 'error');

// 修改后
if (ok) {
    nm.show('知乎 Access Secret 验证成功，搜索服务可用', 'success');
} else {
    nm.show('知乎搜索连接失败，请检查 Access Secret 是否正确', 'warning');
}
// catch 分支保持但去掉 "连接失败:" 前缀（后端已经带了）
// 改为: nm.show(e.message || e, 'error');
```

## 改动总结

| # | 文件 | 改动 | 说明 |
|---|------|------|------|
| 1 | `internal/services/ai_service.go` | 非 2xx 状态码返回具体错误 | 让前端 catch 到 HTTP 状态码 |
| 2 | `frontend/src/main.js` L1857-1882 | 优化 AI 测试提示 | 含 provider 名、区分地址/Key 问题 |
| 3 | `frontend/src/main.js` L2062-2081 | 优化 Tavily 测试提示 | 增加 ok 检查、含服务名 |
| 4 | `frontend/src/main.js` L2146-2169 | 优化知乎测试提示 | 含服务名、去掉冗余前缀 |

## 验证

1. `go build ./...` 编译通过
2. 确认前端无语法错误
