# 清理 note_type 残余 + 构建验证

## 总结

`note_type` 字段已从后端完全移除，但前端 `main.js` 仍有 4 处残余引用（mock 回退路径和缓存更新），文档中也有历史描述需要清理。最后需要执行构建验证并删除旧数据库文件。

## 当前状态分析

### 后端（已彻底清理 ✓）
- `models/note.go` — 无 `NoteType` 字段
- `note_service.go` — 所有方法改用 `fileExt`，无 `noteType` 参数
- `app.go` — 绑定方法签名已更新
- `db.go` — 无回填迁移
- `tools/seed/main.go` — 使用 `FileExt` 字段（除了一处 SQL 注释中的 `note_type`，那是笔记内容文本，无害）

### 前端（仍有残余）

文件：`frontend/src/main.js`

| 行号 | 代码 | 性质 |
|------|------|------|
| 975 | `note_type: noteTypeFromFileExt(...)` | `CreateNote` mock 回退路径（死代码） |
| 1022 | `note.note_type = noteTypeFromFileExt(...)` | `UpdateNote` mock 回退路径（死代码） |
| 2467 | `function noteTypeFromFileExt(ext)` | 辅助函数定义（仅被上面两处调用） |
| 3969 | `cached.note_type = noteTypeFromFileExt(...)` | 本地缓存同步（设置 JS 对象属性，不持久化） |

以上代码在真实运行中不会导致错误（Wails 绑定存在时走的是真实路径），但属于死代码，应当在重构中一并清理。

### 文档
- `AGENTS.md` 有多处 `note_type`/`NoteType` 历史描述，需要更新

## 修改计划

### 1. 清理前端残余

**文件：** `frontend/src/main.js`

**1a. 删除 mock 回退路径中的 `note_type` 赋值**

- L975：`note_type: noteTypeFromFileExt(els.editorFileExt.textContent),` → 直接删除此行（mock 对象的其他字段保持不变）
- L1022：`note.note_type = noteTypeFromFileExt(els.editorFileExt.textContent);` → 直接删除此行

**1b. 删除本地缓存中的 `note_type` 赋值**

- L3969：`cached.note_type = noteTypeFromFileExt(els.editorFileExt.textContent);` → 直接删除此行

**1c. 删除 `noteTypeFromFileExt` 辅助函数**

- L2467-2469：整个函数删除

### 2. 清理旧 DB + 构建验证

- 删除 `~/.jot/data/jot.db`（让 AutoMigrate 重建无 `note_type` 的表）
- 运行 `go build ./...` 验证后端编译
- 运行 `npx vite build` 验证前端编译

### 3. 更新 AGENTS.md（可选）

更新文档中的历史描述，移除对 `note_type` 的引用。

## 假设与决策

- mock 回退路径（L975、L1022）是前端开发调试用的死代码，Wails 绑定正常时不会执行，直接删除赋值行即可
- `cached.note_type`（L3969）仅用于同步 `state.notes` 中的 JS 对象，没有持久化意义，删除后不会影响任何功能
- 旧数据库文件删除后，首次启动时 GORM AutoMigrate 会自动创建新表

## 验证步骤

1. 确认前端 4 处 `note_type` 残余全部清理
2. 删除旧 DB 文件
3. `go build ./...` 通过
4. `npx vite build` 通过
