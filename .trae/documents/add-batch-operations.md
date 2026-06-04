# 多选批量操作功能实现计划

## 一、需求摘要

- 在 topbar 操作区新增一个"批量管理"按钮（切换式）
- 默认状态下，卡片右上角 hover 显示置顶按钮（保持现有行为）
- 点击"批量管理"进入批量模式，所有卡片的置顶按钮变成复选框
- 选中卡片后，顶部出现批量操作栏（批量删除 + 批量标签）

---

## 二、当前状态分析

| 模块 | 当前状态 | 需要变更 |
|------|----------|----------|
| **卡片渲染** (`main.js:renderCardGrid`) | 右上角 `.card-actions` 置顶按钮 | 新增判断：batchMode 时渲染 checkbox |
| **状态管理** (`main.js:state`) | 无批量相关状态 | 新增 `batchMode` + `selectedNoteIds` |
| **Topbar** (`index.html` / `main.js`) | 只有 + 和 ☰ 按钮 | 在 ± 前新增"批量管理"切换按钮 |
| **后端批量删除** | 仅有单条 `DeleteNote(id)` | 新增 `BatchDeleteNotes(ids []uint)` |
| **后端批量标签** | 仅有单条 `AddTagToNote(noteID, tagID)` | 新增 `BatchAddTagToNotes(noteIDs []uint, tagID uint)` |
| **CSS** | `.card-actions`, `.card-action-btn` 等 | 新增 batch 模式复选框、选中态、batch bar 样式 |

---

## 三、变更清单

### 3.1 后端 Go 代码

#### 文件 `services/note_service.go`
在 `Delete` 方法后新增：
```go
// BatchDelete 批量软删除指定 ID 数组的笔记（移入回收站）
func (s *NoteService) BatchDelete(ids []uint) error {
    result := s.db.Where("id IN ?", ids).Delete(&models.Note{})
    return result.Error
}
```

#### 文件 `services/tag_service.go`
在 `AddTagToNote` 方法后新增：
```go
// BatchAddTagToNotes 批量将指定标签添加到多篇笔记
func (s *TagService) BatchAddTagToNotes(noteIDs []uint, tagID uint) error {
    tag, err := s.getTagByID(tagID)
    if err != nil {
        return err
    }
    for _, noteID := range noteIDs {
        note, err := s.getNoteByID(noteID)
        if err != nil {
            continue // 跳过不存在的笔记
        }
        _ = s.db.Model(note).Association("Tags").Append(tag)
    }
    return nil
}
```

#### 文件 `app.go`
在批量操作相关区域新增两个绑定方法：
```go
// BatchDeleteNotes 批量软删除笔记
func (a *App) BatchDeleteNotes(ids []uint) error {
    return a.noteService.BatchDelete(ids)
}

// BatchAddTagToNotes 批量添加标签到笔记
func (a *App) BatchAddTagToNotes(noteIDs []uint, tagID uint) error {
    return a.tagService.BatchAddTagToNotes(noteIDs, tagID)
}
```

---

### 3.2 前端 HTML

#### 文件 `frontend/index.html`

**1. Topbar 新增"批量管理"按钮**
在 `newNoteBtn` 和 `moreMenuBtn` 之间插入：
```html
<button id="batchModeBtn" class="topbar-btn" title="批量管理">☑</button>
```

**2. 批量操作栏**（在 `</header>` 之后、`<main>` 之前插入）
```html
<!-- 批量操作栏 -->
<div id="batchBar" class="batch-bar" style="display:none;">
    <span class="batch-bar-info">已选 <span id="batchCount">0</span> 条</span>
    <div class="batch-bar-actions">
        <button class="btn btn-sm batch-btn btn-danger" id="batchDeleteBtn">批量删除</button>
        <button class="btn btn-sm batch-btn" id="batchTagBtn">批量标签</button>
        <button class="btn btn-sm batch-btn batch-cancel" id="batchCancelBtn">退出批量</button>
    </div>
    <!-- 批量标签下拉面板 -->
    <div id="batchTagPanel" class="batch-tag-panel" style="display:none;">
        <div class="batch-tag-panel-inner">
            <label class="batch-tag-label">选择标签添加到选中笔记：</label>
            <div class="batch-tag-list" id="batchTagList"></div>
        </div>
    </div>
</div>
```

