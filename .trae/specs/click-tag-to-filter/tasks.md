# Tasks

## Task 1: 重写 `searchByTag` 函数 — 调用 `GetNotesByTag` API 按标签 ID 筛选

- [x] 修改 `window.searchByTag(tagName)` → `window.searchByTag(tagId, tagName, tagColor)`
  - 不再打开搜索弹窗 (`openSearchModal`)
  - 改为调用 `window.go.main.App.GetNotesByTag(tagId, 1, 100, state.sortBy)` 获取笔记
  - 后端不可用时降级：`state.notes.filter(n => n.tags && n.tags.some(t => t.id === tagId))`
  - 将 `state.filterTag` 设为 `{id: tagId, name: tagName, color: tagColor}`
  - 调用 `renderSearchResults(items, '', state.filterTag)`（空 keyword 避免高亮）
  - 调用 `switchView('search')` 切换到搜索视图

## Task 2: 修改 `renderSearchResults` — 支持显示标签筛选指示器

- [x] 函数签名新增 `filterTag` 参数（可选）：`renderSearchResults(results, keyword, filterTag)`
- [x] 在搜索结果列表上方渲染 `.tag-filter-indicator` DOM：
  - 显示标签 chip（颜色圆点 + 名称）+ 关闭按钮（×）
  - 关闭按钮点击调用 `clearTagFilter()`：清 `state.filterTag` → `switchView('grid')`
- [x] `switchView('grid')` 时也清空 `state.filterTag`
- [x] 无 `filterTag` 时不渲染指示器（兼容现有搜索场景）
- [x] 样式：标签 chip 与 `.card-tag` 一致，居中上方，带 `8px` 左右内边距

## Task 3: 修改卡片和搜索结果中的标签 onclick 传递 tagId

- [x] `renderCardGrid()` 中标签 onclick：`window.searchByTag(${tag.id}, '${escapeHtml(tag.name)}', '${escapeHtml(tag.color || '#6366f1')}')`
- [x] `renderSearchResults()` 中标签 onclick：同上

## Task 4: 验证与回归

- [x] 卡片网格点击标签 → 正确按标签 ID 筛选 → 显示标签指示器 → 结果准确
- [x] 搜索视图点击结果标签 → 重新按标签 ID 筛选 → 标签指示器更新
- [x] 点击标签指示器关闭按钮 → 回到网格首页 → filterTag 清空
- [x] 批量模式下点击标签 → 仅 stopPropagation → 不触发筛选
- [x] 搜索弹窗功能不受影响（独立入口，完全不变）
- [x] 后端不可用时 Mock 降级正常

# Task Dependencies

- [Task 1]、[Task 2] 可并行编码，但 [Task 2] 的渲染逻辑需与 [Task 1] 的 `renderSearchResults` 调用对齐
- [Task 3] 依赖 [Task 1] 的签名变更
- [Task 4] 依赖所有前置任务完成