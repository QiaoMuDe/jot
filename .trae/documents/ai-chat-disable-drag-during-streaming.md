# AI 流式回复期间禁用拖拽上传

## 概要

在 AI 流式回复期间（`isStreaming === true`），禁用 AI 聊天区域的拖拽文件上传功能，改为提示用户。

## 当前状态分析

拖拽上传涉及两条代码路径，均未检查 `isStreaming`：

1. **视觉层** — `frontend/src/js/ai-chat.js` 中 `initAiChatFileDrop()` 函数（L4375-L4420）
   - `dragenter` 事件无条件显示 `#aiChatDropOverlay` 遮罩
   - `dragleave` / `drop` 事件无条件隐藏遮罩

2. **功能层** — `frontend/src/js/ai-chat.js` 中 `window.handleAiChatFileDrop()` 函数（L4423-L4453）
   - 由 `frontend/src/main.js` 的 `OnFileDrop` 回调（L7677-L7701）调用
   - 无条件调用后端 `ReadAIChatFiles` 读取文件并推入 `uploadedFiles[]`

`isStreaming` 标志在 L35 定义，生命周期管理完善（5 个复位点：done/error/catch/stop），已用于 20+ 处守卫。

## 修改方案

两处修改，各加一行守卫逻辑：

### 修改 1：`handleAiChatFileDrop` — 功能层硬阻断

**文件**: `frontend/src/js/ai-chat.js` L4423-L4453
**位置**: 函数开头，`_aiDragCounter` 重置之后，`paths` 判空之前
**改动**: 增加 `isStreaming` 判断，true 时弹通知提示并 return
**原因**: 这是最终防线，无论拖拽从哪个路径进来，drop 后都能拦截

```js
// AI 正在回复时禁止上传文件
if (isStreaming) {
    window.showNotification?.('AI 正在回复中，请稍后再试', 'warning');
    return;
}
```

### 修改 2：`initAiChatFileDrop` dragenter — 视觉层抑制

**文件**: `frontend/src/js/ai-chat.js` L4379-L4389
**位置**: `dragenter` 回调开头，`e.preventDefault()` 之后，`!e.dataTransfer.types.includes('Files') return` 之前
**改动**: 增加 `isStreaming` 判断，true 时直接 return（不显示遮罩）
**原因**: 遮罩不出现，用户就不会产生"可以上传"的预期，体验更一致

```js
// AI 正在回复时不显示拖拽遮罩
if (isStreaming) return;
```

## 决策依据

- `isStreaming` 已有完善的生命周期管理，无需额外状态同步
- 通知使用已有的 `showNotification` 基础设施，`warning` 类型已在 CSS 中定义
- 视觉层抑制 + 功能层阻断形成双重守卫，覆盖所有入口路径

## 验证步骤

1. 在 AI 对话输入框中拖拽文件 → 正常显示遮罩、正常上传
2. 发送消息，AI 流式回复期间拖拽文件 → 遮罩不显示
3. 流式回复期间释放文件 → 弹出"AI 正在回复中，请稍后再试"通知
4. 流式回复结束后拖拽文件 → 恢复正常上传行为
5. 测试所有 5 个 `isStreaming = false` 的复位路径（done/error/catch/stop/停止按钮）
