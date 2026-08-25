# MCP 服务器导入后端化 + 通知文案优化

## Summary

将 MCP 导入的 JSON 解析、字段校验、循环入库逻辑从**前端迁到后端**，利用现有 `fastlog` 直接写 `logs/app.log`，**不再需要为前端日志新建 binding**。同时按用户决策调整通知文案与按钮状态：

* 失败通知仅列**失败条目名称**（不列具体原因），指明"详情见日志"

* 导入期间**禁用导入按钮**，结束后恢复

## Current State Analysis

### 现状要点

| 项              | 现状                                                                                   | 位置                                                                                                                                                                                                   |
| -------------- | ------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 前端解析           | 100+ 行 `parseMCPServersImportJSON` 三格式容错 + 字段校验                                      | [main.js#L9538-L9587](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9538-L9587)                              |
| 前端循环入库         | `for` 循环逐条 `await SaveMCPServer`，收集 success/failed                                   | [main.js#L9621-L9684](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9621-L9684)                              |
| 失败日志           | `console.error('[MCP 导入失败详情]', failDetails)`                                         | [main.js#L9668-L9670](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9668-L9670)                              |
| 按钮状态           | 已禁用，但**未**显示"导入中..."文本                                                               | [main.js#L9641-L9642, L9681](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9641-L9642)                       |
| 通知文案           | `已导入 N 条，失败 M 条，详情见应用日志`                                                             | [main.js#L9679-L9680](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9679-L9680)                              |
| 现有 Service     | `mcpServerService.Save(server *MCPServer) error`                                     | [mcp\_server\_service.go#L44-L107](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/mcp_server_service.go#L44-L107) |
| 现有 App binding | `GetMCPServers / SaveMCPServer / DeleteMCPServer / TestMCPServer / WarmupMCPServers` | [app.go#L2323-L2428](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/app.go#L2323-L2428)                                             |
| 后端 logger      | `a.LogSvc.Logger.Errorw / Infow` 等                                                   | 全 app.go 大量使用                                                                                                                                                                                        |
| 日志文件           | `<数据目录>/logs/app.log`                                                                | [log\_service.go#L41](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/log_service.go#L41)                          |

### 关键约束（来自项目记忆）

* Wails 绑定**复杂数据用 struct 不用多返回值**（避免数据截断）

* 错误消息必须走 `ClassifyError` 提供**中文友好提示**

* Import 结果通知**不展示逐行详细**，仅聚合 + "详情见应用日志"

* CSS/HTML 改动后必须 `npm run build` + `wails build`

## Proposed Changes

### 1. 后端：新增导入 service 函数

**新文件** `internal/services/mcp_import.go`：

```go
package services

import (
    "encoding/json"
    "errors"
    "fmt"
    "strings"

    "gitee.com/MM-Q/fastlog"
    "github.com/jot/internal/models"  // 按实际 import 路径
)

// rawMCPServer 导入 JSON 的中间结构
type rawMCPServer struct {
    Name      string            `json:"name"`
    Transport string            `json:"transport"`
    Command   string            `json:"command"`
    Args      []string          `json:"args"`
    Env       map[string]string `json:"env"`
    URL       string            `json:"url"`
    Headers   map[string]string `json:"headers"`
}

// ImportMCPServerItem 单条导入结果
type ImportMCPServerItem struct {
    Index int    `json:"index"` // 1-based
    Name  string `json:"name"`
    OK    bool   `json:"ok"`
    Error string `json:"error"`
}

// ImportMCPServers 解析 JSON 并批量入库,所有错误自动写 logs/app.log
// input 支持: 裸数组 / {servers:[...]} / 单个对象
// 返回: 每条入库结果(JSON 整体解析失败时返回单条 {ok:false, error:...})
func ImportMCPServers(logSvc *LogService, mcpSvc *MCPServerService, input string) []ImportMCPServerItem {
    logSvc.Logger.Debugw("ImportMCPServers", fastlog.Int("inputLen", len(input)))

    raws, err := parseMCPImportInput(input)
    if err != nil {
        logSvc.Logger.Errorw("ImportMCPServers JSON 解析失败", fastlog.Error(err))
        return []ImportMCPServerItem{{Index: 0, OK: false, Error: "JSON 解析失败: " + err.Error()}}
    }
    if len(raws) == 0 {
        return []ImportMCPServerItem{{Index: 0, OK: false, Error: "未找到任何服务器配置"}}
    }

    results := make([]ImportMCPServerItem, 0, len(raws))
    for i, raw := range raws {
        res := ImportMCPServerItem{Index: i + 1, Name: strings.TrimSpace(raw.Name)}

        server, errMsg := buildMCPServerFromRaw(raw)
        if errMsg != "" {
            res.Error = errMsg
            logSvc.Logger.Errorw("ImportMCPServers 字段校验失败",
                fastlog.Int("index", i+1),
                fastlog.String("name", res.Name),
                fastlog.String("reason", errMsg))
        } else {
            if err := mcpSvc.Save(server); err != nil {
                res.Error = err.Error()
                logSvc.Logger.Errorw("ImportMCPServers 入库失败",
                    fastlog.Int("index", i+1),
                    fastlog.String("name", server.Name),
                    fastlog.Error(err))
            } else {
                res.OK = true
                logSvc.Logger.Infow("ImportMCPServers 入库成功",
                    fastlog.Int("index", i+1),
                    fastlog.String("name", server.Name))
            }
        }
        results = append(results, res)
    }
    return results
}

// parseMCPImportInput 三格式容错
func parseMCPImportInput(s string) ([]rawMCPServer, error) {
    s = strings.TrimSpace(s)
    if s == "" {
        return nil, errors.New("输入为空")
    }

    // 1) 裸数组
    var arr []rawMCPServer
    if err := json.Unmarshal([]byte(s), &arr); err == nil && len(arr) > 0 {
        return arr, nil
    }
    // 2) {servers:[...]}
    var wrapped struct {
        Servers []rawMCPServer `json:"servers"`
    }
    if err := json.Unmarshal([]byte(s), &wrapped); err == nil && len(wrapped.Servers) > 0 {
        return wrapped.Servers, nil
    }
    // 3) 单对象
    var single rawMCPServer
    if err := json.Unmarshal([]byte(s), &single); err == nil &&
        (single.Name != "" || single.Command != "" || single.URL != "") {
        return []rawMCPServer{single}, nil
    }
    return nil, errors.New("无法识别为 [..] / {servers:[..]} / 单个对象")
}

// buildMCPServerFromRaw 字段校验 + 推导 transport + 构造 MCPServer
// 返回 nil + errMsg 表示校验失败;返回 server + "" 表示成功
func buildMCPServerFromRaw(raw rawMCPServer) (*models.MCPServer, string) {
    name := strings.TrimSpace(raw.Name)
    if name == "" {
        return nil, "名称不能为空"
    }

    // 推导 transport
    transport := strings.ToLower(strings.TrimSpace(raw.Transport))
    if transport == "" {
        switch {
        case raw.Command != "":
            transport = "stdio"
        case raw.URL != "":
            transport = "sse" // 默认 sse,与项目既有约定一致
        default:
            return nil, "缺少 command 或 url"
        }
    }
    if transport != "stdio" && transport != "sse" && transport != "http" {
        return nil, fmt.Sprintf("transport 非法: %q", raw.Transport)
    }

    // 字段冲突清零(与服务层保持一致)
    server := &models.MCPServer{
        Name:    name,
        Enabled: false, // 导入默认禁用,需用户手动启用
    }
    switch transport {
    case "stdio":
        if raw.Command == "" {
            return nil, "stdio 模式必须提供 command"
        }
        server.Transport = "stdio"
        server.Command = raw.Command
        if len(raw.Args) > 0 {
            server.Args = append([]string(nil), raw.Args...)
        }
        if len(raw.Env) > 0 {
            server.Env = make(map[string]string, len(raw.Env))
            for k, v := range raw.Env {
                server.Env[k] = v
            }
        }
    case "sse", "http":
        if raw.URL == "" {
            return nil, transport + " 模式必须提供 url"
        }
        server.Transport = transport
        server.URL = raw.URL
        if len(raw.Headers) > 0 {
            server.Headers = make(map[string]string, len(raw.Headers))
            for k, v := range raw.Headers {
                server.Headers[k] = v
            }
        }
    }
    return server, ""
}
```

### 2. 后端：App 层新增 binding

**修改** `app.go`，在 `SaveMCPServer` 旁新增：

```go
// ImportMCPServers 接收用户粘贴的 JSON,解析后批量入库 MCP 服务器
// 解析/校验/入库错误均自动写入 logs/app.log
// 返回每条处理结果(成功/失败+原因),仅供 UI 聚合通知
func (a *App) ImportMCPServers(jsonStr string) []services.ImportMCPServerItem {
    return services.ImportMCPServers(a.LogSvc, a.mcpServerService, jsonStr)
}
```

**字段访问确认**（不改动结构）：

* `a.LogSvc` ← 现有字段（log\_service 实例）

* `a.mcpServerService` ← 现有字段（小写，复用现有 service）

### 3. 前端：删除解析/循环逻辑，简化为单次 binding 调用

**修改** `frontend/src/main.js`：

#### 3.1 删除 `parseMCPServersImportJSON`（约 50 行，[L9538-L9587](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9538-L9587)）

#### 3.2 重写 `handleMCPImport`（替换 [L9621-L9684](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9621-L9684)）

```javascript
async function handleMCPImport() {
    const dialog = document.getElementById('mcpServerImportDialog');
    const input = document.getElementById('mcpServerImportInput');
    if (!input) return;
    const text = (input.value || '').trim();
    if (!text) {
        nm.show('请粘贴 JSON', 'error');
        shakeMCPFormInput(input);
        input.focus();
        return;
    }

    // 防重复提交:禁用并显示"导入中"
    const confirmBtn = document.getElementById('mcpServerImportConfirmBtn');
    if (confirmBtn) {
        confirmBtn.disabled = true;
        confirmBtn.textContent = '导入中...';
    }

    try {
        // 后端做:解析 + 校验 + 循环入库 + 写日志(统一在 logs/app.log)
        const results = await window.go.main.App.ImportMCPServers(text);

        // 聚合结果
        const success = results.filter(r => r.ok).length;
        const failed = results.filter(r => !r.ok);
        const totalFailed = failed.length;

        // 通知文案:仅列失败条目名称,不列具体原因
        let summary;
        if (totalFailed === 0) {
            summary = `已导入 ${success} 条 MCP 服务器`;
        } else {
            const failedNames = failed
                .map(r => r.name || `第${r.index}条`)
                .join('、');
            summary = `已导入 ${success} 条,失败 ${totalFailed} 条: ${failedNames},详见日志`;
        }
        const level = totalFailed === 0 ? 'success' : (success === 0 ? 'error' : 'warn');
        nm.show(summary, level, totalFailed === 0 ? 3000 : 5000);

        // 关闭对话框(不论成功失败,避免阻塞)
        closeMCPImportDialog();

        // 刷新列表与池
        try {
            await loadMCPServers();
            await warmupMCPServers();
        } catch (e) { /* 刷新失败不影响主通知 */ }
    } catch (e) {
        // 整体调用失败(binding 未就绪 / panic)
        const msg = mcpErrMsg(e);
        nm.show('导入失败: ' + msg, 'error', 5000);
        shakeMCPFormInput(input);
    } finally {
        if (confirmBtn) {
            confirmBtn.disabled = false;
            confirmBtn.textContent = '导入';
        }
    }
}
```

### 4. 不动项

* `index.html` 导入对话框 DOM（含按钮文案"导入"/`id="mcpServerImportConfirmBtn"`）：**不动**

* `settings-panel.css` 按钮 + 弹窗样式：**不动**

* `parseMCPServersImportJSON` 之外的旧分享/解析辅助函数：**不动**

* `mcpServerService.Save` 内部逻辑：**不动**（继续用其"ID==0 新增 / 名称唯一 / 字段冲突清零"全部校验）

* 后端 `LogFrontend` binding：**本次不需要**（项目记忆提到但不引入）

## Assumptions & Decisions

| #  | 决策                                                         | 理由                                                      |
| -- | ---------------------------------------------------------- | ------------------------------------------------------- |
| 1  | 解析/校验/入库全部后端做，前端仅一次 binding 调用                             | 用户明确选择；用现有 `Errorw` 写日志，无需新增 `LogFrontend`              |
| 2  | 返回 `[]ImportMCPServerItem` 用 struct（index/name/ok/error）   | 符合 Wails "复杂数据用 struct" 约定；避免多返回值截断                     |
| 3  | 失败通知只列**条目名称**，不列 error 详细                                 | 用户决策："就写失败了哪些，详见日志就行"                                   |
| 4  | 导入按钮在调用期间 disabled + 文案切"导入中..."                           | 用户决策；防止连点；提示用户状态                                        |
| 5  | 导入默认 `enabled: false`                                      | 与前端现有行为一致（L9657 `enabled: false`），需要用户主动启用避免自动连接未审核的服务器 |
| 6  | `transport` 缺省时按 `command` 存在→`stdio`、否则 `url` 存在→`sse` 默认 | 沿用前端既有 `parseMCPServersImportJSON` 行为，保持兼容              |
| 7  | `enabled: false` 写入                                        | 沿用前端既有逻辑                                                |
| 8  | `closeMCPImportDialog` 总是调用（成功/部分失败/全部失败均关）                | 避免阻塞用户处理；失败详情已写日志                                       |
| 9  | 解析失败返回单条 `{ok:false, error:"JSON 解析失败: ..."}` 而非抛错         | 错误结构化更友好；前端统一处理；避免 throw 触发不同代码路径                       |
| 10 | 不删 `mcpErrMsg` 等既有工具函数                                     | 仍用于 `addMCPServer` 提交等场景；本次只在导入流程不再使用                   |

## Verification

### Step 1: 后端编译

```bash
cd d:/资源池/下水道/Dev/本地项目/jot
go build ./...
```

预期：exit 0，无编译错误。

### Step 2: 前端构建

```bash
cd d:/资源池/下水道/Dev/本地项目/jot/frontend
npm run build
```

预期：exit 0。

### Step 3: wails build

```bash
cd d:/资源池/下水道/Dev/本地项目/jot
wails build
```

预期：exit 0。

### Step 4: 运行时手动验证

打开应用 → 设置页 → MCP 服务器：

| 场景                           | 预期                                                                              |
| ---------------------------- | ------------------------------------------------------------------------------- |
| 粘贴 `[]`（空数组）                 | 通知 "未找到任何服务器配置"，按钮恢复                                                            |
| 粘贴 1 条合法 stdio               | 通知 "已导入 1 条 MCP 服务器"（success），列表多一项 disabled                                    |
| 粘贴 1 条 url 缺 transport       | 默认推导为 sse 成功（与前端既有行为一致）                                                         |
| 粘贴 3 条：2 条合法 + 1 条 `name` 为空 | 通知 "已导入 2 条,失败 1 条: 第3条,详见日志"；`logs/app.log` 出现 "字段校验失败 name=第3条 reason=名称不能为空" |
| 粘贴 2 条同名 stdio               | 通知 "已导入 1 条,失败 1 条: name2,详见日志"；日志出现 "名称 name2 已存在"                             |
| 粘贴非法 JSON（如 `{a:}`）          | 通知 "JSON 解析失败: ..."，按钮恢复，对话框不关                                                  |
| 粘贴 `{servers:[{...},{...}]}` | 成功导入两条                                                                          |
| 粘贴单个对象 `{...}`               | 成功导入一条                                                                          |
| 导入期间连点导入按钮                   | 第二次点击无响应（disabled）                                                              |
| 导入完成后按钮                      | 文案恢复"导入"，disabled 解除                                                            |
| `logs/app.log` 内容            | 出现 "ImportMCPServers" 系列日志（Debugw + Errorw/Infow）                               |

### Step 5: 回归测试

| 项                       | 预期                |
| ----------------------- | ----------------- |
| 现有 MCP 添加/编辑/测试/删除/分享功能 | 不受影响              |
| 既有 console.error 列表     | 79 处不变（本计划不动其他位置） |
| 其他设置页                   | 不受影响              |

## Files Changed

| 文件                                | 变更类型   | 增量                                                                  |
| --------------------------------- | ------ | ------------------------------------------------------------------- |
| `internal/services/mcp_import.go` | **新增** | \~150 行                                                             |
| `app.go`                          | 修改     | +1 binding（约 5 行），位置在 `SaveMCPServer` 旁                             |
| `frontend/src/main.js`            | 修改     | -50 行（删 `parseMCPServersImportJSON`）+ +约 40 行（重写 `handleMCPImport`） |

## 已知边界 / 不做

* ❌ 不引入前端 `LogFrontend` binding（本计划前一轮已评估，但因解析后端化而不再需要）

* ❌ 不批量替换其他 78 处 `console.error`（保留作为后续独立计划）

* ❌ 不在导入流程做"跳过已存在 / 覆盖"交互（沿用现有 service：name 重名直接报错）

* ❌ 不做导入前的格式预览（用户决策保持简洁：粘贴即入，错误聚合展示）

## 实施顺序

1. 新建 `internal/services/mcp_import.go`（含 3 函数 + 1 类型 + 1 struct）
2. `app.go` 新增 `ImportMCPServers` binding
3. `go build ./...` 验证后端
4. `main.js` 删除 `parseMCPServersImportJSON` + 重写 `handleMCPImport`
5. `npm run build` + `wails build`
6. 运行时手动验证（Step 4 全部场景）

