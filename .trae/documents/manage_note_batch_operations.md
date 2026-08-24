# manage\_note 批量操作功能实现计划

## 概述

为 AI 助手模块的 `manage_note` 工具增加批量操作功能，支持通过笔记 ID 数组进行批量移动、批量打标签、批量删除标签。

## 当前状态分析

### 已有的批量方法（后端服务层）

| 功能     | 服务层方法                                                        | 位置                                  |
| ------ | ------------------------------------------------------------ | ----------------------------------- |
| 批量移动笔记 | `BatchMoveToNotebook(noteIDs []uint, targetNotebookID uint)` | `internal/services/note_service.go` |
| 批量打标签  | `BatchAddTagToNotes(noteIDs []uint, tagID uint)`             | `internal/services/tag_service.go`  |
| 批量删除标签 | `BatchRemoveTagFromNotes(noteIDs []uint, tagID uint)`        | `internal/services/tag_service.go`  |

### 当前 AI Agent 工具层（需要修改）

`internal/agent/tools/manage_note.go` 中的 `move`、`add_tag`、`remove_tag` 三个 action **只接受单个** **`id`** **参数**，不支持批量。

## 实现方案

### 设计决策

**采用"统一数组"方案**：将 `id` 参数直接改为 `ids` 数组形式：

* 单条操作时传 `ids: [1]`

* 批量操作时传 `ids: [1, 2, 3]`

这样设计最简洁统一，AI 模型更容易理解。

### 修改文件

`internal/agent/tools/manage_note.go`

### 具体改动

#### 1. 更新文件头部注释（第30-32行）

**原内容**：

```
//   - add_tag / remove_tag：给笔记添加/移除标签（id 必填、正整数，tag_id 必填、正整数）。
// 与 recall_notes 的边界：recall_notes 用于语义召回笔记片段回答知识类问题，
// manage_note 用于结构化操作笔记库。本工具不包含删除类/批量动作（spec 明确不暴露）。
```

**改为**：

```
//   - add_tag / remove_tag：给笔记添加/移除标签（ids 笔记编号数组必填，tag_id 必填、正整数）。
// 与 recall_notes 的边界：recall_notes 用于语义召回笔记片段回答知识类问题，
// manage_note 用于结构化操作笔记库。本工具不包含删除类动作（spec 明确不暴露）。
// move/add_tag/remove_tag 支持批量操作：单条时传 ids=[id]，批量时传 ids=[id1,id2,...]。
```

#### 2. 更新工具描述（Info 方法，第149行）

更新相关 action 的描述：

* `view=查看笔记全文（需提供 ids 笔记编号数组，通常传 [id]）`

* `update=更新笔记标题/扩展名（需提供 ids 笔记编号数组与 title/file_ext 至少其一）`

* `edit=编辑笔记正文（需提供 ids 笔记编号数组）`

* `pin=置顶/取消置顶笔记（需提供 ids 笔记编号数组）`

* `move=移动笔记到目标笔记本（需提供 ids 笔记编号数组与 notebook_id 目标笔记本）`

* `add_tag=给笔记添加标签（需提供 ids 笔记编号数组与 tag_id 标签编号）`

* `remove_tag=从笔记移除标签（需提供 ids 笔记编号数组与 tag_id 标签编号）`

#### 3. 修改参数定义（第249-253行）

将 `id` 参数改为 `ids` 数组：

**原内容**：

```go
"id": {
    Type:     schema.Number,
    Desc:     "笔记编号（正整数，列表中的 [数字] 即为 id），action=view / update / edit / pin / move / add_tag / remove_tag 时必填",
    Required: false,
},
```

**改为**：

```go
"ids": {
    Type:     schema.Array,
    ElemInfo: &schema.ParameterInfo{Type: schema.Number},
    Desc:     "笔记编号数组（正整数列表，列表中的 [数字] 即为 id）；单条操作时传 [id]，批量操作时传 [id1,id2,...]；action=view / update / edit / pin / move / add_tag / remove_tag 时必填",
    Required: false,
},
```

#### 4. 修改参数解析结构体（第291行）

将 `ID float64` 改为 `IDs []float64`：

```go
var args struct {
    // ... 现有字段 ...
    IDs         []float64 `json:"ids"`  // 笔记编号数组
    // ... 其他字段 ...
}
```

#### 5. 修改 switch 分发逻辑（第334-353行）

统一使用 `args.IDs`：

```go
switch args.Action {
case "create":
    return m.createNote(args.Title, args.Content, args.FileExt, args.NotebookID, args.TagIDs)
case "list":
    return m.listNotes(args.Keyword, int(args.Page), int(args.PageSize), int(args.NotebookID), args.SortBy, args.StartDate, args.EndDate, args.TagIDs)
case "view":
    return m.viewNote(args.IDs)
case "update":
    return m.updateNote(args.IDs, args.Title, args.FileExt)
case "edit":
    return m.editNote(args.IDs, args.Find, args.Replace, args.Count, args.ReplaceAll, args.LineStart, args.LineEnd)
case "pin":
    return m.pinNote(args.IDs)
case "move":
    return m.moveNote(args.IDs, args.NotebookID)
case "add_tag":
    return m.addTag(args.IDs, args.TagID)
case "remove_tag":
    return m.removeTag(args.IDs, args.TagID)
}
```

