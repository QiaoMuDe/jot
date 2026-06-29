# 后端标签搜索

## 概述

将搜索弹窗中的标签过滤逻辑从**前端客户端过滤**迁移到**后端 SQL** 中，解决当前标签筛选后分页失效的问题，使标签过滤与关键词/笔记本/日期筛选一样支持完整的分页。

## 当前状态分析

### 客户端过滤（待移除）

[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L5086-L5108)

```js
// 当前：后端返回全部结果 → 前端按 AND 语义过滤 → 分页失效
if (state.searchModalTagIds && state.searchModalTagIds.size > 0) {
    notes = notes.filter(n => {
        const noteTags = n.tags || n.Tags || [];
        if (!Array.isArray(noteTags) || noteTags.length === 0) return false;
        return Array.from(selectedTagIds).every(tagId =>
            noteTags.some(t => (t.id !== undefined ? t.id : t.ID) === tagId)
        );
    });
}
if (state.searchModalTagIds && state.searchModalTagIds.size > 0) {
    state.searchModalHasMore = false; // 分页失效
    // ... total 也按当前已加载的算，不准确
}
```

**问题**：客户端过滤后无法知道总条数，`hasMore` 被设为 `false` 导致滚动加载失效。

### 后端搜索函数（待修改）

[note_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/note_service.go#L249-L314)

`Search()` 和 `SearchByNotebook()` 两个函数当前接受 `keyword, page, pageSize, sortBy, startDate, endDate`（或额外 `notebookID`），**不接受 tagIDs**。

### 数据模型

笔记-标签多对多关联，中间表 `note_tags`：

```go
// Note 结构体
type Note struct {
    Tags []Tag `gorm:"many2many:note_tags;" json:"tags,omitempty"`
}
```

### 前后端绑定

```go
// app.go 第 138 行
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string)
```

前端调用：
```js
window.go.main.App.SearchNotes(kw, page, pageSize, notebookId, sortBy, startDate, endDate)
```

TypeScript 声明：
```ts
SearchNotes(arg1:string,arg2:number,arg3:number,arg4:number,arg5:string,arg6:string,arg7:string)
```

## 变更方案

### 涉及文件

| 文件 | 变更类型 |
|------|----------|
| `internal/services/note_service.go` | 修改 `Search()` 和 `SearchByNotebook()` + 新增 `SearchByTags()` |
| `app.go` | 修改 `SearchNotes()` 绑定签名 |
| `frontend/src/main.js` | 移除客户端过滤 + 传 tagIds 给后端 |
| `frontend/wailsjs/go/main/App.d.ts` | 更新 TS 类型声明 |

### 后端

#### 1. `note_service.go` — 提取公共查询构建方法

新增 `buildSearchQuery` 内部方法，提取 `Search` 和 `SearchByNotebook` 共用的查询构建逻辑（keyword 模糊匹配、日期范围、标签 AND 过滤）。

或更简单的方式：在 `Search` 和 `SearchByNotebook` 中各自添加标签过滤逻辑。

#### 2. `Search()` 和 `SearchByNotebook()` 增加 `tagIDs []uint` 参数

**核心 SQL 逻辑**：使用子查询或 JOIN 实现 AND 语义

```go
// 方式一：子查询（推荐，不影响 Preload("Tags")）
if len(tagIDs) > 0 {
    subQuery := s.db.Table("note_tags").
        Select("note_id").
        Where("tag_id IN ?", tagIDs).
        Group("note_id").
        Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs))
    query = query.Where("id IN (?)", subQuery)
}
```

使用子查询的原因：
- 不影响 GORM 的 `Preload("Tags")`（不会因 JOIN 导致预加载数据重复）
- 语义清晰，AND 条件由 `COUNT = len(tagIDs)` 保证
- 性能好（`note_tags` 表有外键索引）

#### 3. `app.go` — 修改 `SearchNotes` 绑定签名

```go
// 当前
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string)

// 改为
func (a *App) SearchNotes(keyword string, page, pageSize int, notebookID uint, sortBy string, startDate, endDate string, tagIDs []uint)
```

在函数体内，将 `tagIDs` 透传给 `Search()` 和 `SearchByNotebook()`。

#### 4. 后端变更汇总

| 函数 | 变更 |
|------|------|
| `NoteService.Search(keyword, page, pageSize, sortBy, startDate, endDate, tagIDs)` | 增加 `tagIDs` 参数，非空时加子查询 |
| `NoteService.SearchByNotebook(keyword, page, pageSize, notebookID, sortBy, startDate, endDate, tagIDs)` | 同上 |
| `App.SearchNotes(keyword, page, pageSize, notebookID, sortBy, startDate, endDate, tagIDs)` | 增加 `tagIDs` 参数并透传 |

### 前端

#### 5. `main.js` — 移除客户端标签过滤

删除 `searchModalLoadPage()` 中第 5086-5108 行的客户端过滤代码（`if (state.searchModalTagIds && ...)` 两个代码块）。

恢复正常的 `hasMore` 和 `total` 计算逻辑（即后端返回的 `total` 直接可用）。

#### 6. `main.js` — 传 tagIds 给后端

在 `searchModalLoadPage()` 的 `SearchNotes` 调用中增加第 8 个参数：

```js
// 当前
const result = await window.go.main.App.SearchNotes(kw, page, pageSize, notebookId, sortBy, startDate, endDate);

// 改为
const tagIds = state.searchModalTagIds && state.searchModalTagIds.size > 0
    ? Array.from(state.searchModalTagIds)
    : [];
const result = await window.go.main.App.SearchNotes(kw, page, pageSize, notebookId, sortBy, startDate, endDate, tagIds);
```

#### 7. TypeScript 类型声明

`frontend/wailsjs/go/main/App.d.ts` 中更新：

```ts
// 当前
SearchNotes(arg1:string,arg2:number,arg3:number,arg4:number,arg5:string,arg6:string,arg7:string):Promise<services.PaginatedResult>;

// 改为
SearchNotes(arg1:string,arg2:number,arg3:number,arg4:number,arg5:string,arg6:string,arg7:string,arg8:Array<number>):Promise<services.PaginatedResult>;
```

（Wails 编译时会自动生成，手动更新仅用于编辑器智能提示）

## 假设与决策

1. **AND 语义**：选中多个标签时，笔记必须包含**所有**选中标签。当前前端已是 AND 语义，后端保持一致。
2. **子查询方案**不使用 JOIN：避免破坏 `Preload("Tags")` 的预加载结果。
3. **`[]uint` 类型**：Wails v2 支持 `[]uint` 映射到 JS 的 `Array<number>`。如果 Wails 不原生支持，可改用 `[]int64` 或 `string`（逗号分隔后拆分）。假设支持 `[]uint`。
4. **不修改 `noteThinSelect`**：当前 `SUBSTR(content, 1, 200)` 在列表中只取前 200 字符用于卡片预览，和搜索逻辑无关，无需改动。

## 验证步骤

1. **无标签筛选**：不选标签，正常搜索关键词 → 结果与之前一致
2. **单标签筛选**：选中 1 个标签 + 关键词 → 只返回带该标签且匹配关键词的笔记
3. **多标签 AND 筛选**：选中 2 个标签 → 只返回同时带这两个标签的笔记
4. **分页正常**：标签筛选 + 滚动加载多页 → `hasMore` / `total` 正确
5. **标签+笔记本+日期组合筛选**：组合条件正常
6. **搜索弹窗标签下拉**：切换标签后自动刷新结果
