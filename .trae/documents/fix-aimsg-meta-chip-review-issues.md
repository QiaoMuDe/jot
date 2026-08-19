# 修复 AI 消息 Meta Chip 代码审查问题

## Summary

修复 `ai-chat.js` 与 `ai-chat.css` 中 6 个确认问题和 2 个理论隐患，全部是小改动，零风险。

## Current State Analysis

已完成代码审查与逐条核实，结论（详细见对话历史）：

| 编号 | 问题 | 核实结论 |
|---|---|---|
| M4 | `.ai-msg-text` `flex: 1 1 auto` 抢占第一行空间，chip 被挤到第二行 | ✅ 真问题 |
| S2 | `.ai-msg-chip-tag` 全 CSS 目录零命中，截断角标无样式 | ✅ 真问题 |
| S1 | `createChipElement` 不过滤空 label | ✅ 真问题 |
| M1 | `applyEdit` 渲染两次（rerender + cancelEdit） | ✅ 真问题 |
| M3 | `cancelEdit` 找不到 chatHistory 条目时无任何日志 | ✅ 真问题 |
| R4 | `userMsgId=0`（保存失败）时编辑按钮无效 | ✅ 真问题 |
| R1 | `parseInt` 缺参返回 `NaN` | 🤔 理论隐患 |
| R3 | `bindMsgContextMenu` 无去重检查 | 🤔 理论隐患 |

## Proposed Changes

### Fix 1: M4 — chip 错位（CSS 一行）
**文件**：`frontend/src/css/components/ai-chat.css` L110
**改动**：把 `.ai-msg-text` 的 `flex: 1 1 auto` 改为 `flex: 0 1 auto`
**原因**：当前 `flex: 1 1 auto` 让文本段 grow 抢满第一行，chip 总是换行；改为 `0 1 auto` 后文本按内容宽度走，chip 自然跟在后面
```css
/* before */
.ai-msg-user .msg-content .ai-msg-text {
    white-space: pre-wrap;
    flex: 1 1 auto;   /* 抢占剩余空间，导致 chip 换行 */
    min-width: 0;
    line-height: 1.6;
}
/* after */
.ai-msg-user .msg-content .ai-msg-text {
    white-space: pre-wrap;
    flex: 0 1 auto;   /* 不增长，可收缩；chip 同行排列 */
    min-width: 0;
    line-height: 1.6;
}
```

### Fix 2: S2 — 截断角标 CSS（新增样式块）
**文件**：`frontend/src/css/components/ai-chat.css` 在 `.ai-msg-chip-label` 样式后追加
**改动**：新增 `.ai-msg-chip-tag` 样式，半透明白色背景圆角小标签
```css
/* 截断角标（仅 file 类型 truncated=true 时显示） */
.ai-msg-user .ai-msg-chip-tag {
    margin-left: 4px;
    padding: 0 5px;
    background: rgba(255, 200, 0, 0.32);
    border: 1px solid rgba(255, 200, 0, 0.55);
    border-radius: 3px;
    font-size: 0.72em;
    color: #fff;
    opacity: 0.92;
}
```

### Fix 3: S1 — createChipElement 空 label 过滤
**文件**：`frontend/src/js/ai-chat.js` L3227
**改动**：在 `createChipElement` 函数开头加空值检查；调用方处理 `null` 返回值
```js
function createChipElement(item) {
    const text = item.title || item.name || item.label || item.id || '';
    if (!text) return null;  // 空 label 直接跳过
    const chip = document.createElement('span');
    // ... 其余代码保持不变
}
```
**调用方 L3274 附近**（`renderUserMessageWithChips` 的 for 循环）加 null 跳过：
```js
for (const it of items) {
    const chip = createChipElement(it);
    if (!chip) continue;  // 跳过空 label
    contentEl.appendChild(chip);
}
```

### Fix 4: M1 — applyEdit 重复渲染
**文件**：`frontend/src/js/ai-chat.js` L4880-4885
**改动**：`applyEdit` 末尾不再调用 `cancelEdit`，改为局部清理编辑态（移除 textarea、恢复 actions/more-btn），因为 `rerenderUserMessageChips` 已经完成了内容渲染
```js
// 之前
rerenderUserMessageChips(msgId, newMeta);
msgEl.dataset.originalContent = newContent;
cancelEdit(msgEl);  // ← 第二次渲染

// 之后
rerenderUserMessageChips(msgId, newMeta);
msgEl.dataset.originalContent = newContent;
exitEditModeWithoutRerender(msgEl);  // 仅清理编辑态 DOM
```

**新增辅助函数**（在 `cancelEdit` 附近）：
```js
function exitEditModeWithoutRerender(msgEl) {
    const contentDiv = msgEl.querySelector('.msg-content');
    if (contentDiv) {
        const textarea = contentDiv.querySelector('.ai-msg-edit-textarea');
        if (textarea) textarea.remove();
    }
    const actions = msgEl.querySelector('.msg-actions');
    if (actions) actions.style.display = '';
    const editActions = msgEl.querySelector('.ai-msg-edit-actions');
    if (editActions) editActions.remove();
    const moreBtn = msgEl.querySelector('.msg-more-btn');
    if (moreBtn) moreBtn.style.display = '';
}
```

### Fix 5: M3 — cancelEdit 加 console.warn
**文件**：`frontend/src/js/ai-chat.js` L4775-4779
**改动**：`chatHistory.find` 找不到时加 warn 日志
```js
if (msgId) {
    const found = chatHistory.find(m => m.id === msgId);
    if (found) metaJson = found.meta || '';
    else console.warn('[AI Chat] cancelEdit: chatHistory 未找到 msgId', msgId);
}
```