#### 6. 添加辅助函数 `resolveNoteIDs`

```go
// resolveNoteIDs 从 ids 数组中提取有效笔记 ID：过滤 <= 0 的无效值，返回去重后的 []uint。
// 调用方须确保返回值非空（入参校验在各 action 方法中完成）。
func resolveNoteIDs(ids []float64) []uint {
    seen := make(map[uint]bool)
    var result []uint
    for _, v := range ids {
        if v <= 0 {
            continue
        }
        uid := uint(v)
        if !seen[uid] {
            seen[uid] = true
            result = append(result, uid)
        }
    }
    return result
}
```

#### 7. 修改 `viewNote` 方法

```go
// viewNote 查看笔记全文：ids 必填，通常只传一个元素 [id]。
func (m *manageNoteTool) viewNote(ids []float64) (string, error) {
    noteIDs := resolveNoteIDs(ids)
    if len(noteIDs) == 0 {
        return "", errors.New("manage_note 查看笔记缺少有效的 ids")
    }
    if len(noteIDs) > 1 {
        return "", errors.New("manage_note 查看笔记只支持单条操作，请在 ids 中传入一个笔记编号")
    }
    id := noteIDs[0]
    // ... 原有逻辑 ...
}
```

#### 8. 修改 `updateNote` 方法

```go
// updateNote 更新笔记标题/扩展名：ids 必填，通常只传一个元素 [id]。
func (m *manageNoteTool) updateNote(ids []float64, title, fileExt string) (string, error) {
    noteIDs := resolveNoteIDs(ids)
    if len(noteIDs) == 0 {
        return "", errors.New("manage_note 更新笔记缺少有效的 ids")
    }
    if len(noteIDs) > 1 {
        return "", errors.New("manage_note 更新笔记只支持单条操作，请在 ids 中传入一个笔记编号")
    }
    id := noteIDs[0]
    // ... 原有逻辑 ...
}
```

#### 9. 修改 `editNote` 方法

```go
// editNote 编辑笔记正文：ids 必填，通常只传一个元素 [id]。
func (m *manageNoteTool) editNote(ids []float64, find, replace string, count float64, replaceAll bool, lineStart, lineEnd float64) (string, error) {
    noteIDs := resolveNoteIDs(ids)
    if len(noteIDs) == 0 {
        return "", errors.New("manage_note 编辑笔记缺少有效的 ids")
    }
    if len(noteIDs) > 1 {
        return "", errors.New("manage_note 编辑笔记只支持单条操作，请在 ids 中传入一个笔记编号")
    }
    id := noteIDs[0]
    // ... 原有逻辑 ...
}
```

#### 10. 修改 `pinNote` 方法

```go
// pinNote 置顶/取消置顶笔记：ids 必填，通常只传一个元素 [id]。
func (m *manageNoteTool) pinNote(ids []float64) (string, error) {
    noteIDs := resolveNoteIDs(ids)
    if len(noteIDs) == 0 {
        return "", errors.New("manage_note 置顶笔记缺少有效的 ids")
    }
    if len(noteIDs) > 1 {
        return "", errors.New("manage_note 置顶笔记只支持单条操作，请在 ids 中传入一个笔记编号")
    }
    id := noteIDs[0]
    // ... 原有逻辑 ...
}
```

#### 11. 修改 `moveNote` 方法（支持批量）

```go
// moveNote 移动笔记到目标笔记本：ids 必填，支持单条 [id] 或批量 [id1,id2,...] 操作。
func (m *manageNoteTool) moveNote(ids []float64, notebookID float64) (string, error) {
    if notebookID <= 0 {
        return "", errors.New("manage_note 移动笔记缺少有效的 notebook_id")
    }
    noteIDs := resolveNoteIDs(ids)
    if len(noteIDs) == 0 {
        return "", errors.New("manage_note 移动笔记缺少有效的 ids")
    }

    // 批量操作
    if len(noteIDs) > 1 {
        if err := m.note.BatchMoveToNotebook(noteIDs, uint(notebookID)); err != nil {
            return "", err
        }
        return fmt.Sprintf("已将 %d 篇笔记移动到笔记本 #%d", len(noteIDs), uint(notebookID)), nil
    }

    // 单条操作
    if err := m.note.MoveToNotebook(noteIDs[0], uint(notebookID)); err != nil {
        return "", err
    }
    return fmt.Sprintf("已将笔记 #%d 移动到笔记本 #%d", noteIDs[0], uint(notebookID)), nil
}
```

