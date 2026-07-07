# 修复重置后点击 AI 助手闪一下旧对话内容的问题

## 问题描述

恢复出厂设置后，点击 AI 助手选项卡，会短暂显示重置前的旧对话页面，然后才闪一下切换到「尚未配置 AI 服务」的空状态。

## 根因分析

### 时间线

```
视图切换（switchView）
  │  #viewAiChat 变得可见（.active → display: flex）
  │  子元素 display 状态来自上一次会话：
  │    messagesEl.style.display = ''（上次 hideEmptyState 设置）
  │    messagesEl.innerHTML = 旧消息 HTML
  │
  ├── 50ms ──→ onAIChatViewActivated() 开始
  │              ├── syncToolbarState()
  │              └── await GetAIConfig()      ← 异步调用，旧消息**仍然可见**
  │
  └── 响应返回 → 无 API Key → showEmptyState()
                   messagesEl.style.display = 'none'
                   emptyEl.style.display = ''
```

**关键问题**：`onAIChatViewActivated()` 在第一个 `await` 之前没有隐藏消息区。从视图可见到 `GetAIConfig()` 返回之间有异步等待时间，旧消息一直暴露着。

## 修改方案

**文件**：`frontend/src/js/ai-chat.js` — `onAIChatViewActivated()` 函数顶部

在函数入口处、`syncToolbarState()` 之前，立即隐藏所有内容区域：

```javascript
export async function onAIChatViewActivated() {
    if (!messagesEl) return;

    // 立即隐藏所有内容区，防止切换视图时残留旧数据闪烁
    messagesEl.style.display = 'none';
    if (inputAreaEl) inputAreaEl.style.display = 'none';
    if (emptyEl) emptyEl.style.display = 'none';
    if (welcomeEl) welcomeEl.style.display = 'none';

    // 从 DOM 重新同步工具栏状态变量
    syncToolbarState();
    // ... 后续代码不变
}
```

### 为什么要同时隐藏多个元素

| 元素 | 为什么隐藏 |
|------|-----------|
| `messagesEl` | 残留旧消息 HTML |
| `inputAreaEl` | 残留旧输入区域（visible） |
| `emptyEl` | HTML 默认 `display:none`，但确保万无一失 |
| `welcomeEl` | HTML 默认 `display:none`，但确保万无一失 |

### 后续谁会恢复显示

| 条件 | 恢复函数 | 恢复的元素 |
|------|----------|-----------|
| 无 API Key | `showEmptyState()` | `emptyEl` 显示 |
| 有 API Key | `hideEmptyState()` | `messagesEl` 显示，`inputAreaEl` 显示 |
| 有 Key + 空会话 | `showWelcome()` | `welcomeEl` 显示 |

## 涉及文件

| 文件 | 修改内容 |
|------|----------|
| `frontend/src/js/ai-chat.js` | `onAIChatViewActivated()` 入口处新增 4 行隐藏逻辑 |

## 验证步骤

1. 正常使用 AI 聊天，确保有对话内容
2. 恢复出厂设置
3. 点击 AI 助手选项卡
4. 确认：**不再闪烁旧对话内容**，直接显示「尚未配置 AI 服务」页面
5. 配置 API Key 后再次点击 AI 助手
6. 确认：正常显示欢迎页或对话列表
