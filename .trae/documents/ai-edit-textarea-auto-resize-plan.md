# 用户消息编辑框自适应高度方案

## 问题

编辑用户消息时，textarea 固定高度，`max-height: 200px` 限制了显示区域，长内容需要滚动或手动拖拽，体验差。

## 变更

### 1. CSS — 移除 `max-height` 限制

**文件**：`frontend/src/css/components/ai-chat.css`

**改动**：将 `.ai-msg-edit-textarea`（第 3293 行）的 `max-height: 200px` 移除，改为 `max-height: none`，让 textarea 可以自由扩展。

**为何不改用** **`min-height`** **或** **`height`** **固定值**：因为 `resize: vertical` 已保留，用户仍可手动拖拽调整高度，移除 max-height 只是解除上限。

### 2. JS — 添加 auto-resize 逻辑

**文件**：`frontend/src/js/ai-chat.js`

**改动**：在 `enterEditMode()` 函数（第 4055-4068 行）中：

1. 定义一个 `autoResizeTextarea(textarea)` 辅助函数，将 textarea 高度重置为 `auto` 再设为 `scrollHeight + 'px'`，确保内容撑开
2. 在 `textarea.value = originalContent` 之后立即调用 `autoResizeTextarea(textarea)`
3. 添加 `input` 事件监听，用户输入时自动调整高度
4. 移除 `setTimeout` 中的 `textarea.select()` 行（可选），但保留 focus

## 文件修改清单

| 文件                                        | 修改                                                | 说明                   |
| ----------------------------------------- | ------------------------------------------------- | -------------------- |
| `frontend/src/css/components/ai-chat.css` | 第 3293 行 `max-height: 200px` → `max-height: none` | 解除高度上限               |
| `frontend/src/js/ai-chat.js`              | `enterEditMode()` 中添加 auto-resize 逻辑              | 进入编辑时自适应高度 + 输入时实时调整 |

## 验证

1. 编辑一条短消息（< 3 行），textarea 高度约 60px（min-height）
2. 编辑一条长消息（> 10 行），textarea 自动撑开到全部内容可见
3. 编辑时继续输入，textarea 随之自动增高
4. 手动拖拽 resize 手柄仍可自由调整高度

