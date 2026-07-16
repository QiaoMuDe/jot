# 代码块滚动条粗细修复计划

## 根因分析

`pre` 元素上同时存在两套滚动条控制系统：

1. **`scrollbar-width: thin`** — CSS Scrollbar Styling 规范（Chromium 121+ 支持）
2. **`::-webkit-scrollbar { width: 4px; height: 4px }`** — 旧版 WebKit 定制滚动条

在 Chromium（WebView2）中，当 `scrollbar-width: thin` 存在时，`::-webkit-scrollbar` 的宽高**被忽略**，浏览器使用系统定义的 thin 滚动条尺寸。

但 `.cm-scroller` 也有 `scrollbar-width: thin`，用户却说它细。关键区别在于：

| 元素 | overflow-y | scrollbar-width: thin 效果 |
|------|-----------|--------------------------|
| `.cm-scroller` | `auto`（垂直滚动条存在） | ✅ 正常工作 |
| `pre` | `hidden`（垂直滚动条隐藏） | ❌ **失效**，回退默认滚动条 |

当 `overflow-y: hidden` 时，Chromium 的 `scrollbar-width: thin` 对水平滚动条**不生效**，回退到默认滚动条（~12px）。这就是为什么修改 `::-webkit-scrollbar` 的宽高（4px/6px/auto）全无效果——因为 `scrollbar-width: thin` 失效后，`::-webkit-scrollbar` 又被它压制，导致 `pre` 使用浏览器默认的粗滚动条。

## 解决方案

**移除 `pre` 上的 `scrollbar-width: thin`**，让 `::-webkit-scrollbar` 完全控制滚动条尺寸。

### 改动文件

#### 1. `frontend/src/css/components/editor.css`

**位置 1**（第 366-379 行）：`.md-rendered pre` 基础样式
- 移除 `scrollbar-width: thin;`
- 保留 `scrollbar-color: transparent transparent;`（无副作用，有 `::-webkit-scrollbar-thumb` 兜底）
- 保留 `transition: scrollbar-color 0.3s ease;`

**位置 2**（第 403-416 行）：`.md-rendered pre` hover 样式
- 移除 `scrollbar-color: var(--scrollbar-thumb) transparent;`（不再依赖 scrollbar-color）
- hover 时 `::-webkit-scrollbar-thumb` 的 `background` 规则已存在，足以控制颜色

#### 2. `frontend/src/css/components/ai-chat.css`

**位置 1**（第 181-195 行）：`.ai-msg-assistant pre` 基础样式
- 移除 `scrollbar-width: thin;`
- 保留 `scrollbar-color: transparent transparent;`
- 保留 `transition: scrollbar-color 0.3s ease;`

**位置 2**（第 218-221 行）：`.ai-msg-assistant pre:hover` 样式
- 移除 `scrollbar-color: var(--scrollbar-thumb) transparent;`

### 期望效果

移除 `scrollbar-width: thin` 后，`pre` 的滚动条完全由 `::-webkit-scrollbar` 控制：
- 尺寸：`width: 4px; height: 4px`
- 默认：`::-webkit-scrollbar-thumb { background: transparent }`（透明隐藏）
- Hover：`::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb) }`（显示）

4px 的滚动条应该比当前默认滚动条细得多。

### 验证步骤

1. 构建前端：`cd frontend && npm run build`
2. 运行 Wails 应用
3. 打开笔记编辑页，找到有代码块的预览区
4. 水平滚动代码块，观察滚动条粗细
5. 打开 AI 助手，找到有代码块的 AI 消息，同样验证