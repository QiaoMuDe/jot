# 计划：系统提示词注入当前时间并移除 get\_current\_time 工具

## Summary

AI 助手模块目前通过 `get_current_time` 工具在 ReAct 循环中获取时间（一次工具往返 + 长 Desc token 开销 + 依赖模型"必须调用"的非确定性约束）。本计划改为：在共享的系统提示词组装函数 `buildAIContextInstruction` 末尾注入当前时间（含日期、星期、时分、时区），Chat 与 Agent 两种模式同时受益；随后删除时间工具及其全部引用。

## Current State Analysis

* 时间工具实现：[internal/agent/tools/current\_time.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/current_time.go)，返回"当前日期/当前时间/当前年份"，Desc 约 200 token 且带强制调用场景列表

* 工具注册：[internal/agent/registry.go#L189](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go#L189) `{"get_current_time", tools.WrapWithError(...)}`

* 工具展示元数据：[internal/agent/tools/meta.go#L19](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/meta.go#L19)（前端工具开关列表数据源）

* Agent 模式提示词中的强制调用规范：[app.go#L2112-L2115](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2112-L2115)

* 共享提示词组装：[app.go#L2249-L2339](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2249-L2339) `buildAIContextInstruction`，Chat 模式（`CallAIStream` L2366）与 Agent 模式（`CallAIAgentStream` L2092）均调用；当前函数体以技能提示词注入结束，`return` 前 **无** 时间信息

* 包注释引用：[internal/agent/doc.go#L5](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/doc.go#L5)、[internal/agent/tools/doc.go#L5](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/doc.go#L5) 的工具清单中列有 `get_current_time`

* 前端无 `get_current_time` 文案映射（已确认 frontend/src 无引用），无需前端改动

* `playground/agent-demo`、`playground/agent-subagent-demo` 中的同名工具是独立演示程序，不在主程序构建内，**不动**

* Chat 模式现状：无任何工具，问"今天几号"只能靠模型训练知识瞎猜 —— 注入后此缺口补齐

## Proposed Changes

### 1. `app.go` — `buildAIContextInstruction` 末尾注入当前时间

位置：技能提示词注入之后、`return instruction.String()` 之前（即函数尾部，避免扰动前部稳定内容，利于提示词前缀缓存）。

注入格式（一行环境信息段）：

```
【环境信息】当前时间：2026-09-02 21:40 星期三（Asia/Shanghai，UTC+08:00）
```

实现要点：

* `time.Now()` 取本地时间（桌面应用本地时区即用户时区）

* 日期 `2006-01-02`、时分 `15:04`、中文星期（复用现有 `weekdays` 思路，在 app.go 内联实现或使用常量数组）、时区用 `Location().Zone()` 取名称 + `Format("UTC-07:00")` 取偏移

* 提示词说明时间仅作背景参考（不写"必须调用工具"之类规范，工具已删除）

### 2. `app.go` — 删除 Agent 强制调用时间工具规范段

删除 [app.go#L2112-L2115](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2112-L2115) 的 `【工具使用规范 - get_current_time 时间工具（强制调用）】` 整段 WriteString。

### 3. 删除工具实现文件

删除 `internal/agent/tools/current_time.go`（`weekdays` 变量随之消失；若步骤 1 需要中文星期，在 app.go 注入处独立实现，不跨包复用未导出变量）。

### 4. `internal/agent/registry.go` — 移除注册行

删除 L189 `{"get_current_time", tools.WrapWithError("get_current_time", tools.NewGetCurrentTime(), p.ctx)},`。

### 5. `internal/agent/tools/meta.go` — 移除展示条目

删除 L19 `{Name: "get_current_time", Label: "获取当前时间/日期/星期"},`。前端工具开关列表自动少一项；用户设置 `ai_agent_tools_disabled` 中可能残留 `get_current_time` 字符串，解析按未知名忽略，无害，不做迁移。

### 6. 包注释清理

* [internal/agent/doc.go#L5](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/doc.go#L5)：只读工具清单去掉 `get_current_time`

* [internal/agent/tools/doc.go#L5](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/doc.go#L5)：工具清单去掉 `get_current_time`，如有对时间工具的描述一并删除

## Assumptions & Decisions

* **决策**：注入 + 移除工具（非混合方案）。理由：应用无"运行中反复获取秒级时间"的场景；保留工具带来持续 token 开销与非确定性冗余调用风险；将来若恢复子代理，在子代理提示词同样注入一行即可

* **注入位置**：Instruction 尾部（技能提示词之后），保证 Chat/Agent 一致且不破坏前部内容的前缀缓存

* **时间精度**：到分钟（`15:04`）即可，秒级无实际需求

* **不做**：前端改动、playground 演示程序改动、`ai_agent_tools_disabled` 残留数据迁移

* 编译产物为纯 Go 变更，重新 `wails build` 即可生效，无需 `npm run build`

## Verification

1. `go build ./...` 编译通过（确认无残留引用）
2. `go test ./internal/agent/... ./internal/agent/tools/...` 通过
3. `wails build` 后手动验证：

   * **Chat 模式**：问"今天几号星期几"——直接给出正确日期（此前 Chat 模式无法得知真实时间），无工具调用

   * **Agent 模式**：问"现在几点"——直接回答，不再触发任何工具调用；工具开关列表中不再出现"获取当前时间"

   * 问"明天是几号"——基于注入时间正确推算

