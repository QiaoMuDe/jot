# 导入期间禁用文件拖入

## Summary

导入文件期间（`ImportFiles` 执行中）或冲突弹窗显示期间，用户再次拖入文件会触发新的导入，可能导致并发冲突。需要在这段时间内禁用笔记导入的拖入处理，等一切完成后恢复。

## Current State

`handleFileDropPaths`（[main.js#L9064](frontend/src/main.js#L9064)）是笔记导入的唯一入口，由 Wails `OnFileDrop` 回调在"编辑器未打开"分支调用。当前没有任何防重复触发机制。

## Proposed Changes

### `frontend/src/main.js`

1. **在** **`showImportConflictDialog`** **函数外层（与** **`resolved`** **同级）声明一个模块级标记**：不需要，改用函数级方案更简单。

2. **在** **`handleFileDropPaths`** **入口加 flag 守卫**：

   * 函数开头检查 `_importing` 标志，若为 `true` 则 `return`

   * 函数开头设 `_importing = true`

   * 在所有退出路径（正常完成、冲突完成、冲突取消、异常）都设 `_importing = false`

3. **具体 flag 管理**：

   ```javascript
   // 模块级变量
   let _importing = false;

   function handleFileDropPaths(paths, notebookId) {
       if (_importing) return;
       _importing = true;
       // ... 原有逻辑 ...

       // 在 showImportResults 的 onComplete 回调中：
       _importing = false;
   }
   ```

4. **拖入时视觉反馈 + 通知提示**：在 `dragenter` 事件处理中，如果 `_importing` 为 true，仍显示 `#dropOverlay` 但改为禁用样式（如变灰），并显示提示文字"导入进行中，请稍候"。在 `drop` 事件处理（`OnFileDrop` 回调）中，如果 `_importing` 为 true，调用 `nm.show("导入进行中，请稍候", "warning")` 提示用户。

## Affected Files

| 文件                     | 改动                                                                 |
| ---------------------- | ------------------------------------------------------------------ |
| `frontend/src/main.js` | 新增 `_importing` 标志 + `handleFileDropPaths` 入口守卫 + `dragenter` 禁用判断 |

## Assumptions & Decisions

* 只拦截笔记导入路径（编辑器未打开时的 drop），不拦截 AI 聊天上传和编辑器图片/文本插入

* flag 在 `showImportResults` 的 `onComplete` 回调中清除，覆盖了所有场景（无冲突直接完成、有冲突弹窗完成、有冲突弹窗取消）

* 不需要显示额外的"正在导入"提示，因为导入过程本身很快（后端并发处理），用户几乎感知不到等待

## Verification

1. 拖入文件导入中 → 再次拖入 → 遮罩变灰 + 提示"导入进行中，请稍候"
2. 冲突弹窗显示期间 → 再次拖入 → 同上
3. 导入完成后 → 再次拖入 → 正常导入
4. 冲突弹窗取消后 → 再次拖入 → 正常导入
5. AI 聊天上传文件不受影响
6. 编辑器内拖入图片/文本不受影响

