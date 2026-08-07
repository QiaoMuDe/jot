# 设置页 API 连接问题修复 Spec

## Why

设置页「API 链接」部分（对话连接 + 量化连接）经检查发现 6 组问题：空 URL 可静默保存、斜杠修正后不自动保存、预设切换小不对称、召回处 kwCards 死代码、量化任务校验强度不一致、多处后端重复实现。均为非阻塞缺陷，统一修复提升健壮性与代码质量。

## What Changes

1. **URL 保存校验补全**（main.js `initApiConnectionModule`）：URL `change` 分支补空值拦截（空 URL 不保存 + 提示）；`input` 事件在从「斜杠错误态」恢复时立即自动保存一次，消除"修正后需再次失焦"。
2. **预设切换对齐**（main.js `switchProfile`）：补隐藏模型下拉搜索框，与 `switchProfileEmbed` 行为一致。
3. **移除 kwCards 死代码**（app.go `CallAIStream`）：删除恒空的 `kwCards` 拼接段，直接使用 `vectorResult.Cards`；保留 `recallCardsJSON` 赋值（L2263 存 DB 用）与 `MergeRecallCards` 函数本身（无其他调用者但作为多路合并工具保留）。
4. **startVectorIndex 校验对齐**（app.go）：复用 `GetEmbedConfig()` 读取四键，并补 provider/base_url 空值校验，与 `ValidateCardRecall` 强度一致。
5. **合并测试连通性重复实现**（app.go）：`TestAIBaseURL` 与 `TestAIConnection` 抽公共内部函数，两个 Wails 绑定方法保留签名做薄封装。
6. **抽取 HTTP 请求公共函数**（ai_service.go）：openai/ollama 的 test/fetch 共用统一 `httpGetJSON` 请求辅助，消除 4 组重复 HTTP 模板。

## Impact

- Affected specs: AI 设置（对话/量化连接）、API 配置预设、卡片召回
- Affected code: `frontend/src/main.js`、`app.go`、`internal/services/ai_service.go`

## ADDED Requirements

### Requirement: URL 输入自动保存校验

系统 SHALL 在 Base URL 输入框 `change` 时：以 `/` 结尾 → 报错不保存（现状保留）；**为空 → 提示「请先填写 API 地址」且不保存**。

#### Scenario: 清空 URL 后失焦

- **WHEN** 用户清空 Base URL 输入框并失焦（触发 `change`）
- **THEN** 不执行保存，Toast 提示「请先填写 API 地址」
- **AND** 后端配置保持原值不变

### Requirement: 斜杠错误修正后自动保存

系统 SHALL 在 Base URL 输入框从「斜杠结尾错误态」恢复为非斜杠结尾时，自动保存一次并移除错误样式。

#### Scenario: 修正斜杠后不再次失焦

- **WHEN** 用户将 `https://x/v1/` 修正为 `https://x/v1`（输入框处于 `input-error` 状态）
- **THEN** 错误样式移除，且立即自动保存（Toast「AI 配置已保存」）
- **AND** 用户直接关闭设置页也不会丢失修改

### Requirement: 预设切换后模型搜索框隐藏对齐

系统 SHALL 在 `switchProfile`（对话预设切换）后隐藏模型下拉搜索框，与 `switchProfileEmbed` 一致。

#### Scenario: 切换对话预设

- **WHEN** 用户从预设下拉选择任一对话预设
- **THEN** 模型下拉清空、label 重置为占位符，且搜索框隐藏（`display: none`）

### Requirement: startVectorIndex 配置校验对齐

系统 SHALL 在 `startVectorIndex` 启动量化前校验量化 provider/base_url/model 三者齐全，缺失时返回可读错误且不发起任务。

#### Scenario: 仅配置了模型未配置地址

- **WHEN** 用户设置了 `ai_embed_model` 但 `ai_embed_base_url` 为空时发起量化
- **THEN** 任务不启动，返回「请先在设置中配置量化连接与量化模型」

## MODIFIED Requirements

### Requirement: 召回卡片合并逻辑

移除关键词召回死代码，卡片合并直接从向量召回结果 `vectorResult.Cards` 生成；`recallCardsJSON` 继续用于消息 DB 持久化，`ai:recall-cards` 事件发射与 `TruncateRecallCardsPreview(200)` 截断行为不变。

### Requirement: 连通性测试与模型获取

`TestAIBaseURL`/`TestAIConnection` 保留对外签名与各自日志前缀，内部共用同一测试实现；openai/ollama 的 test/fetch 共用统一 HTTP 请求辅助，行为（端点、超时、状态码判断）不变。

## REMOVED Requirements

### Requirement: kwCards 关键词卡片合并

**Reason**: 关键词召回已移除，`kwCards` 恒为空切片，合并无实际效果。
**Migration**: 直接使用 `vectorResult.Cards`；`MergeRecallCards` 函数保留（多路合并工具，无其他调用者）。
