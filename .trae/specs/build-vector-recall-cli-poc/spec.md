# 向量召回 CLI 测试工具（vec-poc）Spec

## Why

验证"SQLite 向量检索 + Ollama 本地 embedding + 互联网模型对话"整条技术路线，在不接触用户库的前提下快速试跑。核心待验证点：

1. `glebarez/sqlite`（底层 `modernc.org/sqlite`）下 sqlite-vec 扩展是否可用
2. Ollama 本地 embedding 的质量、切块粒度与召回效果
3. 向量召回结果 + 互联网模型（OpenAI 兼容）回答的端到端链路

## What Changes

- **ADDED** 项目根目录新建独立测试目录 `vec-poc/`（独立 go module `vec-poc`，不 import 主项目代码、不连接用户库）
- **ADDED** 命令行 CLI：支持添加文件、重建索引、提问（向量召回 + 互联网模型回答）
- **ADDED** 在当前目录打开新的 SQLite 文件（默认 `vec-test.db`），表结构：`documents` + `chunks`
- **ADDED** 向量检索双实现：sqlite-vec 扩展可用时走 SQL 距离函数，不可用时自动回退纯 Go 余弦相似度（保证 POC 可跑通）

## Impact

- Affected specs: 无（全新独立工具）
- Affected code: 仅新增 `vec-poc/` 目录；主项目（`app.go`、`internal/`、`frontend/`）零改动
- 新增依赖仅存在于 `vec-poc/go.mod`，不影响主项目 go.mod

## ADDED Requirements

### Requirement: 独立 CLI 工具

系统 SHALL 提供位于 `vec-poc/` 的命令行程序，支持以下交互：

- 启动参数（flag）+ 交互式命令（REPL）双模式
- 子命令：`add <文件路径>` 添加文本文件并索引、`index` 重建全部索引、`ask <问题>` 向量召回并对话回答、`list` 列出已添加文档、`status` 显示配置与索引状态、`help`、`quit`

#### Scenario: 交互式使用

- **WHEN** 用户运行 `go run .` 进入 REPL 并输入 `add ./test.md`
- **THEN** 程序读取文件、切块、调用 Ollama embedding 并写入向量表，输出"已添加 X 个块"

#### Scenario: 提问回答

- **WHEN** 用户输入 `ask 我的项目部署步骤是什么`
- **THEN** 程序用 Ollama 对问题做 embedding → 向量召回 TopN 块 → 调用互联网模型生成回答并打印，同时打印召回来源块

### Requirement: 配置项

系统 SHALL 通过命令行 flag 与环境变量双重方式提供配置：

| 配置 | flag | 环境变量 | 默认值 |
|---|---|---|---|
| DB 文件路径 | `--db` | `VEC_DB` | `./vec-test.db` |
| Ollama 地址 | `--ollama-url` | `VEC_OLLAMA_URL` | `http://localhost:11434` |
| Embedding 模型 | `--embed-model` | `VEC_EMBED_MODEL` | `bge-m3` |
| LLM 接口地址 | `--llm-base-url` | `VEC_LLM_BASE_URL` | 必填 |
| LLM API Key | `--llm-api-key` | `VEC_LLM_API_KEY` | 必填 |
| LLM 模型名 | `--llm-model` | `VEC_LLM_MODEL` | 必填 |
| 召回块数 | `--topk` | `VEC_TOP_K` | `5` |

#### Scenario: flag 优先

- **WHEN** flag 与环境变量同时提供同一配置
- **THEN** flag 值生效

### Requirement: 独立数据库文件

系统 SHALL 打开当前目录（或 `--db` 指定的）独立 SQLite 文件，初始化两张表：

- `documents`：`id, name, source_path, content, created_at`
- `chunks`：`id, doc_id, chunk_index, text, embedding BLOB, dim, model, created_at`

该文件与用户库 `~/.jot/data/jot.db` 完全隔离，不读取、不写入用户数据。

### Requirement: 文本切块

系统 SHALL 提供简单切块函数：

- 按 Markdown 标题（`##`/`###`）与空行分段
- 单块上限 500 字（rune 安全），超长段落按 500 字硬切
- 保留每块的原始文本（供注入上下文与展示）

#### Scenario: 长文件切块

- **WHEN** 添加一个 2000 字的文件
- **THEN** 生成 ≥ 4 个文本块，每块 ≤ 500 字

### Requirement: Ollama 本地 Embedding

系统 SHALL 调用 Ollama 的批量 Embedding 接口（`api.EmbedRequest`，`Input` 为文本数组）为文本块生成向量：

- 模型由 `--embed-model` 指定（默认 `bge-m3`）
- 支持批量输入，提升索引速度
- 记录向量维度 `dim` 与模型名 `model` 到 `chunks` 表

#### Scenario: Ollama 不可用

- **WHEN** 启动 `add`/`ask` 时 Ollama 连接失败
- **THEN** 打印明确错误提示（含连接地址），不 panic

### Requirement: 向量存储与检索（含回退）

系统 SHALL 在 `vec-poc/` 提供向量存储层：

- 优先尝试 sqlite-vec 扩展：`SELECT vec_version()` 探针检测；可用时使用 `vec_f32` / `vec_distance_cosine` 等 SQL 函数完成 KNN 查询（`ORDER BY distance LIMIT topk`）
- 扩展不可用时回退：向量以 BLOB（float32 LE）存 `chunks.embedding`，查询时全表扫描、在 Go 侧计算余弦相似度并排序（个人笔记量级性能可接受）
- 抽象 `VectorStore` 接口，两种实现可切换，`status` 显示当前使用哪种实现
- 索引重建：`index` 命令清空 `chunks` 后对全部 `documents` 重新切块 + embedding
- 召回结果按块返回：`doc_id, chunk_index, text, distance`

#### Scenario: 扩展可用性

- **WHEN** 程序启动并执行 `SELECT vec_version()`
- **THEN** 记录结果并在 `status` 中显示"sqlite-vec 可用/不可用（使用纯 Go 回退）"

### Requirement: 互联网模型对话召回

系统 SHALL 调用 OpenAI 兼容接口（`sashabaranov/go-openai`，非流式 `CreateChatCompletion`）完成问答：

- system message 包含召回块内容，格式：`以下是相关知识片段（按相关度排序）：\n\n--- 来源: <文件名> 块<i> ---\n<块文本>...`，并注明"优先基于这些片段回答，片段不足时如实说明"
- user message 为问题原文
- BaseURL / APIKey / Model 来自配置

#### Scenario: 召回为空

- **WHEN** 向量召回无结果
- **THEN** 直接调用互联网模型回答（不加上下文），并提示"未召回相关知识"

### Requirement: 端到端可运行

系统 SHALL 完成一次真实链路验证：添加测试文件 → 重建索引 → 提问 → 输出召回块与模型回答，全程无 panic、无用户库访问。

## MODIFIED Requirements

无。

## REMOVED Requirements

无。
