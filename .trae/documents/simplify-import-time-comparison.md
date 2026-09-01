# 简化导入时间对比规则：时间戳对齐文件 mtime + 哈希兜底

## Summary

修复拖拽导入的误报冲突问题：首次导入后笔记 `UpdatedAt` = 导入时刻，重导入同一文件时时间对比必然误判为"笔记更新"而弹冲突窗。

新方案（用户已确认）：

* **导入写入（创建/覆盖）时，把笔记的** **`CreatedAt`/`UpdatedAt`** **对齐为文件的修改时间** **`ModTime()`**，而非当前导入时刻。时间戳本身成为同步基准，无需新增任何字段。

* **内容哈希兜底**：导入时用 go-kit 的 `hash` 包对笔记内容与文件内容（规范化后）计算 SHA256，一致则直接 `skipped`，不进入时间对比。

对比规则（两级）：

```
① 哈希一致 → skipped
② 哈希不一致 → fileTime > UpdatedAt → "updated" 覆盖
              fileTime < UpdatedAt → "conflict" 弹窗
              相等               → "skipped"
```

## Current State Analysis

* [app.go:3616](d:\峡谷\Dev\本地项目\jot\app.go) `processImportFile`：读文件 → `FindByTitleAndExt` 查匹配笔记 → 按 `fileModTime` vs `existingNote.UpdatedAt` 对比，分支为 `updated`/`conflict`/`skipped`，无匹配则 `CreateWithNotebook` 创建（`UpdatedAt` 被 GORM 刷成 now → 问题根源）。

* [app.go:3774](d:\峡谷\Dev\本地项目\jot\app.go) `ResolveImportConflict(noteID, overwrite, title, content, fileExt)`：用户确认覆盖后调用 `noteService.Update`（`db.Save` 同样把 `UpdatedAt` 刷成 now）。

* [note\_service.go:887](d:\峡谷\Dev\本地项目\jot\internal\services\note_service.go) `CreateWithNotebook`：不设置时间字段，GORM 自动填充 now。

* [note\_service.go:43](d:\峡谷\Dev\本地项目\jot\internal\services\note_service.go) `Update`：`db.Save` 刷新 `UpdatedAt` 为 now。

* [main.js:9135/9194](d:\峡谷\Dev\本地项目\jot\frontend\src\main.js)：冲突弹窗逐项/批量两处调用 `ResolveImportConflict`，item 上已有 `file_time` 字段可回传。

* `gitee.com/MM-Q/go-kit v0.0.24` 已在 [go.mod](d:\峡谷\Dev\本地项目\jot\go.mod) 依赖中，`hash` 子包可直接使用（`hash.HashString(data, hash.SHA256)`）。

* `UpdateColumn` 在 note\_service.go 中已有使用先例（L450、L580、L829-830），风格一致。

* AGENTS.md 约定：修改 `app.go` 后需执行 `wails generate module` 重新生成 `frontend/wailsjs` 绑定。

## Proposed Changes

### 1. `internal/services/note_service.go` — 新增两个导入专用方法

**新增** **`CreateWithNotebookAt`**：与 `CreateWithNotebook` 相同，但预设 `CreatedAt`/`UpdatedAt` 为传入时间：

```go
func (s *NoteService) CreateWithNotebookAt(title, content, fileExt string, notebookID uint, t time.Time) (*models.Note, error) {
    note := models.Note{Title: title, Content: content, FileExt: fileExt, NotebookID: notebookID, CreatedAt: t, UpdatedAt: t}
    if err := s.db.Create(&note).Error; err != nil { ... }
    return &note, nil
}
```

⚠️ GORM v2 对 AutoCreateTime 字段仅在零值时填充、非零值保留；实现后需用测试验证保留生效。若实测被覆盖，兜底写法：Create 后追加 `UpdateColumns` 修正两列。

**新增** **`UpdateWithTime`**：用 `UpdateColumns` 一条 SQL 同时写内容与时间（绕过 GORM 自动刷 `UpdatedAt`）：

```go
func (s *NoteService) UpdateWithTime(id uint, title, content, fileExt string, t time.Time) (*models.Note, error) {
    // 先 GetByID 校验存在
    // s.db.Model(&models.Note{}).Where("id = ?", id).UpdateColumns(map[string]interface{}{
    //     "title": title, "content": content, "file_ext": fileExt, "updated_at": t,
    // })
    // 再 GetByID 返回最新笔记（CreatedAt 不变）
}
```

