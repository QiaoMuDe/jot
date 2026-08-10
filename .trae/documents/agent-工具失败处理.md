# Agent 模式工具失败处理（方案 A + B）

## 摘要

修复 Agent 模式工具执行失败时"直接中断整个回复"的问题：

- **方案 A（主修复）**：用 Eino 官方 `utils.WrapInvokableToolWithErrorHandler` 包装三个工具，把工具错误转成 tool 消息回填给模型，让 ReAct 循环继续，模型自主决定"调整策略 / 直接回答 / 说明失败原因"。
- **方案 B（展示）**：工具失败时发射 `tool_error` 状态事件，前端工具条显示 ❌「联网搜索失败」等失败态；历史回放保持一致。

## 现状分析

- [agent.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go) 中 `Run` 构造 `web_search` / `recall_notes` / `refine_search_query` 三个工具（L101-112）后直接放入 `ToolsNodeConfig.Tools`，**未做错误处理包装**。
- Eino `ToolsNode.Invoke` 对工具返回 error 的默认行为是**直接传播错误**（[tool_node.go L1099-L1102](file:///D:/AppData/gopath/pkg/mod/github.com/cloudwego/eino@v0.9.13/compose/tool_node.go#L1099-L1102)）→ ReAct 循环终止 → `agent.Run` 返回 err → app.go 走 `stream-error`，前端移除整个流式气泡并弹通用"AI 调用失败"。对比 Chat 模式（全部源失败仍用模型知识兜底），Agent 更脆弱。
- Eino 提供官方包装 `WrapInvokableToolWithErrorHandler(t tool.InvokableTool, h ErrorHandler) tool.InvokableTool`（[error_handler.go L81-L89](file:///D:/AppData/gopath/pkg/mod/github.com/cloudwego/eino@v0.9.13/components/tool/utils/error_handler.go#L81-L89)），`h` 为 `func(ctx, error) string`，将错误转成字符串结果返回给模型且不返回 error。
- 前端实时工具状态：`ai:tool-status` 事件目前只处理 `tool_start` / `tool_result` 两种 action（[ai-chat.js L2531-L2544](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2531-L2544)），`streamToolRecords` 全量收集（含落库 `tool_calls`）。
- 历史回放：`renderToolCalls` 只渲染 `action === 'tool_start'` 记录并统一显示"已完成X"（[ai-chat.js L3917-L3960](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3917-L3960)），无失败态。
- CSS：`.ai-tool-status-item.is-done` 绿色勾样式已存在（[ai-chat.css L456-L462](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L456-L462)），无 `is-error` 样式。

## 变更清单

### 1. `internal/agent/agent.go` — 包装工具 + 失败事件

**import**：新增 `"github.com/cloudwego/eino/components/tool/utils"`。

**工具构造处（L101-112）**：写一个内部辅助函数 `wrapToolWithError(name string, t tool.InvokableTool) tool.InvokableTool`，闭包捕获 `emit` 与 `toolRecords`：

```go
// wrapToolWithError 包装工具：执行失败时不中断 ReAct 循环，
// 错误文本回填给模型继续推理，同时发射 tool_error 事件供前端展示失败态。
func wrapToolWithError(name string, t tool.InvokableTool, emit EmitFn, records *[]toolCallRecord) tool.InvokableTool {
    return utils.WrapInvokableToolWithErrorHandler(t, func(ctx context.Context, err error) string {
        // 用户取消：不误报失败，直接返回错误文本（循环会随 ctx 终止）
        if ctx.Err() != nil {
            return err.Error()
        }
        rec := toolCallRecord{
            Action: "tool_error",
            Name:   name,
            Result: truncateRunes(err.Error(), maxToolResultLen),
        }
        *records = append(*records, rec)
        if b, err := json.Marshal(rec); err == nil {
            emit("ai:tool-status", string(b))
        }
        return "工具执行失败：" + err.Error() + "。请依据错误信息调整策略，或直接基于已有信息回答用户。"
    })
}
```

构造处改为：

```go
tools := []tool.BaseTool{
    wrapToolWithError("refine_search_query", &refineSearchQueryTool{ai: s.deps.AI}, emit, &toolRecords),
    wrapToolWithError("web_search", &webSearchTool{...}, emit, &toolRecords),
    wrapToolWithError("recall_notes", &recallNotesTool{...}, emit, &toolRecords),
}
```

> 注：`toolRecords` 在构造前已声明（需把当前 `var toolRecords []toolCallRecord` 声明位置上移到工具构造之前）。

**`emitToolResult`（L369-378）**：增加"失败跳过"——若 `*records` 中该 name 最近一条记录为 `tool_error`，则不发射 `tool_result`（避免 `✓` 覆盖 `❌`）：

```go
func emitToolResult(emit EmitFn, records *[]toolCallRecord, name, result string) {
    // 该工具刚失败（已发 tool_error），跳过结果事件，保持失败态
    for i := len(*records) - 1; i >= 0; i-- {
        if (*records)[i].Name != name {
            continue
        }
        if (*records)[i].Action == "tool_error" {
            return
        }
        break
    }
    // ...原有逻辑不变
}
```

### 2. `frontend/src/js/ai-chat.js` — 失败态实时展示 + 历史回放

**实时事件处理（L2539-2543）**：新增 `tool_error` 分支：

```js
} else if (payload.action === 'tool_error') {
    showToolStatusError(payload);
}
```

**新增 `showToolStatusError`（放在 `showToolStatusDone` 之后）**：

```js
/** 展示工具调用失败状态（tool_error） */
const showToolStatusError = (payload) => {
    const name = payload.name || 'tool';
    let item = toolStatusItems[name];
    if (!item) {
        const list = ensureToolStatusList();
        item = { el: null, iconEl: null, textEl: null };
        item.el = document.createElement('div');
        item.el.className = 'ai-tool-status-item';
        item.el.dataset.name = name;
        item.iconEl = document.createElement('span');
        item.iconEl.className = 'ai-tool-status-icon';
        item.textEl = document.createElement('span');
        item.textEl.className = 'ai-tool-status-text';
        item.el.appendChild(item.iconEl);
        item.el.appendChild(item.textEl);
        list.appendChild(item.el);
        toolStatusItems[name] = item;
    }
    let label = '工具执行失败';
    if (name === 'web_search') label = '联网搜索失败';
    else if (name === 'recall_notes') label = '笔记检索失败';
    else if (name === 'refine_search_query') label = '搜索词精炼失败';
    const reason = payload.result ? String(payload.result) : '';
    if (reason) label += '：' + (reason.length > 40 ? reason.slice(0, 40) + '…' : reason);
    item.el.classList.remove('is-done');
    item.el.classList.add('is-error');
    item.iconEl.textContent = '❌';
    item.textEl.textContent = label;
    scrollToBottom();
};
```

**历史回放 `renderToolCalls`（L3917-3960）**：先收集每个 name 是否有失败记录，再渲染：

```js
// 失败名集合：tool_error 记录优先展示失败态
var failedNames = {};
for (var i = 0; i < toolCalls.length; i++) {
    if (toolCalls[i].action === 'tool_error' && toolCalls[i].name) {
        failedNames[toolCalls[i].name] = true;
    }
}
```

渲染 `tool_start` 记录时判断：若 `failedNames[name]` → `item.className = 'ai-tool-status-item is-error'`，icon `❌`，文案「联网搜索失败 / 笔记检索失败 / 搜索词精炼失败」；否则维持现有 `✓ 已完成X` 逻辑。

### 3. `frontend/src/css/components/ai-chat.css` — 失败态样式

在 `.is-done` 样式块后新增：

```css
.ai-tool-status-item.is-error .ai-tool-status-icon {
    color: var(--danger, #e5484d);
}

.ai-tool-status-item.is-error .ai-tool-status-text {
    color: var(--danger, #e5484d);
}
```

### 不改动的部分

- `types.go`：`toolCallRecord` 已有 `Action/Name/Result` 字段，`tool_error` 直接复用，无需新增。
- `tools.go`：工具内部逻辑不变（单源失败仍跳过、refine 仍降级返回原词）；`web_search` 全源失败、`recall_notes` 失败返回的 error 由包装层接管。
- 后端落库链路：`tool_calls` 序列化 `toolRecords`，`tool_error` 记录随之持久化，无需改 `app.go`。

## 假设与决策

1. **失败文案回填模型**：`工具执行失败：<原因>。请依据错误信息调整策略，或直接基于已有信息回答用户。` 引导模型继续推理而非复读错误。
2. **用户取消不误报**：`ctx.Err() != nil` 时 handler 只回填错误文本、不发 `tool_error`；循环随 ctx 取消正常终止，与现有取消分支一致。
3. **失败态优先级**：同一工具 `tool_error` 覆盖 `tool_result`（后端跳过 result 事件 + 前端回放优先渲染失败态），保证实时与历史一致。
4. **文案前缀**：前端失败标签统一「联网搜索失败 / 笔记检索失败 / 搜索词精炼失败」，附原因截断 40 字符，避免工具条超长。
5. **不做**：不在 `agent.go` 对"单源失败"做提示（其余源成功时结果可用，无感更优）；不改 `stream-error` 兜底路径。

## 验证步骤

1. `go build ./...` 通过（验证 import 与包装签名）。
2. 手动测试（运行 Wails 应用）：
   - **失败可见 + 继续回答**：把 Tavily / 知乎密钥配错（或断网）→ Agent 提问需联网问题 → 工具条显示 ❌「联网搜索失败」→ 模型仍输出回答（可含"未能联网"说明），不弹通用错误、不中断。
   - **回归成功场景**：正常配置 → 搜索成功 → 工具条显示 ✓「已完成搜索」。
   - **取消不误报**：发送后点停止 → 无 ❌ 失败条误显示。
   - **历史回放一致**：发送一条含失败工具的 Agent 消息 → 切换会话再切回（或重开会话）→ 历史消息工具条显示 ❌ 失败态，与实时一致。
3. 前端无编译步骤（原生 JS/CSS），以上手动验证即可。
