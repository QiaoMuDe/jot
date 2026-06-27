# 用 file_ext 替代 note_type 计划

## 摘要

移除整个 `note_type` 体系，纯靠 `file_ext` 决定笔记行为：`.md` → Markdown 模式，其他（`.txt`, `.py` 等）→ 纯文本模式。移除顶部 `T`/`M` 切换按钮，简化编辑器逻辑。

## 现状分析

`note_type` 在 7 个文件中被引用：

| 文件 | 用途 |
|---|---|
| `models/note.go` | 模型字段定义 |
| `services/note_service.go` | Create/Update 接收 noteType 参数，自动推导 fileExt |
| `app.go` | Wails 绑定，传递 noteType 参数 |
| `main.js` (25+ 处) | state.noteType 贯穿前端逻辑 |
| `index.html` | 顶部 T/M 切换按钮 |
| `seed/main.go` | 种子数据分配 |
| `models.ts` | 自动生成 |

核心矛盾：`note_type` 和 `file_ext` 本质上存储同一信息（`.md` ↔ markdown，`.txt/.py/.go` ↔ text），却需要双向同步。

## 变更方案

### 设计决策

- **仅 `.md` 扩展名触发 Markdown 模式**，其他所有扩展名（`.txt`, `.py`, `.go` 等）均为纯文本模式
- **移除顶部 `T`/`M` 切换按钮**，类型完全由扩展名决定
- **DB 列保留**（不执行 DROP COLUMN），仅从 Go struct 移除字段，避免 SQLite 表重建
- 前端保留一个**内部计算属性** `_noteType`，每次使用时从 `file_ext` 推导

### 具体变更

#### 1. 后端: `internal/models/note.go`
- **删除** `NoteType` 字段定义（`size:20;default:text`）
- GORM 将不再读写该列，SQLite 中该列保持不动

#### 2. 后端: `internal/services/note_service.go`
- **`Create()`**: 删除 `noteType` 参数；FileExt 由调用方传入
- **`Update()`**: 删除 `noteType` 参数；只更新 `FileExt`（不再自动覆盖）
- **`CreateWithNotebook()`**: 删除 `noteType` 参数
- **删除 `fileExtFromNoteType()`** 辅助函数
- **`noteThinSelect`**: 移除 `note_type` 列

#### 3. 后端: `app.go`
- **`CreateNote()`**: 签名从 `(title, content, noteType, notebookID)` → `(title, content, fileExt, notebookID)`
- **`UpdateNote()`**: 签名从 `(id, title, content, noteType)` → `(id, title, content, fileExt)`
- **`ImportFiles()`**: 移除 `noteType` 变量，直接用 `.md` 判断逻辑设置 `fileExt`

#### 4. 前端: `frontend/index.html`
- **删除** `#editorTypeToggle` 整行（第 142 行的 `<button class="editor-header-btn editor-type-toggle" ...>`）

#### 5. 前端: `frontend/src/main.js`
- **删除 `state.noteType`** 状态变量
- **新增** `function noteTypeFromFileExt(ext)` = `ext === '.md' ? 'markdown' : 'text'`
- **删除 `fileExtFromNoteType()`**（反向映射不再需要）
- **删除 `switchNoteType()`**（不再有手动切换）
- **`saveFileExt()`**: 移除 `switchNoteType` 调用（改为直接刷新 UI）
- **`openEditor()`**: 从 `noteData.file_ext` 推导行为 → `const isMd = (noteData && noteData.file_ext) === '.md'`，替代 `state.noteType === 'markdown'`
- **`createNote()`/`updateNote()`**: 参数从 `state.noteType` 改为 `els.editorFileExt.textContent`
- **`saveBeforeClose()`**: 同上传参改为 `fileExt`
- **Ctrl+L 快捷键**: 从 `state.noteType === 'markdown'` 改为 `els.editorFileExt.textContent === '.md'`
- **MD 语法练习编辑器**: 直接设置 `els.editorFileExt.textContent = '.md'`
- **所有 `state.noteType` 引用** → 替换为 `noteTypeFromFileExt(els.editorFileExt.textContent)`
- **保存按钮处理**: 传递 `els.editorFileExt.textContent` 而非 `state.noteType`
- **缓存更新**: `cached.note_type = 'markdown'|'text'` → 用推导值填充

#### 6. 种子数据: `tools/seed/main.go`
- **删除 `NoteType` 字段** 及其所有赋值（~46 处）
- 改为直接设置 `FileExt`（`.md` 或 `.txt`）

#### 7. 类型绑定: `frontend/wailsjs/go/models.ts`
- Wails 自动生成，重新 `wails generate module` 后自动移除 `note_type`
- 若手动更新：删除 `note_type: string;` 和 `this.note_type = source["note_type"];`

#### 8. 文档: `d:\峡谷\Dev\本地项目\jot\AGENTS.md`
- 更新顶部统计（缩减 JS 逻辑行数）
- 在 §十七 或新增章节记录此次变更

### 不变的部分
- `#editorMode` 按钮（edit/preview）继续保留，可见性由 `fileExt === '.md'` 控制
- `#mdRendered` 显示/隐藏逻辑不变，改用 `fileExt` 做判断
- CM6 语法高亮逻辑不变，改用 `fileExt` 做判断

## 验证步骤

1. `go build ./...` 编译通过
2. `npx vite build` 构建通过
3. 创建 `.txt` 笔记 → 状态栏显示 `.txt`，底部无编辑/预览按钮
4. 修改后缀为 `.md` → 底部显示编辑/预览按钮，预览可用
5. 创建 `.md` 笔记 → 状态栏显示 `.md`，底部有编辑/预览按钮
6. 修改后缀为 `.py` → 底部按钮隐藏，按纯文本处理
7. 导出功能正确使用 `file_ext`
8. 种子数据正常插入