现有 `Create`/`Update`/`CreateWithNotebook` 签名与行为不动（其他调用方：app.go L424/453/2917、manage\_note.go L401/404 均不受影响）。

### 2. `app.go` — `processImportFile` 对比逻辑改造（L3700-L3762）

* 文件内容读取完成后，计算文件内容哈希（含规范化）：

```go
import "gitee.com/MM-Q/go-kit/hash"

func importContentHash(s string) (string, error) {
    normalized := strings.ReplaceAll(s, "\r\n", "\n")
    normalized = strings.TrimSpace(normalized)
    return hash.HashString(normalized, hash.SHA256)
}
```

* 匹配到已有笔记后，**先哈希对比**：`importContentHash(existingNote.Content)` 与文件哈希相等 → 直接返回 `status: "skipped"`（`Success: true`），不进时间对比。

* 哈希不一致 → 保留现有 `fileTimeUnix` vs `noteTimeUnix` 的 switch 逻辑不变。

* **覆盖分支**（`fileTimeUnix > noteTimeUnix`）：改调 `noteService.UpdateWithTime(existingNote.ID, title, content, fileExt, fileModTime)`。

* **创建路径**（L3756）：改调 `noteService.CreateWithNotebookAt(title, content, fileExt, notebookID, fileModTime)`。

* conflict 分支返回值（FileTime/NoteTime/Content/FileExt）不变。

### 3. `app.go` — `ResolveImportConflict` 增加文件时间参数（L3774）

* 签名改为 `ResolveImportConflict(noteID uint, overwrite bool, title, content, fileExt string, fileTime int64)`。

* 覆盖分支改调 `noteService.UpdateWithTime(noteID, title, content, fileExt, time.Unix(fileTime, 0))`，保证冲突覆盖后时间戳同样对齐文件 mtime。

### 4. `frontend/src/main.js` — 回传 `file_time`

两处调用（L9135、L9194）追加第 6 个参数 `item.file_time`：

```js
await window.go.main.App.ResolveImportConflict(item.note_id, overwrite, item.title, item.content, item.file_ext, item.file_time);
```

冲突弹窗 UI、状态处理逻辑零改动。

### 5. 重新生成 Wails 绑定

执行 `wails generate module` 更新 `frontend/wailsjs/go/main/App.d.ts` 与 `App.js` 中 `ResolveImportConflict` 的签名。

## Assumptions & Decisions

1. **UI 显示行为变化（已确认接受）**：导入的笔记显示文件的修改时间（如"昨天更新"）而非导入时刻，列表按 `updated_at` 排序也随之按文件时间。
2. **残留取舍（已确认接受）**：

   * 用户编辑笔记后重导入未变的文件 → 仍弹一次 conflict（内容不同 + 笔记时间更新），选"保留"即可；

   * 用户编辑笔记且文件其后被修改 → 自动覆盖，本地编辑丢失（无字段约束下无解）。
3. **哈希规范化**：`\r\n → \n` + `TrimSpace`，避免换行符/尾部空行造成假性不一致；哈希运行时计算，不持久化。
4. **哈希失败降级**：`hash.HashString` 出错时记日志并退回纯时间对比（不阻断导入）。
5. **不新增字段、不做 migration**；旧数据（`UpdatedAt` 为真实编辑时间或旧导入时间）与新规则天然兼容。

## Verification

1. `go build ./...` 编译通过。
2. `go test ./internal/services/...` 通过；为 `CreateWithNotebookAt`/`UpdateWithTime` 补充测试：创建后读回 `CreatedAt`/`UpdatedAt` 应等于传入时间（同时验证 GORM 不覆盖预设时间戳）。
3. `wails generate module` 后检查 `App.d.ts`/`App.js` 签名已更新。
4. `wails dev` 手动验证：

   * 拖入新文件 → 创建笔记，笔记时间 = 文件修改时间；

   * 原样再次拖入同一文件 → 静默 skipped，无弹窗；

   * 修改文件后再次拖入 → 直接覆盖（无弹窗），笔记时间更新为新的文件时间；

   * 导入后在应用内编辑笔记，再拖入未变的文件 → 弹冲突窗，选"跳过"笔记不变；

   * 导入办公文件（docx→md）重导入 → skipped。

