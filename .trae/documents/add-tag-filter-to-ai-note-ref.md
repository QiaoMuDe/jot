# AI 引用笔记选择器添加标签筛选条件

## 概述

在 AI 助手引用笔记选择器的过滤器栏中，笔记本下拉框旁边增加一个标签筛选下拉菜单，允许用户按标签 AND 过滤笔记。

## 当前状态

### 过滤器栏现有结构
- `.ai-note-ref-search-row`：水平 flex 布局（gap:6px, padding:10px 20px）
  - 搜索输入框（flex:1）
  - 笔记本下拉框（min-width:110px, flex-shrink:0）

### 标签 API 已就绪
- `GetAllTags()` → `[]models.Tag`（后端已有，无需改动）
- `SearchNotes` 第 8 参数 `tagIDs []uint` 已支持标签 AND 过滤（已在搜索弹窗中使用）

### 参考实现
搜索弹窗 (`main.js`) 中已有完整的标签筛选实现：
- 按钮 + 下拉菜单模式
- `state.searchModalTagIds` (Set) 存储选中标签
- `renderTagFilterDropdown()` 渲染下拉选项
- `loadTags()` → `GetAllTags()` 加载标签列表
- 标签变更 → `_triggerFilterSearch()` 立即搜索

## 修改计划

### 文件 1：`frontend/index.html`

**位置**：`.ai-note-ref-search-row` 内，笔记本下拉框之后

**改动**：在笔记本 `<select>` 后面添加标签筛选按钮和下拉菜单结构

```html
<!-- 标签筛选 -->
<div class="ai-note-ref-filter" id="aiNoteRefTagFilter">
    <button class="ai-note-ref-filter-btn" id="aiNoteRefTagBtn">
        <span id="aiNoteRefTagLabel">标签</span>
        <svg class="chevron" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
    </button>
    <div class="ai-note-ref-filter-dropdown" id="aiNoteRefTagDropdown"></div>
</div>
```

**为什么这样设计**：
- 使用独立的类名（`.ai-note-ref-filter-*`）而非复用搜索弹窗的类名，避免 CSS 冲突
- 结构与搜索弹窗一致：按钮显示当前选中状态 + chevron 箭头 + 下拉菜单
- `gap:6px` 的 flex 容器自然容纳第三个控件

### 文件 2：`frontend/src/css/components/ai-chat.css`

**位置**：`.ai-note-ref-notebook-select` 样式之后（约第 1218 行）

**改动**：新增标签筛选按钮和下拉菜单的完整样式

```css
/* ── 标签筛选按钮 ── */
.ai-note-ref-filter {
    position: relative;
    display: flex;
    align-items: center;
}
.ai-note-ref-filter-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 7px 10px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background: var(--input-bg);
    color: var(--text-primary);
    font-size: 0.78rem;
    font-family: inherit;
    cursor: pointer;
    white-space: nowrap;
    transition: border-color 0.15s, background 0.15s, color 0.15s;
}
.ai-note-ref-filter-btn:hover {
    border-color: var(--accent-light);
    background: var(--hover-bg);
}
.ai-note-ref-filter-btn.active {
    border-color: var(--accent);
    color: var(--accent);
    background: var(--accent-light);
    font-weight: 500;
}
.ai-note-ref-filter-btn .chevron {
    transition: transform 0.2s ease-out;
    opacity: 0.6;
    flex-shrink: 0;
}
.ai-note-ref-filter-btn.active .chevron {
    transform: rotate(180deg);
    opacity: 1;
}

/* ── 标签下拉菜单 ── */
.ai-note-ref-filter-dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 150px;
    max-height: 200px;
    overflow-y: auto;
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
    z-index: 10;
    opacity: 0;
    transform: translateY(-4px);
    pointer-events: none;
    transition: opacity 0.15s ease-out, transform 0.15s ease-out;
    padding: 4px 0;
}
.ai-note-ref-filter.open .ai-note-ref-filter-dropdown {
    opacity: 1;
    transform: translateY(0);
    pointer-events: auto;
}
.ai-note-ref-filter-option {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 7px 12px;
    font-size: 0.78rem;
    color: var(--text-primary);
    cursor: pointer;
    user-select: none;
    transition: background 0.1s;
}
.ai-note-ref-filter-option:hover {
    background: var(--accent-light);
    color: var(--accent);
}
.ai-note-ref-filter-option.selected {
    background: var(--accent-light);
    color: var(--accent);
    font-weight: 500;
}
.ai-note-ref-filter-option.selected::before {
    content: '✓';
    font-weight: 700;
    margin-right: 2px;
}
```

**为什么这样设计**：
- 按钮与笔记本下拉框视觉一致（同 `padding`、同 `font-size`、同 `border-radius`）
- 下拉菜单位置、动画、阴影与搜索弹窗保持一致
- 选中项显示 `✓` 前缀 + accent 色背景
- 高度限制 200px 可滚动，避免标签过多遮挡

### 文件 3：`frontend/src/js/ai-chat.js`

#### 3a. 新增变量（约第 59 行，`_refSearchTimer` 之后）

```javascript
let refTagBtn = null;           // #aiNoteRefTagBtn
let refTagLabel = null;         // #aiNoteRefTagLabel
let refTagDropdown = null;      // #aiNoteRefTagDropdown
let refTagFilter = null;        // #aiNoteRefTagFilter
let _refTagIds = new Set();     // 已选标签 ID 集合
let _tagsCache = null;          // 标签列表缓存
```

#### 3b. `initAIChat()` 中获取 DOM 引用（约第 160 行）

```javascript
refTagBtn = document.getElementById('aiNoteRefTagBtn');
refTagLabel = document.getElementById('aiNoteRefTagLabel');
refTagDropdown = document.getElementById('aiNoteRefTagDropdown');
refTagFilter = document.getElementById('aiNoteRefTagFilter');
```

