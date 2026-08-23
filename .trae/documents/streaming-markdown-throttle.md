# 流式 Markdown 渲染节流优化

## Summary

对 `ai-chat.js` 中 `handleStreamChunk` 的流式 Markdown 渲染添加 50ms 节流，将每次 chunk 到达都触发的全量 `marked.parse` + `innerHTML` 替换降低为最多每 50ms 渲染一次，减少 \~80% 的 DOM 操作次数，提升长回复的流式体验流畅度。

## Current State

* **文件**: `frontend/src/js/ai-chat.js`

* **位置**: `handleStreamChunk` 函数（L2289-2324）

* **当前行为**: 每个 chunk 到达 → 立即 `marked.parse(streamingContent)` → `innerHTML` 全量替换 → `scrollToBottom()`

* **开销**: \~100 次 chunk × (\~3-8ms/次) = \~300-800ms 总渲染时间

## Proposed Changes

### 修改 1：`handleStreamChunk` 添加节流逻辑

**文件**: `frontend/src/js/ai-chat.js`（L2289-2324）

将 `handleStreamChunk` 改为缓冲 chunk + 定时渲染模式：

1. 新增两个闭包变量（在 `handleStreamChunk` 定义前）：

   * `_streamRenderTimer` — setTimeout ID

   * `_pendingStreamChunks` — 未渲染的 chunk 缓冲区（字符串）

2. 修改 `handleStreamChunk`：

   * 首个 chunk 到达时（`!hasReceivedChunk` 分支）：保持原逻辑（清除打字指示器、停止思考计时、淡出工具条）

   * 将 chunk 追加到 `_pendingStreamChunks` 缓冲区

   * 若 `_streamRenderTimer` 未设置：启动 50ms 定时器，到期后将缓冲区追加到 `streamingContent`，执行 `marked.parse` + `innerHTML` + `scrollToBottom()`，清空缓冲区和定时器

```javascript
// 新增变量（在 handleStreamChunk 前）
let _streamRenderTimer = null;
let _pendingStreamChunks = '';

const handleStreamChunk = (chunk) => {
    if (!chunk) return;
    if (!hasReceivedChunk) {
        // ... 首个 chunk 的初始化逻辑保持不变 ...
    }
    // 缓冲 chunk，定时批量渲染
    _pendingStreamChunks += chunk;
    if (!_streamRenderTimer) {
        _streamRenderTimer = setTimeout(() => {
            _streamRenderTimer = null;
            streamingContent += _pendingStreamChunks;
            _pendingStreamChunks = '';
            contentDiv.innerHTML = marked.parse(streamingContent);
            scrollToBottom();
        }, 50);
    }
};
```

### 修改 2：`stream-done` 清理定时器

**文件**: `frontend/src/js/ai-chat.js`（`ai:stream-done` 事件处理器内，L2531 附近）

在 `unsubs.forEach(fn => fn())` 之后添加：

```javascript
// 清理流式渲染节流定时器，flush 残留 chunk
if (_streamRenderTimer) {
    clearTimeout(_streamRenderTimer);
    _streamRenderTimer = null;
    streamingContent += _pendingStreamChunks;
    _pendingStreamChunks = '';
}
```

注意：`stream-done` 后面已有 `renderMarkdown(contentDiv, finalContent)`（L2548）做最终渲染，所以此处 flush 的作用是确保 `streamingContent` 是完整的（供后续可能的引用），最终 DOM 由 `renderMarkdown` 覆盖。

### 修改 3：`stream-error` 清理定时器

**文件**: `frontend/src/js/ai-chat.js`（`ai:stream-error` 事件处理器内，L2648 附近）

在 `unsubs.forEach(fn => fn())` 之后添加：

```javascript
// 清理流式渲染节流定时器
if (_streamRenderTimer) {
    clearTimeout(_streamRenderTimer);
    _streamRenderTimer = null;
    _pendingStreamChunks = '';
}
```

错误路径会移除整个流式气泡（L2659 `streamingEl.remove()`），不需要 flush。

## Assumptions & Decisions

* **节流间隔 50ms**：LLM 流式速度约 15-35ms/chunk，50ms 意味着每 2-3 个 chunk 渲染一次，视觉上无感知延迟

* **首个 chunk 不节流**：保持当前行为——第一个 chunk 到达时立即执行初始化逻辑（清除打字指示器等），但渲染本身也会走节流路径（因为是第一个 chunk，timer 会立即设置）

* **不改变** **`stream-done`** **的最终渲染**：`renderMarkdown(contentDiv, finalContent)` 在流结束后做最终完整渲染，保证最终状态正确

* **不改变** **`scrollToBottom`**：仍由节流后的渲染回调触发，保持滚动行为一致

## Verification

1. 启动应用，发送一条会触发长回复的问题（如"详细介绍一下..."）
2. 观察流式输出是否流畅，无明显卡顿或跳帧
3. 确认代码块、表格等复杂 Markdown 在流式过程中正确渲染
4. 流结束后确认最终渲染正确（无残留格式问题）
5. 测试停止按钮：点击停止后流式气泡正常清理，无残留定时器
6. 测试错误场景：AI 报错时气泡正常移除
7. 测试 ask\_user 反问：反问后续答的流式渲染正常

