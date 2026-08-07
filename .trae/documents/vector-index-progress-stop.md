# 笔记量化弹窗：单篇进度 50% 起步 + 关闭可确认停止

## Summary

修复笔记量化（向量索引）弹窗进度视图的两个交互问题：

1. **单篇量化时主进度条停在 0%**：embedding 期间 `done=0`，主进度一直显示 0%，处理完才直接跳 100%。改为 embedding 阶段按"当前篇处理到一半"计，单篇量化时进度从 50% 起步，完成变 100%；多篇时进度平滑递增。
2. **量化进行中关闭一律拦截**：目前无论点击遮罩还是右上角关闭按钮都只弹"请等待完成"提示，无法停止。改为分层处理：点遮罩（周围空白）保持拦截提示；点右上角关闭按钮弹出"是否停止量化"确认框，确认后调用后端取消并关闭弹窗。

## Current State Analysis

### 进度上报链路（后端 → 前端）
- [vector_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/vector_service.go#L34-L131) `IndexNotes`：每篇笔记回调三次 —
  - 开始：`progressCb(i, total, title, "embedding", 0, len(chunks))`
  - 块级（每批）：`progressCb(i, total, title, "embedding", doneChunk, totalChunk)`
  - 完成：`progressCb(i+1, total, title, "done", len(chunks), len(chunks))`
- [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L1594-L1616) `startVectorIndex` goroutine：将回调转为 `vector:index-progress` 事件推送
- [data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js#L976-L1026) `updateVectorIndexProgress`：`percent = done / total * 100` — 这就是单篇停在 0% 的原因（embedding 期间 done 恒为 0，total=1）

### 关闭拦截现状
- [data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js#L606-L624) `closeVectorIndexModal`：`vectorIndexRunning` 时一律提示并 return
- [data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js#L696-L699) 事件绑定：关闭按钮 / 遮罩 / Done 按钮全部绑定 `closeVectorIndexModal`

### 后端取消支持现状（已具备取消链路）
- [App struct](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L98-L100)：已有 `vectorIndexMu` + `vectorIndexRunning` 防重入锁，但无 cancel 源；`startVectorIndex` 用 `context.Background()`（不可取消）
- `IndexNotes` 循环顶部已检查 `ctx.Err()`（L61-63）；`EmbedWithProgress` 分批调用 `ollamaEmbed/openaiEmbed`，两 Provider 均已支持 ctx 取消（[client.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/aicli/client.go#L119-L140)、[openai.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/aicli/openai.go#L57-L58)、[ollama.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/aicli/ollama.go#L29-L31)）
- 确认对话框：`showConfirmDialog(msg, okText='确定', cancelText='取消')` 位于 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L1128)，支持自定义按钮文案

## Proposed Changes

### 1. 单篇进度 50% 起步（仅前端，[data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js#L976-L985)）

`updateVectorIndexProgress` 顶部进度计算改为按 stage 区分：

```js
const stage = p.stage;
// embedding 阶段当前篇按"处理到一半"计：单篇量化时进度从 50% 起步，
// 避免 embedding 期间长时间停在 0%；done/error 阶段按实际完成篇数计
const isEmbedding = stage === 'embedding';
const percent = total > 0
    ? Math.min(100, Math.round((isEmbedding ? done + 0.5 : done) / total * 100))
    : 0;
```

- 单篇：embedding → 50%，done → 100% ✓（用户要求）
- 多篇：第 1 篇处理中 0.5/N → 完成 1/N → … 平滑递增
- 下方 `stageMap[p.stage]` 复用局部 `stage` 变量即可，无需其他改动

### 2. 关闭拦截分层（前端 + 后端）

#### 后端 [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go)

a) App struct 增加取消源字段：

```go
vectorIndexCancel context.CancelFunc // 量化任务取消源（受 vectorIndexMu 保护）
```

b) `startVectorIndex` 改为可取消 ctx（在置 `vectorIndexRunning = true` 后）：

```go
ctx, cancel := context.WithCancel(context.Background())
a.vectorIndexMu.Lock()
a.vectorIndexCancel = cancel
a.vectorIndexMu.Unlock()
```

c) `release` 中清空并触发取消（防御性，正常路径无副作用）：

```go
release := func() {
    a.vectorIndexMu.Lock()
    a.vectorIndexRunning = false
    if a.vectorIndexCancel != nil {
        a.vectorIndexCancel()
        a.vectorIndexCancel = nil
    }
    a.vectorIndexMu.Unlock()
}
```

d) goroutine 错误分支：用户取消不报"失败"（不发 `vector:index-error`，避免前端误显示）：

