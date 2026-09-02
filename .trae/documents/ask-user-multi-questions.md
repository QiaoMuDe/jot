# ask\_user 工具支持多问题（1-3 条）实施计划

## Summary

将 `ask_user` 工具从"一次只能问一个问题"升级为"一次可携带 1-3 个问题"。采用**单次调用携带** **`questions`** **数组**方案：后端 schema 改为数组、事件负载改数组；前端渲染多问题表单面板，用户逐题作答后一次性提交；等待机制（ClaimAsk / WaitForAnswer / AnswerAskUser / drainAsk）零改动——仍是一次抢占、一次阻塞、一个答案投递。

## Current State Analysis

* [ask\_user.go](../../../internal/agent/tools/ask_user.go)：单条 `question` + `options`(2-6) + `selection`(single/multiple)；`ClaimAsk()` 原子抢占后发射 `ai:ask-user` 事件并阻塞等待一个字符串答案

* [agent.go#L121-L232](../../../internal/agent/agent.go)：`askCh`（容量 1）+ `askPending` 互斥；`AnswerAskUser(sessionID, answer string)` 投递单字符串——**本方案不改此层**

* [ai-chat.js#L4218](../../../frontend/src/js/ai-chat.js)：`showAskPanel(question, options, selection)` 渲染单问题卡片；单选点击即发、多选勾选后按钮提交；经 `App.AnswerAskUser` 同轮投递

* [ai-chat.css#L3873](../../../frontend/src/css/components/ai-chat.css)：`.ai-ask-panel` 悬浮面板样式，无高度上限

* eino `ParameterInfo`（v0.9.13）支持 `ElemInfo`（数组）+ `SubParams`（对象子字段），`questions: [{question, options, selection}]` 可完整表达；但**不支持 minItems/maxItems**，1-3 上限需运行时校验

* 反问面板为瞬态 UI（不落库、历史不存卡片），无数据迁移问题

* 现有测试 [session\_test.go](../../../internal/agent/session_test.go) 只测会话层（不变），无 ask\_user 工具级测试

## Proposed Changes

### 1. 后端：internal/agent/tools/ask\_user.go（核心重写）

**参数 schema**：

* 新参数 `questions`：Array，ElemInfo 为 Object（SubParams）：

  * `question`（String，必填）：问题文本（问句形式，简洁明确）

  * `options`（Array of String，可选）：候选选项（2-6 项，可省略让用户自由输入）

  * `selection`（String，可选）：`single` | `multiple`，缺省 `single`

  * `questions` Desc 明确："1-3 条问题；相关联的信息尽量合并到一次提问；仅当问题相互独立时才拆为多条"

* 保留 `reason`（调试说明，可选）

* 工具 Desc 更新："一次可问 1-3 个问题，需一次性收集多项信息时合并为一次调用"，删除"一次只能问一个问题"

**新增常量**：`maxAskUserQuestions = 3`

**参数解析与兼容**：

```go
var args struct {
    Questions []struct {
        Question  string   `json:"question"`
        Options   []string `json:"options"`
        Selection string   `json:"selection"`
    } `json:"questions"`
    // 旧格式兜底：模型偶发仍发单条字段
    Question  string   `json:"question"`
    Options   []string `json:"options"`
    Selection string   `json:"selection"`
    Reason    string   `json:"reason"`
}
```

* `questions` 为空且旧 `question` 非空 → 包装为单条

* 两者皆空 → 返回错误 `"ask_user 参数缺少 questions"`

* 每条校验：question 去空 + `maxToolShortText` 长度；options 走现有 `normalizeAskUserOptions`（≤6 项、每项 ≤200 字符）；selection 非 `multiple` 视为 `single`

* 条数 >3 → 返回错误 `"ask_user 最多支持 3 个问题，请精简为 ≤3 条"`（错误回填模型自纠）

**事件负载**（新旧格式都发，前端双解析）：

```json
{"questions": [{"question":"...","options":[...],"selection":"single"}],
 "question": "首条问题", "options": [...], "selection": "首条 selection"}
```

顶层旧字段取第一条问题的值（兼容期间冗余，后续可移除）。

**ActionText**：解析 `questions`，多条返回 `"向用户提问（共N问）：{首问截断30字}"`，单条/解析失败回退现有行为。

**答案回填（关键设计）**：

* 前端提交格式约定：单问题 = 原始答案文本（与现状一致）；多问题 = 每题一行、行内不含换行（输入框单行、多选拼接为一行），即 `"答案1\n答案2\n答案3"`

* 后端 `strings.Split(answer, "\n")` 按行数与问题数配对：

  * 行数 == 问题数 → 工具结果逐题映射：`"用户已回答你的全部提问：\n1. 问题一\n   用户回答：xxx\n2. 问题二\n   用户回答：yyy。请结合你的问题与用户的回答继续完成用户的原始请求，直接给出最终回答或继续调用后续工具，不要重复提问。"`

  * 行数不匹配 → 整体作为单条答案返回并附提示（防御性兜底）

* 非 AskWaiter 路径（非交互/测试）：确认文案改为"我需要向你确认 N 个问题：…"

**文件头注释**：第 9 行"一次只能问一个问题，选项 2-6 个" → "一次可问 1-3 个问题（每题选项 2-6 个）"。

### 2. 后端文档同步

* [meta.go#L28](../../../internal/agent/tools/meta.go)：Label 改 `"向用户发起澄清提问（1-3 个问题，单选/多选）"`

* [TOOLS.md §4.7 L270](../../../internal/agent/TOOLS.md)：负载描述改 `{questions: [{question, options, selection}], ...}`

* [EVENTS.md §4](../../../internal/agent/EVENTS.md)：更新负载格式、面板描述（多问题表单、逐题作答一次提交）、答案投递格式约定

* 全局 grep `一次只能问一个问题` 清理残留

### 3. 前端：frontend/src/js/ai-chat.js

**事件处理器（\~L2930）**：

```js
// 双格式解析：{questions:[...]} 新格式优先；旧 {question, options, selection} 包装为单条
```

解析后统一为 `questions` 数组传入 `showAskPanel(questions)`。

**showAskPanel 重写**（签名改为 `showAskPanel(questions)`，questions 为 `[{question, options, selection}]`）：

* **单问题（length===1）**：完全保留现有 UI/交互（标题 = 问题、单选点击即发、多选勾选 + 按钮提交、底部输入行）

* **多问题（length≥2）**：

  * 标题行："请回答以下 N 个问题" + 右上角 × 关闭（取消本轮，逻辑不变）

  * 每题一个分组 `.ai-ask-qgroup`：编号题目标题（`1. 问题文本`）+ 该题选项区（按该题 selection 渲染单选/多选）+ 该题自定义输入框（placeholder "自定义答案（可选）"）

  * 单问题模式下的"单选点击即发"在多问题模式**不适用**：所有题统一为"选中后待全局提交"（单选=radio 语义互斥选中；多选=checkbox 勾选）

  * 底部唯一"确认提交"按钮：收集全部答案 → 校验每题已有答案（选项选中或自定义输入，二者取一；缺题时对该题输入框 shake 提示并终止）→ 按题序拼装 `"答案1\n答案2..."` 经 `doSend` 投递（复用现有 `AnswerAskUser` 通道与 `submitting` 防重逻辑）

* 多选拼装沿用现有格式："我选择：A、B"，自定义输入拼 "。补充说明：x"

* `hideAskPanel` / 关闭按钮 / 面板互斥逻辑不变

### 4. 前端：frontend/src/css/components/ai-chat.css

* `.ai-ask-panel` 增加 `max-height: 60vh; overflow-y: auto;`（3 题 × 6 选项防超屏）

* 新增 `.ai-ask-qgroup`：`padding-top/bottom + border-top: 1px solid var(--border)`（首组无上边框）、组间间距

* 新增 `.ai-ask-qtitle`：编号题目标题样式（复用 `.ai-ask-question` 的字重/颜色基调，稍小字号）

* 全部使用现有 CSS 变量（`--card-bg` / `--border` / `--text-primary` 等），保证 14 主题适配

## Assumptions & Decisions

1. **否决"并行多条 ask\_user 调用"方案**：需改 ClaimAsk 互斥语义、答案路由、多面板互覆，复杂度高收益低（已与用户对齐）
2. **答案按行拆分配对**：前后端同仓库同发版（wails 一起构建），行协议由本侧两端共同约定，输入框单行保证无换行歧义；配对失败有兜底
3. **保留旧格式兼容**：后端解析兼容单条 `question` 字段、事件负载冗余旧顶层字段、前端双格式解析——防模型偶发输出旧格式，成本低
4. **多问题模式下单选不即发**：改为统一全局提交，保证用户能答完全部问题
5. 会话层（agent.go）、app.go 绑定、landing 页（仅展示工具名徽章）均无需改动

## Verification

1. `go build ./...` 编译通过
2. `go test ./internal/agent/...` 全部通过（session\_test.go 不受影响）
3. `grep -r "一次只能问一个问题"` 无残留
4. `npm run build`（frontend）+ `wails build`（项目约定，前端资源需重新打包）
5. 手动验收：

   * Agent 模式触发需要 2-3 项信息的请求 → 面板渲染多问题分组、每题独立选项/输入

   * 逐题作答（单选互斥、多选勾选、自定义输入混合）→ 提交后同轮续答，模型正确逐题理解答案

   * 单问题场景回归：单选点击即发、多选勾选提交，与现状一致

   * 缺题提交 → 对应题输入框 shake，不发送

   * × 关闭 / 停止按钮 → 本轮取消无悬挂

