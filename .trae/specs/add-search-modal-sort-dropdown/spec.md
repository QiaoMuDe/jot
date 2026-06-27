# 搜索弹窗添加排序下拉菜单 Spec

## Why

搜索弹窗结果目前固定按 `pinned DESC, updated_at DESC` 排序，用户无法在搜索时切换排序方式。需要像主页一样支持按更新时间、创建时间、名称排序，且所有排序都保持置顶优先。

## 排序选项建议

| 排序条件 | 对应 sortBy 值 | 后端 ORDER BY | 说明 |
|---------|---------------|--------------|------|
| 更新时间 | `updated_at` | `pinned DESC, updated_at DESC` | 默认，和主页一致 |
| 创建时间 | `created_at` | `pinned DESC, created_at DESC` | 和主页一致 |
| 名称 | `title` | `pinned DESC, title ASC` | 和主页一致 |

3 个选项与设置面板一致，用户认知无额外学习成本。

## What Changes

### 后端 — `note_service.go`
- `Search()` 添加 `sortBy string` 参数，调用 `buildSortOrder(sortBy)` 替换硬编码的 `Order("pinned DESC, updated_at DESC")`
- `SearchByNotebook()` 同理添加 `sortBy string` 参数

### 后端 — `app.go`
- `SearchNotes()` 添加 `sortBy string` 参数，透传给 `Search()` / `SearchByNotebook()`

### 前端 — `index.html`
- 在 `.search-modal-filters` 末尾添加排序筛选器：
  ```html
  <div class="search-modal-filter" data-filter="sort">
      <span class="search-modal-filter-label">排序:</span>
      <button type="button" class="search-modal-filter-btn" id="searchModalSortBtn">
          <span id="searchModalSortLabel">更新时间</span>
          <svg class="chevron" ...></svg>
      </button>
      <div class="search-modal-filter-dropdown" id="searchModalSortDropdown">
          <button class="search-modal-filter-option" data-sort="updated_at">更新时间</button>
          <button class="search-modal-filter-option" data-sort="created_at">创建时间</button>
          <button class="search-modal-filter-option" data-sort="title">名称</button>
      </div>
  </div>
  ```

### 前端 — `main.js`
- 添加状态 `state.searchModalSortBy = 'updated_at'`
- 添加元素引用 `els.searchModalSortBtn`、`els.searchModalSortLabel`、`els.searchModalSortDropdown`
- 初始化排序下拉：单击选项切换 `state.searchModalSortBy`、更新按钮 label、高亮选中项、触发搜索
- `searchModalLoadPage()` 调用 `SearchNotes` 时传入 `state.searchModalSortBy` 作为 sortBy 参数
- `openSearchModal()` 重置排序为 `updated_at`
- 绑定点击遮罩/外部关闭排序下拉

### 前端 — `search-modal.css`
- 新增排序下拉无需额外样式，复用 `.search-modal-filter` / `.search-modal-filter-dropdown` / `.search-modal-filter-option` 现有类即可
- 可添加 `.search-modal-filter-option.active` 高亮当前选中项样式（文字颜色变 accent）

## Impact
- Affected specs: 搜索弹窗
- Affected code:
  - `internal/services/note_service.go` — Search / SearchByNotebook 签名变更
  - `internal/app/app.go` — SearchNotes 签名变更
  - `frontend/src/main.js` — searchModalLoadPage 逻辑
  - `frontend/index.html` — HTML 结构