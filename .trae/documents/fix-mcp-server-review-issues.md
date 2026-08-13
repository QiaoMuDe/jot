# 修复 MCP 服务器代码审查发现的 6 个问题

## 摘要

按上一轮审查结论修复 6 项：created\_at 覆写（高）、GetTools 无超时（中）、前端死代码（低）、字符校验缺失（低）、传输切换字段残留（低）、Enabled 注释不一致（信息）。全部为局部改动，不改变交互逻辑。

## 现状分析（基于已完成的源码级探索）

| # | 问题                 | 位置                                                                               | 根因                                                                                                                                                      |
| - | ------------------ | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | 编辑保存覆写 created\_at | `internal/services/mcp_server_service.go:67`                                     | `s.db.Save(server)` 全字段 SET（GORM v1.31.1 源码确认：Save → Selects=\["\*"]，update 回调对 AutoCreateTime 字段直接写入结构体零值）；前端表单 payload 无 created\_at → 覆写为 0001-01-01 |
| 2 | GetTools 无超时       | `internal/mcpserver/tools.go:34`                                                 | `Connect` 有 10s 超时，但 `mcpp.GetTools` 使用调用方 ctx（通常无 deadline），ListTools 挂起会无限阻塞装配                                                                        |
| 3 | 前端死代码 + 误导注释       | `frontend/src/main.js:9496-9501`                                                 | Wails v2 绑定中 Go error 会 reject Promise，`if (result)` 恒为 false，错误实际走 catch                                                                               |
| 4 | 名称/KEY 无字符校验       | `internal/services/mcp_server_service.go:35-54`、`frontend/src/main.js:9397-9480` | name 含空白会污染工具名 `mcp_{name}_{tool}`；Env/Header KEY 含空白/`=` 会导致请求或进程启动出错                                                                                  |
| 5 | 切换传输方式旧字段残留        | `internal/services/mcp_server_service.go:64-67`                                  | Save 全字段写入，stdio→sse 后 command/args/env 仍存库                                                                                                             |
| 6 | Enabled 注释不一致      | `internal/models/mcp_server.go:15`                                               | 注释写「默认 false，安全考量」，前端新增实际传 true                                                                                                                         |

## 修改方案

### 1. 【高】created\_at 覆写 — `internal/services/mcp_server_service.go`

第 67 行 `s.db.Save(server)` 改为：

```go
// Omit("created_at")：避免 Save 全字段更新把 created_at 覆写为零值（前端表单不带该字段）。
// 新增路径（ID==0 走 Create）时 GORM 仍会为 AutoCreateTime 字段自动填充当前时间，不受 Omit 影响。
return s.db.Omit("created_at").Save(server).Error
```

### 2. 【中】GetTools 无超时 — `internal/mcpserver/tools.go`

`OpenSession` 内对工具发现包独立超时（复用 `ConnectTimeout`），与 Connect 的超时策略一致：

```go
cli, err := Connect(ctx, s)
if err != nil {
    return nil, err
}
// 工具发现（ListTools）是独立网络往返，单独包超时，避免远程服务器挂起阻塞整轮装配
discoverCtx, cancel := context.WithTimeout(ctx, ConnectTimeout)
defer cancel()
baseTools, err := mcpp.GetTools(discoverCtx, &mcpp.Config{Cli: cli})
if err != nil {
    _ = cli.Close()
    return nil, fmt.Errorf("MCP 服务器 %s 工具发现失败: %w", s.Name, err)
}
```

### 3. 【低】前端死代码 — `frontend/src/main.js`

`saveMCPServerForm` 中删除 `if (result)` 死分支与误导注释，错误统一走 catch：

```js
try {
    await window.go.main.App.SaveMCPServer(payload);
    closeMCPServerForm(true); // 保存成功后跳过未保存修改确认
    await loadMCPServers();
    nm.show(mcpFormMode === 'create' ? 'MCP 服务器已添加' : 'MCP 服务器已更新', 'success');
} catch (e) {
    // 后端校验/存储错误（Wails 以异常形式返回）
    nm.show(mcpErrMsg(e), 'error');
} finally {
    mcpFormSaving = false;
}
```

### 4. 【低】字符校验

**后端** **`internal/services/mcp_server_service.go`**：在现有校验后、查重前增加：

```go
// 名称不能含空白：名称直接拼入工具名前缀 mcp_{name}_{tool}，空白会破坏工具名
if strings.ContainsAny(server.Name, " \t\r\n") {
    return fmt.Errorf("MCP 服务器名称不能包含空格等空白字符")
}
```

并在 env/headers 解析前增加 KEY 校验。env/headers 为 `map[string]string`，在 Save 入口遍历校验：

```go
for key := range server.Env {
    if strings.ContainsAny(key, " \t\r\n=") {
        return fmt.Errorf("环境变量 KEY「%s」不能包含空白或等号", key)
    }
}
for key := range server.Headers {
    if strings.ContainsAny(key, " \t\r\n=") {
        return fmt.Errorf("请求头 KEY「%s」不能包含空白或等号", key)
    }
}
```

（需新增 `strings` import）

**前端** **`frontend/src/main.js`**：保存表单时补充同等提示（输入即反馈，而非等后端报错）：

* name 校验后追加：`if (/\s/.test(name)) { nm.show('服务器名称不能包含空格等空白字符', 'error'); ... return; }`

* env/headers 循环内 KEY 校验追加：`if (/\s/.test(key)) { nm.show('KEY 不能包含空白字符', 'error'); ... return; }`（现有空 KEY 判断保留）

### 5. 【低】传输切换字段残留 — `internal/services/mcp_server_service.go`

在传输类型校验通过后、查重前，按 transport 清零非相关字段：

```go
// 按传输类型清零非相关字段，避免切换传输方式后旧字段残留（数据保持干净）
switch server.Transport {
case "stdio":
    server.URL = ""
    server.Headers = nil
case "sse", "http":
    server.Command = ""
    server.Args = nil
    server.Env = nil
}
```

### 6. 【信息】Enabled 注释 — `internal/models/mcp_server.go:15`

```go
Enabled bool `json:"enabled"` // 是否启用（前端新增默认 true；编辑沿用原值）
```

## 假设与决策

* **字符校验规则**：仅拒绝空白字符与 KEY 中的 `=`，不限制 `[a-zA-Z0-9_-]` 白名单——避免破坏中文名等现有合法用法（工具名 UTF-8 合法，主要风险是空白）。

* **第 4 项前端 name 校验**：`nameInput.value.trim()` 已去除首尾空白，`/\s/` 只需检测中间空白。

* **第 5 项清零范围**：stdio 保留 command/args/env，sse/http 保留 url/headers；与 config.go validate 的必需字段规则一致。

* 其余 3 处连接/装配逻辑（client.go、agent.go、config.go）不在本次修复范围，行为不变。

## 验证步骤

1. `go build ./...` — 编译通过。
2. `go test ./internal/mcpserver/ ./internal/services/` — 相关包测试通过。
3. `cd frontend && npm run build` — 前端构建通过。
4. `wails build` — 重新编译 `build\bin\jot.exe`。
5. 手工验证：

   * 编辑已有服务器并保存 → 数据库 created\_at 保持不变（可在编辑前后对比）

   * 名称输入含空格 → 前端即时提示；请求头 KEY 含空格 → 提示

   * 新增 stdio 服务器保存后再编辑改为 http → 列表仅显示 URL，无 command 残留

