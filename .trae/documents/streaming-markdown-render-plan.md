# 流式 Markdown 渲染 — 实现计划

## 当前状态

- `ai:stream-chunk` 处理块时使用 `contentDiv.textContent = streamingContent` → 用户看到**原始 MD 标记**（`**粗体**`、`` `代码` `` 等），只有流结束后才调用 `renderMarkdown()` 转换为格式化 HTML
- `renderMarkdown()` 函数内部每次都会调用 `marked.setOptions()`（重复但无害）→ 应将选项初始化移到 `initAIChat()` 中只执行一次
- 使用 `marked@18.0.5` + `highlight.js`，`marked.parse()` 能正确处理不完整 Markdown（未闭合的 `**` 会按原文显示）

## 改动

### 1. 初始化化 `marked` 选项（一次）

**文件：** `frontend/src/js/ai-chat.js`

**什么：** 将 `marked.setOptions({breaks, gfm, highlight})` 从 `renderMarkdown()` 内移到 `initAIChat()` 末尾，只执行一次

**为什么：** 避免每次渲染都重复设置全局选项；纯重构，无行为变化

### 2. 流式块实时渲染 Markdown

**文件：** `frontend/src/js/ai-chat.js`

**什么：** `unsubChunk` 中将：
```js
// 原代码（第 816 行）
contentDiv.textContent = streamingContent;
```
改为：
```js
contentDiv.innerHTML = marked.parse(streamingContent);
```

**为什么：** 每收到一个块就解析并渲染为 HTML，用户即时看到格式化内容（列表、粗体、代码块等）。`marked.parse()` 能优雅处理不完整 Markdown。

### 3. 最终渲染保留代码语言徽章

**文件：** `frontend/src/js/ai-chat.js`

**什么：** `unsubDone` 中保留 `renderMarkdown(contentDiv, finalContent)` 调用

**为什么：** 流结束后重新渲染一次，确保：a) 代码块语言徽章正确附加；b) 高亮完整正确；c) 按住管家最后一次渲染保证完整性

### 4. 选用方案：逐块完整重渲染（vs 增量差异渲染）

**决策：** 每次块到达时对整个 `streamingContent` 调用 `marked.parse()` 并替换 `innerHTML`。不采用 diff/patch 策略（复杂度与收益不匹配）。AI 对话场景下每个块约几~几十个字符，全长通常 <10K，性能可接受。视觉上可能会出现轻微 DOM 闪烁，但这是大多数 AI Chat UI（ChatGPT/Claude）采用的方案。

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/src/js/ai-chat.js` | 3 处改动（详见上述） |
| 后端 | 无改动 |
| CSS | 无改动 |

## 验证步骤

1. 运行 `go build ./...` 确认编译通过
2. 启动应用，发送 AI 消息，观察：
   - 流式过程中 Markdown 格式（列表、粗体、代码块、标题）实时渲染
   - 流结束后代码块出现语言徽章
   - 总耗时标签正常显示
3. 检查历史消息（加载已有 session）渲染正常