#### 3c. `bindEvents()` 中绑定事件

**标签按钮点击**（toggle 下拉菜单）：
```javascript
if (refTagBtn) {
    refTagBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        refTagFilter.classList.toggle('open');
        if (refTagFilter.classList.contains('open')) {
            renderRefTagDropdown();
        }
    });
}
```

**点击外部关闭下拉菜单**（在 `bindEvents` 末尾或已有类似逻辑处）：
```javascript
document.addEventListener('click', (e) => {
    if (refTagFilter && !refTagFilter.contains(e.target)) {
        refTagFilter.classList.remove('open');
    }
});
```
> 注意：如果已有全局 click 监听，则合并到其中；若无，单独添加。

#### 3d. 新增 `loadAllRefTags()` 函数（与 `loadAllNotebooks` 模式一致，带缓存）

```javascript
/**
 * 加载所有标签到缓存（带缓存，仅首次调用时向后端请求）
 */
async function loadAllRefTags() {
    try {
        if (_tagsCache) return;
        const tags = await window.go.main.App.GetAllTags() || [];
        _tagsCache = tags;
    } catch (_) { /* 静默 */ }
}
```

#### 3e. 新增 `renderRefTagDropdown()` 函数

```javascript
/**
 * 渲染标签筛选下拉菜单
 */
function renderRefTagDropdown() {
    if (!refTagDropdown) return;
    const tags = _tagsCache || [];
    let html = '';
    // "全部"选项
    const allSelected = _refTagIds.size === 0;
    html += `<div class="ai-note-ref-filter-option${allSelected ? ' selected' : ''}" data-tag-id="all">全部</div>`;
    // 各标签选项
    tags.forEach(tag => {
        const selected = _refTagIds.has(tag.id);
        html += `<div class="ai-note-ref-filter-option${selected ? ' selected' : ''}" data-tag-id="${tag.id}">#${tag.name || ''}</div>`;
    });
    refTagDropdown.innerHTML = html;

    // 绑定点击事件
    refTagDropdown.querySelectorAll('.ai-note-ref-filter-option').forEach(opt => {
        opt.addEventListener('click', (e) => {
            e.stopPropagation();
            const tagId = opt.dataset.tagId;
            if (tagId === 'all') {
                _refTagIds.clear();
            } else {
                const id = parseInt(tagId);
                if (_refTagIds.has(id)) {
                    _refTagIds.delete(id);
                } else {
                    _refTagIds.add(id);
                }
            }
            // 关闭下拉
            if (refTagFilter) refTagFilter.classList.remove('open');
            // 更新按钮 active 样式 + label
            updateRefTagFilterBtn();
            // 刷新列表
            loadNoteList();
        });
    });
}

/**
 * 更新标签筛选按钮状态
 */
function updateRefTagFilterBtn() {
    if (!refTagBtn || !refTagLabel) return;
    const count = _refTagIds.size;
    refTagBtn.classList.toggle('active', count > 0);
    if (count === 0) {
        refTagLabel.textContent = '标签';
    } else if (count === 1) {
        const id = Array.from(_refTagIds)[0];
        const tag = (_tagsCache || []).find(t => t.id === id);
        refTagLabel.textContent = '#' + (tag ? tag.name : '');
    } else {
        refTagLabel.textContent = count + ' 个标签';
    }
}
```

#### 3f. 修改 `openNoteRefModal()` 重置标签状态（第 1141 行 `_refPendingRefresh = false` 之后）

```javascript
_refTagIds.clear();
```

并在 `Promise.all` 中并行加载标签：

```javascript
await Promise.all([
    loadAllNotebooks(),
    loadAllRefTags(),
    loadNoteList()
]);
```

#### 3g. 修改 `loadNoteList()` 传递标签参数（第 1256 行）

将：
```javascript
result = await window.go.main.App.SearchNotes(query, page, _refPageSize, notebookId, 'updated_at', '', '', []);
```
改为：
```javascript
const tagIds = _refTagIds.size > 0 ? Array.from(_refTagIds) : [];
result = await window.go.main.App.SearchNotes(query, page, _refPageSize, notebookId, 'updated_at', '', '', tagIds);
```

### 无需改动的后端文件

- `app.go` — `SearchNotes` 已接受 `tagIDs []uint` 参数
- `note_service.go` — `Search()` 和 `SearchByNotebook()` 已实现标签 AND 过滤子查询
- `tag_service.go` — `GetAll()` 已存在

## 影响范围

| 改动点 | 影响 |
|--------|------|
| index.html 新增标签筛选 DOM | 无副作用，初始 `display` 由 CSS 控制 |
| ai-chat.css 新增标签筛选样式 | 仅作用于 `.ai-note-ref-filter-*`，不影响其他元素 |
| ai-chat.js 新增变量/函数 | 仅在使用引用笔记选择器时生效 |
| `loadNoteList` 传参改动 | 搜索弹窗不受影响（main.js 已正常传参） |

## 验证步骤

1. 打开 AI 助手 → 点击引用笔记按钮
2. 确认过滤器栏出现第三个控件"标签"按钮
3. 点击"标签"按钮 → 下拉菜单展开，显示所有标签
4. 点击某个标签 → 下拉关闭，按钮显示 `#标签名`，笔记列表按该标签过滤
5. 再次点击标签按钮 → 选择另一个标签 → 笔记列表按 AND 逻辑过滤
6. 点击"全部" → 清除标签过滤
7. 组合测试：搜索关键词 + 笔记本 + 标签 → 三者同时生效
8. 关闭弹窗重新打开 → 标签筛选状态重置
