# AI 助手流式回复慢修复方案

## 问题根因

**根本原因**：`ai-chat.js:822` 每个 chunk 都全量运行 `marked.parse(streamingContent)` + `innerHTML` 替换。

Ollama HTTP API 按**逐 token** 粒度发 chunk（1-3 字符/次），而 OpenAI 兼容服务（DeepSeek）每次发 20-50 字符。Ollama 产生更多次 chunk 事件，每次 marked.parse 的耗时随文本长度线性增长，整体呈 O(n²)。

## 方案

两套 chunk 处理路径，根据 `window._aiProvider` 切换：

### OpenAI 兼容 路径（保持现有行为）
每个 chunk 照常走 `marked.parse` + `innerHTML`，实时渲染 markdown 格式。

### Ollama 路径
- **chunk 事件**：只追加纯文本节点 `document.createTextNode(chunk)`，不做任何 markdown 解析
- **done 事件**：只做一次 `contentDiv.innerHTML = marked.parse(finalContent)` + `renderMarkdown()` 建复制按钮和高亮

## 修改文件

### `frontend/src/main.js` — 3 处改动

1. `loadAISettings()` 中 `cfg.provider` 赋值处（约 1632 行）：
   ```js
   window._aiProvider = cfg.provider || 'openai';
   ```

2. `saveAIConfig()` 保存成功后（约 1861 行后）：
   ```js
   window._aiProvider = provider;
   ```

3. URL change 和 Key change 自动保存中也同步更新 `window._aiProvider`（约 1874、1887 行处）

### `frontend/src/js/ai-chat.js` — chunk 事件分支（约 811-824 行）

```js
// 替换现有的 contentDiv.innerHTML = marked.parse(streamingContent)
const provider = window._aiProvider || 'openai';
if (provider === 'ollama') {
    // Ollama：纯文本追加，不做 markdown 解析
    if (!streamingTextPre) {
        streamingTextPre = document.createElement('div');
        streamingTextPre.className = 'streaming-text';
        contentDiv.appendChild(streamingTextPre);
    }
    streamingTextPre.appendChild(document.createTextNode(chunk));
} else {
    // OpenAI 兼容：保持原有 marked.parse 行为
    contentDiv.innerHTML = marked.parse(streamingContent);
}
```

### `frontend/src/js/ai-chat.js` — done 事件调整（约 840 行）

Ollama 路径在 done 时只做一次完整渲染：
```js
if (window._aiProvider === 'ollama' && streamingTextPre) {
    contentDiv.innerHTML = marked.parse(finalContent);
}
renderMarkdown(contentDiv, finalContent);
```

## 不修改的部分

- 后端 `app.go`、`ai_service.go` — 不变
- 前端 HTML/CSS — 不变
- OpenAI 路径的 thinking 处理 — 不变
- Ollama 路径的 thinking 处理 — 仍通过 `ai:stream-thinking` 事件实时显示

## 验证步骤

1. Ollama 提问 → 文字逐字流畅出现，无卡顿
2. 流结束后 → markdown 格式正确（粗体、代码等）
3. 切换回 OpenAI 兼容（DeepSeek）→ 原有实时 markdown 渲染行为不变
4. 深度思考在两种模式下均正常工作
