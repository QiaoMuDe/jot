# NewApp 初始化失败时的资源清理方案

## Summary

当前 `NewApp()` 中失败路径有 `os.Exit(1)` 和 `panic(err)` 两种退出方式，均会绕过 Wails 的 `OnShutdown` 回调，导致 Logger 缓冲日志未落盘、DB 连接未关闭。方案 B：将 `os.Exit` 改为 `panic`，通过 `defer` 兜底完成资源清理后再 re-panic。

## Current State

`NewApp()` (app.go#L50-97) 中的失败退出路径：

| 行号 | 场景 | 退出方式 | Logger 状态 | DB 状态 |
|---|---|---|---|---|
| 57 | `Init` 失败 | `os.Exit(1)` | 无效 | 未打开 |
| 61 | Logger 为 nil | `os.Exit(1)` | 为 nil | 未打开 |
| 68 | DB 路径获取失败 | `panic(err)` | 正常 | 未打开 |
| 73 | `InitDB` 失败 | `panic(err)` | 正常 | 未打开 |

`logSvc.Close()` 在 `app.shutdown()` 中，但无论 `os.Exit` 还是 `panic` 都不会触发 `OnShutdown` 回调。

fastlog 的 `Close()` 是同步阻塞的，会执行 `flushLocked()` 确保缓冲区（256KB）全部写入 OS 内核，之后关闭文件句柄。

## Proposed Changes

### 文件：`app.go` — `NewApp()` 函数

**改动 1：`defer` 兜底**

在 `NewApp()` 最顶部、创建 `logSvc` 之后立即添加：

```go
defer func() {
    if r := recover(); r != nil {
        if logSvc.Logger != nil {
            logSvc.Close()
        }
        println("启动失败:", r)
        os.Exit(1)
    }
}()
```

**改动 2：`os.Exit(1)` 替换为 `panic`**

- app.go#L57：`os.Exit(1)` → `panic(err)`
- app.go#L61：`os.Exit(1)` + `println` → `panic("日志实例为空，无法继续启动")`

**不需要在 defer 中关闭 DB 的原因**：

- 第 67-68 行（DB 路径失败）：`db` 尚未创建，无 DB 可关
- 第 72-74 行（DB Init 失败）：`InitDB` 返回 error，`db` 为 nil，无 DB 可关
- DB Init 成功后到 `return &App{}` 之间没有任何会 panic 的代码（`strconv.Atoi` 返回 error、`SetLevel` 有 nil 防护），所以不存在"DB 已开但未能返回"的路径

### 文件：`app.go` — `shutdown()` 函数

**无需修改。** 正常退出路径（`shutdown()`）已包含 Logger + DB 关闭的完整逻辑。

## Assumptions & Decisions

1. **`os.Exit` 全部改为 `panic`**：Go 的 `defer` 机制让 `panic` 可拦截、可清理后 re-panic，而 `os.Exit` 直接绕过所有 Go 级清理
2. **清理后 `os.Exit(1)` 而非 re-panic**：错误已由原始调用输出（`println`/`Errorw`），兜底只负责资源清理 + 退出，不丢 Go 栈跟踪给用户
3. **不处理 DB Close**：经过逐路径分析，在 panic 到达 `defer` 时 DB 要么未创建、要么已成功返回 App，不存在中间态

## Verification

1. 编译通过：`go build ./...`
2. 确认 `NewApp()` 中不再有 `os.Exit` 调用（仅保留 `defer` 中的 `recover` + `panic`）
3. 确认 `defer` 中 `logSvc.Logger != nil` 检查可防御性调用
