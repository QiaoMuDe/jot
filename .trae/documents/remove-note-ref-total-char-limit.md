# 移除笔记引用总字符数限制逻辑

## 问题

`BuildNoteRefContext()` 中仍存在**总字符数限制**逻辑：用 `max_file_size`（MB）换算为字节，作为所有引用笔记内容的总长度上限。超出上限的笔记被截断并标记为 `Truncated: true`。

这与之前移除各场景截断逻辑的路线不一致——笔记引用应该全量注入，不应有总长度限制。

## 当前状态

[note_service.go#L141-L182](file:///d:/%E5%B3%A1%E8%B0%B7/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/internal/services/note_service.go#L141-L182)

```go
	// 从设置动态读取最大文件限制数（转换为字节），默认 1MB
	maxTotalChars := 1 * 1024 * 1024
	if s.settingService != nil {
		if val := s.settingService.Get("max_file_size"); val != "" {
			if n, err := strconv.Atoi(val); err == nil && n > 0 && n <= 100 {
				maxTotalChars = n * 1024 * 1024
			}
		}
	}

	notes := make([]NoteRefInfo, 0, len(rows))
	var parts []string
	totalLen := 0

	for _, row := range rows {
		noteText := row.Content
		truncated := false

		block := fmt.Sprintf("--- 📄 《%s》 ---\n%s", row.Title, noteText)

		// 总长度截断
		if totalLen+len(block) > maxTotalChars {
			parts = append(parts, fmt.Sprintf("--- 📄 《%s》 ---\n...(内容已截断，超出上下文长度限制)", row.Title))
			notes = append(notes, NoteRefInfo{
				ID:           row.ID,
				Title:        row.Title,
				Truncated:    true,
				NotebookName: row.NotebookName,
			})
			// 剩余笔记不再处理
			break
		}

		parts = append(parts, block)
		totalLen += len(block)
		notes = append(notes, NoteRefInfo{
			ID:           row.ID,
			Title:        row.Title,
			Truncated:    truncated,
			NotebookName: row.NotebookName,
		})
	}
```

## 修改方案

### 改动 1: 移除总字符数限制逻辑

移除：
- 第 141-149 行：`maxTotalChars` 计算（从 `max_file_size` 读取）
- 第 153 行：`totalLen` 变量声明
- 第 157 行：`truncated` 变量声明（已无用）
- 第 161-172 行：总长度截断检查 + 截断处理

改为：所有引用笔记无限制地全量拼接，`Truncated` 始终为 `false`。

```go
	notes := make([]NoteRefInfo, 0, len(rows))
	var parts []string

	for _, row := range rows {
		block := fmt.Sprintf("--- 📄 《%s》 ---\n%s", row.Title, row.Content)

		parts = append(parts, block)
		notes = append(notes, NoteRefInfo{
			ID:           row.ID,
			Title:        row.Title,
			Truncated:    false,
			NotebookName: row.NotebookName,
		})
	}
```

### 不修改

- `NoteRefInfo` 结构体：保留 `Truncated` 字段，虽然始终为 `false`，但它是 API 合约的一部分，移除需同步更新前端 `models.ts`，非必要不改
- `max_file_size` 的其余使用（`readAIChatFiles`、`ImportNotes`）保持不变

## 验证

1. 引用多条笔记时，所有笔记内容应完整注入，无截断提示
2. 即使引用内容总长度超过 10MB，也不应截断
3. 前端 chip 始终显示未截断状态