# Tasks

- [ ] Task 1: 后端 — Search/SearchByNotebook 添加 sortBy 参数
  - [ ] `note_service.go`: `Search()` 签名添加 `sortBy string`，用 `buildSortOrder(sortBy)` 替代硬编码 ORDER BY
  - [ ] `note_service.go`: `SearchByNotebook()` 同步修改
  - [ ] `app.go`: `SearchNotes()` 添加 `sortBy string` 参数并透传

- [ ] Task 2: 前端 — 搜索弹窗添加排序下拉菜单
  - [ ] `index.html`: 在 `.search-modal-filters` 末尾添加排序筛选器 HTML
  - [ ] `main.js`: 添加 state/els 引用、初始化排序下拉事件、openSearchModal 重置排序
  - [ ] `main.js`: `searchModalLoadPage()` 传入 `state.searchModalSortBy`
  - [ ] `search-modal.css`: 添加 `.search-modal-filter-option.active` 高亮样式

- [ ] Task 3: 联动 — 搜索弹窗通过标签打开时同步默认排序
  - [ ] 确保点击笔记标签打开搜索弹窗时排序下拉初始状态正确

## Task Dependencies
- Task 1 → Task 2（前端传参依赖后端新签名）
- Task 3 depends on Task 2