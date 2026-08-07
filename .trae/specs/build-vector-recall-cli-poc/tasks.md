# Tasks

- [x] Task 1: 初始化 vec-poc 独立模块：在项目根目录创建 `vec-poc/`，初始化 go module `vec-poc`，引入依赖（`github.com/glebarez/sqlite`、`gorm.io/gorm`、`github.com/ollama/ollama/api`、`github.com/sashabaranov/go-openai`、`modernc.org/sqlite/vec` 的 blank import 路径）。
  - [x] SubTask 1.1: 创建 `vec-poc/` 目录与 `go.mod`（go 1.26，module `vec-poc`）
  - [x] SubTask 1.2: `go get` 上述依赖并确认编译通过（`go build ./...`）
  - [x] SubTask 1.3: 建 `cmd/vec-poc/main.go` 空入口，保证能编译运行

- [x] Task 2: sqlite-vec 扩展可用性探针：编写探针，blank import `modernc.org/sqlite/vec` 后通过 glebarez 驱动执行 `SELECT vec_version()`。
  - [x] SubTask 2.1: 实现 `internal/store/probe.go`，返回扩展版本或错误
  - [x] SubTask 2.2: 在 `status` 中展示探针结果；若扩展不可用，记录原因（供回退路径使用）

- [x] Task 3: 文本切块模块：实现 `internal/chunk/chunk.go`。
  - [x] SubTask 3.1: 按 Markdown 标题 + 空行分段
  - [x] SubTask 3.2: 单块上限 500 字（rune 安全），超长硬切
  - [x] SubTask 3.3: 单元测试：2000 字输入 → ≥ 4 块且每块 ≤ 500 字

- [x] Task 4: Ollama embedding 客户端：实现 `internal/embed/ollama.go`。
  - [x] SubTask 4.1: 使用 `ollama/api` 的 `Embed`（`api.EmbedRequest{Model, Input []string}`）批量生成向量
  - [x] SubTask 4.2: 连接失败时返回可读错误（含地址）
  - [x] SubTask 4.3: 支持配置 Ollama 地址与模型名

- [x] Task 5: 向量存储与检索层：实现 `internal/store/`。
  - [x] SubTask 5.1: 定义 `VectorStore` 接口（`AddDocument`/`Rebuild`/`Search`/`ListDocs`/`Status`）
  - [x] SubTask 5.2: GORM + glebarez 打开独立 DB（`--db`），AutoMigrate `documents`/`chunks` 两张表
  - [x] SubTask 5.3: sqlite-vec 实现：`vec_f32` 写入 BLOB、`vec_distance_cosine` KNN 查询
  - [x] SubTask 5.4: 纯 Go 回退实现：BLOB 存 float32 LE，全表扫描余弦相似度排序
  - [x] SubTask 5.5: 启动时按探针结果选择实现，可强制 `--force-brute` 切换

- [x] Task 6: 互联网模型对话客户端：实现 `internal/llm/openai.go`。
  - [x] SubTask 6.1: `sashabaranov/go-openai` 非流式 `CreateChatCompletion`
  - [x] SubTask 6.2: 组装 system message（召回块 → 上下文格式，见 spec "互联网模型对话召回"）
  - [x] SubTask 6.3: 召回为空时无上下文直接回答，并提示"未召回相关知识"

- [x] Task 7: CLI 主程序与 REPL：实现 `cmd/vec-poc/main.go` 交互逻辑。
  - [x] SubTask 7.1: flag 解析（`--db`/`--ollama-url`/`--embed-model`/`--llm-base-url`/`--llm-api-key`/`--llm-model`/`--topk`/`--force-brute`）+ 环境变量兜底，flag 优先
  - [x] SubTask 7.2: 单次命令模式：`add <file>` / `index` / `ask <问题>` / `list` / `status`（flag 子命令或 `-cmd`）
  - [x] SubTask 7.3: 交互 REPL：`add`/`index`/`ask`/`list`/`status`/`help`/`quit`
  - [x] SubTask 7.4: `ask` 流程：问题 embedding → 召回 TopN 块 → 打印召回块 → 调用 LLM → 打印回答

- [x] Task 8: 端到端验证：真实链路试跑。
  - [x] SubTask 8.1: 在 `vec-poc/` 准备一个测试文本文件（`sample.md`，含明确主题内容）
  - [x] SubTask 8.2: `add sample.md` + `index` 成功，输出块数（沙箱无法启动 Ollama，以 store 集成测试 `TestStoreAddSearchBothImpls`/`TestStoreRebuild` 用确定性伪向量验证等价链路：添加→索引→重建均通过，sqlite-vec 与 pure-go-brute 双实现）
  - [x] SubTask 8.3: `ask` 输出召回块与模型回答（沙箱无 LLM key 且 Ollama 不可启动；已验证：无 LLM 配置时 `ask` 输出清晰错误提示；召回链路由集成测试覆盖；LLM 回答需用户在正常环境提供 `--llm-base-url/--llm-api-key/--llm-model` 后实测）
  - [x] SubTask 8.4: 整理运行说明（`help` 输出完整命令与 flag 说明；最终交付说明含使用示例）

# Task Dependencies

- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 1]
- [Task 4] depends on [Task 1]
- [Task 5] depends on [Task 1, Task 2]
- [Task 6] depends on [Task 1]
- [Task 7] depends on [Task 3, Task 4, Task 5, Task 6]
- [Task 8] depends on [Task 7]

（Task 2/3/4/6 相互独立，可在 Task 1 后并行）
