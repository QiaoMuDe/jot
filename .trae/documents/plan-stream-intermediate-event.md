# 中间文本与最终回答事件分离

## 概述

将 ReAct 循环中模型的中间文本（工具调用前的过渡性输出）与最终回答分离为独立事件，前端临时展示中间文本，最终回答到达时自动替换。

## 当前状态

* 中间文本和最终回答都通过 `ai:stream-chunk` 推送，前端无法区分

* 前端用 `hasReceivedToolStart` hack 清除中间文本

* 后端 `consumeAssistantStream` 内部直接 emit `ai:stream-chunk`，调用方无法控制

## 改动文件

### 1. `internal/agent/agent.go` — 后端事件分离

#### 1a. `consumeAssistantStream` 签名变更

**位置**: L941-982

**当前**: 函数内部直接 `emit("ai:stream-chunk", chunk.Content)`
**改为**: 不再 emit 正文 chunk，改为收集后返回；由调用方根据是否有 ToolCalls 决定 emit 哪个事件

具体改动：

* 移除 L962 的 `emit("ai:stream-chunk", chunk.Content)`

* 新增返回值：将 `full` 拆为返回 `full`（合并消息）+ `content`（正文字符串），或者直接不改签名，让调用方从 `full.Content` 读取

* **推荐方案**：不改签名，而是在 `Run()` 循环中处理——`consumeAssistantStream` 仍然正常返回 `full`，但**不再内部 emit 正文 chunk**，由 `Run()` 根据 ToolCalls 决定 emit

具体代码变更：

```go
// consumeAssistantStream 内部：移除 L962 的 emit("ai:stream-chunk", ...)
// 保留 emit("ai:stream-thinking", ...) 不变
```

#### 1b. `Run()` 流式路径：分支 emit

**位置**: L690-747

**当前**:

```go
streamedContent += full.Content
allStreamedContent += full.Content
if len(full.ToolCalls) > 0 {
    for _, tc := range full.ToolCalls { emitToolStart(...) }
    // ...
} else if full.Content != "" {
    finalContent = full.Content
}
```

**改为**:

```go
streamedContent += full.Content
allStreamedContent += full.Content
if len(full.ToolCalls) > 0 {
    // 中间文本：模型在工具调用前的过渡性输出
    if full.Content != "" {
        emit("ai:stream-intermediate", full.Content)
    }
    for _, tc := range full.ToolCalls { emitToolStart(...) }
    // ...
} else if full.Content != "" {
    // 最终回答：正常流式推送
    emit("ai:stream-chunk", full.Content)
    // 计划完成检测...
    finalContent = full.Content
}
```

#### 1c. `Run()` 非流式路径：同样分支

**位置**: L748-788

**当前**: L775 `emit("ai:stream-chunk", mv.Message.Content)` 在 ToolCalls 之外
**改为**: ToolCalls 分支内 emit `ai:stream-intermediate`，无 ToolCalls 分支 emit `ai:stream-chunk`

#### 1d. 注册新事件到 EventsOff 清理列表

**位置**: app.go 中 `EventsOff` 调用处（如有），以及前端清理列表

### 2. `frontend/src/js/ai-chat.js` — 前端新事件处理

#### 2a. 新增 `ai:stream-intermediate` 监听器

**位置**: `ai:stream-chunk` 处理器附近（L2551）

**逻辑**:

```js
const unsubIntermediate = window.runtime.EventsOn('ai:stream-intermediate', (streamGen, content) => {
    if (streamGen !== myGen) return;
    // 在气泡内创建/更新临时中间文本区域
    let interimEl = contentDiv.querySelector('.ai-stream-intermediate');
    if (!interimEl) {
        interimEl = document.createElement('div');
        interimEl.className = 'ai-stream-intermediate';
        contentDiv.insertBefore(interimEl, contentDiv.firstChild);
    }
    interimEl.textContent = content;
});
unsubs.push(unsubIntermediate);
```

#### 2b. `ai:stream-chunk` 处理器：到达时清除中间文本

**位置**: L2501-2503（`hasReceivedChunk` 首次设置处）

**新增**: 首个 `ai:stream-chunk` 到达时，移除 `.ai-stream-intermediate` 元素

```js
if (!hasReceivedChunk) {
    hasReceivedChunk = true;
    contentDiv.innerHTML = '';
    // ... 现有逻辑
}
```

`contentDiv.innerHTML = ''` 已经会清除中间文本元素，无需额外代码。

#### 2c. 移除 `hasReceivedToolStart` hack

**位置**: L2416（变量声明）、L2717-L2724（tool\_start 清除逻辑）

因为中间文本现在通过独立事件处理，`tool_start` 时不再需要清除。移除 `hasReceivedToolStart` 变量和 L2717-L2724 的清除逻辑。

#### 2d. 注册到事件清理列表

**位置**: L2404 的 `EventsOff` 数组，追加 `'ai:stream-intermediate'`

### 3. `internal/agent/EVENTS.md` — 文档更新

新增 `ai:stream-intermediate` 事件说明：

* 触发时机：ReAct 循环中有 ToolCalls 的 assistant 消息

* 负载：中间文本内容（string）

* 前端行为：临时展示在气泡内，最终回答到达时清除

### 4. CSS（可选）

**文件**: `frontend/src/css/components/ai-chat.css`

`.ai-stream-intermediate` 样式：灰色小字、轻微动画，与最终回答视觉区分。

## 验证步骤

1. Go 编译通过：`go build ./...`
2. 正常对话（无工具调用）：行为不变，`ai:stream-chunk` 正常渲染
3. 工具调用场景：中间文本通过 `ai:stream-intermediate` 临时展示，`tool_start` 后最终回答到达时中间文本消失
4. Plan 模式：计划生成 spinner → 中间文本 → 最终回答，流程正常
5. ask\_user 场景：同轮续答不受影响
6. 库中内容：不含中间文本

