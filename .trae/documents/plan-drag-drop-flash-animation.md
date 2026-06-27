# 拖拽导入改为卡片红色闪烁动画

## Summary

拖拽文件导入后，不再自动打开编辑器预览最后一条笔记。改为：导入成功后刷新列表，所有成功导入的笔记卡片以红色边框闪烁动画醒目标记。用户可自行点击查看。

## Current State Analysis

当前 `handleFileDropPaths()` ([main.js:5888-5933](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L5888-L5933)) 流程：
```
后端 ImportFiles → 收集结果 → loadNotes() + loadNotebooks() → openEditor(lastNoteId, true)
```

卡片渲染 `renderCardGrid()` ([main.js:2185-2291](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L2185-L2291))：
- 卡片 `<div class="note-card">` 上**没有** `data-id` 属性
- note.id 只出现在内联 `onclick`/`oncontextmenu` 参数中
- `.note-card` 默认 `border: 1px solid transparent` ([main-content.css:189](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/main-content.css#L189))

现有动画：
- `dangerFlash` keyframes ([main-content.css:567](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/main-content.css#L567-L571))：闪烁背景色，但暂不需要
- `cardEnter`：卡片入场动画（JS 内联）

## Proposed Changes

### 1. CSS — 新增 `cardFlash` 红色边框闪烁动画

**文件**: `d:\峡谷\Dev\本地项目\jot\frontend\src\css\components\main-content.css`

在 Section C（卡片网格）末尾新增 `@keyframes cardFlash`：
```css
@keyframes cardFlash {
    0%   { border-color: transparent; }
    20%  { border-color: #EF4444; box-shadow: 0 0 8px rgba(239, 68, 68, 0.4); }
    40%  { border-color: transparent; box-shadow: none; }
    60%  { border-color: #EF4444; box-shadow: 0 0 8px rgba(239, 68, 68, 0.4); }
    80%  { border-color: transparent; box-shadow: none; }
    100% { border-color: transparent; }
}
/* 闪烁后保留微弱的残留态，方便用户定位 */
.note-card.flash-done {
    border-color: rgba(239, 68, 68, 0.3);
}
.note-card.flash-done:hover {
    border-color: var(--border);
}
```

- 硬编码 `#EF4444` 红色，不依赖主题变量，确保醒目
- 闪烁 2 次（20%/40%/60%/80%），总时长约 1.2s
- `flash-done` 类保留微弱红色边框残留，hover 时恢复

### 2. JS — 卡片模板添加 `data-id` 属性

**文件**: `d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`

在 `renderCardGrid()` 的卡片模板字符串中（[line 2226](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L2226)），为 `<div class="note-card">` 添加 `data-id`：

```
改动前: <div class="note-card${...}" onclick="${...}" oncontextmenu="${...}">
改动后: <div class="note-card${...}" data-id="${note.id}" onclick="${...}" oncontextmenu="${...}">
```

### 3. JS — 重写 `handleFileDropPaths` 闪烁逻辑

**文件**: `d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`

改动 `handleFileDropPaths()`：
1. 收集所有成功导入的 note ID 到数组 `importedNoteIds[]`
2. 移除 `lastNoteId` 和 `openEditor(lastNoteId, true)` 调用
3. `loadNotes()` + `loadNotebooks()` 刷新后，调用新函数 `flashNoteCards(importedNoteIds)`

新增函数 `flashNoteCards(noteIds)`：
```js
function flashNoteCards(noteIds) {
    if (!noteIds || noteIds.length === 0) return;
    // 延迟到 DOM 更新完毕（loadNotes 异步可能刚完成渲染）
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            noteIds.forEach((id, index) => {
                const card = els.cardGrid.querySelector(`.note-card[data-id="${id}"]`);
                if (card) {
                    // 清除可能存在的入场动画，应用闪烁动画
                    card.style.animation = `cardFlash 1.2s ease-out forwards`;
                    card.style.animationDelay = `${index * 150}ms`; // 多文件交错闪烁
                    // 动画结束后添加残留态
                    card.addEventListener('animationend', function handler() {
                        card.removeEventListener('animationend', handler);
                        card.classList.add('flash-done');
                        card.style.animation = '';
                    });
                }
            });
        });
    });
}
```

### 4. 通知保持不变

摘要通知（`nm.show('N 个文件导入成功', 'success')`）保留，用户同时看到通知 + 卡片闪烁，双重反馈。

## Assumptions & Decisions

| 决定 | 理由 |
|------|------|
| 红色边框闪烁而非背景闪烁 | 更醒目且不影响卡片内容可读性，与 `cardFlash` 命名一致 |
| 闪烁 2 次 + 残留态 | 闪烁提醒注意，残留态方便定位（多文件时容易忘记哪些是新导入的） |
| 多文件交错闪烁（150ms 间隔） | 避免所有卡片同时闪烁看起来杂乱 |
| 硬编码 `#EF4444` 红色 | 与 danger/错误语义一致，不依赖主题确保始终醒目 |
| `data-id` 属性方式定位卡片 | 比遍历查找更高效，QSA 选择器直接命中 |
| 保留 `flash-done` class | 用户可自行 hover 消除，也可点击卡片查看后自然消失 |

## Verification

1. 拖入单个 `.md` 文件 → 卡片网格刷新，该卡片红色边框闪烁 2 次 → 残留淡红色边框 → 鼠标悬停残留消失
2. 拖入多个文件（3-5 个）→ 所有卡片依次交错闪烁 → 通知显示成功数
3. 拖入混合文件（含失败）→ 仅成功导入的卡片闪烁 → 失败文件显示通知
4. 导入完成后不打开编辑器
5. `go build ./...` 编译通过