---

### 3.3 前端 CSS

#### 文件 `frontend/src/style.css`
在文件末尾追加：

```css
/* ===== 批量操作栏 ===== */
.batch-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 24px;
    background: #fff;
    border-bottom: 1px solid #e2e8f0;
    gap: 12px;
    flex-shrink: 0;
    animation: batchSlideDown 0.15s ease;
}

@keyframes batchSlideDown {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
}

.batch-bar-info {
    font-size: 13px;
    color: #475569;
    font-weight: 500;
    white-space: nowrap;
}

.batch-bar-actions {
    display: flex;
    align-items: center;
    gap: 8px;
}

.batch-btn {
    background: #f1f5f9;
    color: #334155;
    border-color: #e2e8f0;
}

.batch-btn:hover {
    background: #e2e8f0;
    color: #1e293b;
}

.batch-btn.btn-danger {
    background: #fef2f2;
    color: #dc2626;
    border-color: #fecaca;
}

.batch-btn.btn-danger:hover {
    background: #fee2e2;
}

.batch-cancel {
    background: transparent;
    color: #94a3b8;
    border-color: transparent;
}

.batch-cancel:hover {
    color: #64748b;
    background: #f1f5f9;
}

/* 批量标签下拉面板 */
.batch-tag-panel {
    position: absolute;
    top: calc(100% + 4px);
    right: 24px;
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
    padding: 14px 16px;
    min-width: 220px;
    z-index: 2000;
}

.batch-tag-label {
    display: block;
    font-size: 12px;
    color: #64748b;
    margin-bottom: 10px;
}

.batch-tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
}

/* ===== 批量模式复选框（替换置顶按钮位置，右上角） ===== */
.batch-checkbox {
    width: 18px;
    height: 18px;
    cursor: pointer;
    accent-color: #6366f1;
    margin: 0;
    flex-shrink: 0;
}

/* 批量模式下隐藏置顶按钮，显示复选框 */
.note-card.batch-mode .card-action-btn {
    display: none !important;
}

.note-card.batch-mode .batch-checkbox {
    display: block;
}

/* 默认隐藏复选框 */
.batch-checkbox {
    display: none;
}

/* 选中态卡片 */
.note-card.selected {
    box-shadow: 0 0 0 2px #6366f1, 0 4px 12px rgba(99, 102, 241, 0.15);
    transform: translateY(-1px);
}

/* 批量模式按钮激活态 */
#batchModeBtn.active {
    background-color: #6366f1;
    color: #fff;
    border-color: #6366f1;
}
```

---

### 3.4 前端 JS 逻辑

#### 文件 `frontend/src/main.js`

**1. 状态扩展**

```js
const state = {
    // ... 现有状态 ...
    batchMode: false,           // 是否处于批量管理模式
    selectedNoteIds: new Set(), // 选中的笔记 ID 集合
};
```

**2. DOM 引用扩展**

```js
const els = {
    // ... 现有引用 ...
    // 批量操作
    batchModeBtn: $('batchModeBtn'),
    batchBar: $('batchBar'),
    batchCount: $('batchCount'),
    batchDeleteBtn: $('batchDeleteBtn'),
    batchTagBtn: $('batchTagBtn'),
    batchCancelBtn: $('batchCancelBtn'),
    batchTagPanel: $('batchTagPanel'),
    batchTagList: $('batchTagList'),
};
```

**3. `renderCardGrid()` 修改**

在卡片 `.card-actions` 内部，将置顶按钮改为条件渲染：

```js
// 原代码（置顶按钮）:
// <button class="card-action-btn" onclick="...">${note.pinned ? '★' : '☆'}</button>

// 改为：
${state.batchMode
    ? `<input type="checkbox" class="batch-checkbox" ${state.selectedNoteIds.has(note.id) ? 'checked' : ''}
          onclick="event.stopPropagation(); window.toggleNoteSelection(${note.id})">`
    : `<button class="card-action-btn" onclick="event.stopPropagation(); window.togglePin(${note.id})" title="${note.pinned ? '取消置顶' : '置顶'}">
           ${note.pinned ? '★' : '☆'}
       </button>`
}
```

同时，在 `.note-card` div 上条件添加 `batch-mode` 和 `selected` class：

