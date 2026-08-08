# 分块元数据前缀注入 Spec

## Why
当前向量化分块仅包含标题链 + 正文，缺少笔记标题、标签、创建时间等元信息。向量 embedding 和关键词 LIKE 检索都无法匹配到笔记标题/标签中的关键词，导致命中率不高。在每个分块前注入元数据前缀可同时提升向量语义检索和关键词检索的召回质量。

## What Changes
- `chunk.go`：`ChunkContent` 函数签名新增 `ChunkMeta` 参数，每个分块前面注入元数据前缀模板
- `chunk.go`：新增 `ChunkMeta` 结构体（Title / Tags / CreatedAt）
- `vector_service.go`：`IndexNotes` 查询笔记时加 `Preload("Tags")`，构造 `ChunkMeta` 传入 `ChunkContent`
- 单块上限从 500 rune 调整为 600 rune（补偿元数据前缀占用的 ~80-100 rune）
- **BREAKING**：`ChunkContent` 签名变更，已有调用方（`chunk_test.go`）需同步更新

## Impact
- Affected code: `internal/services/chunk.go`（核心改动）、`internal/services/vector_service.go`（调用方）、`internal/services/chunk_test.go`（测试同步）
- 不影响：前端、app.go 绑定层、数据库 schema
- 已有向量数据需重新量化才能生效（chunk_text 格式变了）

## ADDED Requirements

### Requirement: ChunkMeta 结构体
系统 SHALL 定义 `ChunkMeta` 结构体，包含笔记元数据用于分块前缀注入：
- `Title string`：笔记标题
- `Tags []string`：标签名称列表
- `CreatedAt time.Time`：创建时间

#### Scenario: 构造 ChunkMeta
- **WHEN** IndexNotes 处理每篇笔记
- **THEN** 从 note 对象构造 ChunkMeta{Title: note.Title, Tags: tagNames, CreatedAt: note.CreatedAt}

### Requirement: 元数据前缀模板
系统 SHALL 在每个分块的正文前注入格式化元数据前缀，模板如下：
```
笔记标题：{title}
分类标签：{tag1, tag2}
创建时间：{2006-01-02}
笔记核心内容：
{标题链 + 分段正文}
```

规则：
- 标签用中文逗号分隔（`、`），无标签时整行省略
- 创建时间格式 `2006-01-02`（精确到天）
- `笔记核心内容：` 后换行，接原有的标题链 + 正文（保留 ChunkContent 现有标题链拼接逻辑）
- 元数据前缀计入 maxRunes 限制

#### Scenario: 有标签的笔记
- **GIVEN** note.Title="数据库设计", tags=["架构","后端"], CreatedAt=2026-08-07
- **WHEN** ChunkContent 切块
- **THEN** 每块格式为 `笔记标题：数据库设计\n分类标签：架构、后端\n创建时间：2026-08-07\n笔记核心内容：\n{标题链+正文}`

#### Scenario: 无标签的笔记
- **GIVEN** note.Title="日记", tags=[], CreatedAt=2026-08-07
- **WHEN** ChunkContent 切块
- **THEN** 每块格式为 `笔记标题：日记\n创建时间：2026-08-07\n笔记核心内容：\n{标题链+正文}`（分类标签行省略）

### Requirement: 单块上限调整
系统 SHALL 将 `ChunkContent` 的默认单块上限从 500 rune 调整为 600 rune，补偿元数据前缀占用的空间。

#### Scenario: IndexNotes 调用 ChunkContent
- **WHEN** IndexNotes 调用 ChunkContent
- **THEN** maxRunes 参数传 600

## MODIFIED Requirements

### Requirement: ChunkContent 函数签名
`ChunkContent` 签名从 `func ChunkContent(content string, maxRunes int) []string` 改为 `func ChunkContent(content string, maxRunes int, meta ChunkMeta) []string`。切块逻辑不变，仅在 flush 时将元数据前缀拼接到每块前面。`maxRunes` 计算包含元数据前缀长度。

### Requirement: IndexNotes 笔记查询
`IndexNotes` 查询笔记时增加 `.Preload("Tags")` 预加载标签关系，并从 `note.Tags` 提取标签名称列表构造 `ChunkMeta`。
