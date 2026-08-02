# 调整卡片标题字号 & 新增标签 3 个限制

## Summary

1. 将笔记卡片标题字号从 `1.1rem` 调小至 `1rem`，降低在首页的视觉突兀感
2. 在新建/编辑笔记时，限制最多选择 3 个标签，超出时拒绝选择并调用通知提示
3. 在批量标签弹窗中，追加标签时同样限制最多选择 3 个标签

## Current State Analysis

### 卡片标题字号
- 文件：`frontend/src/css/components/main-content.css` 第 216 行
- 当前值：`font-size: 1.1rem; font-weight: 700;`（方案 G 设定）
- 用户反馈：在首页展示略显突兀，需要稍微调小

### 编辑器标签选择逻辑
- 文件：`frontend/src/main.js` 第 5056-5068 行
- 函数：`window.toggleEditorTag(tagId, el)` — 切换编辑器标签选择器的选中状态
- 当 `state.selectedTags.indexOf(tagId) > -1` 时取消选中，否则添加
- `state.selectedTags` 是编辑器侧维护的选中标签 ID 数组

### 批量标签选择逻辑
- 文件：`frontend/src/main.js` 第 5371-5377 行
- 函数：`onBatchTagClick(el)` — 切换批量弹窗中标签芯片的选中态
- 通过 `batchTagAction` 区分 `'add'`（追加）和 `'remove'`（移除）模式
- `confirmBatchTagAction`（第 5382 行）读取所有选中芯片并执行操作

### 通知系统
- 文件：`frontend/src/js/notification.js` 第 176 行
- `window.showNotification(msg, type, duration)` — 全局通知函数
- 支持类型：`'info'`、`'warning'`、`'error'`、`'success'`

## Proposed Changes

### Change 1: 调小卡片标题字号

**文件**: `frontend/src/css/components/main-content.css`

| 位置 | 当前值 | 修改为 |
|------|--------|--------|
| 第 216 行 `.card-title` | `font-size: 1.1rem` | `font-size: 1rem` |

**Why**: `1rem` 等于根字号（16px），比当前 `1.1rem`（约 17.6px）略小但依然保持 700 字重，维持标题与正文的区分度，同时降低突兀感。

### Change 2: 编辑器标签选择器增加 3 个限制

**文件**: `frontend/src/main.js`

**函数**: `window.toggleEditorTag`（第 5056-5068 行）

**修改逻辑**:
```js
window.toggleEditorTag = function (tagId, el) {
    const idx = state.selectedTags.indexOf(tagId);
    if (idx > -1) {
        // 取消选中：无限制
        state.selectedTags.splice(idx, 1);
        el.classList.remove('active');
    } else {
        // 新增选中：检查是否已达上限 3
        if (state.selectedTags.length >= 3) {
            window.showNotification('一篇笔记最多选择 3 个标签', 'warning');
            return; // 拒绝选择
        }
        state.selectedTags.push(tagId);
        el.classList.add('active');
    }
    // 点击脉冲动画（保持不变）
    el.classList.add('clicked');
    setTimeout(() => el.classList.remove('clicked'), 250);
};
```

**Why**: 在 `else` 分支（新增标签）开头增加长度检查，当 `state.selectedTags.length >= 3` 时拒绝添加并调用 `showNotification` 提示。取消选中无限制。

### Change 3: 批量标签选择器增加 3 个限制

**文件**: `frontend/src/main.js`

**函数**: `onBatchTagClick`（第 5371-5377 行）

**修改逻辑**:
```js
function onBatchTagClick(el) {
    const isAdd = batchTagAction === 'add';
    // 追加模式下，如果芯片当前未选中且已选 ≥ 3，拒绝
    if (isAdd && !el.classList.contains('selected')) {
        const count = els.batchTagList.querySelectorAll('.batch-tag-chip.selected').length;
        if (count >= 3) {
            window.showNotification('一篇笔记最多选择 3 个标签', 'warning');
            return;
        }
    }
    el.classList.toggle('selected');
    const count = els.batchTagList.querySelectorAll('.batch-tag-chip.selected').length;
    const label = isAdd ? '确定添加' : '确定移除';
    els.batchTagConfirmBtn.textContent = count > 0 ? `${label}（${count}）` : label;
}
```

**Why**: 追加模式下，当芯片从「未选中」变为「选中」时检查已选数量是否 ≥ 3。移除模式无需限制（移除标签不会超过 3 个）。取消选择也无需限制。

### 不涉及修改
- 批量标签弹窗 `confirmBatchTagAction` — 前端选择已限制，确认时无需再检查
- 只读模式下的标签渲染（`renderTagSelector(readOnly=true)`）— 仅展示，不可交互

## Verification

1. 构建验证：运行 `npx vite build` 确认无编译错误
2. 视觉验证：笔记首页卡片标题字号从约 17.6px 降到 16px，是否更协调
3. 功能验证：
   - 新建笔记时尝试选择第 4 个标签，应被拒绝并弹出提示
   - 编辑笔记时尝试选择第 4 个标签，同样拒绝
   - 批量标签弹窗（追加模式）尝试选择第 4 个标签，同样拒绝
   - 取消选中标签后，可重新选择其他标签（不超过 3 个）
   - 批量移除模式不受限制