```js
`<div class="note-card ${state.batchMode ? 'batch-mode' : ''} ${state.selectedNoteIds.has(note.id) ? 'selected' : ''}" ...>`
```

**4. 新增函数**

```js
/**
 * 切换批量管理模式
 */
function toggleBatchMode() {
    state.batchMode = !state.batchMode;
    if (!state.batchMode) {
        // 退出批量模式：清空选中
        clearSelection();
    }
    els.batchModeBtn.classList.toggle('active', state.batchMode);
    renderCardGrid();
}

/**
 * 切换笔记选中状态
 */
window.toggleNoteSelection = function (id) {
    if (state.selectedNoteIds.has(id)) {
        state.selectedNoteIds.delete(id);
    } else {
        state.selectedNoteIds.add(id);
    }
    updateBatchBar();
    renderCardGrid();
};

/**
 * 更新批量操作栏
 */
function updateBatchBar() {
    const count = state.selectedNoteIds.size;
    els.batchCount.textContent = count;
    els.batchBar.style.display = state.batchMode ? 'flex' : 'none';
    if (count === 0 && state.batchMode) {
        // 批量模式下无选中也显示 bar，只是不显示计数动作
        els.batchBar.style.display = 'flex';
        els.batchCount.textContent = '0';
    }
    if (count === 0) {
        els.batchTagPanel.style.display = 'none';
    }
}

/**
 * 取消选中
 */
function clearSelection() {
    state.selectedNoteIds.clear();
    updateBatchBar();
    if (state.batchMode) {
        renderCardGrid();
    }
}

/**
 * 批量删除选中的笔记
 */
async function batchDeleteSelected() {
    const ids = Array.from(state.selectedNoteIds);
    if (ids.length === 0) return;
    if (!confirm(`确定要删除选中的 ${ids.length} 条笔记吗？`)) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchDeleteNotes) {
            await window.go.main.App.BatchDeleteNotes(ids);
        } else {
            mockNotes = mockNotes.filter(n => !ids.includes(n.id));
        }
    } catch (err) {
        console.error('批量删除失败:', err);
    }
    clearSelection();
    await loadNotes();
}

/**
 * 打开批量标签选择面板
 */
async function showBatchTagPanel() {
    // 确保标签已加载
    await loadTags();
    if (state.tags.length === 0) {
        els.batchTagList.innerHTML = '<div style="color:#94a3b8;font-size:12px;">暂无可用标签</div>';
    } else {
        els.batchTagList.innerHTML = state.tags.map(tag =>
            `<span class="card-tag" style="background-color:${tag.color || '#6366f1'};cursor:pointer;"
                   onclick="window.applyBatchTag(${tag.id})">${escapeHtml(tag.name)}</span>`
        ).join('');
    }
    els.batchTagPanel.style.display = 'block';
}

/**
 * 对选中的笔记批量应用标签
 */
window.applyBatchTag = async function (tagId) {
    const ids = Array.from(state.selectedNoteIds);
    if (ids.length === 0) return;
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.BatchAddTagToNotes) {
            await window.go.main.App.BatchAddTagToNotes(ids, tagId);
        } else {
            const tag = state.tags.find(t => t.id === tagId);
            if (tag) {
                mockNotes.forEach(n => {
                    if (ids.includes(n.id)) {
                        if (!n.tags) n.tags = [];
                        if (!n.tags.find(t => t.id === tagId)) {
                            n.tags.push({ ...tag });
                        }
                    }
                });
            }
        }
    } catch (err) {
        console.error('批量添加标签失败:', err);
    }
    els.batchTagPanel.style.display = 'none';
    clearSelection();
    await loadNotes();
};
```

**5. 事件绑定追加**

在 `initEventListeners()` 末尾追加：

```js
    // 批量管理模式
    els.batchModeBtn.addEventListener('click', toggleBatchMode);
    els.batchDeleteBtn.addEventListener('click', batchDeleteSelected);
    els.batchTagBtn.addEventListener('click', showBatchTagPanel);
    els.batchCancelBtn.addEventListener('click', () => {
        if (state.batchMode) toggleBatchMode();
    });

    // 点击外部关闭批量标签面板
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.batch-tag-panel') && !e.target.closest('#batchTagBtn')) {
            els.batchTagPanel.style.display = 'none';
        }
    });
```

