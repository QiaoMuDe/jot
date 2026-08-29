# AI 助手 Chat 模式（精简问答模式）回归实施计划

> 性质：实施计划文档（含逐文件改动清单），评审通过后按此执行。
> 背景：之前因功能迭代移除 Chat 模式，AI 助手现为 Agent-only。痛点：① 很多模型不支持深度思考（`reasoning_effort=high` 可能被拒）或工具调用（Plan 模式必然失败，普通模式指令臃肿）；② 本地模型跑 ReAct 多轮循环慢/易超时。故恢复"单次请求、不调用工具"的精简 Chat 模式。

***

## 1. 目标与边界

### 1.1 目标

* 恢复 Chat（问答）模式：**单次流式调用、不注册任何工具、不加载 MCP**，任何 OpenAI 兼容模型（含本地 Ollama/LM Studio）都能快速出结果。

* 模式切换 UI：三段控件 **Chat / Agent / Plan**（用户已确认）。

* Chat 模式**不注入任何工具使用规范**（用户已确认），仅保留身份层 + 技能/角色扮演/引用/上传/追问上下文。

### 1.2 明确不做（与旧 Chat 模式的差异）

* **不恢复**联网搜索状态机（搜索已 MCP 化）、卡片召回编排（走 Agent 的 `recall_notes`）、搜索源/召回开关 UI、`ai:search-*`/`ai:recall-*`/`ai:refined-*` 事件。

* **不动** Agent/Plan 现有逻辑与数据，零回归。

* **不做**模型能力自动检测/降级（方案 B 被否，见讨论记录）。

***

## 2. 现状分析（关键事实）

