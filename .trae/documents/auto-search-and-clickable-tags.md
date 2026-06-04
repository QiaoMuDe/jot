# 自动搜索与可点击标签实施计划

## 概述
实现搜索框自动搜索（防抖 250ms）、卡片标签可点击触发搜索、搜索结果展示标签并可点击。

## 当前状态分析

### 后端 — 无需修改
- `services/note_service.go:Search()` 已通过子查询支持标签名搜索（INNER JOIN `note_tags` + `tags`）
- `app.go` 已暴露 `SearchNotes` 绑定方法

### 前端 — 需要修改
- **搜索触发**：`keydown(Enter)` + 按钮点击（`frontend/src/main.js:727-731`）
- **卡片标签**：`card-tag` span 无 onclick（`frontend/src/main.js:469-474`）
- **搜索结果**：不显示标签（`frontend/src/main.js:508-518`）
- **工具函数**：`debounce()` 已存在（`frontend/src/main.js:113-119`）

## 修改方案

### 1. `frontend/src/main.js` — 搜索框自动搜索

将 `searchInput` 的 `keydown` 事件替换为 `input` 事件 + 防抖：

```js
// 移除（约第 727-731 行）：
els.searchInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        searchNotes(els.searchInput.value);
    }
});

// 替换为：
els.searchInput.addEventListener('input', debounce(function () {
    const kw = this.value.trim();
    if (kw) {
        searchNotes(kw, 'input');
    } else {
        state.searchKeyword = '';
        switchView('grid');
    }
}, 250));
```

保留搜索按钮 `click` 不变，但改为立即搜索（无防抖）。

### 2. `frontend/src/main.js` — 卡片标签可点击

在 `renderCardGrid()` 函数中，修改标签渲染（约第 469-474 行）：

```js
// 原来的 span：
<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}">${escapeHtml(tag.name)}</span>

// 改为：
<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}" 
      onclick="event.stopPropagation(); window.searchByTag('${escapeHtml(tag.name)}')">${escapeHtml(tag.name)}</span>
```

### 3. `frontend/src/main.js` — 搜索结果展示标签

在 `renderSearchResults()` 函数中，为每个搜索结果项添加标签行（约第 514 行后）：

```js
// 在 result-time 之前添加标签区域：
${(note.tags || []).length > 0 ? `
<div class="result-tags">
    ${(note.tags || []).map(tag => 
        `<span class="card-tag" style="background-color: ${tag.color || '#6366f1'}" 
               onclick="event.stopPropagation(); window.searchByTag('${escapeHtml(tag.name)}')">${escapeHtml(tag.name)}</span>`
    ).join('')}
</div>` : ''}
```

### 4. `frontend/src/main.js` — 添加 `searchByTag` 全局函数

在 `window.permanentDeleteNote` 附近添加：

```js
window.searchByTag = function (tagName) {
    els.searchInput.value = tagName;
    searchNotes(tagName, 'tag');
};
```

### 5. `frontend/src/style.css` — 添加标签样式

将 `.card-tag` 添加 `cursor: pointer` 样式，并添加搜索结果标签容器样式：

```css
.card-tag {
    cursor: pointer;  /* 新增 */
}

.result-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 8px;
}
```

## 涉及文件
| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/main.js` | 编辑 | 4 处改动：事件绑定、卡片标签、搜索结果标签、全局函数 |
| `frontend/src/style.css` | 编辑 | 添加 cursor:pointer 和 .result-tags 样式 |

## 验证步骤
1. 在搜索框输入文字，观察 250ms 后是否自动触发搜索
2. 清空搜索框，观察是否返回网格视图
3. 点击卡片上的标签，观察是否触发标签搜索并显示结果
4. 点击搜索结果中的标签，观察是否跳转到标签搜索
5. 确认搜索按钮点击仍然正常工作