**6. 键盘快捷键扩展**

在 `handleKeyboardNavigation` 中追加：

```js
    // Escape: 退出批量模式
    if (e.key === 'Escape' && state.batchMode) {
        e.preventDefault();
        toggleBatchMode();
        return;
    }
```

**7. 视图切换时退出批量模式**

在 `switchView()` 开头追加：

```js
function switchView(view) {
    // 切换视图时强制退出批量模式
    if (state.batchMode) {
        state.batchMode = false;
        els.batchModeBtn.classList.remove('active');
        clearSelection();
        els.batchBar.style.display = 'none';
    }
    // ... 原有逻辑 ...
}
```

---

## 四、交互流程

### 4.1 进入/退出批量模式

```
用户点击 topbar "☑" 按钮
  → toggleBatchMode()
    → state.batchMode = true
    → batchModeBtn 加 active 高亮
    → renderCardGrid() 重新渲染
      → 所有卡片右上角置顶按钮 → 复选框
      → 卡片加 batch-mode class
    → updateBatchBar() → 显示 batchBar（count=0）

用户再次点击 "☑" 按钮（active 态）
  → toggleBatchMode()
    → state.batchMode = false
    → clearSelection()
    → 隐藏 batchBar
    → renderCardGrid() 恢复正常（置顶按钮恢复）

用户点击 "退出批量" 按钮 → 同上

用户按 Escape → 同上

切换视图（搜索/设置/数据/回收站）→ 自动退出批量模式
```

### 4.2 选中/取消选中

```
批量模式下，用户点击卡片右上角复选框
  → toggleNoteSelection(id)
    → state.selectedNoteIds 更新
    → updateBatchBar() → 更新计数
    → renderCardGrid() → 选中卡片加 selected class

用户再次点击已选卡片的复选框 → 取消选中
```

### 4.3 批量删除

```
用户选中笔记 → 点击"批量删除"
  → confirm("确定要删除选中的 X 条笔记吗？")
    → [确认]
      → batchDeleteSelected()
        → BatchDeleteNotes(ids) → 后端软删除
        → clearSelection() → 退出批量模式
        → loadNotes()
    → [取消] 无操作
```

### 4.4 批量标签

```
用户选中笔记 → 点击"批量标签"
  → showBatchTagPanel()
    → 加载标签 → 渲染到 batchTagList
    → 显示 batchTagPanel 下拉面板

用户点击某个标签 chip
  → applyBatchTag(tagId)
    → BatchAddTagToNotes(ids, tagId) → 后端批量关联
    → 关闭面板 → clearSelection() → 退出批量模式
    → loadNotes()
```

---

## 五、假设与决策

| 编号 | 决策 | 理由 |
|------|------|------|
| D1 | 批量标签只支持添加，不支持移除 | 更常见的使用场景；移除可后续单独操作 |
| D2 | 复选框替换置顶按钮位置（右上角） | 用户要求"置顶键变成复选框"，保持位置一致性 |
| D3 | 批量操作栏在 topbar 下方自然流，非 fixed | 避免覆盖内容 |
| D4 | 批量标签使用下拉面板而非弹窗 | 更轻量，操作路径更短 |
| D5 | 后端批量删除使用 `IN (?)` 单条 SQL | 性能好，原子性强 |
| D6 | 切换视图时自动退出批量模式 | 避免跨视图状态混乱 |
| D7 | 进入批量模式时不自动全选 | 让用户自主选择 |

---

## 六、验证步骤

1. **后端编译检查**：`go build ./...` 确认无编译错误
2. **前端构建检查**：`cd frontend && npx vite build` 确认无语法错误
3. **功能验证**：
   - 点击"☑" → 进入批量模式，置顶按钮变为复选框
   - 点击复选框 → 卡片高亮 + 底部 bar 显示计数
   - 选中多条 → 计数正确
   - 批量删除 → 确认 → 笔记移至回收站
   - 批量标签 → 选标签 → 笔记关联成功
   - 退出批量 → 恢复正常
   - Escape 退出批量模式
   - 切换视图自动退出批量
4. **Mock 降级验证**：独立前端运行时，批量模式 Mock 数据下正常工作
