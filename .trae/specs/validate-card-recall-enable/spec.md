# 向量召回替代关键词召回 + 卡片召回开启校验 Spec

## Why
1. 现有问答存在两套召回并行：关键词召回（gse 分词 + LIKE 整篇注入，token 占用大）与向量召回（语义相似度 + 命中块注入）。关键词召回逻辑冗余，且其"整篇笔记注入"正是最初要解决的 token 占用问题。
2. 启用"卡片召回"开关后问答执行向量检索，但若量化配置缺失、量化表为空或当前量化模型在量化表中无记录，召回必然无结果，用户无感知。

综上：**彻底移除关键词召回，由向量召回按现有 UI 设置（开关 + 笔记本下拉 + 条数）全权接管**，并在开关启用时做前置校验，不满足则拒绝开启并通知提示。

## What Changes
- **移除关键词召回**：删除 gse 分词、停用词表、`CardRecallSearch`、`SearchFull`、`combinedQuery` 及其依赖（`github.com/go-ego/gse`）
- **向量召回受"卡片召回"开关控制**：`cardRecallEnabled == true` 时才执行 `VectorRecall`（替代原先无条件执行）；开关名称"卡片召回"、UI 结构、会话配置键（`enable_card_recall`/`recall_notebook_ids`）、召回条数设置（`ai_card_recall_limit`）全部保持不变
- **按现有 UI 笔记本选择限定召回范围**：`VectorRecall` 继续接收前端下拉勾选的 `recallNotebookIDs`，勾选了指定笔记本 → 只检索这些笔记本的已量化内容；全选/未勾选 → 检索全部已量化内容（JOIN notes 过滤软删除）
- **引用功能保持独立**：`referencedNotes` → `BuildNoteRefContext` 整篇注入逻辑不受影响，与召回互不相干
- **新增后端校验** `ValidateCardRecall()`，完成三类校验（配置基础 → 模型类型 → 量化表内容），返回 `{ ok, message }`
- **前端三处"启用召回"入口接入校验**，失败时回滚 UI 状态并通知提示

## Impact
- 受影响代码：
  - `internal/services/recall_service.go`：删除 gse/停用词/`tokenize`/`CardRecallSearch`/`maxRecallKeywords`；保留 `RecallCard`/`CardRecallResult`/`MergeRecallCards`/`TruncateRecallCardsPreview`/`TruncateSearchSourcesPreview`
  - `internal/services/note_service.go`：删除 `SearchFull`（仅关键词召回在用）
  - `go.mod`：移除 `github.com/go-ego/gse`
  - `internal/services/vector_service.go`：新增按模型计数方法
  - `app.go`：删除关键词召回块与 `combinedQuery`；向量召回块包进 `cardRecallEnabled` 门控；新增 `ValidateCardRecall` 绑定方法
  - `frontend/src/main.js`：设置页开关 `aiSettingCardRecallToggle` 接入校验
  - `frontend/src/js/ai-chat.js`：对话内开关 `aiChatCardRecallToggle` + 笔记本下拉勾选自动开启接入校验

## ADDED Requirements

### Requirement: 移除关键词召回
系统 SHALL 彻底移除关键词召回链路，包括 gse 分词（`stopWords`/`isStopWord`/`initGseSegmenter`/`tokenize`）、`maxRecallKeywords`、`CardRecallSearch`、`NoteService.SearchFull`、`combinedQuery` 拼接及 `github.com/go-ego/gse` 依赖。
- `recallCardsJSON`、`ai:recall-cards` 事件、卡片面板展示逻辑保留，由向量召回结果驱动
- `refinedQuery`（联网搜索精炼）保留，仅删除其在关键词召回的拼接使用

#### Scenario: 移除后编译
- **WHEN** 删除全部关键词召回代码
- **THEN** `go build ./...` 通过，无残留引用

### Requirement: 向量召回受卡片召回开关控制
系统 SHALL 将向量召回从"无条件执行"改为由 `cardRecallEnabled` 开关门控：
- 开关开启 → 执行 `VectorRecall`（按 `recallNotebookIDs` 限定笔记本范围，空则全库；条数用 `ai_card_recall_limit`）
- 开关关闭 → 跳过召回，不注入、不发射卡片
- 召回结果继续走 `MergeRecallCards` 合并（保留合并逻辑，向量结果单路时同样生效）→ `recallCardsJSON` 持久化 → `TruncateRecallCardsPreview(200)` 后发射 `ai:recall-cards`

#### Scenario: 开关开启
- **WHEN** 卡片召回开关开启且校验通过
- **THEN** 问答执行向量召回，命中块注入 system message，卡片事件正常发射

#### Scenario: 开关关闭
- **WHEN** 卡片召回开关关闭
- **THEN** 不执行任何召回，`ai:recall-cards` 不发射

