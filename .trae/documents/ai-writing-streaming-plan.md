# AI 写作流式输出改造 — 实施计划

> 目标：将编辑器「操作」菜单中 AI 写作分组（润色/续写/扩写/缩写/校对/改写/翻译）从"一次性等待输出"改为"流式输出"，实现打字机式实时增量替换，且 Ctrl+Z 一步还原原文、中途取消恢复原文。
> 涉及模块：后端 `app.go`、前端 `editor-actions.js`、`ai-writing.js`、wailsjs 绑定重新生成。
> 无新依赖、无数据库改动。

***

## 一、现状分析

### 1.1 当前实现（一次性调用）

* **后端** [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2874-L2922) `AITextOperation(text, operation) (string, error)`：

  * 校验 AI 配置 → 按 operation 构造 system prompt → 构造 messages → 调 `a.aiService.CallAI`（einocli **非流式** `Generate`）→ 一次性返回全文。

  * 60s 超时 + 复用 `a.aiStreamCancel` 支持取消。

* **前端** [ai-writing.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions/ai-writing.js)：8 个操作项，`handler` 均为 `await window.go.main.App.AITextOperation(text, op)`。

* **执行引擎** [editor-actions.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions.js#L189-L283) `executeAction`：

  * `await handler(sourceText)` 拿到全文 → 一次性 `dispatch` 替换选区。

  * 期间：脉冲圆球指示器 + 锁定编辑器（`setAIEditorLock`）+ 禁用按钮；点击圆球 → `CancelAIStream()`。

  * AI 操作强制要求有选中文本（`actionType === 'ai' && !hasSelection` 提示返回）。

### 1.2 流式基础设施（已就绪，可复用）

| 层      | 组件                                                                                                                              | 位置                                                                                |
| ------ | ------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| 流式调用   | einocli `Client.Stream`（回调 OnChunk/OnThinking/OnDone/OnError）                                                                   | [chat.go](file:///d:/峡谷/Dev/本地项目/jot/internal/einocli/chat.go#L83-L166)           |
| 服务封装   | `AIService.CallAIStream(ctx, messages, thinkingEnabled, onChunk, onThinking, onDone, onError)`                                  | [ai\_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L233) |
| 事件推送范式 | `CallAIStream` 内 goroutine + `runtime.EventsEmit("ai:stream-chunk", streamGen, chunk)`；取消兜底 `if ctx.Err() != nil { emit done }` | [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2293-L2379)                           |
| 前端事件范式 | `EventsOn` 注册（回调首参判 `streamGen !== myGen` 丢弃旧流）+ `EventsOff` 逐个清理                                                               | [ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2375-L2810)   |
| 取消     | `a.aiStreamCancel` + `CancelAIStream()`                                                                                         | [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2864-L2870)                           |

### 1.3 CM6 undo 机制（方案根基，已读源码验证）

项目使用 `@codemirror/commands` v6.10.3 的 `history()`（[main.js L12/L85](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L12)）：

* `Transaction.addToHistory === false` 的事务不会进历史，但会触发 `state.addMapping(tr.changes.desc)`，把该次变更的映射**累积到既有历史事件上**（[index.js L240-241](file:///d:/峡谷/Dev/本地项目/jot/frontend/node_modules/@codemirror/commands/dist/index.js#L240-L241)）。

* 因此：**第一条 chunk 用正常事务记录（形成 undo 锚点，反转即还原原文），后续 chunk 全部** **`addToHistory: false`**——undo 事件会被自动映射到最新文档内容，Ctrl+Z 仍精确还原操作前的原文。

* 自定义 `userEvent: 'ai.op'` 不匹配 `joinableUserEvent = /^(input\.type|delete)($|\.)/`，**不会**与用户操作前的输入合并成一条 undo，独立成事件。

* CM6 `readOnly` 只阻止用户输入，`dispatch` 程序化写回不受影响（[editor-actions.js L122](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions.js#L122) 注释已确认）。

***

## 二、目标行为（UX 规格）

1. 选中文本 → 触发 AI 写作操作 → 生成结果**逐块实时写入**编辑器替换选区（打字机效果）。
2. 操作期间保持现有锁定态：脉冲圆球指示器、编辑器只读、操作按钮禁用。
3. 点击圆球取消：**恢复原文**（丢弃全部已生成内容），不弹错误提示。
4. 生成完成：最终内容已全部写入，光标定位到结果末尾（选中范围保持选中态）。
5. 出错（API 配置缺失/鉴权失败/超时等）：**恢复原文** + 右上角通知错误信息。
6. 全程 Ctrl+Z **一步**还原原文；Ctrl+Shift+Z 重做（重做仅还原首块，属可接受边界）。
7. 与 AI 对话流互不干扰（独立事件命名空间、独立取消通道）。

***

## 三、事件协议（新增，与聊天流完全隔离）

| 事件              | 负载                         | 语义                                |
| --------------- | -------------------------- | --------------------------------- |
| `ai:aiop-chunk` | `(streamGen, chunk)`       | 生成增量块                             |
| `ai:aiop-done`  | `(streamGen, fullContent)` | 正常完成；取消时 fullContent 为空串          |
| `ai:aiop-error` | `(streamGen, errMsg)`      | 失败（errMsg 可能是 aierrors JSON 或纯文本） |

* 复用聊天流的 `streamGen` 关联机制：前端每次操作自增 `aiStreamGen`，回调首参不匹配即丢弃。

* 不用 `ai:stream-*` 事件名，避免与 ai-chat.js 的全局监听串台。

***

## 四、后端改动（app.go）

### 4.1 新增绑定 `AITextOperationStream(streamGen int, text string, operation string)`（无返回值，fire-and-forget）

替换旧的 `AITextOperation`（**删除**该绑定；`AIService.CallAI` 保留，`RefineSearchQuery` 仍在用）。

实现要点（对照 `CallAIStream` 外壳）：

1. **配置校验**：`a.aiService.GetConfig()` 三要素缺一 → 立即 `runtime.EventsEmit(a.ctx, "ai:aiop-error", streamGen, "请先配置 AI 服务（API 地址 / API Key / 模型）")` 并 return。
2. **prompt 构造**：把现有 `AITextOperation` 的 operation→prompt switch 抽为包级函数 `aiTextOpSystemPrompt(operation string) (string, error)`（`default` 分支返回 `fmt.Errorf("不支持的操作: %s", operation)` → 同样 emit `ai:aiop-error` 返回）。
3. **可取消 context**：`ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)`；存入**新字段** **`a.aiEditorCancel`**（不能复用 `a.aiStreamCancel`——可能与后台聊天流冲突，见 §五）。
4. **goroutine 内调流**：

   ```go
   go func() {
       a.aiService.CallAIStream(ctx, messages, false, // thinkingEnabled=false，写作不需要思维链
           func(chunk string) { runtime.EventsEmit(a.ctx, "ai:aiop-chunk", streamGen, chunk) },
           func(string) {}, // OnThinking 忽略
           func(content string, _, _ float64) { runtime.EventsEmit(a.ctx, "ai:aiop-done", streamGen, content) },
           func(err string) { runtime.EventsEmit(a.ctx, "ai:aiop-error", streamGen, err) },
       )
       // 兜底：取消/超时导致 OnDone/OnError 均未触发时
       if ctx.Err() != nil {
           if errors.Is(ctx.Err(), context.DeadlineExceeded) {
               runtime.EventsEmit(a.ctx, "ai:aiop-error", streamGen, "AI 处理超时，请重试")
           } else { // Canceled（用户点击圆球）
               runtime.EventsEmit(a.ctx, "ai:aiop-done", streamGen, "")
           }
       }
   }()
   ```

   > 注意：`CallAIStream` 的 `Stream` 在取消时走 `ctx.Err() != nil` 分支不回调 OnError，因此必须依赖此兜底发射终态事件。

### 4.2 新增字段与取消绑定

* App 结构体新增 `aiEditorCancel context.CancelFunc`（与 `aiStreamCancel` 并列）。

* 新增绑定 `CancelAIEditorOperation()`：`if a.aiEditorCancel != nil { a.aiEditorCancel(); a.aiEditorCancel = nil }`。

* `CancelAIStream()` **保持不变**（仅取消聊天流），避免编辑器圆球误杀后台聊天流。

***

## 五、前端改动

### 5.1 `editor-actions/ai-writing.js`：handler 改为 `op` 字段

* 8 个操作项删除 `handler`，新增 `op` 字段（`'polish'`/`'continue'`/`'expand'`/`'condense'`/`'proofread'`/`'rewrite'`/`'translate'`/`'translate-en'`），与后端 switch 分支一一对应。

* 保留 `group: 'AI 写作'`、`label`、`errorLabel: 'AI 操作'`、`type: 'ai'`。

### 5.2 `editor-actions.js`：新增流式执行引擎

**a. 模块级状态**

```js
let aiStreamGen = 0;          // 每次 AI 操作自增，关联事件
let aiStreamActive = false;   // 防重入
let _aiOpOriginalText = '';   // 操作前原文（取消/失败恢复用）
let _aiOpFrom = 0, _aiOpTo = 0;
```

**b. 新增** **`runAIStreamAction(op, text, from, to)`**

流程：

1. `aiStreamActive` 校验（true 则直接返回，按钮本就禁用，双保险）。
2. `aiStreamGen++`；快照 `_aiOpOriginalText = text; _aiOpFrom = from; _aiOpTo = to`。
3. `createAIStatusIndicator()` + `setAIEditorLock(true)` + 禁用按钮（复用现有代码）。
4. 注册事件（`EventsOn`，回调首参 `g !== aiStreamGen` 即丢弃）：

   * **chunk**：`acc += chunk`；首次 chunk 用 `dispatch({changes:{from,to,insert:acc}, selection:{anchor:from+acc.length}, userEvent:'ai.op'})`（**记入历史**），后续用同参数追加 `addToHistory: false`（**不入历史**）。

   * **done**：若 `_aiOperationCancelled` → 恢复原文；否则最后再 dispatch 一次 `{from,to,insert:fullContent}`（`addToHistory:false`，保证文档与最终内容一致，幂等）。

   * **error**：恢复原文 + 通知（见 §5.3）。

   * 三个终态分支统一调 `cleanupAIStream()`：`EventsOff` 逐个清除三个监听 → `removeAIStatusIndicator()` → `setAIEditorLock(_aiEditorWasReadOnly)` → 按钮恢复 → `aiStreamActive = false`。
5. 调用 `window.go.main.App.AITextOperationStream(text, op, aiStreamGen)`（**不 await**，结果走事件）。
6. 返回值：Promise（await 直到终态，供 `executeAction` 收尾）。实现用 `new Promise(resolve => { 终态分支内 resolve(); })`。

**c. 恢复原文辅助函数** **`restoreOriginal(silent)`**

`dispatch({changes: {from: _aiOpFrom, to: _aiOpTo, insert: _aiOpOriginalText}, selection: {anchor: _aiOpFrom, head: _aiOpTo}, addToHistory: false, userEvent: 'ai.op'})`。

* 恢复用 `addToHistory: false`：文档回到原文，历史栈中首块 undo 事件仍在；此时 Ctrl+Z 会应用"还原原文"的反转（对已是原文的文档无可见变化）——**可接受边界**，文档注明。

**d.** **`executeAction`** **的 AI 分支改造**

```js
if (actionType === 'ai') {
    await runAIStreamAction(action.op, sourceText, from, to);
    return; // 不再走通用的 dispatch 写回
}
```

同时：

* 点击处理（L322）改为 `executeAction(action, action.type)`，把整个 action 对象传入（内部取 `action.op`/`action.handler`）。

* 圆球点击（L167-173）：`CancelAIStream()` → `CancelAIEditorOperation()`。

* `_aiOperationCancelled` 标志沿用（点击圆球时置 true；取消后静默不通知）。注意取消时序：先置 true 再调后端取消，后端兜底发 `ai:aiop-done("")`，前端按取消分支恢复原文。

**e.** **`action.op`** **缺失防御**

`executeAction` 内 `if (actionType === 'ai' && !action.op)` → 通知"操作配置缺失"并 return（防止手误新增 AI 项漏配 op）。

### 5.3 错误通知解析

`ai:aiop-error` 的 errMsg 可能是 aierrors JSON（`{"category":...,"user_msg":...}`）或纯文本。新增小工具 `formatAIErrorMsg(errMsg)`：`errMsg` 以 `{` 开头时 `JSON.parse` 取 `user_msg`，失败/非 JSON 则原样返回。通知文案沿用 `AI 处理失败: ${msg}`。

### 5.4 wailsjs 绑定

`wails dev` / `wails build` 会自动重新生成 `frontend/wailsjs/go/main/App.js` 与 `App.d.ts`（删除旧 `AITextOperation`、新增 `AITextOperationStream`/`CancelAIEditorOperation`）。**无需手改** wailsjs 文件；若 dev 未自动刷新，手动执行 `wails generate module`。

***

## 六、边界与异常处理汇总

| 场景             | 处理                                                                                             |
| -------------- | ---------------------------------------------------------------------------------------------- |
| 未配置 AI 服务      | 后端绑定立即 emit `ai:aiop-error` → 前端恢复原文 + 通知                                                      |
| 不支持的 operation | 后端 prompt 构造返回 error → emit `ai:aiop-error`                                                    |
| 用户点圆球取消        | 置 `_aiOperationCancelled` → `CancelAIEditorOperation()` → 后端兜底发 `ai:aiop-done("")` → 前端恢复原文、静默 |
| 60s 超时         | 后端兜底分支（DeadlineExceeded）emit `ai:aiop-error("AI 处理超时，请重试")` → 恢复原文 + 通知                        |
| 流中网络/鉴权错误      | einocli OnError → `ai:aiop-error` → 恢复原文 + 通知                                                  |
| 后台聊天流并行        | 事件命名空间隔离 + 独立 `aiEditorCancel`，互不影响                                                            |
| Ctrl+Z 一步还原    | 首块记历史 + 后续 `addToHistory:false`（CM6 addMapping 机制自动追踪）                                         |
| 取消后 Ctrl+Z     | 对已是原文的文档无可见变化（可接受边界，不额外处理）                                                                     |

***

## 七、假设与决策记录

1. **实时增量替换**（用户已确认）：打字机效果直写编辑器；不用"末尾一次性替换"或"悬浮预览气泡"。
2. **取消恢复原文**（用户已确认）：丢弃全部已生成内容。
3. **删除旧** **`AITextOperation`** **绑定**：前端同步切流式后无引用，避免死代码；`AIService.CallAI` 保留（`RefineSearchQuery` 在用）。
4. **独立取消通道** `aiEditorCancel` + `CancelAIEditorOperation`：`CancelAIStream` 保持仅管聊天流，杜绝误杀。
5. **独立事件命名空间** `ai:aiop-*`：与 `ai:stream-*` 隔离，不串台。
6. **thinking 关闭**：写作操作不需要思维链。
7. 假设编辑器 AI 操作同一时刻最多一个（按钮禁用 + `aiStreamActive` 双保险）。

***

## 八、任务清单（实施顺序）

* [ ] T1 后端：app.go 新增 `aiEditorCancel` 字段；抽取 `aiTextOpSystemPrompt(operation)` 公共函数

* [ ] T2 后端：新增 `AITextOperationStream` 绑定（含配置校验、事件发射、取消/超时兜底）

* [ ] T3 后端：新增 `CancelAIEditorOperation` 绑定；删除旧 `AITextOperation`（先 grep 确认无其他引用）

* [ ] T4 前端：`ai-writing.js` 8 个操作项 handler → `op` 字段

* [ ] T5 前端：`editor-actions.js` 新增流式执行引擎（`runAIStreamAction`/`restoreOriginal`/`cleanupAIStream`/事件注册与增量 dispatch）

* [ ] T6 前端：`executeAction` AI 分支改流式 + 点击处理传 action 对象 + 圆球取消改 `CancelAIEditorOperation` + 错误消息解析

* [ ] T7 验证：`go build ./...`、`go vet ./...`、`cd frontend && npm run lint`

* [ ] T8 手动功能验证（见 §九）

* [ ] T9 更新 AGENTS.md「已实现」列表 + 记忆点（按维护规范）

***

## 九、验证步骤

1. **构建检查**：`go build ./...`、`go vet ./...`、`cd frontend && npm run lint`（0 error）。
2. **绑定生效**：`wails dev` 启动，确认 wailsjs 已含 `AITextOperationStream`/`CancelAIEditorOperation`。
3. **功能**：

   * 新建笔记，选中一段文本，逐个触发 8 个 AI 操作 → 观察**打字机式增量写入**、锁定态、完成后光标定位。

   * **Ctrl+Z 一次** → 恢复操作前原文；Ctrl+Shift+Z 重做。

   * **流中点击圆球取消** → 原文恢复、无错误弹窗、锁定解除。

   * **未配置 AI** 时触发 → 原文保持 + 通知提示。

   * **错误 Key** 触发 → 原文恢复 + 通知显示错误。
4. **并发隔离**：先在 AI 对话开启一条流式回复（后台生成），再切到编辑器触发 AI 写作 → 两边均正常；编辑器取消不影响对话流。
5. **回归**：格式化/文本转换/MD 语法等非 AI 操作不受影响；菜单渲染正常。

***

## 十、涉及文件清单

| 文件                                                                                                      | 改动                                                                                                                   |
| ------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go)                                                             | 新增 `aiEditorCancel` 字段、`aiTextOpSystemPrompt`、`AITextOperationStream`、`CancelAIEditorOperation`；删除 `AITextOperation` |
| [editor-actions/ai-writing.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions/ai-writing.js) | 8 项 handler → `op` 字段                                                                                                |
| [editor-actions.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/editor-actions.js)                       | 流式引擎 + executeAction AI 分支 + 取消/错误处理                                                                                 |
| frontend/wailsjs/go/main/App.js、App.d.ts                                                                | 自动重新生成（`wails dev`/`wails generate module`）                                                                          |
| [AGENTS.md](file:///d:/峡谷/Dev/本地项目/jot/AGENTS.md)                                                       | 已实现列表 + 记忆点更新                                                                                                        |

