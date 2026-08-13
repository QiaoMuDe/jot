# 通知置顶：修复 nm.show 通知显示在遮罩之下

## 摘要

全局通知（`nm.show`）的容器 `.notification-container` z-index 为 2000，低于 MCP 表单对话框（10000）、预设弹窗（10000）、确认弹窗（10001）、锁屏/拖拽遮罩（9999）等浮层，导致弹窗打开时通知被遮罩盖住。修复：将通知容器 z-index 提升为 99999，确保通知永远显示在最上层。

## 现状分析

浮层层级全景（frontend/src/css 各文件）：

| z-index  | 元素                                         | 位置                              |
| -------- | ------------------------------------------ | ------------------------------- |
| **2000** | `.notification-container`（通知）              | modals.css:617                  |
| 3000     | 移动笔记本弹窗                                    | sidebar.css:416                 |
| 9999     | `.drop-overlay`（拖拽导入遮罩）、`.lock-screen`（锁屏） | modals.css:662 / modals.css:856 |
| 10000    | 预设弹窗、MCP 表单对话框                             | settings-panel.css:1200 / 1856  |
| 10001    | `.confirm-overlay`（确认弹窗，最高业务层）             | modals.css:499                  |

通知容器结构（notification.js:8-15）：`position: fixed` 直接挂载于 `document.body`，位于根层叠上下文。因此只要提高其 z-index，即可越过所有对话框/遮罩。

## 修改方案

单文件改动：`frontend/src/css/components/modals.css`

`.notification-container`（第 613-623 行）z-index 由 `2000` 改为 `99999`，并更新注释说明置顶语义：

```css
.notification-container {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 99999;   /* 通知永远置顶：高于所有业务弹窗（确认弹窗 10001 / 预设与 MCP 表单 10000 / 锁屏 9999） */
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
  max-width: 380px;
}
```

## 假设与决策

* **决策**：采用单一高值 99999，而非精确取 10002。原因：通知是全局提示，语义上应高于一切弹窗/遮罩；且避免未来新增浮层时再次撞层级。

* **假设**：锁屏（lock-screen，9999）期间若有通知触发，通知显示在锁屏之上——应用锁定场景通常无通知触发，影响可忽略；若后续有异议可单独降级。

* 不改动 notification.js 及任何调用点；通知容器为 body 直接子元素，提高 z-index 即全局生效。

## 验证步骤

1. `cd frontend && npm run build` —— 构建通过。
2. `wails build` —— 重新编译 `build\bin\jot.exe`。
3. 重启应用验证：

   * 打开 MCP 服务器新增/编辑对话框，触发校验失败通知（如名称为空保存）→ 通知显示在遮罩之上。

   * 打开确认弹窗（如删除 MCP 服务器）时触发通知 → 通知仍在最上层。

   * 通知可正常点击关闭，不影响遮罩交互。

