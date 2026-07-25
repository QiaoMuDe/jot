# AI 助手优化按钮问题修复计划

## 问题描述

1. **发送按钮未被禁用**：点击「优化」按钮开始优化后，旁边的发送按钮仍可点击，可能导致冲突。
2. **优化无法取消**：点击「优化」后只能等待 AI 返回结果，无法中途取消。

## 当前状态分析

### 相关文件

| 文件                                | 位置                                                              | 用途                                                         |
| --------------------------------- | --------------------------------------------------------------- | ---------------------------------------------------------- |
| `frontend/src/js/ai-chat.js`      | L10-22, L420-545, L443-463                                      | 前端 UI 变量、优化按钮、发送按钮、停止按钮事件处理                                |
| `frontend/index.html`             | L1170-1184                                                      | 输入行 DOM：#aiChatPolishBtn / #aiChatSendBtn / #aiChatStopBtn |
| `app.go`                          | L55-65(App 结构体), L1573-1576(CallAI), L2430-2437(CancelAIStream) | 后端 App 结构体、CallAI 非流式绑定、取消方法                               |
| `internal/services/ai_service.go` | L150-177(CallAI)                                                | 后端 AI 服务层，内部创建 `context.WithTimeout`                       |

### 关键发现

* **发送按钮** `sendBtnEl` 在优化点击处理中完全未被引用，优化期间保持原有状态

* **优化调用**使用 `window.go.main.App.CallAI()`（非流式），后端 `ai_service.go` 内部创建 `context.WithTimeout(context.Background(), 60s)`，外部无法取消

* **已有取消模式**：流式调用 `CallAIStream` 使用 `a.aiStreamCancel context.CancelFunc`，通过 `CancelAIStream()` 触发

* **停止按钮** `#aiChatStopBtn` 初始 `display:none`，目前仅用于流式生成的取消

* **优化和流式互斥**：优化按钮点击时检查 `isStreaming`，不会同时运行

## 修改方案

### 变更 1：后端 — 为 `CallAI` 添加取消能力（app.go + ai\_service.go）

**app.go**：

* 修改 `App.CallAI` 方法：不再直接委托给 `aiService.CallAI`，而是：

  * 创建 `context.WithTimeout(context.Background(), 60s)`，存储 cancel 函数到 `a.aiStreamCancel`

  * 将 `ctx` 传递给 `aiService.CallAI`

  * defer 中调用 `cancel()` 并清空 `a.aiStreamCancel`

* `CancelAIStream()` 保持不变，它会同时适用于流式和非流式调用（两者互斥）

**ai\_service.go**：

* 修改 `CallAI` 方法签名：`CallAI(ctx context.Context, messages []Message) (string, error)`

* 移除内部 `context.WithTimeout` 创建逻辑，直接使用传入的 `ctx`

* 在 service 层**不**处理 cancel 错误，原样返回让 app.go 处理

**关键点**：复用 `aiStreamCancel` 字段是安全的，因为优化和流式生成互斥。

### 变更 2：前端 — 优化期间禁用发送按钮（ai-chat.js）

在优化按钮点击处理中：

* 调用 `CallAI` **之前**：设置 `sendBtnEl.disabled = true`

* 在所有出口（成功/失败/取消）恢复发送按钮状态

恢复逻辑：

* 如果 `inputEl.value.trim()` 非空 → `sendBtnEl.disabled = false`

* 如果为空 → `sendBtnEl.disabled = true`

### 变更 3：前端 — 添加优化取消支持（ai-chat.js）

**新增变量**：

* `let isPolishOptimizing = false;` — 标记是否正在优化中，供 catch 块区分取消与错误

**优化按钮点击处理改动**：

* 调用 `CallAI` 前：设置 `isPolishOptimizing = true`

* 显示停止按钮：`stopBtnEl.style.display = ''; sendBtnEl.style.display = 'none'`

* 成功/失败出口：隐藏停止按钮，恢复发送按钮，`isPolishOptimizing = false`

**停止按钮点击处理改动**（L445-463）：

在停止按钮处理中增加分支判断：

```js
if (isPolishOptimizing) {
    // 1. 隐藏停止按钮，显示发送按钮
    // 2. isPolishOptimizing = false
    // 3. 调用 CancelAIStream()
    // 4. 恢复输入框内容为 polishOriginalText
    // 5. 恢复优化按钮状态（移除 is-loading，文字恢复"优化"）
    // 6. 恢复发送按钮状态
    return; // 不执行后续流式取消逻辑
}
```

**优化 catch 块改动**：

```js
catch (e) {
    if (!isPolishOptimizing) return; // 已被停止按钮处理，跳过
    isPolishOptimizing = false;
    // 恢复 UI（stop btn / send btn 显示切换）
    // 恢复优化按钮状态
    // 原本的错误通知逻辑不变
}
```

## 验证步骤

1. 点击优化 → 发送按钮应立即置灰不可点击
2. 优化进行中点击停止按钮 → 输入框恢复原文，优化按钮恢复为「优化」，发送按钮可点击
3. 优化正常完成 → 打字机效果正常，发送按钮在输入框有内容时可点击
4. 优化失败 → 显示错误通知，发送按钮恢复正常
5. 已有流式生成功能不受影响（停止按钮在流式模式下的行为不变）

