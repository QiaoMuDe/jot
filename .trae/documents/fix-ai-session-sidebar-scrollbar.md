# 修复 AI 会话侧栏滚动条显示问题

## 摘要

AI 会话侧栏的 `.ai-session-list` 滚动条在进入页面时短暂显示约 0.5 秒后消失，原因是该容器未纳入滚动条样式管理系统，布局稳定前后滚动条出现/消失导致闪烁。

## 当前状态分析

### 相关文件
- `frontend/src/css/scrollbar.css` — 全局滚动条样式 + 自动隐藏系统
- `frontend/src/css/components/ai-chat.css` — AI 对话页面全部样式
- `frontend/src/main.js` — `initScrollbarAutoHide()` 函数

### 滚动条自动隐藏系统

`scrollbar.css` 中有一套针对特定容器的自动隐藏机制：

```
第 40-44 行：将容器的 thumb 设为 transparent（默认不可见）
第 47-51 行：容器有 .scrolling 类时 thumb 恢复 visible（滚动时可见）
第 53-57 行：hover 时 thumb 变深
```

受此系统管理的容器（第 40-44 行）：
- `#mainContent` 
- `.search-results`
- `.ai-chat-messages`

`initScrollbarAutoHide()` 函数（`main.js`）中注册 scroll 事件的容器：
- `els.mainContent`（`#mainContent`）
- `document.querySelector('.ai-chat-messages')`

### 问题根因

**`.ai-session-list` 不在自动隐藏系统的目标列表中**，也没有独立的滚动条样式覆盖。

具体表现为：
1. 页面首次加载时，`.ai-session-list` 内容未完全填充/布局未稳定，此时容器高度未确定，浏览器临时显示滚动条
2. 布局稳定后，`flex: 1` 容器高度确定，内容可能恰好占满或溢出减少，浏览器自动隐藏滚动条（`overflow-y: auto` 原生行为）
3. 全局 `::-webkit-scrollbar-thumb` 的 `background: var(--scrollbar-thumb)` 理论上应让 thumb 始终可见，但由于无专有规则保护，其可见性受父容器 `overflow: hidden` 和布局抖动影响

## 修改方案

### 方案：为 `.ai-session-list` 添加独立滚动条样式（始终可见）

在 `ai-chat.css` 中添加 `.ai-session-list` 的专用滚动条样式，使其脱离自动隐藏系统影响，始终稳定可见。

#### 修改文件 1: `frontend/src/css/components/ai-chat.css`

在 `.ai-session-list` 规则块中（约第 830-834 行）添加：

```css
.ai-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px 1px 6px 8px;
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar-thumb) transparent;
}

.ai-session-list::-webkit-scrollbar {
  width: 6px;
}

.ai-session-list::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
  border-radius: 3px;
}

.ai-session-list::-webkit-scrollbar-track {
  background: transparent;
}
```

使用 `scrollbar-width: thin`（Firefox）和 `6px` 宽（WebKit）与全局统一。

## 验证方式

1. 进入 AI 助手页面，观察侧栏会话列表滚动条是否稳定显示，不再闪烁消失
2. 滚动列表时确认滚动条 thumb 正常跟随
3. 切换主题，确认滚动条颜色跟随 `--scrollbar-thumb` 变量变化
4. 确认右侧 `.ai-chat-messages` 的自动隐藏行为不受影响

## 备注

- 不将 `.ai-session-list` 纳入自动隐藏系统（`initScrollbarAutoHide`），因为侧栏导航列表的滚动条应该常显，与内容阅读区的"滚动时显示、静止后淡出"设计意图不同
- 不改动 `scrollbar.css` 和 `main.js`，隔离影响范围
