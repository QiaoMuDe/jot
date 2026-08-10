# 新增 get\_current\_time 工具

## Summary

为 Agent 新增一个 `get_current_time` 工具：模型在 ReAct 循环中调用它获取当前日期、时间、星期、年份等，用于回答"今天几号""现在几点""今年是哪年"或依赖当前时间背景的问题。无参数、无外部依赖（仅标准库 `time`），遵循 tools 子包既有模式（每文件一工具 + `NewXxx` 构造器 + registry 注册）。

## Current State Analysis

* 工具架构：[tools 子包](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools) 每文件一个工具，均提供导出构造器，由父包 [registry.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/registry.go#L19-L25) 统一装配注册（现有 `refine_search_query` / `web_search` / `recall_notes`）。

* 工具实现模式（参考 [recall\_notes.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/recall_notes.go#L33-L49)）：结构体 + 编译期断言 `var _ tool.InvokableTool` + `Info()`（名称/描述/参数 Schema）+ `InvokableRun()`（执行并返回文本）+ `NewXxx()` 构造器。

* 注册方式：`tools.WrapWithError("工具名", tools.NewXxx(...), p.ctx)` 一行注册；`WrapWithError` 保证失败回填模型不中断循环。

* [doc.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/doc.go#L1-L11) 维护工具清单说明，需同步补充。

* eino schema 支持无参工具：`ParamsOneOf` 传空 map 的 `NewParamsOneOfByParams(map[string]*schema.ParameterInfo{})`（避免某些模型对 nil ParamsOneOf 的处理差异）。

* 项目无统一时间格式惯例（仅备份文件名用 `2006-01-02`），本工具自定输出格式即可。

## Proposed Changes

### 1. 新建 `internal/agent/tools/current_time.go`

遵循既有工具模式，无依赖注入（不需要 ai/setting/ctx），纯 `time.Now()` 读取本地时间：

* 结构体 `currentTimeTool` + 编译期断言 `var _ tool.InvokableTool = (*currentTimeTool)(nil)`。

* `Info()`：

  * `Name: "get_current_time"`

  * `Desc`: 说明用途与调用时机——"获取当前日期与时间（年份、日期、星期、时分秒）。当用户询问现在几点、今天几号/星期几、今年是哪年，或问题需要以当前时间作为背景时调用。无参数。"

  * `ParamsOneOf`: `schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{})`（空参数）。

* `InvokableRun()`：忽略入参，取 `time.Now()`，返回固定格式文本：

```
当前日期：2026-08-10（星期一）
当前时间：22:37:10
当前年份：2026
```

实现要点：

* 星期用中文映射数组 `[星期日, 星期一, ..., 星期六]` 由 `now.Weekday()` 索引取值（Go 的 Weekday 返回英文，直接输出不友好）。

* 日期格式 `now.Format("2006-01-02")`，时间格式 `now.Format("15:04:05")`，年份 `now.Year()`。

* 返回 `string, nil`，无失败路径。

* 构造器 `func NewGetCurrentTime() tool.InvokableTool`。

### 2. `internal/agent/registry.go`：注册工具

`buildTools` 工具列表追加一行：

```go
tools.WrapWithError("get_current_time", tools.NewGetCurrentTime(), p.ctx),
```

### 3. `internal/agent/tools/doc.go`：更新工具清单说明

* 顶部职责说明中工具列表补充 `get_current_time`。

* 新增一行说明本文件实现 `get_current_time` 时间工具。

## Assumptions & Decisions

1. **无参数**：模型调用即获得完整时间信息（日期/时间/星期/年份一次返回），无需细分参数；这符合"获取当前的时间、日期或年份等"的整体意图。
2. **使用本地时间**：App 为本地桌面应用，直接使用 `time.Now()`（本地时区 Asia/Shanghai），不做时区参数。
3. **中文友好输出**：星期用中文；信息多行结构化，方便模型解析。
4. **不注入依赖**：工具无状态，构造器无参数，`WrapWithError` 包装后失败路径实际不会触发。
5. **不修改其他工具 / 前端 / app.go**：范围最小。

## Verification

1. `go build ./...` 通过。
2. `go vet ./internal/agent/...` 通过。
3. 手动验证（Agent 模式）：向模型提问"今天是几号 / 现在几点"，观察模型调用 `get_current_time` 工具并基于返回时间作答；工具状态条正常显示工具名与结果。

