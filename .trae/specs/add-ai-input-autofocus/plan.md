# 计划：程序启动后自动获取焦点

## 问题分析

经过深入排查，发现 `element.focus()` 在启动时失效的**根本原因**：

在 Wails/WebView2 环境下，程序刚启动时**嵌入式 WebView 控件本身未获得操作系统的键盘焦点**。JavaScript 的 `element.focus()` 只能设置文档内的"逻辑焦点"，但如果 WebView 控件没有 OS 级键盘焦点，键盘事件不会传递到页面 —— 因此 `Ctrl+9` 等快捷键无效。

**证据**：
- `main.js` 第 2580 行的 `openEditor()` 中已有 `document.hasFocus()` 守卫检查，注释写明"仅在窗口已激活时生效，启动时自动打开的快速笔记跳过"
- Wails runtime 提供了 `WindowShow()` API，可将窗口带到前台并赋予焦点

## 解决方案

### 1. 导入 Wails `WindowShow()` 

`main.js` 的 import 行添加 `WindowShow`：
```js
import { ..., WindowShow } from '../wailsjs/runtime/runtime.js';
```

### 2. 创建 `ensureFocus()` 辅助函数

在 `main.js` 中添加一个轻量轮询函数：

```js
function ensureFocus(el) {
    if (!el) return;
    let retries = 0;
    function tryFocus() {
        WindowShow();          // 强制窗口/WebView 获取 OS 焦点
        el.focus({ preventScroll: true });
        if (document.activeElement === el || retries >= 20) return;
        retries++;
        requestAnimationFrame(tryFocus);
    }
    tryFocus();
}
```

逻辑：
- 每次尝试先调用 `WindowShow()` 确保窗口有 OS 焦点
- 然后 `el.focus()` 设置逻辑焦点
- 检查 `document.activeElement` 确认是否成功
- 失败则重试（最多 20 帧 ≈ 300ms）

### 3. 三处替换

| 文件 | 位置 | 原代码 | 替换为 |
|------|------|--------|--------|
| `frontend/src/main.js` | import 行 | 无 `WindowShow` | 添加 `WindowShow` |
| `frontend/src/main.js` | init() L5794 | `setTimeout(() => els.viewGrid?.focus(), 50)` | `ensureFocus(els.viewGrid)` |
| `frontend/src/js/ai-chat.js` | onAIChatViewActivated L1328 | `setTimeout(() => inputEl?.focus(), 100)` | `ensureFocus(inputEl)` |
| `frontend/src/js/ai-chat.js` | switchSession L770 | `inputEl?.focus()` | `ensureFocus(inputEl)` |

### 4. 在 `ai-chat.js` 中复用 `ensureFocus`

`ai-chat.js` 没有直接访问 `WindowShow` 和 `ensureFocus`。有两种方式：

**方式 A（推荐）**：在 `main.js` 中将 `ensureFocus` 挂到 `window` 上，`ai-chat.js` 直接调用 `window.ensureFocus()`。

即在 `main.js` 的 `ensureFocus` 定义之后加：
```js
window.ensureFocus = ensureFocus;
```

然后在 `ai-chat.js` 中将两处 `inputEl?.focus()` 替换为 `window.ensureFocus(inputEl)`。

## 改动清单

| 文件 | 改动 |
|------|------|
| `frontend/src/main.js` | import 添加 `WindowShow`；新增 `ensureFocus()` 函数并暴露到 `window`；替换 `els.viewGrid` 的聚焦调用 |
| `frontend/src/js/ai-chat.js` | 2 处 `focus()` 替换为 `window.ensureFocus()` |

## 验证

1. 启动应用 → 立即按 `Ctrl+9` → 跳转到 AI 助手界面
2. 切换到 AI 聊天视图 → 输入框自动聚焦
3. 切 AI 会话 → 输入框仍自动聚焦
