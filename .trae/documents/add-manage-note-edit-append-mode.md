# manage\_note edit 动作新增追加模式（append\_content）计划

## Summary

为 manage\_note 的 `edit` 动作新增第三种模式：`append_content`（末尾追加）。解决当前"追加内容"需模型持有全文/文末锚点的痛点——长笔记场景模型看不到文末，追加实际不可行。改动集中在 `internal/agent/tools/manage_note.go` 单文件。

## Current State Analysis

* [manage\_note.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/manage_note.go) 的 `edit` 动作当前为双模式（互斥）：

  * 全量：`content` 非空 → 整篇替换正文

  * 片段：`find`+`replace`+`count` → 定位第 N 次出现的片段替换（`replace` 空=删除）

* 追加现状：全量拼接（短笔记）或 find 文末锚点（长笔记 view 截断后模型看不到文末 → 不可行）

* 服务层 `NoteService.Update(id, title, content, fileExt)` 非空才更新，`GetNoteContent(id)` 读全文，均现成

* 已确认文件当前状态与本会话上次修改一致（无用户侧改动）

## Proposed Changes

全部改动在 `internal/agent/tools/manage_note.go`，共 4 处：

### 1. 头注释（edit 动作说明，约 L21-L22）

`edit` 动作说明由"双模式互斥"改为"三模式互斥"，新增：`content` 与 `find` 均为空、`append_content` 非空时为追加模式，在正文末尾拼接文本（无需先读全文）。

### 2. Info() 参数（L107-L131 区域）

* `content` 的 Desc 追加互斥说明：改为"笔记内容，action=create 时必填；action=edit 时非空即整篇替换正文（与 find、append\_content 互斥）"

* 新增参数：

```go
"append_content": {
    Type:     schema.String,
    Desc:     "追加到笔记末尾的文本，仅 action=edit 且需追加内容时使用（与 content、find 互斥），无需先获取全文",
    Required: false,
},
```

* Info() 的 `Desc`（L94）中 edit 描述补一句：末尾追加用 `append_content`（无需先读全文）。

### 3. InvokableRun（结构体 + 分发）

* 参数结构体新增字段：`AppendContent string \`json:"append\_content"\`\`

* edit 分发调用签名扩展：`m.editNote(args.ID, args.Content, args.Find, args.Replace, args.AppendContent, args.Count)`

### 4. editNote 方法与追加分支

* 签名扩展：`editNote(id float64, content, find, replace, appendContent string, count float64)`

* 模式判定改为三选一互斥（非空模式计数 >1 报错"三种模式不可混用"；=0 报错"无可更新内容"）

* 新增追加分支：

```go
// 追加模式：将 append_content 拼接到正文末尾（无需先读全文）
current, err := m.note.GetNoteContent(uint(id))
if err != nil {
    return "", err
}
newContent := current
if strings.TrimSpace(current) != "" {
    newContent += "\n\n"
}
newContent += appendContent
if _, err := m.note.Update(uint(id), "", newContent, ""); err != nil {
    return "", err
}
return fmt.Sprintf("笔记 #%d 已在末尾追加内容", uint(id)), nil
```

注意：追加分支与片段分支都需先 `GetNoteContent`，各自独立读取，不做合并优化（保持逻辑简单）。

## Assumptions & Decisions

* **三模式互斥**：`content` / `find` / `append_content` 同时传多个 → 报错，避免模型混淆

* `append_content` 为空字符串视为未提供（追加空文本无意义，走"无可更新内容"报错）

* 追加使用 `\n\n` 分隔（与既有正文分隔风格一致）；正文为空时直接写入追加文本

* 动作数不变（仍为 edit），`ActionText`（"编辑笔记正文"）、meta.go、registry.go、doc.go 均无需改动

* 并发覆盖风险与现状一致，不做版本校验（本地单机）

## Verification

1. `go build ./...` 通过
2. `go vet ./internal/agent/...` 通过
3. `GetDiagnostics` 检查 manage\_note.go 无错误
4. 手动验证路径（可选，重启应用后）：

   * Agent 消息"在这篇笔记末尾追加：xxx" → 模型调用 `edit` + `append_content` → 正文末尾出现追加内容

   * 同时传 `content` 与 `append_content` → 返回"三种模式不可混用"错误并回填模型

