# 卡片召回三问题修复计划

## Why

AI 助手对话框的向量召回（替代关键词召回后）存在三个逻辑问题：

1. **状态事件语义冲突**：召回复用了联网搜索的状态通道 `ai:search-status`。只开卡片召回、不开联网搜索时，后端因 `cardRecallEnabled` 发射 `refining` 事件，前端显示「正在优化输入...」（地球图标，搜索专用 UI），与实际执行的向量召回无关，纯视觉误导；召回耗时期间用户一直看着无关动画。
2. **召回失败静默降级**：`VectorRecall` 所有失败路径统一返回 `nil`，前端不注入、不发射、无提示。开关开着但运行中 embedding/SQL 出错（服务故障、模型切换维度不匹配等）时，用户毫无感知（此前 JOIN 语法错误正是这样被吞掉、只能翻日志）。
3. **笔记本范围语义冲突**：开关开启但用户取消全部笔记本勾选 → `recallNotebookIds` 空集 → 后端无过滤 → **全库召回**。「全不选」与「全选」范围相同，与 UI 预期（不选 = 不召回）冲突。

## Current State Analysis

- 后端召回段：[app.go L1867-L1869](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L1867-L1869) `if len(searchSources) > 0 || cardRecallEnabled { searching = true; emit "ai:search-status" "refining" }`；[L2117](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2117) `if searching { emit "done" }`；[L2124-L2172](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2124-L2172) 召回执行段（`VectorRecall` → `if vectorResult != nil` 注入 + 发射 `ai:recall-cards`）
- `VectorRecall`：[vector_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go) 返回 `(*CardRecallResult)`，共 9 个返回点：预期跳过（query 空/limit<=0、embedClient nil、无向量数据、无命中、卡片空）与意外错误（embedding 失败、SQL 失败、查笔记失败、查块失败）混在一起全部返回 `nil`
- 前端开关/下拉：[ai-chat.js L869-L918](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L869-L918) 开启时默认全选、关闭时清空集合；空集合时 `Array.from(recallNotebookIds)` 传空数组 → 后端全库
- 前端事件通道：[ai-chat.js L2157](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2157) `ai:search-status` 监听（refining/searching/done 三段替换消息区）；`isStreaming` 在 [L36](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L36) 定义，用于通知守卫

## Proposed Changes

### 1. 后端 `VectorRecall` 签名改为返回错误（vector_service.go）

`func (s *VectorService) VectorRecall(...) (*CardRecallResult, error)`，按严重度分类：

| 场景 | 返回 |
|------|------|
| query 空 / limit<=0 / embedClient nil / 无向量数据 / 无命中 / 卡片空 | `(nil, nil)` — 预期跳过 |
| embedding/序列化失败 / SQL 检索失败 / 查笔记失败 / 查块失败 | `(nil, err)` — 意外错误 |

**Why**：让调用方区分「配置未就绪/无内容」（可静默）与「运行中出错」（应提示），是修复静默降级的前提。
**How**：逐个返回点改造，错误用 `fmt.Errorf` 包装保留现场（沿用现有 `s.logger.Errorw` 日志）。

### 2. 后端召回段脱离搜索状态机 + 空集跳过 + 错误事件（app.go）

- **L1867**：`if len(searchSources) > 0 || cardRecallEnabled` → `if len(searchSources) > 0`。召回不再进入搜索状态机，消除「正在优化输入...」误导。
- **L2124 召回段**：
  - `len(recallNotebookIDs) == 0` → 跳过召回（`Logger.Debugw` 记录），不发射任何事件。
  - 非空 → 发射 `ai:recall-status "searching"` → `VectorRecall`：
    - `err != nil` → 发射 `ai:recall-status "error", err.Error()`
    - `vectorResult == nil`（预期跳过）→ 发射 `ai:recall-status "done"`（无卡片）
    - 成功 → 注入 system message + 发射 `ai:recall-cards`（现有逻辑保留）+ 发射 `ai:recall-status "done"`
- 保留现有 `TruncateRecallCardsPreview` 截断与 DB 全量存储逻辑。

**Why**：召回有独立事件通道（`ai:recall-status`），状态语义干净；空集按用户预期「不召回」；错误经事件通道直达前端提示。

### 3. 前端新增 `ai:recall-status` 事件处理（ai-chat.js）

- **L2157 EventsOff 列表**：加入 `'ai:recall-status'`（跟随现有发送前重新注册的隔离模式，不带 streamGen，与 `ai:search-status`/`ai:recall-cards` 一致）。
- **注册新监听** `window.runtime.EventsOn('ai:recall-status', (status, detail) => {...})`：
  - `searching`：消息区当前为打字点（无搜索指示器、未收到 chunk）时，替换为召回指示器「正在检索本地笔记...」
  - `done`：恢复打字点（`createTypingDots()`）
  - `error`：恢复打字点；`isStreaming` 为 true 时 `showNotification('卡片召回失败: ' + detail, 'warning')`
- **新增 `createRecallIndicator()`**：复用 NOTE_ICON（书图标）+ 文案「正在检索本地笔记...」，容器复用打字点布局（保持消息区高度稳定），最小 CSS 改动。

**Why**：召回期间用户有真实反馈（不再看无关搜索动画）；运行中失败有明确提示（解决静默降级）。

## Assumptions & Decisions

- **空集语义**：空笔记本集合 = 不召回（非全库）。前端开启时默认全选、用户主动全取消说明不想召回该会话，此语义最符合直觉。
- **预期跳过静默**：未配置/无向量数据/无命中视为正常态，不打扰用户；仅意外错误提示。
- **事件不带 streamGen**：与 `ai:search-status`、`ai:recall-cards` 现有模式一致，靠发送前重新注册隔离；召回事件发生在 LLM 流开始前，竞争窗口极小。
- **不做**：D4/D5 展示不一致（历史消息卡片全量 vs 预览、面板位置）、死字段 `Truncated` 清理——不在本次范围。

## Verification

1. `go build ./...` + `golangci-lint run ./...` 通过
2. `npm run build` 通过（前端产物更新）
3. 手动验证（`wails dev`）：
   - 只开卡片召回、不开联网搜索 → 消息区显示「正在检索本地笔记...」，**不再出现**「正在优化输入...」
   - 召回成功后恢复打字点并正常输出，消息末尾带召回卡片
   - 全取消笔记本勾选（开关保持开）→ 发送消息：不召回任何内容（后端 Debug 日志「未选择笔记本」）
   - 人为断开 embedding 服务 → 发送消息：弹出「卡片召回失败: ...」通知，回复正常继续
   - 联网搜索 + 召回同时开启 → 搜索动画流程不变，召回指示器在搜索完成后短暂显示
