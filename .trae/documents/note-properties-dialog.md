# 笔记属性弹窗（只读）实施计划

## Summary

在笔记首页右键菜单新增"属性"项，打开仿 Windows 资源管理器风格的只读属性弹窗，展示笔记的类型、位置、大小、统计、标签、时间等信息。后端新增专用 `GetNoteProperties` API（不含 content 全文，统计后端算好），前端新增弹窗 UI。

**菜单项放置**（用户明确要求不放"删除"之后）：放在右键菜单**第一组（打开组）的"查看"之后**——"查看"与"属性"同为只读信息类操作，语义同组，远离危险区。

## Current State Analysis

- 右键菜单：[index.html#L2351-L2365](../../frontend/index.html) `#contextMenu`，四组结构（编辑/查看 | 置顶/移动到/管理标签 | 复制内容/导出/创建副本 | 删除），`data-action` 分发
- 菜单逻辑：[main.js#L5235-L5308](../../frontend/src/main.js) `showContextMenu` + `handleContextAction`
- Note 模型：[note.go#L10-L22](../../internal/models/note.go) — id/title/content/file_ext/pinned/notebook_id/created_at/updated_at/tags/notebook
- 服务层：`NoteService.GetByID`（note_service.go L95）只 `Preload("Tags")`，且 gorm 软删除默认过滤——回收站笔记查不到，无法显示"已删除"状态
- 弹窗模式：[modals.css#L494-L792](../../frontend/src/css/components/modals.css) `.confirm-*` 与 `.import-conflict-*` 两套 overlay + dialog + `active` 类切换可参考
- wailsjs 绑定：新后端方法需手动同步 `frontend/wailsjs/go/main/App.js` / `App.d.ts`（沿用上次 `ResolveImportConflict` 的做法）

## Proposed Changes

### 1. 后端：新增属性 API — app.go + note_service.go

**note_service.go** 新增专用方法（不动现有 `GetByID`，避免影响既有调用方）：

```go
// GetNoteWithRelations 按 ID 取笔记（含 Tags、Notebook，Unscoped 使回收站笔记也可查）
func (s *NoteService) GetNoteWithRelations(id uint) (*models.Note, error) {
    var note models.Note
    err := s.db.Unscoped().Preload("Tags").Preload("Notebook").First(&note, id).Error
    ...
}
```

**app.go** 新增方法与结构体：

```go
type NoteProperties struct {
    ID           uint      `json:"id"`
    Title        string    `json:"title"`
    FileExt      string    `json:"file_ext"`
    NotebookName string    `json:"notebook_name"`
    Pinned       bool      `json:"pinned"`
    Tags         []string  `json:"tags"`
    SizeBytes    int       `json:"size_bytes"`   // len(content)
    CharCount    int       `json:"char_count"`   // utf8.RuneCountInString
    LineCount    int       `json:"line_count"`   // strings.Count + 1
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    Deleted      bool      `json:"deleted"`      // DeletedAt.Valid
}

func (a *App) GetNoteProperties(noteID uint) (*NoteProperties, error)
```

统计（字节数/字符数/行数）在后端计算，前端纯展示。

### 2. wailsjs 绑定同步

- `frontend/wailsjs/go/main/App.js`：新增 `GetNoteProperties(noteID)` 导出
- `frontend/wailsjs/go/main/App.d.ts`：新增对应签名
- 优先尝试 `wails generate module`，失败则手动同步（与上次相同）

### 3. 前端：菜单项 + 弹窗 HTML — index.html

**菜单项**：在"查看"（`data-action="view"`，L2354）之后插入：

```html
<div class="context-menu-item" data-action="properties"><svg ...信息图标.../>属性</div>
```

位置：第一组内"查看"之后，其后仍是既有 divider——不新增分组，不靠近"删除"。

**弹窗骨架**（放在 `#contextMenu` 附近的全局弹窗区域）：

```
#notePropertiesOverlay (.note-properties-overlay)
  └ .note-properties-dialog
      ├ .note-properties-header   → 图标 + 标题 + 类型副标题（"Markdown 笔记 (.md)"）
      ├ .note-properties-body     → 信息行列表（.note-properties-row：label + value）
      └ .note-properties-footer   → 单个"关闭"按钮
```

展示字段（顺序固定）：
类型（Markdown 笔记 (.md) / 文本笔记 (.txt)，副标题已有则 body 不重复）、位置（笔记本名）、大小（格式化 KB/MB + 原始字节数）、字符数、行数、标签（无则"无"）、置顶（是/否）、创建时间、修改时间、状态（正常/已删除，已删除时红色强调）、笔记 ID

### 4. 前端：逻辑 — main.js

- `handleContextAction` 新增 `case "properties"` → `showNoteProperties(state.contextNoteId)`（复用 `showContextMenu` 记录的当前笔记 ID，与其它 case 取值方式一致）
- 新增 `showNoteProperties(noteId)`：
  1. 调 `window.go.main.App.GetNoteProperties(noteId)`
  2. 填充 DOM（时间格式化复用现有 `formatTime`/等价函数；大小用现成格式化或本地实现）
  3. 加 `active` 类显示；"关闭"按钮 / 点击遮罩 / Escape 关闭（与 `.import-conflict-overlay` 交互模式一致）
  4. 请求失败走现有 toast/错误提示惯例

### 5. 样式 — modals.css

新增 `.note-properties-*` 系列类（overlay 遮罩、dialog 容器、header/body/footer、row 两列布局），风格对齐 `.confirm-dialog`（宽度约 420px、圆角、主题变量配色）。只读无表单控件。

## Assumptions & Decisions

1. 菜单项放第一组"查看"之后（用户要求不放删除后，此位置语义最贴切）
2. 回收站笔记同样可用（`Unscoped` 查询，状态行显示"已删除"红色）
3. 不复用列表数据，每次打开实时调 API（数据准确、传输小，content 不出后端）
4. 显示笔记 ID（放最后一行，调试用）
5. 弹窗纯只读，唯一交互是关闭

## Verification

1. `go build ./...` 编译通过
2. `go test ./...` 全部通过
3. `wails dev` 手动验证：
   - 正常笔记：右键 → 属性 → 各字段正确（对比列表显示的时间、标签）
   - 大笔记：大小/字符数/行数正确
   - 回收站笔记：右键 → 属性 → 状态显示"已删除"
   - Esc / 遮罩 / 关闭按钮三种方式均可关闭
   - 菜单中"属性"位于"查看"之后、"删除"远端