### Requirement: 按选中笔记本检索量化内容
系统 SHALL 在向量召回时按前端下拉勾选的 `recallNotebookIDs` 限定检索范围：
- 勾选部分笔记本 → `VectorRecall` JOIN notes 过滤，仅检索这些笔记本的已量化块（跳过软删除）
- 全选或未勾选 → 检索全部已量化块

#### Scenario: 指定笔记本召回
- **WHEN** 用户在下拉中仅勾选笔记本 A 并开启召回
- **THEN** 向量检索仅在笔记本 A 的已量化内容中执行

### Requirement: 后端校验方法 ValidateCardRecall
系统 SHALL 提供 `ValidateCardRecall()` Wails 绑定方法，返回 `{ ok: bool, message: string }`。校验按以下顺序执行，任一不满足即返回 `ok=false` 与对应提示：

1. **基础判断**：`ai_embed_provider`、`ai_embed_base_url`、`ai_embed_model` 任一为空（量化连接未设置或量化模型未选择）→ 拒绝，提示"请先在设置中配置量化连接与量化模型"
2. **模型类型判断**：
   - `ai_embed_provider == "openai"`：`ai_embed_api_key` 为空 → 拒绝，提示"请先填写量化 API Key"
   - `ai_embed_provider == "ollama"`：**不检查** API Key
3. **量化表内容判断**：
   - `note_vectors` 表总记录数为 0 → 拒绝，提示"量化表为空，请先在数据管理中量化笔记"
   - `note_vectors` 表中 `model = 当前量化模型` 的记录数为 0 → 拒绝，提示"当前量化模型「X」暂无量化数据，请先使用该模型量化笔记"
4. 全部通过 → 返回 `ok=true`

#### Scenario: 未配置量化连接
- **WHEN** 量化模型未设置时点击开启召回
- **THEN** 拒绝开启，通知提示"请先在设置中配置量化连接与量化模型"

#### Scenario: OpenAI 未填 Key
- **WHEN** provider 为 openai 且量化 API Key 为空时点击开启召回
- **THEN** 拒绝开启，通知提示"请先填写量化 API Key"

#### Scenario: Ollama 无需 Key
- **WHEN** provider 为 ollama 且 API Key 为空时点击开启召回
- **THEN** Key 检查跳过，进入量化表校验

#### Scenario: 量化表为空
- **WHEN** `note_vectors` 表无任何记录时点击开启召回
- **THEN** 拒绝开启，通知提示"量化表为空，请先在数据管理中量化笔记"

#### Scenario: 当前模型无量化数据
- **WHEN** 量化表有数据但当前量化模型无对应记录时点击开启召回
- **THEN** 拒绝开启，通知提示当前模型暂无量化数据

#### Scenario: 全部通过
- **WHEN** 量化配置完整、量化表有数据且当前模型有记录时点击开启召回
- **THEN** 正常开启，无提示（或按现状提示开启成功）

### Requirement: 设置页开关校验
设置页"卡片召回"开关 `aiSettingCardRecallToggle` 被点击且**即将开启**（当前为关闭态）时，SHALL 先调用 `ValidateCardRecall()`：
- `ok=true`：正常开启（含同步工具栏开关、全选笔记本）
- `ok=false`：**回滚开关状态**（不开启）、不保存设置、调用通知方法 `nm.show(message, 'warning')`

#### Scenario: 设置页开启被拒
- **WHEN** 在设置页点击开启且校验失败
- **THEN** 开关保持关闭态，弹出 warning 通知显示拒绝原因

### Requirement: 对话内开关校验
对话内"卡片召回"开关 `aiChatCardRecallToggle` 的 knob 区域被点击且**即将开启**时，SHALL 先调用 `ValidateCardRecall()`：
- `ok=true`：正常开启（含全选笔记本、持久化会话配置）
- `ok=false`：**回滚开关状态**、不执行全选、不持久化，调用通知方法提示

#### Scenario: 对话内开启被拒
- **WHEN** 在对话工具栏点击开启且校验失败
- **THEN** 开关保持关闭态，弹出 warning 通知显示拒绝原因

### Requirement: 笔记本下拉勾选自动开启校验
对话内召回笔记本下拉 `aiChatRecallDropdown` 勾选笔记本，导致召回状态从**关闭变为开启**（`recallNotebookIds.size` 从 0 变 > 0 且此前 `enableCardRecall == false`）时，SHALL 先调用 `ValidateCardRecall()`：
- `ok=true`：正常勾选并开启
- `ok=false`：**回滚该复选框勾选**、恢复开关关闭态、不持久化，调用通知方法提示

#### Scenario: 下拉勾选被拒
- **WHEN** 在召回下拉勾选首个笔记本且校验失败
- **THEN** 复选框回滚为未勾选，开关保持关闭，弹出 warning 通知显示拒绝原因

## 校验方法说明
- `ValidateCardRecall` 读取 `ai_embed_*` 四键（复用 `GetEmbedConfig`），量化表检查走 `VectorService` 新增的按模型计数方法
- 校验逻辑在后端完成（需读 DB），前端仅做 UI 回滚与通知展示
