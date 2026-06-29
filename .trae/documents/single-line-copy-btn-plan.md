# 单行代码块复制按钮垂直居中 — 实现计划

## 当前状态

AI 聊天消息中的代码块复制按钮固定 `top: 4px; right: 8px`。单行代码块（如 `` ```python print("hello")``` ``）中按钮偏左上，视觉上不对称。

编辑器预览模式已有处理方案（`main.js` `_applyPreviewDOMHelpers`）：
- 检测 `!codeEl.textContent.trim().includes('\n')` 判断是否为单行
- 单行时添加 `.copy-code-btn--single` 类，用 `top: 50%; transform: translateY(-50%)` 垂直居中

## 改动

### 1. `frontend/src/js/ai-chat.js` — `renderMarkdown()`

**什么：** 创建复制按钮时加单行检测逻辑

**改动位置**：第 941-960 行（创建 `copyBtn` 处）

**具体修改：**
```js
// 原代码
copyBtn.className = 'code-copy-btn';

// 改为
const isSingleLine = code && !code.textContent.trim().includes('\n');
copyBtn.className = 'code-copy-btn' + (isSingleLine ? ' code-copy-btn--single' : '');
```

### 2. `frontend/src/css/components/ai-chat.css` — 新增单行样式

**什么：** 添加 `.code-copy-btn--single` 垂直居中样式

**改动位置**：`.code-copy-btn` 块之后（约第 229 行后）

**具体修改：**
```css
.ai-msg-assistant .code-copy-btn--single {
    top: 50%;
    transform: translateY(-50%);
}
.ai-msg-assistant .code-copy-btn--single:hover {
    transform: translateY(-50%) scale(1.08);
}
```

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/src/js/ai-chat.js` | 复制按钮创建时加 `isSingleLine` 检测，动态类名 |
| `frontend/src/css/components/ai-chat.css` | 新增 `.code-copy-btn--single` 样式 |

## 验证

1. 发送含单行代码块的消息（如 `` ```python print("hello")``` ``），悬浮时复制按钮应在右侧垂直居中
2. 多行代码块不受影响，按钮仍在右上角
3. Go build 通过