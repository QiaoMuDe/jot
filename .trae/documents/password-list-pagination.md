# 密码列表分页实现计划

## Summary

为密码管理列表添加滚动懒加载分页，复用笔记首页已有的 `page_size` 设置。后端 `List`/`Search` 加 LIMIT/OFFSET，前端监听滚动到底部附近自动加载下一页，底部显示总条数。

***

## Current State Analysis

* **后端** `PasswordService.List()` / `Search()` 无分页参数，返回全量数组

* **前端** `loadPmRecords()` 一次调用拿到全部数据，`renderPmList()` 全量渲染

* **笔记分页模式**：后端返回 `PaginatedResult{Items, Total, Page, PageSize}`，前端滚动监听 + `loadMoreNotes()` 逐页追加，距底部 200px 时触发

* **page\_size 设置**：已有 `GetPageSize()` Wails 方法，前端从设置页 segmented control 读取

***

## Proposed Changes

### 1. 后端 Service 层 — `password_service.go`

**改动**：`List()` 和 `Search()` 加分页参数

* `List(page, pageSize int) ([]PasswordListItem, int64, error)`

  * 先 `db.Model(&PasswordRecord{}).Count(&total)`

  * 再 `db.Offset((page-1)*pageSize).Limit(pageSize).Find(&recs)`

* `Search(keyword string, page, pageSize int) ([]PasswordListItem, int64, error)`

  * 同上模式，Count + Offset/Limit

* 返回 `(items, total, error)`

### 2. 后端 App 绑定层 — `app.go`

**改动**：`ListPasswordRecords` / `SearchPasswordRecords` 签名加参数，返回 `*services.PaginatedResult`

```go
func (a *App) ListPasswordRecords(page, pageSize int) (*services.PaginatedResult, error)
func (a *App) SearchPasswordRecords(keyword string, page, pageSize int) (*services.PaginatedResult, error)
```

返回值从 `[]PasswordListItem` 改为 `*PaginatedResult`（Items 字段为 `[]PasswordListItem`）。

### 3. 前端逻辑 — `password-manager.js`

**新增分页状态变量**（与笔记模式一致）：

```js
let pmCurrentPage = 1;
let pmTotal = 0;
let pmHasMore = true;
let pmLoadingMore = false;
let pmPageSize = 20;
```

**改动** **`loadPmRecords()`**：

* 初始化时调用 `window.go.main.App.GetPageSize()` 获取分页大小（默认 20）

* 重置分页状态（`pmCurrentPage = 1`, `pmHasMore = true`）

* 调用 `ListPasswordRecords(1, pmPageSize)` 或 `SearchPasswordRecords(kw, 1, pmPageSize)`

* 响应为 `PaginatedResult`，从中提取 `items` 赋值 `pmRecords`，记录 `pmTotal`，判断 `pmHasMore`

* `renderPmList()` 正常全量渲染第一页

**新增** **`loadMorePmRecords()`**：

* 防重入（`pmLoadingMore` 守卫）

* `pmCurrentPage++`，调用对应接口请求下一页

* 新数据 `concat` 到 `pmRecords`

* 以 `append` 模式调用 `renderPmList({ append: true })`

* 更新 `pmHasMore`

**改动** **`renderPmList()`**：

* 新增 `{ append = false }` 参数

* `append` 为 false 时：清空 `pmListEl`，全量渲染（当前行为不变）

* `append` 为 true 时：不清空，只渲染新增的记录追加到 `pmListEl`

* 不显示总条数，`pmHasMore` 为 false 时隐藏 loading indicator 即可

**新增滚动监听**：

* 在 `.pm-list-wrap` 上监听 `scroll` 事件

* 距底部 < 200px 时触发 `loadMorePmRecords()`

* 在 `initPasswordManager()` 中绑定

### 4. 前端样式 — `password-manager.css`

新增一个样式（复用笔记首页模式）：

```css
.pm-loading-indicator { /* 加载中 spinner */ }
```

### 5. 前端 HTML — `index.html`

在 `#pmList` 后面（`.pm-list-wrap` 内部）添加：

```html
<div id="pmLoadingIndicator" class="pm-loading-indicator" style="display:none;">
    <span class="pm-loading-spinner"></span> 加载中...
</div>
```

***

## Files to Modify

| 文件                                                 | 改动                                                |
| -------------------------------------------------- | ------------------------------------------------- |
| `internal/services/password_service.go`            | List/Search 加分页参数                                 |
| `app.go`                                           | ListPasswordRecords/SearchPasswordRecords 改签名+返回值 |
| `frontend/src/js/password-manager.js`              | 分页状态 + loadMore + render append + 滚动监听            |
| `frontend/src/css/components/password-manager.css` | 加载指示器样式                                           |
| `frontend/index.html`                              | 添加 loading indicator DOM                          |

***

## Assumptions & Decisions

1. **复用** **`page_size`** **设置**：密码列表使用笔记首页相同的每页条数设置（通过 `GetPageSize()` 获取），不需要单独的密码分页设置
2. **滚动懒加载**：采用与笔记首页相同的无限滚动模式，距底部 200px 时自动加载下一页
3. **排序不变**：密码列表保持 `updated_at DESC, id DESC` 排序，不加排序切换
4. **搜索重置分页**：搜索关键词变化时重置到第 1 页

***

## Verification Steps

1. 后端编译通过：`go build ./...`
2. 前端无语法错误
3. 功能验证：

   * 打开密码管理，列表正常显示第一页（20 条）

   * 滚动到底部附近，自动加载下一页并追加

   * 搜索时列表重置到第 1 页，只显示搜索结果的第一页

   * 搜索后滚动也能加载更多搜索结果

   * 新增/编辑/删除后列表正确刷新（从第 1 页重新加载）

