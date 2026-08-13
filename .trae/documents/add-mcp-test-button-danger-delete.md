# 计划：MCP 服务器列表新增「测试连接」按钮 + 删除按钮警告配色

## Summary

设置页「MCP 服务器」面板的每个列表条目（[buildMCPServerItem](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L9133-L9202)）当前仅有「启用开关 + 编辑 + 删除」。本计划：

1. 在条目操作区新增「测试」按钮：点击后调用后端按该服务器现有配置（stdio 启动命令 / sse、http 地址）实际发起 MCP 连接 + 握手 + 工具发现，返回是否可用及发现的工具数。
2. 删除按钮配色改为警告红：常态（浅红底 + 红字 + 红边）、悬浮（实心红底白字）、按压（深红 + 缩放），全部基于 `--danger` 变量随 14 套主题自适应。

## Current State Analysis

* 后端绑定：`app.go` 已有 `GetMCPServers` / `SaveMCPServer` / `DeleteMCPServer`（[app.go L2697-L2715](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2697-L2715)），但 **未 import** **`jot/internal/mcpserver`**。

* 连接能力：`internal/mcpserver/client.go` 的 `Connect(ctx, Server)` 已完成「构建客户端 + Start + Initialize 握手」，内部带 10s 超时（`ConnectTimeout`），错误文案已中文化；`internal/mcpserver/tools.go` 的 `OpenSession(ctx, Server)` 进一步做工具发现（ListTools），返回 `*Session{Tools, Close()}`，可给出「发现 N 个工具」。

* 配置映射：`models.MCPServer` → `mcpserver.Server` 的字段映射逻辑已存在于 `mcpserver.LoadFromDB`（config.go L45-L70），app.go 侧需一个等价的小型转换。

* 前端测试按钮模式：设置页已有成熟范例（Tavily / 知乎「测试连接」按钮），使用 `setBtnLoading(btn, true/false)` + `nm.show(..., 'success'/'error')`（[main.js L2481-L2504](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2481-L2504)），本计划沿用。

* 按钮基础样式：`.btn:active { transform: scale(0.97) }` 已全局存在（settings-panel.css L16-L18）；`.btn-danger` 是实心红（全局共用，不影响）。删除按钮当前用 `.btn-cancel`（灰），需替换为专用警告样式。

* Wails 绑定：新增绑定方法后需重新生成 `frontend/wailsjs/go/main/App.js` / `App.d.ts`（`wails build` 或 `wails generate module` 会自动生成）。

## Proposed Changes

### 1. `app.go` — 新增 `TestMCPServer` 绑定（后端核心）

位置：`GetMCPServers` 附近（约 L2715 之后）。

**a. import 增加** **`jot/internal/mcpserver`**（现有 import 列表无此项）。

**b. 新增结果结构体 + 转换辅助 + 绑定方法**：

```go
// TestMCPServerResult MCP 服务器连接测试结果（Wails 绑定需用结构体传复杂数据）
type TestMCPServerResult struct {
    OK      bool   `json:"ok"`       // 是否连接可用
    ToolNum int    `json:"tool_num"` // 连接成功时发现的工具数
    Message string `json:"message"`  // 中文提示文案（失败时为具体原因）
}

// toMCPServerConfig models.MCPServer → mcpserver.Server（与 LoadFromDB 的字段映射一致）
func toMCPServerConfig(m models.MCPServer) mcpserver.Server { ... }

// TestMCPServer 按 ID 加载服务器配置并实测连接（连接 + 握手 + 工具发现）。
// 无论服务器是否启用均可测试；内部超时由 ConnectTimeout 兜底，不会卡死。
func (a *App) TestMCPServer(id uint) TestMCPServerResult {
    var rec models.MCPServer
    if err := a.db.First(&rec, id).Error; err != nil {
        return TestMCPServerResult{OK: false, Message: "MCP 服务器不存在或已被删除"}
    }
    sess, err := mcpserver.OpenSession(context.Background(), toMCPServerConfig(rec))
    if err != nil {
        return TestMCPServerResult{OK: false, Message: err.Error()} // 错误已是中文
    }
    defer sess.Close()
    return TestMCPServerResult{OK: true, ToolNum: len(sess.Tools), Message: "连接成功"}
}
```