```go
if err != nil {
    if errors.Is(err, context.Canceled) {
        a.LogSvc.Logger.Infow("向量量化索引已取消")
        return
    }
    // ...原有 error 事件逻辑不变
}
```

e) 新增绑定方法（防重入锁内检查）：

```go
// CancelVectorIndex 停止当前正在进行的量化任务（异步取消，任务在批次间/笔记间退出）
func (a *App) CancelVectorIndex() error {
    a.vectorIndexMu.Lock()
    defer a.vectorIndexMu.Unlock()
    if !a.vectorIndexRunning || a.vectorIndexCancel == nil {
        return errors.New("当前没有正在进行的量化任务")
    }
    a.vectorIndexCancel()
    return nil
}
```

> `vectorIndexRunning` 复位由 goroutine 的 `release` 负责，`CancelVectorIndex` 不直接复位，避免并发复位竞态。

#### 前端 [data-management.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/data-management.js)

a) `closeVectorIndexModal` 增加 `force` 参数（默认 false，保持遮罩拦截行为）：

```js
export function closeVectorIndexModal(force = false) {
    if (vectorIndexRunning && !force) {
        window.nm?.show?.('量化进行中，请等待完成', 'warning');
        return;
    }
    // ...原有关闭逻辑不变（清理 timer / 事件 / 复位状态）
}
```

b) 新增关闭请求处理（右上角 X 按钮专用）：

```js
async function onVectorIndexCloseRequested() {
    if (!vectorIndexRunning) {
        closeVectorIndexModal();
        return;
    }
    const confirmed = await window.showConfirmDialog('量化进行中，确定要停止并关闭吗？', '停止', '继续');
    if (!confirmed) return;
    // 异步停止后端任务，随后立即关闭弹窗并清理事件
    try { await window.go?.main?.App?.CancelVectorIndex?.(); } catch (_) { /* 忽略 */ }
    closeVectorIndexModal(true);
}
```

c) `bindVectorIndexModalEvents` 事件绑定调整：

```js
document.getElementById('vectorIndexClose')?.addEventListener('click', onVectorIndexCloseRequested);
document.getElementById('vectorIndexOverlay')?.addEventListener('click', () => closeVectorIndexModal()); // 遮罩保持拦截
document.getElementById('vectorIndexDoneBtn')?.addEventListener('click', () => closeVectorIndexModal()); // 仅完成/错误后可见，保持原行为
```

### 竞态边界说明（可接受）

- 确认停止后立即关闭弹窗并 `cleanupVectorIndexEvents()`，后端 goroutine 收尾期间的事件已被卸载，无残留
- 若用户停止后极速重开并再次启动，可能短暂命中后端防重入"量化任务正在进行中"（取消极快，概率极低），前端会以错误提示展示，可接受

## Assumptions & Decisions

- **50% 语义**：embedding 阶段当前篇按"处理到一半"计，而非仅单篇特判——多篇进度也更平滑，且实现更简单
- **遮罩仍拦截**：按用户要求，点周围空白保持"阻止退出 + 提示"，只有右上角关闭按钮可确认停止
- **取消后不显示错误**：取消是用户主动行为，后端不发射 `vector:index-error`，前端关闭弹窗无感知
- **不新增 `vector:index-canceled` 事件**：确认后立即关闭弹窗，前端收不到也无需处理取消事件，减少改动面

## Verification

1. `go build ./...` 与 `golangci-lint run ./...` 通过
2. `npm run build` 通过
3. `wails dev` 手动验证：
   - 只勾选 1 篇笔记量化 → 主进度从 50% 起步，完成后变 100%
   - 多篇量化 → 主进度平滑递增、不跳变
   - 量化中点击周围空白（遮罩）→ 拦截并提示"请等待完成"
   - 量化中点击右上角 X → 弹出"是否停止"确认框 → 选"停止"后弹窗关闭、后端停止（日志出现"已取消"）；选"继续"不关闭
   - 停止后重新量化可正常进行
   - 量化完成后点 X / Done → 直接关闭，无确认框