### Fix 6: R4 — userMsgId=0 时编辑入口加守卫
**文件**：`frontend/src/js/ai-chat.js` L1197（context menu `edit` action）
**改动**：进入编辑前检查 msgId，缺失时弹窗提示并退出
```js
} else if (action === 'edit') {
    if (isStreaming) return;
    const _msgId = msgEl ? parseInt(msgEl.dataset.msgId, 10) : NaN;
    if (!Number.isFinite(_msgId) || _msgId <= 0) {
        window.showNotification && window.showNotification('该消息尚未完整保存，无法编辑', 'warn');
        return;
    }
    if (msgEl) enterEditMode(msgEl, content);
}
```

### Fix 7: R1 — parseInt 显式 Number.isFinite
**文件**：`frontend/src/js/ai-chat.js` L4773、L4970 等所有用 `parseInt(msgEl.dataset.msgId)` 的位置
**改动**：用 `Number.isFinite` 显式判断
```js
const msgId = Number.isFinite(parseInt(msgEl.dataset.msgId, 10)) ? parseInt(msgEl.dataset.msgId, 10) : 0;
```
或更简洁地封装：
```js
const msgId = +msgEl.dataset.msgId || 0;  // 一元加号 + 兜底 0
```

### Fix 8: R3 — bindMsgContextMenu 去重守卫
**文件**：`frontend/src/js/ai-chat.js` L3920
**改动**：函数开头加去重标记
```js
function bindMsgContextMenu(msgEl, content, role) {
    if (!msgEl || msgEl._ctxMenuBound) return;
    msgEl._ctxMenuBound = true;
    msgEl.addEventListener('contextmenu', (e) => {
        showAiMsgContextMenu(e, content, role, msgEl);
    });
}
```

## Assumptions & Decisions

| 决策 | 理由 |
|---|---|
| 用 CSS 类 `_ctxMenuBound` 标记而不是 `WeakSet` | 简单直接，与元素生命周期绑定；DOM 移除时自动失效 |
| S1 在 `createChipElement` 内过滤 + `buildUserMessageMeta` 出口层过滤 | 双层保险：单点修复不够，未来加新 type 也安全 |
| M1 新增 `exitEditModeWithoutRerender` 而非删 `cancelEdit` 调用 | `cancelEdit` 在其他地方还可能被纯取消场景调用，不破坏 |
| R4 用 `window.showNotification` | 沿用项目现有提示机制（与 import 通知一致） |
| R1 用 `+msgEl.dataset.msgId \|\| 0` 而非 `Number.isFinite` | 简洁，语义清晰（缺失/无效 = 0，调用方已用 `if (msgId)` 判断） |

## Files Touched

| 文件 | 行数变化 |
|---|---|
| `frontend/src/css/components/ai-chat.css` | +12 行（Fix 1 改 1 行 + Fix 2 新增 11 行）|
| `frontend/src/js/ai-chat.js` | +20 行（Fix 3 改 2 处 + Fix 4 新增 12 行 + Fix 5 改 1 行 + Fix 6 改 1 处 + Fix 7 改若干处 + Fix 8 改 1 处）|

总计：~32 行改动，2 个文件。

## Verification Steps

### 静态检查
1. `go vet ./...` 退出码 0
2. `cd frontend && npm run build` 退出码 0
3. `wails build` 退出码 0，产出 `jot.exe`

### 功能验证（手动）
| 场景 | 预期 |
|---|---|
| Fix 1：发带 1 个 ref 的消息 | chip 与文本**同一行**（修复前 chip 在第二行）|
| Fix 2：上传大文件触发截断 | 出现带橙黄色背景的"截断"小角标 |
| Fix 3：极端空数据 | 不再出现"只有图标的空 chip" |
| Fix 4：编辑消息 → 确认 | 消息气泡**只渲染一次**（DevTools 看 DOM 闪烁消失）|
| Fix 5：制造 chatHistory 缺失 | 浏览器控制台出现 warn 日志 |
| Fix 6：保存失败的消息点编辑 | 弹出"该消息尚未完整保存，无法编辑"提示 |
| Fix 7：人为把 dataset.msgId 设为 'abc' | 走 `if (msgId)` 兜底，meta 走空字符串降级 |
| Fix 8：重复调用 bindMsgContextMenu | `_ctxMenuBound=true` 后第二次早返回，contextmenu 只触发一次 |

### 回归测试（确认旧功能没坏）
- 1、4、5、6、7、8、9、10、11 场景（来自 [spec.md](file:///d:/Users/27766/.trae-cn/memory/projects/-d---------Dev------jot--p2-574e433c27e727c5d3c2/project_memory.md) 既有功能）

## Work Order

按以下顺序修复（每步独立可验证）：

1. **Fix 1**（M4）— 1 行 CSS，立即解决最明显的视觉问题
2. **Fix 2**（S2）— 新增 ~10 行 CSS
3. **Fix 3**（S1）— 2 处 JS
4. **Fix 5**（M3）— 1 行 JS
5. **Fix 6**（R4）— 1 处 JS
6. **Fix 8**（R3）— 1 处 JS
7. **Fix 7**（R1）— 若干处 JS
8. **Fix 4**（M1）— 1 处 JS + 新增 12 行辅助函数（涉及最多文件，最后做）

每步后运行 `npm run build` 确认无编译错误，全部完成后 `wails build` 出包。