#### 12. 修改 `addTag` 方法（支持批量）

```go
// addTag 给笔记添加标签：ids 必填，支持单条 [id] 或批量 [id1,id2,...] 操作。
func (m *manageNoteTool) addTag(ids []float64, tagID float64) (string, error) {
    if tagID <= 0 {
        return "", errors.New("manage_note 添加标签缺少有效的 tag_id")
    }
    noteIDs := resolveNoteIDs(ids)
    if len(noteIDs) == 0 {
        return "", errors.New("manage_note 添加标签缺少有效的 ids")
    }

    // 批量操作
    if len(noteIDs) > 1 {
        if err := m.tag.BatchAddTagToNotes(noteIDs, uint(tagID)); err != nil {
            return "", err
        }
        return fmt.Sprintf("已为 %d 篇笔记添加标签 #%d", len(noteIDs), uint(tagID)), nil
    }

    // 单条操作
    if err := m.tag.AddTagToNote(noteIDs[0], uint(tagID)); err != nil {
        return "", err
    }
    return fmt.Sprintf("已为笔记 #%d 添加标签 #%d", noteIDs[0], uint(tagID)), nil
}
```

#### 13. 修改 `removeTag` 方法（支持批量）

```go
// removeTag 从笔记移除标签：ids 必填，支持单条 [id] 或批量 [id1,id2,...] 操作。
func (m *manageNoteTool) removeTag(ids []float64, tagID float64) (string, error) {
    if tagID <= 0 {
        return "", errors.New("manage_note 移除标签缺少有效的 tag_id")
    }
    noteIDs := resolveNoteIDs(ids)
    if len(noteIDs) == 0 {
        return "", errors.New("manage_note 移除标签缺少有效的 ids")
    }

    // 批量操作
    if len(noteIDs) > 1 {
        if err := m.tag.BatchRemoveTagFromNotes(noteIDs, uint(tagID)); err != nil {
            return "", err
        }
        return fmt.Sprintf("已从 %d 篇笔记移除标签 #%d", len(noteIDs), uint(tagID)), nil
    }

    // 单条操作
    if err := m.tag.RemoveTagFromNote(noteIDs[0], uint(tagID)); err != nil {
        return "", err
    }
    return fmt.Sprintf("已从笔记 #%d 移除标签 #%d", noteIDs[0], uint(tagID)), nil
}
```

## 验证步骤

1. **编译检查**：`go build ./...` 确保无编译错误
2. **单元测试**：如果项目有测试，运行相关测试
3. **功能验证**：

   * 测试单条操作：传 `ids: [1]`

   * 测试批量操作：传 `ids: [1, 2, 3]`

   * 测试写操作确认机制：不传 `confirm=true` 时应拒绝执行

   * 测试无效参数：空数组、负数 ID 等

## 兼容性分析

### 1. 前端页面功能影响

**结论：不影响**

* AI Agent 工具（`manage_note.go`）是独立模块，通过 AI 模型动态调用

* 前端页面通过 Wails API（`app.go`）直接调用后端服务层（`note_service.go`、`tag_service.go`）

* 前端不直接调用 AI Agent 工具，所以修改 AI Agent 工具参数不影响前端功能

### 2. 其他工具兼容性

**结论：不影响**

* `manage_notebook.go`、`manage_tag.go`、`manage_todo.go`、`read_note_section.go` 等工具有各自独立的 `id` 参数

* 各工具参数定义互不干扰，修改 `manage_note.go` 的 `id` 参数不影响其他工具

### 3. AI 模型调用兼容性

**结论：自动兼容**

* AI Agent 工具通过 JSON Schema 描述参数，AI 模型根据描述动态生成调用参数

* 修改参数名和类型后，AI 模型会根据新的工具描述自动适应新的调用方式

* 不存在硬编码调用，所以不存在兼容性问题

### 4. 潜在风险

| 风险                 | 影响 | 应对措施                           |
| ------------------ | -- | ------------------------------ |
| AI 模型可能需要重新学习新参数格式 | 低  | 工具描述会清晰说明新格式，模型会自动适应           |
| 用户可能有自定义的 AI 调用脚本  | 低  | AI Agent 工具主要用于交互式对话，很少有外部脚本调用 |

## 假设与决策

1. **统一数组设计**：将 `id` 改为 `ids` 数组，所有操作统一使用数组形式
2. **单条操作限制**：`view`、`update`、`edit`、`pin` 保持单条操作，传入多个 ID 时报错提示
3. **批量操作支持**：`move`、`add_tag`、`remove_tag` 支持批量操作
4. **错误处理策略**：复用现有服务层的批量方法，遇到错误不中断，最终合并错误返回
5. **去重处理**：`resolveNoteIDs` 会自动去重，避免重复操作
6. **批量操作仍需确认**：因为 `move`、`add_tag`、`remove_tag` 仍属于写操作，批量操作同样需要 `ask_user` 确认

