# 修复停止按钮问题（完整方案）

## 问题总结

用户反馈三个问题：

1. **精炼提示词/联网搜索动画无法停止**：搜索关键词精炼和联网搜索的动画在点击停止后依然持续
2. **点击停止后报错通知**：`context.Canceled` 被当作搜索失败通知给用户
3. **残留气泡 + 不该存库**：LLM 流式阶段点击停止后气泡残留，且不应存库

## 根因分析

### 问题 1 & 2：搜索阶段取消

```
用户点击停止 → ctx 取消
→ 收集循环 for i < len(sources)：
    → r.err != nil → 无条件发射 ai:search-error   ← 不检查 ctx.Err()
    → 前端弹通知 "联网搜索失败: context canceled"   ← 问题2
→ 循环结束 → ctx.Err() 检查 → 发射 ai:stream-done（空）← 动画晚消除
```

### 问题 3：LLM 流式阶段取消（最严重）

```
用户点击停止 → ctx 取消
→ client.Stream() 检测到 ctx.Err()
    → 跳过 OnDone（不保存 assistant 消息到 DB ✅）
    → 跳过 OnError
    → 静默返回 ❌ 不发射任何事件
→ app.go goroutine 退出
→ 前端：收不到 ai:stream-done 也收不到 ai:stream-error
    → streaming bubble 永久残留（打字动画一直在转）
    → 事件监听器泄露（unsubs 永不执行）
```

### 涉及代码位置

| 文件 | 位置 | 问题 |
|------|------|------|
| `app.go` `CallAIStream` | 1659-1668 | 精炼失败直接发射 `ai:stream-error`，不检查 `ctx.Err()` |
| `app.go` `CallAIStream` | 1717-1724 | 搜索收集循环不检查 `ctx.Err()`，无条件发射 `ai:search-error` |
| `app.go` `CallAIStream` | 1878-1947 | `CallAIStream` 返回后无兜底检测，取消时无事件发射 |
| `app.go` `CallAIStreamRegenerate` | 2101-2109 | 同（精炼失败处） |
| `app.go` `CallAIStreamRegenerate` | 2155-2161 | 同（收集循环处） |
| `app.go` `CallAIStreamRegenerate` | 2300-2370 | 同（LLM 返回后无兜底） |
| `ai-chat.js` | 456-468 | 停止按钮未清理搜索动画和气泡 |
| `ai-chat.js` | 2200-2206 | `ai:search-error` 无条件弹通知 |
| `ai-chat.js` | 2244-2340 | `ai:stream-done` 空内容时弹"AI 未返回内容"通知 |

## 改动方案

### 涉及文件

| 文件 | 改动 |
|------|------|
| `app.go` | 6 处：精炼检查(×2) + 收集循环(×2) + LLM 返回后兜底(×2) |
| `ai-chat.js` | 3 处：停止按钮 + 搜索错误处理器 + stream-done 处理器 |

---

### Step 1: 后端 — 精炼失败处增加 ctx 检查

**`app.go` `CallAIStream` ~1659 行** 和 **`CallAIStreamRegenerate` ~2101 行**

`RefineSearchQuery` 返回错误后，先检查 `ctx.Err()`。如果已取消，发射空 `ai:stream-done` 并 return，不报错。

```go
if err != nil {
    if ctx.Err() != nil {
        runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, "", 0.0, 0.0, 0, 0, 0, 0, 0)
        return
    }
    var aiErr *aicli.AIErrorWrapper
    if errors.As(err, &aiErr) {
        runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, aiErr.Err.ToJSON())
    } else {
        ae := aicli.NewAIError(aicli.CategoryUnknown, "搜索关键词精炼失败: "+err.Error())
        runtime.EventsEmit(a.ctx, "ai:stream-error", streamGen, ae.ToJSON())
    }
    return
}
```

### Step 2: 后端 — 搜索收集循环中跳过取消导致的错误

**`app.go` `CallAIStream` ~1717 行** 和 **`CallAIStreamRegenerate` ~2155 行**

在 `r.err != nil` 分支开头检查 `ctx.Err()`。如果已取消，`continue` 跳过，不发射 `ai:search-error`。

```go
if r.err != nil {
    if ctx.Err() != nil {
        continue  // 用户取消，跳过
    }
    errEvent := map[string]interface{}{
        "source": r.source,
        "error":  r.err.Error(),
    }
    errJSON, _ := json.Marshal(errEvent)
    runtime.EventsEmit(a.ctx, "ai:search-error", string(errJSON))
} else if r.result != nil {
    // ... 原有不变
}
```

