# Tasks

- [ ] Task 1: 重新引入 GSE 依赖
  - [ ] SubTask 1.1: `go get github.com/go-ego/gse` 拉取依赖
  - [ ] SubTask 1.2: 在 `vector_service.go` import 中添加 `_ "github.com/go-ego/gse/pkg/embed"`（EMBED 嵌入式词典，编译进二进制）
  - [ ] SubTask 1.3: 确认 go.mod / go.sum 更新成功，`go build` 通过

- [ ] Task 2: 实现 GSE 分词器懒加载
  - [ ] SubTask 2.1: 在 `vector_service.go` 中新增包级 `sync.Once` + `*gse.Tokenizer` 变量
  - [ ] SubTask 2.2: 实现 `getTokenizer()` 函数：sync.Once 内用 EMBED 词典初始化 GSE（词典已嵌入二进制，不会加载失败）
  - [ ] SubTask 2.3: 实现 `tokenize(query string) []string`：调用 GSE Cut 分词
  - [ ] SubTask 2.4: 过滤分词结果：去除停用词（标点/空白/单字符英文）和过短 token（< 2 rune）

- [ ] Task 3: 实现 KeywordRecall 方法
  - [ ] SubTask 3.1: 在 `vector_service.go` 新增 `KeywordRecall(ctx, query, limit, notebookIDs) ([]models.NoteVector, error)`
  - [ ] SubTask 3.2: 调用 tokenize 分词，无有效 token 时返回空列表
  - [ ] SubTask 3.3: 构建 SQL：JOIN notes 过滤软删除 + notebookIDs，WHERE 条件用 OR 拼接各 token 的 `chunk_text LIKE '%token%'`，按命中 token 数降序 + LIMIT
  - [ ] SubTask 3.4: 日志记录分词结果和命中数

- [ ] Task 4: 实现 HybridRecall 方法
  - [ ] SubTask 4.1: 新增 `HybridRecall(ctx, query, limit, embedClient, notebookIDs) (*CardRecallResult, error)`
  - [ ] SubTask 4.2: 向量检索路：复用现有 vec_distance_cosine SQL 逻辑（抽取为内部函数 `vectorSearch`），embedClient 为 nil 或无数据时跳过
  - [ ] SubTask 4.3: 关键词检索路：调用 KeywordRecall
  - [ ] SubTask 4.4: 合并去重：按 (note_id, chunk_index) 去重，标记双命中/仅向量/仅关键词
  - [ ] SubTask 4.5: 排序：双命中 > 仅向量 > 仅关键词，同优先级按向量距离升序（仅关键词按命中 token 数降序）
  - [ ] SubTask 4.6: 相邻块扩展 + 卡片组装：复用现有 hitIndexes / byNote / hitOrder / RecallCard 逻辑
  - [ ] SubTask 4.7: FormattedText 标注来源为"混合检索"

- [ ] Task 5: VectorRecall 委托 HybridRecall
  - [ ] SubTask 5.1: `VectorRecall` 函数体改为调用 `HybridRecall`，保留对外签名不变
  - [ ] SubTask 5.2: 确认 app.go 绑定层无需改动

- [ ] Task 6: 构建验证
  - [ ] SubTask 6.1: `go build ./...` 通过
  - [ ] SubTask 6.2: `golangci-lint run ./...` 0 issues
  - [ ] SubTask 6.3: `npm run build` 通过

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 2]
- [Task 4] depends on [Task 3]
- [Task 5] depends on [Task 4]
- [Task 6] depends on [Task 5]
