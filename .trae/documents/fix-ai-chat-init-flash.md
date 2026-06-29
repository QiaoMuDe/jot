# 修复 AI 助手页面打开时的闪烁问题

## 总结

AI 助手页面每次打开时都会短暂闪现空状态提示（"尚未配置 AI 服务"），原因是 HTML 默认让空状态元素可见，但正确的显示状态需要等待异步调用后端获取配置后才能确定。

## 当前状态分析

### 页面加载时序

```
switchView('ai-chat')
  │
  ├─ view 可见（.active + .view-enter 动画 250ms）
  │   ├─ #aiChatEmpty        ← 可见（HTML 无 display:none）
  │   ├─ #aiChatMessages     ← display:none（hidden）
  │   └─ #aiChatInputArea    ← display:none（hidden）
  │
  └─ setTimeout(50ms) → onAIChatViewActivated()
       │
       ├─ await GetAIConfig()     ← 异步 Wails IPC
       │
       ├─ 配置有效 → hideEmptyState()  → 隐藏空状态，显示消息区+输入区
       │              loadSessionList()
       │              switchSession()  → 加载消息
       │
       └─ 配置无效 → showEmptyState()  → 显示空状态（维持原样）
```

### 根因

**HTML 默认状态与最终状态不一致**（[index.html#L647](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L647)）：

| 元素 | HTML 初始状态 | 配置有效时最终状态 | 配置无效时最终状态 |
|------|--------------|-------------------|-------------------|
| `#aiChatEmpty` | **可见** | 隐藏 | 可见 |
| `#aiChatMessages` | `display:none` | 可见 | 隐藏 |
| `#aiChatInputArea` | `display:none` | 可见 | 隐藏 |

每次进入 AI 聊天页时，空状态会在 `onAIChatViewActivated` 的异步调用完成前短暂可见（约 50ms + Wails IPC 延迟），然后被 `hideEmptyState()` 隐藏，造成闪烁。

## 修改方案

**只改一行**：将 `#aiChatEmpty` 的初始状态改为 `display:none`，与 `#aiChatMessages`、`#aiChatInputArea` 保持一致，三者初始都隐藏。

这样在 `onAIChatViewActivated` 异步完成前，内容区是空白的（动画期间无内容闪烁），异步完成后由 `showEmptyState()` / `hideEmptyState()` 决定正确显示哪个。

### 涉及文件

| 文件 | 行号 | 修改 |
|------|------|------|
| `frontend/index.html` | L647 | `<div id="aiChatEmpty" class="ai-chat-empty">` → `<div id="aiChatEmpty" class="ai-chat-empty" style="display:none;">` |

**无需修改 JS 代码**，`showEmptyState()` / `hideEmptyState()` 已有完整的显示/隐藏逻辑。

### 用户感知变化

- **已配置 AI 的用户**（绝大多数情况）：不再看到"尚未配置 AI 服务"的闪现，直接看到消息列表 + 输入框（异步加载完成后出现，此时动画也基本结束）
- **未配置 AI 的用户**：打开时短暂空白，异步完成后显示空状态提示（比原来闪现后稳定显示更合理）

## 验证方式

1. 打开应用 → 切换到 AI 助手 → 无闪烁
2. 反复切换其他视图和 AI 助手 → 每次切换都无闪烁
3. 清除 AI 配置后重新打开 → 空白 → 空状态提示出现（无闪烁）
