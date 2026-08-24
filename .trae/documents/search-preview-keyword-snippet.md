# 搜索预览优化：关键词居中截取

## Summary

搜索结果的内容预览从"固定取前 200 字符"改为"围绕关键词截取约 200 字符"，让用户一眼看到匹配原因。后端改一处，前端搜索弹窗和 Agent 工具同时受益。

## Current State

### 后端

[note\_service.go L225](file:///d:\峡谷\Dev\本地项目\jot\internal\services\note_service.go#L225):

```go
const noteThinSelect = "id, title, SUBSTR(content, 1, 200) AS content, file_ext, pinned, notebook_id, created_at, updated_at"
```

这是一个**常量**，被 4 个函数共用：

* `GetAll()` — 列表浏览，无关键词

* `Search()` — 搜索，有关键词 → **需要改**

* `SearchByNotebook()` — 按笔记本搜索，有关键词 → **需要改**

* `GetByDate()` — 按日期查询，无关键词

### 前端

[main.js L7825-7831](file:///d:\峡谷\Dev\本地项目\jot\frontend\src\main.js#L7825-L7831):

```js
const snippet = String(content || '').slice(0, 100).replace(/\s+/g, ' ').trim();
snippetEl.innerHTML = highlightModalMatch(snippet, kw);
```

固定取后端返回内容的前 100 字符，然后高亮。

## Proposed Changes

### 1. 后端：`noteThinSelect` 改为函数

**文件**: [note\_service.go](file:///d:\峡谷\Dev\本地项目\jot\internal\services\note_service.go)

将常量改为函数，接受可选 keyword 参数：

```go
// noteThinSelect 列表/搜索查询时使用的 Select，排除全量 Content
// 有 keyword 时围绕关键词首次出现位置截取约200字符作为预览，无 keyword 时取前200字符
func noteThinSelect(keyword ...string) string {
    base := "id, title, %s AS content, file_ext, pinned, notebook_id, created_at, updated_at"
    var contentExpr string
    if len(keyword) > 0 && keyword[0] != "" {
        // INSTR 返回 1-based 位置；MAX(1, pos-100) 确保不越界；SUBSTR 截取约 200 字符
        escaped := strings.ReplaceAll(keyword[0], "'", "''")
        contentExpr = fmt.Sprintf(
            `COALESCE(SUBSTR(content, MAX(1, INSTR(content, '%s') - 100), 200), SUBSTR(content, 1, 200))`,
            escaped,
        )
    } else {
        contentExpr = "SUBSTR(content, 1, 200)"
    }
    return fmt.Sprintf(base, contentExpr)
}
```

逻辑：

* keyword 非空 → 找到关键词在 content 中的首次位置，截取其前 100 + 后 100 字符（共约 200 字符）

* keyword 为空或 GetAll/GetByDate 场景 → 回退到原来的前 200 字符

### 2. 后端：修改 Search() 和 SearchByNotebook()

**文件**: [note\_service.go](file:///d:\峡谷\Dev\本地项目\jot\internal\services\note_service.go)

* `Search()` (约 L294-319)：将 `.Select(noteThinSelect)` 改为 `.Select(noteThinSelect(keyword))`

* `SearchByNotebook()` (约 L340-365)：同上

`GetAll()` 和 `GetByDate()` 不改，继续使用 `noteThinSelect()`（无参数，走原逻辑）。

### 3. 前端：调整预览显示长度

**文件**: [main.js](file:///d:\峡谷\Dev\本地项目\jot\frontend\src\main.js#L7825-L7831)

```js
// 旧：固定取前 100 字符
const snippet = String(content || '').slice(0, 100).replace(/\s+/g, ' ').trim();

// 新：显示后端返回的完整关键词片段（约200字符），仅做空白归一化
const snippet = String(content || '').replace(/\s+/g, ' ').trim();
```

后端已经返回了智能截取的片段，前端不需要再截断，直接显示即可。

### 4. 不需要改的

* `manage_note.go` — Agent 工具的 `listNotes()` 调用 `Search()`/`SearchByNotebook()`，后端改了自动生效

* HTML/CSS — 结构和样式不变

* `highlightModalMatch()` — 已有高亮逻辑，无需修改

## Assumptions & Decisions

1. **SQL 注入防护**：keyword 已在 WHERE 子句中绑定为参数（`?`），但 SQL 片段拼接中需要手动转义单引号。用 `strings.ReplaceAll("'", "''")` 处理。这是 SQLite 的标准转义方式。
2. **截取边界**：`MAX(1, INSTR-100)` 确保起始位置不小于 1。当关键词出现在前 100 字符内时，起始位置为 1，截取前 200 字符。
3. **常量改函数的兼容性**：`GetAll()` 和 `GetByDate()` 调用 `noteThinSelect()`（无参数）保持原行为，不受影响。
4. **GORM SQL 片段兼容性**：GORM 的 `.Select()` 接受原生 SQL 表达式，当前已在使用 `SUBSTR()`，改为更复杂的表达式无兼容性问题。

## Verification

1. `go build ./...` — 确认编译通过
2. `go test ./internal/services/ -run TestSearch` — 运行搜索相关测试
3. `go vet ./...` — 静态检查
4. 手动测试场景：

   * Ctrl+F 搜索"项目管理" → 预览应显示关键词周围的上下文

   * 关键词在标题中但不在内容中 → 预览回退到前 200 字符

   * 无关键词的列表浏览 → 行为不变

