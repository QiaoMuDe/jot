# 提升 manage\_note edit 精度的优化

## 背景

edit 简化为片段替换 + 行级替换两种模式后，模型的编辑精度仍有提升空间：view 截断后缺总行数、edit feedback 缺 diff 预览、未找到时缺相似片段提示、工具描述缺模式选择引导。以下 5 项优化投入产出比高，改动集中。

## 涉及文件

| 文件                                         | 变更                                                  |
| ------------------------------------------ | --------------------------------------------------- |
| `internal/agent/tools/manage_note.go`      | 主要修改（viewNote 截断提示、editNote feedback、未找到提示、工具 Desc） |
| `internal/agent/tools/manage_note_test.go` | 新增/更新测试用例                                           |

## 详细变更

### 1. viewNote 截断提示加入总行数

**文件**: `internal/agent/tools/manage_note.go`，viewNote 函数（约 L503-532）

**现状**: 截断消息只有字符统计：

```
（内容共 25000 字符，已显示前 10000。如需继续阅读...）
```

**改为**: 同时输出总行数（在 lineNumbers 模式下，可通过最后一个行号推算；非行号模式下用 splitNoteLines 计数）：

```
（内容共 25000 字符 / 380 行，已显示前 10000 字符 / 62 行。如需继续阅读...）
```

**实现**: 在截断逻辑前，用 `splitNoteLines(content)` 计算总行数 `totalLines`；截断后如果带行号，从最后一个行号（`行 62: xxx`）提取已显示行数。将两个数字加入 Sprintf 格式串。

### 2. editNote feedback 带上下文预览

**文件**: `internal/agent/tools/manage_note.go`，editNote 函数

#### 行级替换 feedback（约 L629-632）

**现状**:

```
笔记 #123 已替换第 5-8 行（原共 30 行）
```

**改为**: 附带替换后新内容的摘要（替换区域前后各 1 行 + 替换区域本身，限 300 字符）：

```
笔记 #123 已替换第 5-8 行（原共 30 行，现共 29 行）：
行 4: 前文上下文
行 5: 新内容第一行
行 6: 新内容第二行
行 7: 后文上下文
```

**实现**: 在 Update 成功后，从 `newContent` 中提取替换区域（start-1 到 end 新范围）前后各 1 行，用 `numberLines` 格式化，截取后追加到 feedback。新总行数通过 `splitNoteLines(newContent)` 计算。

#### 片段替换 feedback（约 L649、L677）

**现状**:

```
笔记 #123 正文片段已替换（第 1 处，精确匹配）
```

**改为**: 附带替换前后片段摘要（各限 80 字符）：

```
笔记 #123 正文片段已替换（第 1 处，精确匹配）：
-旧: "这是被替换的原文片段内容..."
+新: "这是替换后的新内容..."
```

**实现**: 替换前取 `current[pos:pos+matchedLen]` 截断 80 字符；替换后取 `replace` 截断 80 字符。拼接到 feedback 尾部。`replaceAll` 模式下只展示第一处的 diff。

### 3. 片段替换未找到时返回最相似片段

**文件**: `internal/agent/tools/manage_note.go`，editNote 函数片段模式的两个错误分支（约 L668-671）

**现状**:

```
未在笔记 #123 中找到该片段（已尝试空白归一化匹配），请重新调用 view 获取精确原文后重试
```

**改为**: 找到笔记中与 find 最接近的子串作为提示：

```
未在笔记 #123 中找到该片段（已尝试空白归一化匹配）。笔记中最接近的内容片段：
「这是笔记中最相似的一段文字...」（第 N 行附近）
请确认片段是否正确，或调用 view 获取精确原文后重试
```

**实现**: 新增 `findMostSimilar(content, find string) (string, int)` 辅助函数——用滑动窗口（窗口大小 = len(find) ± 50%）遍历原文，计算每个窗口与 find 的简单字符重叠率（`count common runes / max(len(find), len(window))`），返回得分最高的片段及其在原文中的大致行号（通过计数 `\n` 得到）。只在 find 非空且未找到时调用。

### 4. 工具描述引导模式选择

**文件**: `internal/agent/tools/manage_note.go`，Info() Desc 字符串中 edit 部分（约 L151）

**现状**: edit 描述只列出两种模式的参数，没有引导何时用哪种。

**改为**: 在 edit 描述末尾（行级替换说明之后）加一句引导：

```
（只需修改几个字或一句话用片段替换；需要修改连续多行、整段重写、或无法用简短片段定位时用行级替换）
```

### 5. 行级替换校验 replace 行数

**文件**: `internal/agent/tools/manage_note.go`，editNote 函数行级替换分支（约 L608-632）

**现状**: 无校验，直接传入 `replaceLines`。

**改为**: 在调用 `replaceLines` 前，用 `splitNoteLines(replace)` 计算 replace 的行数 `newLines`，与被替换行数 `end-start+1` 比较。当 `newLines > end-start+1` 时（替换后行数比原来多），在 feedback 中追加提示：

```
（注：替换后行数从 X 行变为 Y 行，新增了 Z 行）
```

这不阻止操作，只在 feedback 中提供信息帮助模型判断是否符合预期。

## 不修改的文件

* 前端 JS：无 edit 相关引用

* `context.go`、`AGENTS.md`：不涉及

* `read_note_section.go`：不涉及（它的行号处理已正确）

## 验证步骤

1. `go build ./...` 编译通过
2. `go test ./internal/agent/tools/... -v` 所有测试通过
3. `go vet ./internal/agent/tools/...` 无静态分析问题
4. 手动检查：确认 viewNote 截断消息格式正确、editNote feedback 包含预期的 diff/上下文信息

