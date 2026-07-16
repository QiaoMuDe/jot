# 大文件 .md 笔记自动切换纯文本模式 — 实施计划

## 概述

对于后缀为 `.md` 的笔记，当内容长度超过用户设置的"引用截断字数"（`ai_ref_max_chars`）时，不再默认进入 Markdown 预览模式，而是保持 CM6 纯文本编辑模式显示，避免大文件 Markdown 渲染导致的卡顿。

## 当前状态分析

### 编辑器打开流程（`openEditor` 函数）

文件：[main.js#L3163-L3407](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3163-L3407)

```
阶段一（同步）：
  - 检测 isMd = true → 设置 editorOverlay.dataset.mode = 'preview'
  - 预览按钮激活，mdRendered 设为 "暂无内容"
  - 显示骨架屏

阶段二（异步）：
  - GetNoteContent(id) → 加载完整内容到 editorContent
  - loadTagsForEditor() → 加载标签
  → 初始化 CM6（纯文本编辑器）
  → 调用 updatePreview() → marked.parse() + hljs 渲染预览
```

### 关键代码位置

| 位置        | 行号                                                                         | 作用                                       |
| --------- | -------------------------------------------------------------------------- | ---------------------------------------- |
| 阶段一设置预览模式 | [L3214-L3229](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3214-L3229) | 只读 `.md` 笔记默认进入预览模式                      |
| 内容加载完成    | [L3343](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3343)             | 阶段二 `Promise.all` 完成                     |
| CM6 初始化   | [L3382-L3384](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3382-L3384) | 初始化 CM6 编辑器                              |
| 触发预览渲染    | [L3398-L3400](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L3398-L3400) | 只读 + `.md` + 预览模式 → 调用 `updatePreview()` |

### 引用截断字数设置（`ai_ref_max_chars`）

* DB 默认值：`10000`，范围 1\~100000

* 前端设置页输入框 id：`#aiRefMaxChars`

* 每次 `loadSettings()` 时从后端 `cfg.ai_ref_max_chars` 加载到 DOM

* 前端 Wails 绑定：`window.go.main.App.GetAIRefMaxChars()` 可直接调用

## 改动方案

### 修改文件

仅需修改 **一个文件**：[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)

### 具体改动

#### 1. 阶段一：默认不进入预览模式（L3214-L3229）

**当前行为**：只读 `.md` 笔记直接在阶段一设置 `dataset.mode = 'preview'`，激活预览按钮。

**改为**：先检查 `editorContent` 是否已知（仅为缓存中的 `noteData.content`，长度有限），如果已知且内容过长，则不进入预览模式。但更重要的改动在阶段二。

实际上，阶段一无法知道完整内容长度（缓存中的 `noteData.content` 只有前 200 字），所以阶段一**不做改动**，保持原样（默认设为 preview 模式，但内容还没加载，渲染不会执行）。

#### 2. 阶段二：内容加载后检查大小，决定是否跳过预览（L3396-L3400 附近）

在当前代码 `// 查看模式：Markdown 预览（CM6 就绪后刷新）` 之前，插入大小检查逻辑：

```javascript
// ── 大文件自动切换纯文本模式 ──
// 只读 + .md + 预览模式下，检查内容是否超过引用截断阈值
if (isReadOnly && isMd && els.editorOverlay.dataset.mode === 'preview') {
    const refMaxChars = parseInt(document.getElementById('aiRefMaxChars')?.value) || 10000;
    if (editorContent.length > refMaxChars) {
        // 内容过长，自动切换为纯文本模式
        switchEditorMode('edit');
        _setPreviewLayout(false);
        _closeToc();
        window.showNotification?.('笔记内容超过引用截断字数，已自动切换为纯文本模式', 'info');
    } else {
        // 内容正常，执行预览渲染
        updatePreview();
    }
}
```

### 阈值来源

* **优先级 1**：DOM 元素 `#aiRefMaxChars` 的当前值（用户设置页中修改的引用截断字数）

* **优先级 2**：`10000`（DB 默认值，与后端 `GetAIRefMaxChars` 的空值回退一致）

### 为什么不用 `GetAIRefMaxChars()` 后端调用

* 该调用是异步的，需要额外 await

* 此时内容已加载完毕，DOM 中的值已经由 `loadSettings` 填充好

* 直接用 DOM 值更简单，零额外开销

### 用户交互

* 大文件自动显示为 CM6 纯文本（只读模式）

* 用户仍可手动点击"预览"按钮切换到 Markdown 预览

* 通过 `showNotification` 提示用户文件过大已自动切换

## 风险与注意事项

1. **用户仍可手动预览**：自动切换后，用户可点击预览按钮手动渲染，这是合理行为
2. **编辑模式不受影响**：编辑模式本来就没有预览渲染
3. **阈值可调**：用户调整引用截断字数后，下次打开笔记即生效
4. **性能收益**：绕过 `marked.parse()` + `hljs` 的大文件渲染开销
5. **通知提示**：避免用户困惑为什么不是预览模式

## 验证步骤

1. 创建一篇内容超过 `ai_ref_max_chars`（默认 10000 字）的 `.md` 笔记
2. 在笔记列表点击查看 → 应自动进入 CM6 纯文本模式，显示提示
3. 手动点击"预览"按钮 → 应正常渲染 Markdown
4. 在设置页将引用截断字数调大（如 100000），重新打开该笔记 → 应正常进入预览模式
5. 创建一篇内容小于阈值的小 `.md` 笔记 → 行为不变，正常预览模式

