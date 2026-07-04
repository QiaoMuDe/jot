# 卡片召回智能截断功能 Spec

## Why

卡片召回目前将匹配笔记的全文注入 AI 上下文，长笔记消耗大量 token 且引入噪音。已有的 `ai_ref_max_chars` 设置（引用截断字符数）仅用于手动笔记引用，卡片召回未使用。复用该设置，在笔记内容超阈值时自动截取匹配关键词附近的上下文，可在不丢失关键信息的前提下大幅减少 token 消耗。

## What Changes

- **修改** `recall_service.go` — `CardRecallSearch` 新增 `maxChars int` 参数；当笔记长度 > maxChars 时，按关键词位置截取前后各 200 字上下文
- **修改** `app.go` — 调用 `CardRecallSearch` 时动态读取 `ai_ref_max_chars` 设置并传入

## Impact

- Affected specs: 卡片召回功能
- Affected code: `internal/services/recall_service.go`、`app.go`

## ADDED Requirements

### Requirement: 卡片召回智能截断

The system SHALL truncate long note content in card recall results based on the `ai_ref_max_chars` setting, extracting context around matched keywords.

#### Scenario: 笔记长度未超阈值
- **WHEN** 笔记内容长度 ≤ `ai_ref_max_chars`
- **THEN** 全文注入（行为不变）

#### Scenario: 笔记长度超过阈值
- **WHEN** 笔记内容长度 > `ai_ref_max_chars` 且有匹配的关键词
- **THEN** 在内容中定位第一个匹配的关键词位置
- **AND** 截取该关键词前后各 200 字（共约 400 字 + 关键词长度）作为注入内容
- **AND** 在截取片段前后添加 `...(内容已截断)` 标记

#### Scenario: 超过阈值但无法定位关键词
- **WHEN** 笔记内容长度 > `ai_ref_max_chars` 但未在内容中找到任何关键词（SQL LIKE 匹配与 2-gram 不完全一致）
- **THEN** 从头截取 `ai_ref_max_chars` 字作为注入内容
- **AND** 在末尾添加 `...(内容已截断)` 标记

#### Scenario: 截取位置在边界
- **WHEN** 关键词位置靠近笔记开头（< 200 字）
- **THEN** 从笔记开头开始截取，不填充空白
- **WHEN** 关键词位置靠近笔记末尾（距末尾 < 200 字）
- **THEN** 截取到笔记末尾

## MODIFIED Requirements

### Requirement: CardRecallSearch 参数

`CardRecallSearch(ctx, query string, limit int, noteService)` 变更为 `CardRecallSearch(ctx, query string, limit int, maxChars int, noteService)`。

新增 `maxChars int` 参数，表示单条笔记的最大字符数阈值：
- 笔记长度 ≤ maxChars → 全文注入
- 笔记长度 > maxChars → 截取上下文

### Requirement: 调用方传值

`app.go` 中调用 `CardRecallSearch` 前，从 `settingService` 动态读取 `ai_ref_max_chars`：
- 从 SettingService 按 key `"ai_ref_max_chars"` 取值
- 取值失败或空值时使用默认值 5000
- 将读取到的值作为 `maxChars` 参数传入 `CardRecallSearch`

### Requirement: 输出标记

`RecallCard` 结构体新增 `Truncated bool` 字段，标记该卡片内容是否被截断。前端召回卡片折叠面板可在卡片标题旁显示截断标识。

## ADDED Code

### 上下文截取算法

在 `recall_service.go` 中新增 `extractContext(content string, keywords []string, contextChars int) string` 函数：

```
输入：笔记全文、召回关键词列表、上下文窗口（200 字）
输出：截取后的文本片段

1. 遍历 keywords，在 content 中查找每个 keyword 首次出现位置
2. 取第一个找到的位置 pos
3. 计算 start = max(0, pos - contextChars)
4. 计算 end = min(len(content), pos + len(keyword) + contextChars)
5. 截取 content[start:end]
6. 如果 start > 0，前面加 "...(内容已截断)"
7. 如果 end < len(content)，后面加 "...(内容已截断)"
8. 返回截取结果
```
