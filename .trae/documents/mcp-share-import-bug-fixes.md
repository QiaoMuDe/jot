# MCP 服务器分享/导入 9 项 Bug 修复计划

## 摘要

修复 MCP 服务器分享/导入功能（含两阶段解析/入库流程）审查中暴露的 9 个问题。涉及前端解析流程、对话框生命周期、事件绑定防御、后端校验一致性。

**不动项**：

* B1（`enabled` 字段导入丢失）— 用户本次未列入修复清单，保留原行为

* B8（闭包引用渲染时快照）— 非 bug

* B9（无取消机制）— "导入应该很快"，暂不做

## 待修复问题

| #   | 描述                                    | 严重    | 文件             |
| --- | ------------------------------------- | ----- | -------------- |
| B2  | 名称空白校验后置（与服务层不一致）                     | 🔴 严重 | mcp\_import.go |
| B3  | `Items[i].ok` 永远 false                | 🔴 严重 | mcp\_import.go |
| B4  | `[]` 返回"无法识别"而非"未找到"                  | 🟡 中  | mcp\_import.go |
| B5  | 阶段 2 失败时对话框已关，无法重试                    | 🟡 中  | main.js        |
| B6  | 分享全部按钮用全局缓存，异步未就绪时返回空                 | 🟡 中  | main.js        |
| B7  | `shareAllBtn.addEventListener` 多次绑定风险 | 🟡 中  | main.js        |
| B10 | `parseMCPImportInput` 三 fallback 重复   | 🟢 小  | mcp\_import.go |
| B11 | `openMCPImportDialog` 立即清空 textarea   | 🟢 小  | main.js        |
| B12 | 错误通知未走统一辅助                            | 🟢 小  | main.js        |

## 现状分析

### 服务层已存在的校验（[mcp\_server\_service.go#L43-L96](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/mcp_server_service.go#L43-L96)）

`Save` 完整校验链：

1. 名称非空
2. 传输方式合法性（stdio/sse/http）
3. stdio 必须 `command`、sse/http 必须 `url`
4. 名称不能含  ` \t\r\n`
5. env/headers KEY 不能含  ` \t\r\n=`
6. 名称全库唯一

但 [mcp\_import.go](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/mcp_import.go) 的 `buildMCPServerFromRaw` **只复刻了 1-3 项**（name 非空 + transport 校验 + 字段必备），未做 4-5 项 → **B2**。

### 当前前端流程（[main.js#L9559-L9634](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9559-L9634)）

```
点击 导入
  ↓
阶段 1 ParseMCPServersImport  → 失败：抖动+通知，对话框不关 ✅
                            → 成功：closeMCPImportDialog() 立刻执行
                                     ↓
                                  阶段 2 ImportMCPServers → 失败：已关无法重试 ❌
```

→ **B5**

### 事件绑定与状态（[main.js#L10166-L10222](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L10166-L10222)）

* `initMCPServerSettings` 每次 panel 初始化都无差别 addEventListener

