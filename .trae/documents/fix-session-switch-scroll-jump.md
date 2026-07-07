# 修复会话切换滚动跳跃问题

## 根因分析

`switchSession()` 中消息渲染完成后立即调用 `scrollToBottom()`，但此时每调用一次 `collapseActionsIfNeeded(el)` 都通过 `requestAnimationFrame` **推迟**了实际的按钮折叠/展开布局计算。

执行时序：

| 时机  | 事件                            | 说明                                                                                           |
| --- | ----------------------------- | -------------------------------------------------------------------------------------------- |
| T0  | 分块渲染循环                        | `addMessage` + `collapseActionsIfNeeded(el)` 每消息调用一次                                         |
| T0a | `collapseActionsIfNeeded(el)` | **仅排队** rAF 回调，不立即执行                                                                         |
| T1  | 循环结束                          | <br />                                                                                       |
| T2  | `scrollToBottom()`            | 读取当前 `scrollHeight` → 设置 `scrollTop = scrollHeight`                                          |
| T3  | **rAF 回调执行**                  | `collapseActionsIfNeeded` 测量宽度 → 添加/移除 `.collapsed` → **消息高度变化 →** **`scrollHeight`** **变小** |
| T4  | 浏览器绘制                         | `scrollTop` 相对于新 `scrollHeight` 过大 → 视口显示底部空白                                                |

## 修复方案

在 `switchSession()` 末尾，用连续两帧 `requestAnimationFrame` 等待所有 `collapseActionsIfNeeded` 回调执行完毕并更新布局后，再调用 `scrollToBottom()`。

## 变更文件

| 文件                           | 变更                                                                         |
| ---------------------------- | -------------------------------------------------------------------------- |
| `frontend/src/js/ai-chat.js` | `switchSession()` 中将最后两行 `scrollToBottom(); inputEl?.focus();` 替换为等待两帧后再滚动 |

## 修改前代码（switchSession 末尾）

```javascript
    scrollToBottom();
    inputEl?.focus();
```

## 修改后代码

```javascript
    // 等待两帧：确保所有 collapseActionsIfNeeded 的 rAF 回调执行完毕并更新布局
    await new Promise(r => requestAnimationFrame(r));
    await new Promise(r => requestAnimationFrame(r));
    scrollToBottom();
    inputEl?.focus();
```

## 验证

1. 切换有大量消息（10+ 条，部分包含代码块）的会话，观察是否不再跳动
2. 验证正常流式发送/接收消息时滚动行为不受影响

