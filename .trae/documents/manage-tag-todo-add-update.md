# 给 manage_tag / manage_todo 增加 Update 动作

## Summary

为两个现有 Agent 工具各增加一个 `update` 动作：

- `manage_tag.update`：按编号 `id` 重命名标签 / 修改标签颜色（name、color 均可选，至少提供一个）
- `manage_todo.update`：按编号 `id` 修改待办文本（text 必填）

复用服务层已存在的 `TagService.Update(id, name, color)` 与 `TodoService.Update(id, text)`，不新增任何后端绑定。无需改注册（`registry.go`）、父包依赖、前端展示（工具名直显英文，默认"执行"动作文案即可）。

## Current State Analysis

### manage_tag（[manage_tag.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/manage_tag.go)）
- 结构：`manageTagTool{ tag *services.TagService, ctx *Context }`
- 现有动作：`create`（name 必填、color 可选校验 `#RRGGBB`、同名查重返回提示）、`list`（全部标签，不分页）
- 参数解析结构体：`{ Action, Name, Color string }`——**无 id 字段**
- 现有查重逻辑（createTag L109-115）：`GetByName` + `errors.Is(err, gorm.ErrRecordNotFound)` 判断不存在；存在则返回提示而非错误

### manage_todo（[manage_todo.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/manage_todo.go)）
- 结构：`manageTodoTool{ todo *services.TodoService, ctx *Context }`
- 现有动作：`create`（text 必填）、`list`（status/keyword 过滤 + 分页）、`toggle`（id 必填、正整数，切换完成/未完成）
- 参数解析结构体：`{ Action, Text, Status, Keyword string; ID, Page, PageSize float64 }`——已有 id 字段模式（toggle 用 `args.ID <= 0` 校验）

### 服务层（已存在，无需修改）
- [tag_service.go L39-59](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/tag_service.go#L39-L59)：`Update(id uint, name, color string)` —— name/color 为空则保留原值；ID 不存在返回 `errors.New("tag not found")`（非 gorm.ErrRecordNotFound）
- [todo_service.go L139-151](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/todo_service.go#L139-L151)：`Update(id uint, text string)` —— 直接覆盖 text；ID 不存在返回 `First` 的原始 gorm 错误

### 文档
- [tools/doc.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/doc.go)：仅列工具名与构造器，**不列动作**，无需改动
- [TOOLS.md §6 表格 L304-306](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/TOOLS.md#L304-L306)：manage_todo / manage_tag 功能描述需补充 update

## Proposed Changes

### 1. manage_tag.go — 增加 `update` 动作

**文件头注释**：在动作列表补一行 `- update：更新标签（id 必填；name 可选新名称、color 可选新颜色，格式 #RRGGBB；至少提供其一；重命名时排除自身的同名查重）`。

**Info()**：
- `Desc`：追加说明 update 动作——"update=更新标签（需提供 id 标签编号，可提供 name 新名称或 color 新颜色，至少其一）"
- `action` Enum：`["create", "list", "update"]`
- `name` Desc：改为"标签名称，action=create 时必填，action=update 时可选（提供则重命名）"
- `color` Desc：改为"标签颜色（格式 #RRGGBB），action=create 时可选（缺省 #3b82f6），action=update 时可选（提供则改色）"
- 新增参数 `id`：`schema.Number`，Desc"标签编号（正整数，列表中的 [数字] 即为 id），action=update 时必填"，非 Required（与其他可选参数一致）

**InvokableRun()**：`args` 结构体加 `ID float64` 字段；`switch` 加 `case "update": return m.updateTag(ctx, args.ID, args.Name, args.Color)`；日志 Debugw 补 `fastlog.Int("id", int(args.ID))`。

**新增方法 `updateTag(ctx, id float64, name, color string)`**：
1. `id <= 0` → `errors.New("manage_tag 更新标签缺少有效的 id")`
2. `strings.TrimSpace` name/color；`name == "" && color == ""` → 返回提示 `"未提供任何更新字段：需要 name 新名称或 color 新颜色"`（非错误，避免无意义调用）
3. `color != "" && !tagColorPattern.MatchString(color)` → `fmt.Errorf("manage_tag 参数非法 color: %s（应为 #RRGGBB 格式）", color)`
4. 用户取消检查：`ctx.Err() != nil` → 返回 ctx.Err()
5. name 非空时查重（排除自身）：`GetByName(name)`；错误非 `gorm.ErrRecordNotFound` 则返回 err；查到时若 `existing.ID != uint(id)` 返回提示 `"标签「%s」已存在（编号 [%d]），无法重命名为同名标签"`（非错误）
6. 调用 `m.tag.Update(uint(id), name, color)`
7. 返回 `fmt.Sprintf("已更新标签「%s」（编号 [%d]）· 颜色 %s", tag.Name, tag.ID, tag.Color)`

### 2. manage_todo.go — 增加 `update` 动作

**文件头注释**：在动作列表补 `- update：修改待办文本（id 必填、text 必填）`。

**Info()**：
- `Desc`：追加 "update=修改待办文本（需提供 id 待办编号与 text 新内容）"
- `action` Enum：`["create", "list", "toggle", "update"]`
- `text` Desc：改为"待办内容，action=create / update 时必填"
- `id` Desc：改为"待办编号（正整数，列表中的 [数字] 即为 id），action=toggle / update 时必填"

**InvokableRun()**：`switch` 加 `case "update"`：
```go
case "update":
    if args.ID <= 0 {
        return "", errors.New("manage_todo 更新待办缺少有效的 id")
    }
    text := strings.TrimSpace(args.Text)
    if text == "" {
        return "", errors.New("manage_todo 更新待办缺少 text")
    }
    t, err := m.todo.Update(uint(args.ID), text)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("已更新待办 #%d：%s", t.ID, t.Text), nil
```

### 3. TOOLS.md §6 表格更新

- `manage_todo` 行功能改为：`创建 / 列出（支持 status/keyword 过滤与 page/pageSize 分页）/ 勾选（完成或取消）/ 修改文本（update）`
- `manage_tag` 行功能改为：`列出全部标签 / 创建标签（name 必填，color 可选 #RRGGBB）/ 更新标签（重命名、改色，name/color 至少其一）`

## Assumptions & Decisions

- **定位方式**：update 用 `id`（列表返回的 `[数字]` 编号）定位，与 toggle 保持一致
- **manage_tag.update 重名冲突**：查重排除自身（`GetByName` 命中但 ID 相同 = 名称未变，不冲突）；其他同名返回提示而非错误，与 create 的既有风格一致
- **manage_tag.update 至少一个字段**：name/color 全空时返回提示（非错误），避免无意义的 `Save`
- **不存在 ID 的错误处理**：直接返回服务层错误（manage_tag 的 "tag not found"、manage_todo 的 gorm 错误），经 `WrapWithError` 回填模型，与现有 toggle 行为一致，不做字符串特判
- **不改前端**：`showToolStatusStart` 无 update 专属动作文案需求，走默认"执行"；工具名直显英文
- **不改 registry.go / agent.go / app.go**：工具依赖与注册均不变
- **不改 tools/doc.go**：该文件只列工具名与构造器，不列动作

## Verification

1. `go build ./...`
2. `go vet ./internal/agent/...`
3. 重启应用，Agent 模式下验证：
   - 创建标签 → `manage_tag` 列出拿 id → update 重命名 → update 改色 → 重命名为已存在标签名应返回提示
   - 创建待办 → `manage_todo` 列出拿 id → update 修改文本 → 确认列表显示新文本