为什么用 `OpenSession` 而非仅 `Connect`：工具发现（ListTools）是判断「该传输方式配置是否真正可用」的完整闭环，成功提示还能附带工具数，UX 更友好。`a.db` 已在 App 结构体中存在，直接使用即可（`mcpServerService` 无按 ID 查询方法，无需新增）。

### 2. `frontend/src/main.js` — 条目操作区新增「测试」按钮 + 删除按钮换类

**a.** **`buildMCPServerItem()`（L9172-L9197）操作区调整**：在 toggle 与 编辑 之间插入测试按钮，并把删除按钮 class 从 `btn btn-sm btn-cancel` 改为 `btn btn-sm mcp-server-del-btn`：

```js
const testBtn = document.createElement('button');
testBtn.className = 'btn btn-sm btn-secondary';
testBtn.textContent = '测试';
testBtn.title = '测试该传输方式的连接是否可用';
testBtn.addEventListener('click', () => testMCPServer(srv, testBtn));
```

（沿用 `.btn-secondary` 与编辑按钮视觉一致，不新增样式。）

**b. 新增** **`testMCPServer(srv, btn)`** **异步函数**（放在 `toggleMCPServer` 之后）：

```js
async function testMCPServer(srv, btn) {
    setBtnLoading(btn, true);
    try {
        const res = await window.go.main.App.TestMCPServer(srv.id);
        if (res && res.ok) {
            const toolText = res.tool_num > 0 ? `，发现 ${res.tool_num} 个工具` : '';
            nm.show(`「${srv.name}」连接成功${toolText}`, 'success');
        } else {
            nm.show(`「${srv.name}」连接失败：${mcpErrMsg((res && res.message) || '')}`, 'error');
        }
    } catch (e) {
        nm.show(`「${srv.name}」测试出错：${mcpErrMsg(e)}`, 'error');
    } finally {
        setBtnLoading(btn, false);
    }
}
```

复用现有 `setBtnLoading`（L2147）与 `mcpErrMsg`（L9094）。

### 3. `frontend/src/css/components/settings-panel.css` — 删除按钮警告配色

在 MCP 服务器列表样式区（`.mcp-server-item-actions` 附近）新增：

```css
/* 条目删除按钮：警告红（常态浅红描边 → 悬浮实心红 → 按压深红） */
.mcp-server-del-btn {
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
  border-color: color-mix(in srgb, var(--danger) 38%, transparent);
}
.mcp-server-del-btn:hover {
  background: var(--danger);
  color: #fff;
  border-color: var(--danger);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--danger) 35%, transparent);
}
.mcp-server-del-btn:active {
  background: color-mix(in srgb, var(--danger) 78%, #1a1a1a);
  color: #fff;
  border-color: var(--danger);
}
```

说明：

* 三态均围绕 `--danger`（红）展开，14 套主题各自定义该变量（variables.css 已确认），无需硬编码色值，暗色主题下依然醒目。

* 按压态背景叠加少量深色形成「更深红」的按压反馈；`.btn:active` 全局的 `scale(0.97)` 保留缩放反馈。

* 不修改全局 `.btn-danger`（确认对话框等处仍在用），避免副作用。

## Assumptions & Decisions

* 测试按钮标签用「测试」（非「测试连接」），避免条目操作区过宽；悬停 title 说明用途。

* 测试按钮与「编辑」同样式（`btn-secondary`），用户未要求测试按钮特殊配色。

* 测试逻辑放在 `app.go`（package main）而非 services 层：`internal/services` 因依赖链不能反向 import `internal/mcpserver`，而 main 包可以。

* 测试不要求服务器已启用（Enabled 忽略），只要配置存在即可实测。

* 使用 `context.Background()`：`Connect`/`OpenSession` 内部各自有 10s 超时兜底，不会无限阻塞。

## Verification

1. 后端编译：`go build ./...`。
2. 前端检查：`cd frontend && npm run lint`（新增代码需 0 error）。
3. 重新生成 Wails 绑定并构建（记忆约束：前端改动必须重编译才生效）：`wails generate module`（或直接 `wails build`）。
4. 手动验证：

   * 新增 stdio 服务器（如 `npx -y @modelcontextprotocol/server-memory`）→ 点「测试」→ 提示「连接成功，发现 N 个工具」。

   * 配置一个不可达的 sse/http 地址 → 点「测试」→ 10s 内提示具体中文失败原因（超时/地址错误），且按钮恢复可点。

   * 删除按钮：常态为浅红描边、悬浮变实心红、按住变深红；明暗主题下均正常。