* `shareAllBtn` 回调使用模块级 `mcpServers`（[main.js#L9409](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9409)），未等待 `loadMCPServers` 完成

* 现有项目惯用模式：直接 `addEventListener`（未做幂等防护），由调用方保证只初始化一次

→ **B6 + B7**

### 对话框清理（[main.js#L9529-L9553](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9529-L9553)）

* `openMCPImportDialog`：进入时**先清空** textarea

* `closeMCPImportDialog`：220ms 后清空 textarea

→ **B11** 用户在 textarea 写了内容 → 按 Esc 关 → 再点导入 → 上次内容没了（vs `mcpServerFormDialog` 行为不一致：表单是 dirty 检测才提示）

### 错误通知（[main.js#L9437-L9440](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9437-L9440)）

项目内 MCP 错误统一用 `mcpErrMsg(err)` 抽取 Wails exception 文本，但部分导入路径未走：

| 位置                      | 当前             | 期望            |
| ----------------------- | -------------- | ------------- |
| L9582 早期空文本             | 硬编码 "请粘贴 JSON" | 硬编码（业务提示，非异常） |
| L9596 parseResult.error | 直传后端中文         | 直传 OK（已是业务中文） |
| L9626 catch             | `mcpErrMsg(e)` | ✅ 已是          |

→ **B12** 实际差异极小，仅做一致性检查/收口。

## 提议的改动

### 改动 1：补齐 `buildMCPServerFromRaw` 校验（B2 + B3 + B10）

**文件**：[internal/services/mcp\_import.go](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/mcp_import.go)

**为什么**：阶段 1 校验不完整 → 阶段 2 才报，UX 割裂

**怎么改**：

1. 名称空白检查（与服务层 `strings.ContainsAny(name, " \t\r\n")` 一致）
2. env/headers KEY 空白/等号检查（与服务层一致）
3. parse 成功时 `res.OK = true`
4. 抽出 `tryUnmarshalArray` 公共函数消除三 fallback 重复

**预计 diff 形状**：

```go
// tryParseInput 统一三格式解析：裸数组 / {servers:[..]} / 单个对象
func tryParseInput(input []byte) ([]rawMCPServer, error) {
    var arr []rawMCPServer
    if err := json.Unmarshal(input, &arr); err == nil && len(arr) > 0 {
        return arr, nil
    }
    var wrapped struct {
        Servers []rawMCPServer `json:"servers"`
    }
    if err := json.Unmarshal(input, &wrapped); err == nil && len(wrapped.Servers) > 0 {
        return wrapped.Servers, nil
    }
    var single rawMCPServer
    if err := json.Unmarshal(input, &single); err == nil &&
        (single.Name != "" || single.Command != "" || single.URL != "") {
        return []rawMCPServer{single}, nil
    }
    return nil, errors.New("无法识别为 [..] / {servers:[..]} / 单个对象")
}
```

```go
func buildMCPServerFromRaw(raw rawMCPServer) (*models.MCPServer, string) {
    name := strings.TrimSpace(raw.Name)
    if name == "" {
        return nil, "名称不能为空"
    }
    if strings.ContainsAny(name, " \t\r\n") {
        return nil, "名称不能包含空格等空白字符"
    }
    transport := strings.TrimSpace(raw.Transport)
    if transport == "" {
        // 隐式推导：有 command → stdio；有 url → sse
        if strings.TrimSpace(raw.Command) != "" {
            transport = "stdio"
        } else if strings.TrimSpace(raw.URL) != "" {
            transport = "sse"
        } else {
            return nil, "transport、command、url 至少需要其一"
        }
    }
    switch transport {
    case "stdio":
        if strings.TrimSpace(raw.Command) == "" {
            return nil, "stdio 传输必须提供 command"
        }
    case "sse", "http":
        if strings.TrimSpace(raw.URL) == "" {
            return nil, transport + " 传输必须提供 url"
        }
    default:
        return nil, "不支持的 transport \"" + transport + "\"（支持 stdio / sse / http）"
    }
    for key := range raw.Env {
        if strings.ContainsAny(key, " \t\r\n=") {
            return nil, "环境变量 KEY「" + key + "」不能包含空白或等号"
        }
    }
    for key := range raw.Headers {
        if strings.ContainsAny(key, " \t\r\n=") {
            return nil, "请求头 KEY「" + key + "」不能包含空白或等号"
        }
    }
    return &models.MCPServer{
        Name:      name,
        Transport: transport,
        Command:   strings.TrimSpace(raw.Command),
        Args:      raw.Args,
        Env:       raw.Env,
        URL:       strings.TrimSpace(raw.URL),
        Headers:   raw.Headers,
        Enabled:   false, // 修复 B1 不在本计划内
    }, ""
}
```

```go
// ParseMCPServersImport 在校验通过路径设置 res.OK = true
items := make([]models.ImportMCPServerItem, 0, len(raws))
allValid := true
for i, raw := range raws {
    res := models.ImportMCPServerItem{Index: i + 1, Name: strings.TrimSpace(raw.Name)}
    if _, errMsg := buildMCPServerFromRaw(raw); errMsg != "" {
        res.Error = errMsg
        allValid = false
    } else {
        res.OK = true  // B3 修复
    }
    items = append(items, res)
}
```

### 改动 2：`parseMCPImportInput` 检测 `[]` 返回友好错误（B4）

**文件**：同一 [mcp\_import.go](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/mcp_import.go)

```go
// 改名为：先 unmarshal 为 interface{} 检查顶层是 []
func parseMCPImportInput(input string) ([]rawMCPServer, error) {
    trimmed := strings.TrimSpace(input)
    if trimmed == "" {
        return nil, errors.New("输入为空")
    }
    // 顶层直接是 [] 但为空数组
    var probe []any
    if err := json.Unmarshal([]byte(trimmed), &probe); err == nil && len(probe) == 0 {
        return nil, errors.New("未找到任何服务器配置（输入为空数组）")
    }
    raws, err := tryParseInput([]byte(trimmed))
    if err != nil {
        return nil, err
    }
    if len(raws) == 0 {
        return nil, errors.New("未找到任何服务器配置")
    }
    return raws, nil
}
```

### 改动 3：阶段 2 失败保留对话框与 textarea（B5）

**文件**：[frontend/src/main.js#L9559-L9634](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9559-L9634)

**为什么**：阶段 2 失败时让用户能看到失败原因 + 可改 JSON 后重导

**怎么改**：把 `closeMCPImportDialog()` 移到阶段 2 调用**成功**之后；先拿结果再决定是否关。

```javascript
// 阶段 2 校验通过 + 实际入库
let results = [];
try {
    results = await window.go.main.App.ImportMCPServers(text);
} catch (e) {
    // binding 异常（不在预期内），关对话框 + 错误通知
    closeMCPImportDialog();
    nm.show('导入失败: ' + mcpErrMsg(e), 'error', 5000);
    return;
}
const safeResults = Array.isArray(results) ? results : [];
const success = safeResults.filter(r => r && r.ok).length;
const failed = safeResults.filter(r => r && !r.ok);

if (failed.length > 0) {
    // 有失败：保留对话框 + textarea,抖动 + 通知（让用户可改后再导）
    const failedNames = failed
        .map(r => (r.name && r.name.trim()) ? r.name : `第${r.index || '?'}条`)
        .join('、');
    nm.show(`已导入 ${success} 条,失败 ${failed.length} 条: ${failedNames},详见日志`, 'error', 5000);
    shakeMCPFormInput(input);
    // 按钮恢复,允许重试
    if (confirmBtn) {
        confirmBtn.disabled = false;
        confirmBtn.textContent = '导入';
    }
    return;
}

// 全部成功：关对话框 + 通知
closeMCPImportDialog();
nm.show(`已导入 ${success} 条 MCP 服务器`, 'success', 3000);
// 刷新列表与全局池
try {
    await loadMCPServers();
    await warmupMCPServers();
} catch (e) { /* 刷新失败不影响主通知 */ }
```

### 改动 4：分享全部按钮用现取 + 幂等绑定（B6 + B7）

**文件**：[frontend/src/main.js#L10213-L10221](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L10213-L10221)

```javascript
// 分享全部按钮 → 复制全部服务器配置为 JSON（点击时现取,避免依赖缓存）
const shareAllBtn = document.getElementById('mcpServerShareAllBtn');
if (shareAllBtn && !shareAllBtn._shareAllBound) {
    shareAllBtn._shareAllBound = true;  // B7 幂等防护
    shareAllBtn.addEventListener('click', async () => {
        let list = mcpServers;
        if (!Array.isArray(list) || list.length === 0) {
            // 缓存为空时现取一次（应对面板首开未加载）
            try {
                list = (await window.go.main.App.GetMCPServers()) || [];
                mcpServers = list;  // 同步全局缓存
            } catch (e) {
                nm.show('获取服务器列表失败: ' + mcpErrMsg(e), 'error');
                return;
            }
        }
        const text = JSON.stringify(buildMCPServersShareJSON(list), null, 2);
        const n = list.length;
        copyMCPServersShare(text, `已复制 ${n} 条服务器配置`, '当前没有可分享的服务器');
    });
}
```

**幂等字段命名**：使用 `shareAllBtn._shareAllBound` 标记避免与未来业务属性冲突（DOM 元素上的私有属性，下划线前缀 + 命名空间）。

**备选方案**：项目里其他按钮（如 addBtn）也用同样模式**没**做防护，保持现状风险低。但用户已列 B7 为要修，按用户意愿加防护。**只针对** **`shareAllBtn`** 防御，不批量改所有按钮（避免越界修改）。

### 改动 5：`openMCPImportDialog` 不主动清空 textarea（B11）

**文件**：[frontend/src/main.js#L9529-L9537](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9529-L9537)

**为什么**：与 `mcpServerFormDialog` 行为一致（表单不主动清空，让用户保留输入）

**怎么改**：

```javascript
function openMCPImportDialog() {
    const dialog = document.getElementById('mcpServerImportDialog');
    if (!dialog) return;
    dialog.style.display = 'flex';
    requestAnimationFrame(() => dialog.classList.add('visible'));
    setTimeout(() => {
        const input = document.getElementById('mcpServerImportInput');
        if (input) input.focus();
    }, 200);
}
```

**附加**：在 `closeMCPImportDialog` 清空逻辑**保留**（成功导入/取消后清理），但确保只对**导入成功**路径触发。简化：成功/失败都清空（保持原逻辑），仅去掉 open 时的清空。

**简化决策**：移除 `openMCPImportDialog` 里的 `input.value = ''`，保留 `closeMCPImportDialog` 里的清空。

### 改动 6：错误通知一致性收口（B12）

**文件**：[frontend/src/main.js#L9576-L9585](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L9576-L9585)

**为什么**：与项目 MCP 错误处理一致

**怎么改**：

* L9582 "请粘贴 JSON" 改为走 helper：硬编码业务提示，**不**走 `mcpErrMsg`（非异常），保留硬编码

* L9596 `parseResult.error` 后端业务中文，保留直传

* L9626 catch 块已走 `mcpErrMsg` ✅

**实际改动**：仅检查确认无遗漏。**预期无代码变更**，仅作为一致性检查清单记录。

## 改动汇总

| 改动 | 文件                                                                                                                                                                                  | 行数        | 修复        |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | --------- |
| 1  | [internal/services/mcp\_import.go](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/mcp_import.go) | +30 / -15 | B2 B3 B10 |
| 2  | 同一文件                                                                                                                                                                                | +10 / -3  | B4        |
| 3  | [frontend/src/main.js](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js)                        | +15 / -10 | B5        |
| 4  | [frontend/src/main.js](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js)                        | +12 / -2  | B6 B7     |
| 5  | [frontend/src/main.js](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%1C%AC%E5%1C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js)                        | +0 / -1   | B11       |
| 6  | 无代码变更                                                                                                                                                                               | 0         | B12 一致性收口 |

总计：约 +57 / -31 行

## 假设与决策

1. **B1 不在范围**：用户未列入修复清单。`enabled` 字段导入仍为 false（与新插入服务器默认一致），用户导入后需手动启用。后续若需可单开任务。
2. **B5 行为变更**："失败保留对话框"是 UX 增强，与项目"导入通知不得展示失败详情"约定无冲突（通知仍只列名称，详情靠日志）。
3. **B7 仅修** **`shareAllBtn`**：不批量改其他按钮（避免越界）。项目其他 addEventListener 也未做防护，但目前未观察到问题。
4. **B11 简化**：仅移除 open 时的清空，保留 close 时的清空。close 清空对全部成功路径生效；失败路径因不关对话框，自然保留。
5. **`mcpServers`** **全局变量**：B6 现取后回填全局，保持与现有 `loadMCPServers` 行为一致。
6. **后端 API 形态不变**：`ImportMCPServers` / `ParseMCPServersImport` 签名不变，TS bindings 无需重生成（struct 字段不变）。

## 验证步骤

### 构建验证

* [ ] `go build ./...` exit 0

* [ ] `npm run build` exit 0

* [ ] `wails build` exit 0

* [ ] 检查 `App.d.ts` 与 `models.ts` 无变化（结构未动）

### 功能验证（人工，按 B2-B12 顺序）

**B2 名称空白校验**：

* 粘贴 `{"name":"foo bar","command":"echo"}` → 阶段 1 报错"名称不能包含空格等空白字符"，对话框不关，textarea 内容保留

* 粘贴 `{"name":"foo","env":{"BAD KEY":"v"}}` → 阶段 1 报错"环境变量 KEY「BAD KEY」..."

**B3 Items\[i].ok**：

* 粘贴 `{"name":"foo","command":"echo"}` → 阶段 1 整体 `ok=true`，items\[0].ok=true

**B4** **`[]`** **错误**：

* 粘贴 `[]` → 阶段 1 报错"未找到任何服务器配置（输入为空数组）"

**B5 失败保留对话框**：

* 库内已有 name="test" 服务器 → 粘贴 `{"name":"test","command":"echo"}` → 阶段 1 通过，阶段 2 失败 → 通知 + 抖动 + 对话框**不关** + textarea **保留**

**B6 现取**：

* 打开设置页后**立即**点"分享全部"（不等待列表加载）→ 触发现取 `GetMCPServers`，不出现"当前没有可分享的服务器"

* 服务端未就绪（异常） → 通知"获取服务器列表失败: ..."

**B7 幂等**：

* 在 console 模拟调用 `initMCPServerSettings()` 多次 → click 按钮只触发 1 次回调

**B10 公共函数**：

* 三种格式（裸数组 / `{servers:[..]}` / 单对象）均能解析

**B11 不清空**：

* 在 textarea 写内容 → 关闭对话框 → 重新打开 → textarea 内容**保留** → 取消后清空

**B12 一致性**：

* Wails exception 路径全部走 `mcpErrMsg(e)`

* 业务中文直接显示

### 回归验证

* [ ] 正常新增/编辑/删除 MCP 服务器无影响

* [ ] 现有 14 主题下 dialog 样式无变化

* [ ] 通知层级（confirm > business）保持

* [ ] Esc 关闭由全局 `handleKeyboardNavigation` 处理仍生效