| 层    | 现状                                                                                                                                                                                                          | 复用点                                                                                             |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| 后端入口 | [CallAIAgentStream](file:///d:/峡谷/Dev/本地项目/jot/app.go#L1978) 是唯一流式入口                                                                                                                                        | 其 Instruction 组装、截断、落库、token 统计、取消处理逻辑全部可复用                                                     |
| 服务层  | [AIService.CallAIStream](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L362) **仍存在**（无工具单次流式，`enable_thinking` 方式，非思考模型安全）                                                                     | 被编辑器写作 [AITextOperationStream](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2418) 使用，是**新 Chat 流包装的直接模板** |
| 客户端  | [einocli Stream](file:///d:/峡谷/Dev/本地项目/jot/internal/einocli/chat.go#L83-L166)：OnChunk/OnThinking/OnDone(content, thinkingElapsed, totalElapsed)/OnError                                                    | 直接可用；OnThinking 逐块推送 `reasoning_content`                                                        |
| 前端   | [ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2323-L2324) `isAgentFlow = true` 硬编码；渲染（气泡/thinking/done/error/历史回放）全共享；工具/计划事件 handler 已有 `if (!isAgentFlow) return` 守卫                | 恢复双分支即可，渲染层零改动                                                                                  |
| UI   | [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1192-L1197) `#aiModeToggle` 现为 Agent/Plan 两按钮 + 分割线；[CSS](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1195-L1233) 样式通用 | 加一个 Chat 按钮 + 一条分割线即可，CSS 无需改                                                                   |
| 数据   | [AISessionConfig](file:///d:/峡谷/Dev/本地项目/jot/internal/models/ai_session_config.go) 只有 `PlanMode`，无 chat 标志；`agent_enabled` 已删                                                                               | 新增 `ChatMode bool` 字段（GORM AutoMigrate 加列，默认 false，零迁移）                                         |

**关键事件契约**：前端 `ai:stream-done` handler（[L2751](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2751)）期望参数 `(gen, content, elapsedThinking, elapsedTotal, totalTokens, userTokens, assistantTokens, userMsgID, assistantMsgID)`；`assistantMsgID=0` 表示"已取消不落库"。新 Chat 流必须发射**相同形状**，前端才能零改动复用。

***

## 3. 变更方案（按文件）

### 3.1 数据层

**文件：`internal/models/ai_session_config.go`**

* `AISessionConfig` 增加字段：`ChatMode bool \`gorm:"default:false" json:"chat\_mode"\`\`

**文件：`internal/services/ai_service.go`**

* `SessionConfig` struct（L46-54）增加：`ChatMode bool \`json:"chat\_mode"\`\`

* `SaveSessionConfig`（L552）Assign map 增加 `"chat_mode": cfg.ChatMode`

* `LoadSessionConfig`（L586-594）返回结构增加 `ChatMode: record.ChatMode`

* `LoadSessionConfig` 空值兜底返回（L578-584）可不加（零值 false 即正确）

> 模式解析规则：`ChatMode=true` → chat；否则 `PlanMode=true` → plan；否则 agent。三态互斥由前端保证（见 3.3）。

### 3.2 后端：新增 Chat 流包装

**文件：`app.go`**

**(a) 提取共享上下文组装 helper（纯抽取，不改行为）**

将 `CallAIAgentStream` 内 L2004-2091（基础提示词 + 角色扮演 + 笔记引用 + 追问 + 上传文件 + 技能注入）抽取为：

```go
// buildAIContextInstruction 组装基础问答上下文（身份层 + 技能/角色扮演/引用/追问/上传文件）。
// 不含任何工具使用规范（Agent 模式在其后追加，Chat 模式直接用）。
func (a *App) buildAIContextInstruction(skillIds []string, roleplayNoteIDs, referencedNoteIDs []uint, followUpRefContent string, uploadedFiles []AIChatFileResult) string
```

* 返回 `instruction strings.Builder` 的最终字符串；`CallAIAgentStream` 改为调用它后再追加 4 个工具规范块（L2093-2114 原样保留）。

* 纯抽取，Agent 行为零变化。

**(b) 新增** **`CallAIStream`（Chat 流式入口）**

```go
// CallAIStream Chat 模式流式对话绑定方法（单次请求、不调用工具）。
// 复用 truncateAIMessages + buildAIContextInstruction；走 einocli 流式（enable_thinking 方式，
// 非思考模型安全）。事件与 Agent 流同形：ai:stream-chunk / ai:stream-thinking / ai:stream-done / ai:stream-error。
func (a *App) CallAIStream(streamGen int, sessionID uint, userText string, thinkingEnabled bool, skillIds []string, referencedNoteIDs []uint, roleplayNoteIDs []uint, followUpRefContent string, uploadedFiles []AIChatFileResult, userMsgID uint)
```

实现步骤（参考 `CallAIAgentStream` 与 `AITextOperationStream` 混合模板）：

1. `ctx, cancel := context.WithCancel(context.Background())`；`a.aiStreamCancel = cancel`（与 Agent 共用取消源，`CancelAIStream` 直接生效）。
2. `messages := a.truncateAIMessages(ctx, sessionID, "AI Chat 滑动窗口截断")`。
3. `userMsgID == 0` 时倒序找回末条 user 消息 ID（同 Agent L1989-1996）。
4. goroutine 内：

   * `instruction := a.buildAIContextInstruction(...)`（**不追加任何工具规范块**）。

   * 组装 `[]services.Message`：`{Role:"system", Content:instruction}` + 历史 user/assistant（跳过 system）+ 当前 `userText`（**去重**：若历史末条 user 内容 == userText 则跳过，复用 `buildMessages` 的去重逻辑）。

   * 取消预检：`if a.handleAICancelled(ctx, sessionID, userMsgID, messages, streamGen) { return }`。

   * `a.aiService.CallAIStream(ctx, msgs, thinkingEnabled, ...)`：

     * `OnChunk` → `ai:stream-chunk`(gen, chunk)

     * `OnThinking` → 追加到本地 `reasoningBuf` + `ai:stream-thinking`(gen, chunk)

     * `OnDone(content, thinkingElapsed, totalElapsed)` → 落库 + 事件（见下）

     * `OnError(errMsg)` → 若 `ctx.Err()!=nil`（取消）走 `handleAICancelled`；否则 `ai:stream-error`(gen, errMsg, estimateUserTokens(messages))

   * `OnDone` 落库逻辑（与 Agent L2191-2242 对齐）：

     * `assistantTokens = estimateTokens(content)`；若 thinking 开启且 `reasoningBuf` 非空则加 `estimateTokens(reasoningBuf)`；`userTokens = estimateUserTokens(messages)`；`totalTokens = userTokens + assistantTokens`。

     * `SaveAIMessage(sessionID, services.Message{Role:"assistant", Content, ReasoningContent:reasoningBuf.String(), ThinkingElapsed, TotalElapsed, Tokens:assistantTokens})`。

     * `UpdateAIMessageTokens(userMsgID, userTokens)`；`SumSessionTokens`；`UpdateSessionContextTokens`。

     * 发射 `ai:stream-done`(gen, content, thinkingElapsed, totalElapsed, totalTokens, userTokens, assistantTokens, userMsgID, assistantMsgID)。

   * 兜底：若 `ctx.Err()!=nil` 且 OnDone/OnError 均未触发，走 `handleAICancelled`（同 `AITextOperationStream` 的兜底思路）。

> token 说明：einocli 流式回调不返回真实 usage，Chat 走估算（与旧 Chat 一致，可接受）。

### 3.3 前端

**文件：`frontend/index.html`（L1192-1197）**

`#aiModeToggle` 改为三按钮两分割线：

```html
<div id="aiModeToggle" class="ai-mode-toggle">
    <button class="ai-mode-btn active" data-mode="chat">Chat</button>
    <span class="ai-mode-divider"></span>
    <button class="ai-mode-btn" data-mode="agent">Agent</button>
    <span class="ai-mode-divider"></span>
    <button class="ai-mode-btn" data-mode="plan">Plan</button>
</div>
```

**文件：`frontend/src/js/ai-chat.js`**

1. **状态**（L42 附近）：新增 `let currentChatMode = false;`（false = 非 Chat）。保留 `currentPlanMode`，两标志互斥（Chat 与 Plan 不同时 true）。

2. **模式切换绑定**（L507-523）：泛化为三态逻辑：

   * 点击 `chat`：`currentChatMode=true; currentPlanMode=false` → `saveCurrentMode()`

   * 点击 `plan`：`currentPlanMode=true; currentChatMode=false` → `saveCurrentMode()`

   * 点击 `agent`：两标志均 false → `saveCurrentMode()`

   * 均先做 `isStreaming` 锁定检查（shake + 提示，保持现状）

3. **`syncModeToggle`**（L5856-5861）active 判定改为：
   `btn.dataset.mode === (currentChatMode ? 'chat' : (currentPlanMode ? 'plan' : 'agent'))`

4. **`saveCurrentPlanMode`**（L5866-5875）替换/泛化为 **`saveCurrentMode()`**：

   ```js
   const cfg = await window.go.main.App.LoadSessionConfig(activeSessionId);
   cfg.plan_mode = currentPlanMode;
   cfg.chat_mode = currentChatMode;
   await window.go.main.App.SaveSessionConfig(activeSessionId, cfg);
   ```

   更新全部调用点：L519（绑定内）、L2868（Plan 执行完毕自动回退 Agent 时——保持"回退 Agent"语义：`currentChatMode=false; currentPlanMode=false`）。

5. **会话恢复**：

   * `switchSession`（L1540-1542）追加 `currentChatMode = !!config.chat_mode;`

   * 新建会话默认配置（L1716-1718）追加 `currentChatMode = !!defaultCfg.chat_mode;`

   * 两处均在 `syncModeToggle()` 之前赋值。

6. **`startStreaming`**：

   * L2324：`const isAgentFlow = true;` 改为

     ```js
     const isChatFlow = currentChatMode;
     const isAgentFlow = !isChatFlow;
     ```

   * 事件监听注册（L2328）与各 handler 无需改（工具/计划 handler 已有 `isAgentFlow` 守卫，Chat 流不触发这些事件）。

   * L2953 调用处分流：

     ```js
     if (isChatFlow) {
         window.go.main.App.CallAIStream(myGen, activeSessionId, userText, enableThinking, skillIds, refNoteIDs, roleNoteIDs, followUpRef, uploadedFiles, userMsgID);
     } else {
         window.go.main.App.CallAIAgentStream(myGen, activeSessionId, userText, enableThinking, skillIds, refNoteIDs, roleNoteIDs, followUpRef, uploadedFiles, Array.from(recallNotebookIds), userMsgID);
     }
     ```

7. **`saveCurrentSessionConfig`**（L5881-5893）不需要改（mode 由 `saveCurrentMode` 单独维护，与 plan 现状一致）。

> `stream-done` handler（L2751-2883）零改动：Chat 后端发射同形事件；`currentPlanMode` 在 Chat 流恒 false，Plan 自动回退块（L2864-2870）天然 no-op。

**文件：`frontend/src/css/components/ai-chat.css`**

* 预计无需改动（`.ai-mode-btn` 通用样式支持任意数量按钮）。执行时目检三按钮是否拥挤，如挤则微调 `.ai-mode-btn` 的 padding（可选，非必须）。

### 3.4 绑定生成

**文件：`frontend/wailsjs/go/main/App.js`** **/** **`App.d.ts`** **/** **`models.ts`**

* 后端新增 `CallAIStream` 后执行 `wails generate module` 重新生成（`models.ts` 同步加 `chat_mode` 字段）。

* 注意：`CallAIStream` 与旧版同名同参（除去掉 `recallNotebookIDs`），旧绑定已删，无冲突。

***

## 4. 假设与决策（已确认）

| #  | 决策                      | 说明                                                      |
| -- | ----------------------- | ------------------------------------------------------- |
| D1 | 三段切换 Chat/Agent/Plan    | 用户已选；会话级持久化 `chat_mode`                                 |
| D2 | Chat 不注入任何工具规范          | 用户已选；只注入身份层 + 技能/引用/角色扮演/上传/追问上下文                       |
| D3 | Chat 不带搜索/召回编排          | 搜索已 MCP 化，召回归 Agent `recall_notes`                      |
| D4 | 模式默认值                   | 新会话默认 Agent（向后兼容存量数据）；`ChatMode` 零值 false               |
| D5 | 深度思考走 `enable_thinking` | 复用 einocli 方式（编辑器写作同款），非思考模型安全忽略                        |
| D6 | Chat 不新增/不删除任何 Wails 事件 | 复用 `ai:stream-*` 四事件，形状与 Agent 一致                       |
| D7 | 取消语义                    | 复用 `a.aiStreamCancel` + `handleAICancelled`，停止按钮对两模式均生效 |

***

## 5. 验证步骤

1. **编译**：`go build ./...` 通过。
2. **绑定**：`wails generate module` 后检查 `App.d.ts` 含 `CallAIStream`、`models.ts` 含 `chat_mode`。
3. **前端构建**：`npm run build`（或 vite dev）无错误。
4. **手动冒烟（本地模型优先）**：

   * Chat 模式 + 本地模型（Ollama）：单次回答、无工具条、无计划面板；深度思考开关切换不报错。

   * Chat 模式 + 技能（翻译/角色扮演）/引用笔记/上传文件/追问：上下文正确注入，回答正常。

   * 停止按钮：Chat 流即时取消，气泡移除、不落库（`stream-done` assistantMsgID=0 路径）。

   * 重新生成/编辑/重发：Chat 流走 `userMsgID=0` 找回逻辑正常。

   * 会话切换/新建：模式正确恢复（chat/plan/agent），Plan 执行完毕自动回退 Agent 正常。

   * 回归 Agent 模式：`recall_notes` 召回、MCP 工具调用、Plan 模式、ask\_user 反问、优化表达全部与改动前一致。

   * 流中锁定：Chat 流期间模式切换按钮锁定（置灰 + 抖动提示）。

***

## 6. 风险与缓解

| 风险                                           | 缓解                                          |
| -------------------------------------------- | ------------------------------------------- |
| 提取 `buildAIContextInstruction` 影响 Agent 指令组装 | 纯抽取 + 全量回归 Agent 冒烟；若存疑可回退为"在 Chat 函数内复制该段" |
| 前端 `currentChatMode`/`currentPlanMode` 互斥被破坏 | 所有切换收敛到绑定内一处 + `saveCurrentMode` 统一落库       |
| wails 绑定与 app.go 不一致                         | 改动后立即 `wails generate module`               |
| 存量会话无 `chat_mode` 列                          | GORM AutoMigrate 加列默认 false；无数据迁移           |
| Chat 流经 `truncateAIMessages` 可能触发摘要生成        | 与 Agent 共用逻辑，行为一致，可接受                       |

***

## 7. 实施顺序

1. 数据层：`models/ai_session_config.go` + `services/ai_service.go`（SessionConfig/Save/Load）
2. 后端：`app.go` 提取 helper → 新增 `CallAIStream` → `wails generate module`
3. 前端：`index.html` 三按钮 → `ai-chat.js`（状态/绑定/syncModeToggle/saveCurrentMode/会话恢复/startStreaming 分流）
4. 编译 + 冒烟（按 §5）

