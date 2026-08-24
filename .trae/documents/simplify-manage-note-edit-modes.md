# 简化 manage\_note edit 模式：从 4 种减为 2 种

## 背景

`manage_note` 的 `edit` 动作当前有 4 种互斥模式（全量替换 / 片段替换 / 行级替换 / 末尾追加），参数 Schema 复杂（8 个参数），模型选择负担重。经分析，"全量替换"是行级替换的特例（start=1, end=total），"末尾追加"也是行级替换的特例（start=total+1），可以合并。

## 变更摘要

* **保留**：片段替换（find+replace）、行级替换（line\_start/line\_end/replace）

* **移除**：全量替换（content 参数）、末尾追加（append\_content 参数）

* edit 模式从 4 选 1 简化为 2 选 1，参数从 8 个减到 6 个

## 涉及文件

| 文件                                         | 变更类型   |
| ------------------------------------------ | ------ |
| `internal/agent/tools/manage_note.go`      | 主要修改   |
| `internal/agent/tools/manage_note_test.go` | 更新测试用例 |
| `internal/agent/tools/context.go`          | 更新注释   |
| `AGENTS.md`                                | 更新记忆点  |

## 详细变更

### 1. `internal/agent/tools/manage_note.go`

#### 1.1 文件头注释（L8-33）

* 更新 edit 模式的描述，从"四模式互斥"改为"双模式互斥"

* 移除全量替换和末尾追加的说明

#### 1.2 `Info()` 工具描述（L148-273）

* `content` 参数（L164-168）：Desc 改为仅描述 `action=create` 时的用途，移除 `action=edit 时非空即整篇替换正文` 的描述

* `append_content` 参数（L209-213）：**整行删除**

* `line_start` 参数（L194-198）：Desc 移除 `与 content、find、append_content 互斥`，改为 `与 find 互斥`；补充说明 start > 笔记总行数时为末尾追加语义

* `find` 参数（L175-178）：Desc 移除 `与 content、append_content、line_start 互斥`，改为 `与 line_start 互斥`

* 工具 Desc 中 edit 部分（L151）：重写 edit 描述，只保留片段替换和行级替换两种模式，明确说明：

  * 片段替换：find+replace（与之前一致）

  * 行级替换：line\_start+line\_end+replace（覆盖删除行、替换行、末尾追加）

  * 末尾追加：传 line\_start 为笔记总行数+1，replace 为追加内容

#### 1.3 参数结构体（L277-301）

* 移除 `AppendContent string \`json:"append\_content"\`\` 字段（L289）

* `Content` 字段保留（create 仍需要），但 edit 逻辑中不再使用

#### 1.4 `editNote` 函数（L588-712）

* 移除 `appendContent` 参数

* 移除模式计数中的 `appendContent` 判断（L604-605）

* 将模式判定从"四选一"改为"二选一"（find 非空 或 lineStart > 0）

* 移除 `content != ""` 的全量替换分支（L620-625）

* 移除 `appendContent != ""` 的追加分支（L628-643）

* 更新错误信息：`请四选一` → `请二选一`，移除 content/append\_content 的提示

* 片段替换和行级替换逻辑保持不变

#### 1.5 `InvokableRun` 中 edit 的调用（L351）

* 移除 `args.AppendContent` 参数传递

#### 1.6 `isManageNoteWriteAction`（L74-81）和 `manageNoteActionCN`（L84-101）

* 不变（edit 动作本身仍在，只是内部模式减少）

### 2. `internal/agent/tools/manage_note_test.go`

#### `TestEditNoteModeValidation`（L233-267）

* 移除 `"content+find 混用"` 测试用例（不再有 content 作为 edit 模式）

* 移除 `"content+append 混用"` 测试用例

* 移除 `"append+line 混用"` 测试用例

* 保留 `"find+line 混用"` 和 `"全空"` 测试用例

* 新增测试用例：

  * `"find+replace 模式"` — 正常路径（mock DB 验证）

  * `"line 模式"` — 正常路径

  * `"append via line_start > total"` — 通过行级替换实现追加

### 3. `internal/agent/tools/context.go`

* L33 常量注释：从 `content / replace / append_content` 改为 `content / replace`

### 4. `AGENTS.md`

* L514 记忆点 20：更新 edit 模式描述，从"四模式互斥"改为"双模式互斥（find+replace 片段替换 / line\_start+line\_end 行级替换，后者覆盖全量替换与末尾追加）"

## 不修改的文件

* 前端 JS：无 edit 模式引用，不需要改

* `context.go` 的 `maxToolLongText` 常量值：不变（replace 仍可能很长）

* `.trae/specs/` 和 `.trae/documents/` 下的历史文档：不修改（保留历史记录）

* `Info()` 中 `content` 参数定义：保留（create 仍需要），仅更新 Desc

## 验证步骤

1. `go build ./...` 确认编译通过
2. 运行 `go test ./internal/agent/tools/... -run TestEditNoteModeValidation` 确认模式互斥测试通过
3. 运行 `go test ./internal/agent/tools/...` 确认所有工具测试通过
4. 运行 `go vet ./...` 确认无静态分析问题