### Step 3: 后端 — LLM 返回后增加兜底检测

**`app.go` `CallAIStream` ~1947 行** 和 **`CallAIStreamRegenerate` ~2370 行**

`a.aiService.CallAIStream(...)` 返回后，检查 `ctx.Err()`。如果已取消（LLM 流式过程中被停止），发射空 `ai:stream-done` 确保前端清理气泡。

```go
        )
        // 兜底：LLM 流中取消导致 OnDone/OnError 都没触发，补发完成事件
        if ctx.Err() != nil {
            runtime.EventsEmit(a.ctx, "ai:stream-done", streamGen, "", 0.0, 0.0, 0, 0, 0, 0, 0)
        }
    }()
}
```

**注意**：这一步是关键修复，解决了用户说的"点击停止后留下空气泡"问题。  
同时，由于 `OnDone` 不会被调用，assistant 消息**不会**被保存到 DB，满足"AI 消息别写入库里"的要求 ✅。

### Step 4: 前端 — 停止按钮增加气泡和动画清理

**`ai-chat.js` ~455 行**

点击停止时，立即清理搜索动画指示器，并移除当前 streaming 气泡（如果有）。

```javascript
stopBtnEl.addEventListener('click', async () => {
    stopBtnEl.style.display = 'none';
    if (sendBtnEl) sendBtnEl.style.display = '';
    isStreaming = false;
    // 立即清理搜索动画和 AI 气泡
    const lastAssistantEl = messagesInnerEl.querySelector('.ai-msg-assistant:last-child');
    if (lastAssistantEl) {
        const indicator = lastAssistantEl.querySelector('.ai-simple-search-indicator');
        if (indicator) {
            lastAssistantEl.remove();  // 搜索阶段：直接移除气泡
        }
    }
    if (polishBtn) {
        polishBtn.disabled = !(inputEl && inputEl.value.trim().length > 0);
    }
    try {
        await window.go.main.App.CancelAIStream();
    } catch (_) {}
});
```

### Step 5: 前端 — 搜索错误处理器增加停止态防护

**`ai-chat.js` ~2200 行**

弹通知前检查 `isStreaming`，已停止则不弹。

```javascript
const unsubSearchError = window.runtime.EventsOn('ai:search-error', (errJSON) => {
    try {
        const data = typeof errJSON === 'string' ? JSON.parse(errJSON) : errJSON;
        searchSourceStates[data.source] = { source: data.source, status: 'error', error: data.error };
        if (!isStreaming) return;  // 用户主动取消，不弹通知
        window.showNotification?.('联网搜索失败 (' + (sourceLabels[data.source] || data.source) + '): ' + data.error, 'error', 5000);
    } catch (_) {}
});
```

### Step 6: 前端 — `ai:stream-done` 空内容时抑制通知

**`ai-chat.js` ~2262 行**

当 `ai:stream-done` 携带空内容（用户主动取消导致），不弹"AI 未返回内容"通知。

```javascript
const isEmptyMsg = !finalContent || !finalContent.trim();
if (isEmptyMsg) {
    // 用户主动取消的不弹通知
    if (!isStreaming) {
        window.showNotification('AI 未返回内容，请尝试重新生成', 'warning');
    }
    // ... 原有清理逻辑不变
}
```

## 不变的部分

- `RefineSearchQuery` 本身的同步调用不变
- 正常流式完成（不点击停止）的 OnDone/OnError 逻辑不变
- 后端 `OnDone` 回调中保存 assistant 消息、更新 token 的逻辑不变
- 前端 `ai:stream-done` 非空内容的处理和渲染逻辑不变
- 不再额外调用 `SaveAIMessage`，也不会保存被取消的 AI 回复到 DB ✅

## 验证步骤

1. **搜索阶段取消**：启用搜索，发送消息后立即点击停止
   - 搜索动画立即消失
   - 不弹出"联网搜索失败"通知
   - 不弹出"AI 未返回内容"通知
   - 无残留气泡
   - DB 无 assistant 消息写入

2. **LLM 流式阶段取消**：发送消息（搜索关闭），AI 开始回复后点击停止
   - 气泡立即消失
   - 不弹出任何错误通知
   - DB 无 assistant 消息写入

3. **正常完成**（不点击停止）→ 原有行为不变

4. **重新生成**：测试 `CallAIStreamRegenerate` 同样上述场景
