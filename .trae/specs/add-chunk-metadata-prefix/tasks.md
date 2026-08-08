# Tasks

- [x] Task 1: 新增 ChunkMeta 结构体 + 元数据前缀拼接函数
  - [x] SubTask 1.1: 在 `chunk.go` 新增 `ChunkMeta` 结构体（Title string, Tags []string, CreatedAt time.Time）
  - [x] SubTask 1.2: 新增 `formatMetaPrefix(meta ChunkMeta) string` 函数：按模板拼接元数据前缀（标签用 `、` 分隔，无标签时省略该行，时间格式 `2006-01-02`）
  - [x] SubTask 1.3: 前缀末尾以 `笔记核心内容：` 结尾，后续接标题链+正文

- [x] Task 2: 修改 ChunkContent 集成元数据前缀
  - [x] SubTask 2.1: `ChunkContent` 签名改为 `(content string, maxRunes int, meta ChunkMeta) []string`
  - [x] SubTask 2.2: 在 `flush` 函数中，每块拼接好标题链+正文后，再在前面拼接 `formatMetaPrefix(meta)` + `\n`
  - [x] SubTask 2.3: maxRunes 计算包含元数据前缀长度（硬切时前缀不计入正文硬切范围，每块都带完整前缀）
  - [x] SubTask 2.4: `splitWithHeading` 硬切的非首段也需拼接元数据前缀

- [x] Task 3: 修改 IndexNotes 调用方
  - [x] SubTask 3.1: 笔记查询加 `.Preload("Tags")` 预加载标签
  - [x] SubTask 3.2: 从 `note.Tags` 提取标签名称列表（`[]string`）
  - [x] SubTask 3.3: 构造 `ChunkMeta{Title: note.Title, Tags: tagNames, CreatedAt: note.CreatedAt}` 传入 `ChunkContent`
  - [x] SubTask 3.4: `maxRunes` 参数从 500 改为 600

- [x] Task 4: 同步更新 chunk_test.go
  - [x] SubTask 4.1: 所有 `ChunkContent(content, maxRunes)` 调用加 `ChunkMeta` 参数（用空 meta 或合理测试数据）
  - [x] SubTask 4.2: 新增测试用例：验证有标签/无标签的元数据前缀格式正确

- [x] Task 5: 构建验证
  - [x] SubTask 5.1: `go build ./...` 通过
  - [x] SubTask 5.2: `go test ./internal/services/...` 通过
  - [x] SubTask 5.3: `golangci-lint run ./...` 0 issues

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 2]
- [Task 4] depends on [Task 2]
- [Task 5] depends on [Task 3, Task 4]
