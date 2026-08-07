# Checklist

## 移除关键词召回
- [x] `recall_service.go` 中 gse 分词全套（stopWords/isStopWord/tokenize/gseSeg 等）、maxRecallKeywords、CardRecallSearch 已删除
- [x] `note_service.go` 中 `SearchFull` 已删除
- [x] `app.go` 中关键词召回块与 `combinedQuery` 已删除，`refinedQuery` 保留
- [x] `go.mod` 已移除 `github.com/go-ego/gse`
- [x] `go build ./...` 通过，无残留关键词召回引用

## 向量召回替代
- [x] 向量召回由 `cardRecallEnabled` 开关门控（开关关不召回、不发射卡片事件）
- [x] 开关开时 `recallNotebookIDs` 传入 `VectorRecall` 限定笔记本范围，条数沿用 `ai_card_recall_limit`
- [x] 召回结果经 `MergeRecallCards` → `recallCardsJSON` → `TruncateRecallCardsPreview(200)` → 发射 `ai:recall-cards`，卡片展示逻辑不受影响

## 校验方法
- [x] VectorService 新增 `CountAllVectors` 与 `CountVectorsByModel` 方法，按 model 精确过滤
- [x] `ValidateCardRecall()` 返回结构化结果 `{ ok, message }`，校验顺序：基础配置 → 模型类型 → 量化表内容
- [x] 基础判断：provider/base_url/model 任一为空时返回拒绝与对应提示
- [x] 模型类型判断：openai 无 Key 拒绝；ollama 跳过 Key 检查
- [x] 量化表为空（总记录数 0）时返回拒绝与对应提示
- [x] 当前量化模型在量化表无记录时返回拒绝与对应提示
- [x] 全部校验通过时返回 `ok=true`，不阻断正常开启

## 前端校验接入
- [x] 设置页开关 `aiSettingCardRecallToggle`：开启被拒时回滚 active、不保存、warning 通知
- [x] 对话内开关 `aiChatCardRecallToggle`：开启被拒时回滚状态、不持久化、warning 通知
- [x] 笔记本下拉勾选自动开启：被拒时回滚 checkbox、恢复关闭态、不持久化、warning 通知
- [x] 校验通过时三处入口维持现有开启流程（全选笔记本、会话持久化等）不受影响

## 整体
- [x] `go build ./...` 编译通过
