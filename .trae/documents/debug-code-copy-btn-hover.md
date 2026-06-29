# AI 聊天多行代码块复制按钮悬浮失效 — 排查与修复计划

## 问题

多行代码块的复制按钮鼠标悬浮不显现，单行正常。目前实现与笔记预览模式（正常工作的参考实现）完全一致：

| 环节 | 笔记预览（正常） | AI 聊天（异常） |
|------|---------|-------|
| 按钮位置 | `pre.appendChild(btn)` | `pre.appendChild(copyBtn)` |
| 包装顺序 | 追加按钮 → 包 wrapper | 追加按钮 → 包 wrapper |
| hover 选择器 | `.md-rendered pre:hover .copy-code-btn` | `.ai-msg-assistant pre:hover .code-copy-btn` |

从代码静态分析无法定位根因（单行/多行共用同一选择器，且单行有效）。需要分层排查。

## 排查方案

### Step 1：确认按钮是否存在
在 CSS 中加 debug 规则使 .code-copy-btn 永久可见（不依赖 hover），确认多行代码块的按钮是否被渲染到 DOM：
```css
.ai-msg-assistant .code-copy-btn {
    opacity: 1 !important;
}
```
- 如果按钮显示 → 按钮存在，问题是 hover 选择器不生效
- 如果按钮不显示 → 按钮未创建，JS 渲染路径有问题

### Step 2：根据 Step 1 结果分叉修复

**情形 A：按钮存在但 hover 不生效**
尝试以下方案之一：
- **A1（调整个别规则）**：
  ```css
  .ai-msg-assistant .code-copy-btn:hover {
      opacity: 1;
  }
  ```
  直接在按钮自身 hover 时显现，不依赖父级选择器。
- **A2（改变定位方式）**：切换按钮位置到 `.pre-wrapper` 级别，用 CSS 相邻兄弟选择器 `pre:hover + .code-copy-btn` 配合 `.code-copy-btn:hover` 兜底
- **A3（加 pointer-events 控制）**：`.pre-wrapper { pointer-events: auto; }`

**情形 B：按钮未创建**
排查 JS 渲染路径——可能是 `hljs.highlightElement` 执行后改变了 `<code>` 的类名，导致 `language-` 检测失效：

```js
// ai-chat.js renderMarkdown() 中，在创建按钮后检查：
console.log('code classes:', code.classList);
console.log('isSingleLine:', !code.textContent.trim().includes('\n'));
```

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/src/css/components/ai-chat.css` | Step 1 debug 规则 + 最终修复 |
| `frontend/src/js/ai-chat.js` | 仅情形 B 需改动 |

## 验证

1. 多行代码块悬浮时复制按钮显现
2. 单行代码块悬浮时复制按钮仍正常