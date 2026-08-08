# 修复 XLS row\.Col() panic + Markdown 表格管道符转义

## 概述

修复 markitdown 库中两个已确认的问题：

1. **#1**：`converter_xls.go` 中 `row.Col()` 调用可因 xls 库内部 bug 导致 panic，当前 `safeRow` 仅保护了 `sheet.Row(i)`，未覆盖后续的 `row.Col()` 调用。
2. **#4**：`converter_csv.go` 中 `renderMarkdownTable` 函数不转义单元格内容中的 `|`，导致含管道符的单元格破坏 Markdown 表格结构。该函数被 CSV / XLS / XLSX / PPTX 四个转换器共享。

## 当前状态

### #1 row\.Col() panic

`converter_xls.go` 第 88-99 行：

```go
for rowIdx := 0; rowIdx <= maxRow; rowIdx++ {
    row := safeRow(sheet, rowIdx)  // ← 仅此处有 recover 保护
    if row == nil { continue }
    var cells []string
    lastCol := row.LastCol()
    for colIdx := 0; colIdx < lastCol; colIdx++ {
        cells = append(cells, row.Col(colIdx))  // ← 未保护，可 panic
    }
    rows = append(rows, cells)
}
```

`row.Col(colIdx)` 内部调用 `ch.String(r.wb)`，存在三条 panic 路径（均在 xls 库 `col.go` / `cell_range.go` 中）：

* `LabelsstCol.String()` → `wb.sst[int(c.Sst)]` 越界

* `MulrkCol.String()` → 空切片后 `strs[0]` 越界

* `HyperLink.String()` → uint16 下溢导致 `make` 分配巨量内存

### #4 表格管道符未转义

`converter_csv.go` 第 79-119 行 `renderMarkdownTable` 函数直接 `b.WriteString(records[0][i])` 写入单元格内容，不转义 `|`。

## 修改方案

### 修改 1：converter\_xls.go — 扩大 recover 保护范围

**文件**：`internal/markitdown/converter_xls.go`

**做法**：将 `safeRow` 替换为 `safeRowCells`，把整行提取（`Row()` + `LastCol()` + `Col()` 循环）包在一个 recover 函数内。任一环节 panic 时返回 nil，该行被跳过。

```go
// safeRowCells 安全地提取一行的所有单元格文本。
// extrame/xls 库在行缺失、SST 索引越界、MULRK 空切片等场景会 panic，
// 这里用 recover 统一兜底，把异常行当作空行跳过。
func safeRowCells(sheet *xls.WorkSheet, rowIdx int) (cells []string) {
	defer func() {
		if recover() != nil {
			cells = nil
		}
	}()
	row := sheet.Row(rowIdx)
	if row == nil {
		return nil
	}
	lastCol := row.LastCol()
	for colIdx := 0; colIdx < lastCol; colIdx++ {
		cells = append(cells, row.Col(colIdx))
	}
	return cells
}
```

调用处改为：

```go
for rowIdx := 0; rowIdx <= maxRow; rowIdx++ {
    cells := safeRowCells(sheet, rowIdx)
    if cells == nil {
        continue
    }
    rows = append(rows, cells)
}
```

删除旧的 `safeRow` 函数。

### 修改 2：converter\_csv.go — 管道符转义

**文件**：`internal/markitdown/converter_csv.go`

**做法**：在 `renderMarkdownTable` 中写入单元格内容前，将 `|` 替换为 `\|`。添加一个内联 helper 函数：

```go
// escapeTableCell 转义 Markdown 表格单元格中的管道符。
func escapeTableCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
```

在 `renderMarkdownTable` 的三处写入位置（header row、data rows）调用 `escapeTableCell`：

* 第 93 行：`b.WriteString(escapeTableCell(records[0][i]))`

* 第 112 行：`b.WriteString(escapeTableCell(row[i]))`

### 修改 3：测试更新

**文件**：`internal/markitdown/converter_xls_test.go`

将 `TestSafeRowMissingRowNoPanic` 更新为测试 `safeRowCells`：

```go
func TestSafeRowCellsMissingRowNoPanic(t *testing.T) {
	sheet := &xls.WorkSheet{}
	for i := 0; i < 10; i++ {
		if cells := safeRowCells(sheet, i); cells != nil {
			t.Fatalf("safeRowCells(%d) = %v, want nil", i, cells)
		}
	}
}
```

**文件**：`internal/markitdown/markitdown_test.go`

新增 `TestRenderMarkdownTablePipeEscaping` 测试管道符转义：

```go
func TestRenderMarkdownTablePipeEscaping(t *testing.T) {
	records := [][]string{
		{"name", "formula"},
		{"a|b", "x|y|z"},
	}
	result := renderMarkdownTable(records)
	// 管道符应被转义
	if strings.Contains(result, "a|b") && !strings.Contains(result, "a\\|b") {
		t.Errorf("pipe in cell not escaped: %s", result)
	}
}
```

## 验证步骤

1. `cd internal/markitdown && go build ./...`
2. `cd internal/markitdown && go test ./... -count=1`（含新测试 + 现有 golden file 测试）
3. `cd internal/markitdown && go vet ./...`
4. `cd .. && go build ./...`（确认主项目编译通过）
5. 检查 golden file 测试是否通过（若 testdata 中的 fixture 不含 `|`，golden 输出应无变化）

