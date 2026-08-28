# Agent 计划执行优化方案

## 概述

优化 Agent 的计划执行机制，解决计划兜底处理冗余、工具执行后未自动检测步骤完成、缺少步骤级别超时/重试、计划状态变更未实时同步等问题。

## 当前状态分析

### 现有机制

1. **计划完成检测机制**（agent.go:599-612）

   * 模型想输出最终回答时，检测计划是否全部完成

   * 未完成则拒绝接受，强制继续执行

2. **计划兜底处理**（agent.go:710-746）

   * 循环结束后自动补标未完成步骤为 done

   * 模型跳过 create\_plan 时自动补建单步计划

3. **update\_plan 工具**（tools/plan.go:242-338）

   * 模型手动调用更新步骤状态

   * 调用后自动推进 Current 指针

### 存在的问题

1. **兜底处理与新机制冗余**：新机制已强制完成计划，兜底逻辑可能不再需要
2. **工具执行后未自动检测**：模型可能忘记调用 update\_plan
3. **缺少步骤级别超时/重试**：步骤卡住时无自动处理
4. **计划状态变更未实时同步**：前端可能看不到最新状态

## 优化方案

### 优化1：简化计划兜底处理

**目标**：移除或简化与新机制冗余的兜底逻辑

**修改文件**：`internal/agent/agent.go`

**具体修改**：

* 移除第 710-735 行的自动补标逻辑（因为新机制已强制模型完成计划）

* 保留第 736-746 行的自动补建逻辑（模型跳过 create\_plan 时仍需兜底）

* 增加日志记录，当模型跳过 create\_plan 时记录警告

**代码位置**：

```go
// agent.go:710-735 需要修改
// agent.go:736-746 保留
```

### 优化2：工具执行后自动检测步骤完成

**目标**：在工具执行后，自动检测当前步骤是否应该标记为完成

**修改文件**：`internal/agent/agent.go`

**具体修改**：

* 在工具执行完成后（第 670 行之后），增加自动检测逻辑

* 如果当前有计划且存在 in\_progress 状态的步骤，自动将其标记为 done

* 发射 ai:plan-updated 事件同步前端

**代码位置**：

```go
// agent.go:670 之后增加
```

**实现逻辑**：

```go
// 工具执行后自动检测步骤完成
if toolCtx.PlanState != nil && name != "create_plan" && name != "update_plan" {
    // 查找当前 in_progress 的步骤
    for i := range toolCtx.PlanState.Steps {
        if toolCtx.PlanState.Steps[i].Status == "in_progress" {
            // 自动标记为 done
            toolCtx.PlanState.Steps[i].Status = "done"
            toolCtx.PlanState.Steps[i].Result = fmt.Sprintf("工具 %s 执行完成", name)
            // 推进 Current 指针
            if i+1 == toolCtx.PlanState.Current+1 {
                toolCtx.PlanState.Current = i + 1
            }
            // 发射更新事件
            emitPlanUpdated(emit, toolCtx.PlanState, &toolCtx.PlanState.Steps[i].ID, "done", toolCtx.PlanState.Steps[i].Result)
            break
        }
    }
}
```

### 优化3：添加步骤级别超时机制

**目标**：为每个步骤设置超时时间，防止步骤卡住

**修改文件**：`internal/agent/agent.go`

**具体修改**：

* 在 agentSession 中添加步骤开始时间记录

* 在每次循环迭代中检测当前步骤是否超时

* 超时后自动跳过当前步骤，标记为 skipped

**代码位置**：

```go
// agent.go:82-99 agentSession 结构体增加字段
// agent.go:534-672 循环中增加超时检测
```

**实现逻辑**：

```go
// agentSession 结构体增加字段
type agentSession struct {
    // ... 现有字段
    stepStartTime time.Time // 当前步骤开始时间
    stepTimeout   time.Duration // 步骤超时时间（默认 5 分钟）
}

// 循环中增加超时检测
if toolCtx.PlanState != nil && sess.stepStartTime.IsZero() {
    sess.stepStartTime = time.Now()
    sess.stepTimeout = 5 * time.Minute // 默认 5 分钟
}

// 检测超时
if !sess.stepStartTime.IsZero() && time.Since(sess.stepStartTime) > sess.stepTimeout {
    // 超时处理：跳过当前步骤
    if toolCtx.PlanState != nil && toolCtx.PlanState.Current < len(toolCtx.PlanState.Steps) {
        step := &toolCtx.PlanState.Steps[toolCtx.PlanState.Current]
        step.Status = "skipped"
        step.Result = "步骤执行超时，自动跳过"
        toolCtx.PlanState.Current++
        sess.stepStartTime = time.Time{} // 重置
        // 发射更新事件
        emitPlanUpdated(emit, toolCtx.PlanState, &step.ID, "skipped", step.Result)
    }
}
```

### 优化4：计划状态变更实时同步

**目标**：在每次工具执行后自动发射计划状态事件，让前端实时更新

**修改文件**：`internal/agent/agent.go`

**具体修改**：

* 在工具执行完成后，自动发射 ai:plan-status 事件

* 事件携带当前计划的完整状态

**代码位置**：

```go
// agent.go:670 之后增加
```

**实现逻辑**：

```go
// 工具执行后同步计划状态
if toolCtx.PlanState != nil {
    if b, err := json.Marshal(toolCtx.PlanState); err == nil {
        emit("ai:plan-status", string(b))
    }
}
```

## 实施顺序

1. **优化1：简化计划兜底处理**（优先级：高）

   * 移除冗余的自动补标逻辑

   * 保留必要的自动补建逻辑

2. **优化2：工具执行后自动检测步骤完成**（优先级：高）

   * 实现自动检测和标记逻辑

   * 确保与现有机制兼容

3. **优化4：计划状态变更实时同步**（优先级：中）

   * 添加 ai:plan-status 事件发射

   * 确保前端能正确处理

4. **优化3：添加步骤级别超时机制**（优先级：低）

   * 实现超时检测逻辑

   * 需要更多测试验证

## 验证步骤

1. **测试优化1**：

   * 运行 Agent 对话，验证计划完成后不再自动补标

   * 验证模型跳过 create\_plan 时仍能正确补建

2. **测试优化2**：

   * 执行多个工具调用，验证步骤自动标记为 done

   * 验证 Current 指针正确推进

3. **测试优化3**：

   * 设置较短的超时时间（如 10 秒）

   * 验证超时后步骤自动跳过

4. **测试优化4**：

   * 打开浏览器开发者工具

   * 验证每次工具执行后都收到 ai:plan-status 事件

## 注意事项

1. **向后兼容**：所有优化都应保持与现有前端的兼容性
2. **日志记录**：关键操作都应记录日志，便于调试
3. **错误处理**：自动检测逻辑不应影响正常的错误处理流程
4. **性能考虑**：实时同步不应显著增加性能开销

