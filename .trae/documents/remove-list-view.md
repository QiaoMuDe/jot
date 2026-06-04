# 移除列表视图

## 概述
删除列表视图功能及相关代码，保留网格视图作为唯一展示模式。

## 涉及文件及改动

### 1. `frontend/index.html` — 1 行

删除列表视图按钮：
```diff
- <button id="listViewBtn" class="topbar-btn" title="列表视图">☰</button>
```

### 2. `frontend/src/main.js` — 4 处

**a) 移除 `els.listViewBtn` 引用（第 68 行）**
```diff
- listViewBtn: $('listViewBtn'),
```

**b) 移除列表视图事件监听与切换逻辑（第 883-890 行）**
```diff
- els.gridViewBtn.classList.remove('active');
- els.gridViewBtn.addEventListener('click', () => { ... });
- els.listViewBtn.classList.remove('active');
- els.cardGrid.classList.remove('list-view');
- els.listViewBtn.addEventListener('click', () => { ... });
- els.cardGrid.classList.add('list-view');
```

**c) 保留 `gridViewBtn` 但去掉 toggle 逻辑 — gridViewBtn 只作为标识，不再响应点击（列表已删，不需要切换了）**
```diff
- els.gridViewBtn.addEventListener('click', () => { ... });
```
（gridViewBtn 按钮本身保留在 HTML 中作为视觉标识，或者也一并移除）

**决定**：gridViewBtn 也可以移除，因为不需要视图切换了。topbar 只留 `+` 和 `☰` 菜单。

### 3. `frontend/src/style.css` — 7 行

删除所有 `.card-grid.list-view` 相关样式（第 215、398-424 行）：
```diff
- .card-grid.list-view { ... }
- .card-grid.list-view .note-card { ... }
- .card-grid.list-view .card-body { ... }
- .card-grid.list-view .card-title { ... }
- .card-grid.list-view .card-content { ... }
- .card-grid.list-view .card-footer { ... }
- .card-grid.list-view .card-actions { ... }
```

另外 `gridViewBtn` 按钮也移除，因此 `topbar-btn.active` 相关保留但不再被网格按钮使用。

## 最终 topbar 效果
```
[Jot]  [搜索框]  [+]  [⋮ 菜单]
```

## 验证
1. 首页正常显示卡片网格（无布局切换按钮）
2. 卡片右键菜单、左击查看、标签搜索等功能正常
3. 无控制台报错
