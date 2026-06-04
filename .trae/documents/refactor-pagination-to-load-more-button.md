# 分页加载改造：快捷键不触发加载

## 摘要

保持当前「触底自动加载下一页」的模式不变，核心改动只有一条：

**键盘快捷键 PgUp/PgDn/Ctrl+Home/Ctrl+End 只操控已加载数据的滚动，不触发任何加载行为。**

例外：Ctrl+End 在 `hasMoreNotes = true` 时仍加载全部，因为它的语义本身就是"跳到最后"，一次性加载完所有剩余页是合理的。

## 当前状态分析

当前 [handleKeyboardNavigation](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) 中：

| 快捷键 | 当前行为 | 问题 |
|--------|----------|------|
| `PgDn` | `scrollTop += clientHeight` | 如果已滚动到底部，触发 scroll 事件 → `loadMoreNotes()` |
| `PgUp` | `scrollTop -= clientHeight` | 不触发加载 ✅ |
| `Ctrl+Home` | `scrollTop = 0` | 不触发加载 ✅ |
| `Ctrl+End` | `hasMoreNotes ? loadAllRemainingNotes() : scrollBottom` | 已处理，保持不变 |

问题本质：PgDn 本身不调用加载函数，但它改变 `scrollTop` 后触发了 scroll 监听器中的 `loadMoreNotes()`。

## 变更方案

### 改动 1：PgDn 滚动前排除加载触发的可能性

**文件**: `frontend/src/main.js` — `handleKeyboardNavigation()`

**做什么**:
PgDn 滚动前判断：如果已触底（`scrollTop + clientHeight >= scrollHeight - 1`），不做任何操作（`return`）。

```js
// PgDn: 向下翻一页（已到底时不操作）
if (e.key === 'PageDown') {
    e.preventDefault();
    const { scrollTop, scrollHeight, clientHeight } = container;
    if (scrollTop + clientHeight >= scrollHeight - 1) return; // 已到底，不触发加载
    container.scrollTop = scrollTop + clientHeight;
    return;
}
```

### 改动 2：Ctrl+End 保持现状

**文件**: `frontend/src/main.js` — `handleKeyboardNavigation()`

已正确处理，不变：
- `hasMoreNotes = true` → `loadAllRemainingNotes()`
- `hasMoreNotes = false` → `scrollBottom`

### 不改的部分

| 部分 | 说明 |
|------|------|
| `initScrollLoading()` | 保持，鼠标滚动触底仍触发 `loadMoreNotes()` |
| `loadMoreNotes()` | 保持 |
| `loadNotes()` | 保持 |
| `loadAllRemainingNotes()` | 保持 |
| `renderCardGrid()` footer | 保持「共 X 条笔记」显示 |
| 后端 | 不改 |

## 验证步骤

1. 首屏加载 pageSize 条
2. 鼠标滚动触底 → 加载下一页 ✅
3. PgDn 滚动到页面底部附近但不触底 → 不加载 ✅
4. PgDn 到底部（无法再向下）→ 不做任何操作 ✅
5. Ctrl+End → 加载全部并到底 ✅
6. PgUp/Ctrl+Home → 只滚动，不加载 ✅
