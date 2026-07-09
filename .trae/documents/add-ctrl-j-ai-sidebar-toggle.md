# Ctrl+J AI 会话侧栏折叠/展开快捷键 实施计划

## 总结
为 AI 助手模块新增 `Ctrl+J` 快捷键（不分大小写），用于控制 AI 会话侧栏的展开与折叠。仅在 AI 助手视图活跃且不在输入框中时生效。

## Current State Analysis

### AI 会话侧栏折叠/展开逻辑
- **位置**: `frontend/src/js/ai-chat.js` 第 592-622 行
- 侧栏切换逻辑内联在 `#aiSidebarToggle` 按钮的 click 事件回调中
- 操作 DOM 元素：`.ai-session-sidebar` 的 `collapsed` CSS 类切换
- 持久化：`localStorage` 键 `ai_sidebar_collapsed`（值 `'true'` 展开，`'false'` 折叠）
- 展开时会调用 `loadSessionList()` 刷新
- 当前没有抽象为可复用的函数

### 键盘快捷键系统
- **位置**: `frontend/src/main.js` 第 5154-5357 行（`handleKeyboardNavigation` 函数）
- 非数字快捷键（Ctrl+S/F/H/N/L/E/Q）：各自为独立的 `if` 块，在 Ctrl+数字模块之前
- Ctrl+数字快捷键（Ctrl+1~8）：统一的 `switch` 块
- 保护：编辑器/查看器激活时屏蔽所有 Ctrl+数字快捷键；不在 `<input>`/`<textarea>`/`[contenteditable]` 内时生效
- Ctrl+J 目前未被占用

### 快捷键帮助页面
- **位置**: `frontend/src/main.js` 第 5562-5589 行（`renderShortcutsPage` 函数）
- 所有快捷键定义在一个静态数组中，按 Ctrl+数字排在最末

## Proposed Changes

### 改动 1: ai-chat.js — 导出可复用的 toggleAISessionSidebar 函数

**文件**: `frontend/src/js/ai-chat.js`（第 592-622 行）

**What**: 将 AI 侧栏折叠/展开的内联逻辑提取为 `toggleAISessionSidebar()` 全局函数，同时保留原有点击事件绑定。

**Why**: 使该逻辑可被键盘快捷键处理函数复用，避免代码重复。

**How**:
1. 将侧栏引用的 DOM 元素（`toggleBtn`、`sidebar`）提升到模块级作用域（函数外部），或通过闭包封装
2. 定义 `window.toggleAISessionSidebar = function() { ... }`，内容复用原有 click 回调的逻辑
3. 简化原有的 click 事件绑定为直接调用 `toggleAISessionSidebar()`

**具体修改**（在原位置重构）：
```javascript
// 侧栏折叠/展开
const toggleBtn = document.getElementById('aiSidebarToggle');
const sidebar = document.querySelector('.ai-session-sidebar');
const chevronLeft = '<svg...>';
const chevronRight = '<svg...>';

// 恢复保存的状态 (默认展开) 
const saved = localStorage.getItem('ai_sidebar_collapsed');
if (saved === 'false') {
    sidebar.classList.add('collapsed');
    toggleBtn.innerHTML = chevronLeft;
    toggleBtn.title = '展开侧栏';
} else {
    sidebar.classList.remove('collapsed');
    toggleBtn.innerHTML = chevronRight;
    toggleBtn.title = '折叠侧栏';
}

// 导出为全局函数，供快捷键使用
window.toggleAISessionSidebar = function() {
    const wasCollapsed = sidebar.classList.contains('collapsed');
    const isCollapsed = sidebar.classList.toggle('collapsed');
    toggleBtn.innerHTML = isCollapsed ? chevronLeft : chevronRight;
    toggleBtn.title = isCollapsed ? '展开侧栏' : '折叠侧栏';
    localStorage.setItem('ai_sidebar_collapsed', String(!isCollapsed));
    if (wasCollapsed && !isCollapsed) {
        loadSessionList();
    }
};

toggleBtn.addEventListener('click', window.toggleAISessionSidebar);
```

### 改动 2: main.js — 添加 Ctrl+J 快捷键处理

**文件**: `frontend/src/main.js`

**What**: 在 `handleKeyboardNavigation` 函数中添加 Ctrl+J 的快捷键处理。

**Where**: 在 Ctrl+E 和 F11 之间（约第 5222 行之后），或放在 Ctrl+Q 之后（约第 5242 行之后），Escape 之前。选择放在 Ctrl+E 之后、F11 之前，与其他非数字 Ctrl 快捷键相邻。

**Why**: 避免与其他快捷键冲突；仅在 AI 助手视图活跃时生效。

**How**:
```javascript
// Ctrl+J: AI 助手侧栏折叠/展开（仅 AI 助手视图生效）
if (e.ctrlKey && (e.key === 'j' || e.key === 'J') && state.currentView === 'ai-chat') {
    e.preventDefault();
    if (typeof window.toggleAISessionSidebar === 'function') {
        window.toggleAISessionSidebar();
    }
    return;
}
```

**注意**: 不加 `!e.target.closest('input, textarea, [contenteditable]')` 检查，因为 AI 聊天输入框是 `contenteditable` 区域，如果在输入中触发 Ctrl+J 会与输入行为冲突。但用户期望在聊天输入时也能折叠侧栏查看更多内容。因此需要判断：**如果焦点在 AI 聊天输入框内，仍然允许触发**，或者更简单地——不考虑输入框保护（因为 Ctrl+J 不是标准文本编辑快捷键）。

**决策**: 去掉对输入框的保护限制（具体来说，不加入 `!e.target.closest(...)` 条件），但在 `state.currentView === 'ai-chat'` 时总是有效。

### 改动 3: main.js — 更新快捷键说明页面

**文件**: `frontend/src/main.js` 第 5562-5589 行

**What**: 在 `renderShortcutsPage` 的快捷键数组中新增 `Ctrl+J` 条目。

**Where**: 插入在 `Ctrl + 8`（AI 助手）条目之后。

**How**:
```javascript
{ key: 'Ctrl + 8', desc: 'AI 助手' },
{ key: 'Ctrl + J', desc: 'AI 侧栏折叠/展开' },
```

## Assumptions & Decisions
- Ctrl+J 在 AI 聊天输入框内也有效（因为 Ctrl+J 不是标准编辑快捷键，且用户可能想在输入时折叠侧栏查看更多内容）
- 字母快捷键统一使用大写 J 在说明页显示
- `state.currentView === 'ai-chat'` 是已有的状态变量，由 `switchView('ai-chat')` 设置
- 不需要双重保护（编辑器激活 + AI 视图的判断互斥，二者不会同时发生）

## Verification
1. `npm run build` 前端构建通过，无语法错误
2. 代码审查确认：
   - `toggleAISessionSidebar` 正确引用 `sidebar` 和 `toggleBtn` 变量
   - Ctrl+J 仅在 `state.currentView === 'ai-chat'` 时触发
   - 快捷键说明页面新增了 Ctrl+J 条目
