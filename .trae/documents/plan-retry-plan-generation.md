# 计划生成重试机制

## 概述

当 `generatePlan()` 中模型输出的 JSON 解析/校验失败时，自动重试最多 3 次，每次重试通过 `ai:plan-generating` 事件通知前端显示进度。

## 当前状态

* `generatePlan()` 只调用一次 LLM，失败直接返回错误

* 前端 `ai:plan-generating` 事件当前 payload 为空字符串 `""`

* 前端仅在 `!hasReceivedChunk` 时显示 spinner，不支持多次更新文案

## 改动文件

### 1. `internal/agent/agent.go` — `generatePlan()` 加重试循环

**位置**: L284\~L393（当前函数体）

**改动内容**:

* 新增常量 `maxPlanRetries = 3`

* 将 LLM 调用 → 清洗 → 解析的逻辑包裹在 `for attempt := 1; attempt <= maxPlanRetries; attempt++` 循环中

* 每次重试前 `emit("ai:plan-generating", fmt.Sprintf("第 %d 次尝试", attempt))` 通知前端

* 仅在解析/校验失败时重试，LLM API 调用失败不重试（网络问题重试无意义）

* 解析失败时往 `msgs` 追加 assistant（模型上次错误输出）+ user（错误提示），让模型针对性修正

* 3 次都失败后返回最后一个错误

**伪代码**:

```
for attempt := 1; attempt <= maxPlanRetries; attempt++ {
    if attempt > 1 {
        emit("ai:plan-generating", fmt.Sprintf("第 %d 次尝试", attempt))
    }

    msg, err := chatModel.Generate(ctx, msgs)
    // LLM 调用失败 → 直接返回（不重试）
    // 空响应 → 作为解析失败处理

    raw = stripCodeBlock(msg.Content)

    plan, err := tools.ParseCreatePlanArgs(raw)
    if err == nil {
        // 成功 → 设置 PlanState + emit ai:plan-created + return nil
    }

    // 失败 → 记录日志
    // 追加到 msgs：assistant(msg.Content) + user("上次输出解析失败：{err}，请严格按格式重新输出 JSON。")
}
return 最后一次错误
```

### 2. `frontend/src/js/ai-chat.js` — `ai:plan-generating` 处理器支持重试文案

**位置**: L2727\~L2745（`ai:plan-generating` 事件处理器）

**改动内容**:

* 当前逻辑：仅在 `!hasReceivedChunk` 时创建 spinner + 固定文案

* 改为：如果 spinner 已存在则更新文案（重试时），不存在则创建

* 解析 payload，如果有内容则追加到文案中（如 "正在制定执行计划...（第 2 次尝试）"）

**伪代码**:

```
EventsOn('ai:plan-generating', (streamGen, payload) => {
    if (streamGen !== myGen) return;
    if (!hasReceivedChunk) {
        let wrap = contentDiv.querySelector('.ai-msg-plan-generating');
        if (!wrap) {
            // 首次：创建 spinner + 文案（现有逻辑）
            wrap = create elements...
            contentDiv.appendChild(wrap);
        }
        // 更新文案（payload 为重试信息时追加）
        const textEl = wrap.querySelector('span:last-child');
        if (payload) {
            textEl.textContent = '正在制定执行计划...（' + payload + '）';
        } else {
            textEl.textContent = '正在制定执行计划...';
        }
    }
});
```

## 验证步骤

1. 编译通过：`go build ./...`
2. 测试正常场景：Plan 模式下正常生成计划，spinner → 计划面板流程不变
3. 测试重试场景：可通过临时修改提示词或使用弱模型触发解析失败，观察：

   * 日志中出现 "计划生成第 N 次尝试失败" debug 日志

   * 前端 spinner 文案更新为 "正在制定执行计划...（第 2 次尝试）"
4. 测试最终失败：3 次都失败后，`ai:stream-error` 正常触发，右上角通知显示错误

