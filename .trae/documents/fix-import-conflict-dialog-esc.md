# 修复导入冲突弹窗 ESC 关闭逻辑

## 问题

1. 冲突弹窗的 ESC 处理绑定在 overlay DOM 的 `keydown` 事件上，但 overlay 不一定获得焦点，导致 ESC 可能不生效
2. 关闭冲突弹窗（ESC 或点击遮罩）后，`showImportResults` 的回调仍然显示"导入完成"通知，即使用户没有处理任何冲突
3. 全局 ESC 链中没有处理导入冲突弹窗的优先级

## 改动方案

### 文件：`frontend/src/main.js`

#### 1. 全局 ESC 链中添加冲突弹窗处理

在 `handleKeyboardNavigation` 的 ESC 分支中（约第 6586 行确认框判断之后），添加冲突弹窗的检测和关闭逻辑：

```javascript
// 导入冲突弹窗打开时：ESC 终止导入（不关闭确认框，由确认框自己的 ESC 处理）
const conflictOverlay = document.querySelector('.import-conflict-overlay');
if (conflictOverlay && conflictOverlay.classList.contains('visible')) {
    // 如果同时有确认框打开，不处理（让确认框的 ESC 优先）
    if (!(els.confirmDialog && els.confirmDialog.classList.contains('visible'))) {
        conflictOverlay._onCancel?.();
    }
    return;
}
```

#### 2. 改造 `showImportConflictDialog` 的 `close()` 逻辑

将 `close()` 拆分为两种情况：

* `close()` — 正常关闭（所有冲突处理完毕），传 `null` 表示完成

* `_onCancel` — 用户 ESC 或点击遮罩取消，传 `false` 表示取消

```javascript
function close() {
    overlay.classList.remove('visible');
    setTimeout(() => {
        overlay.remove();
        if (onComplete) onComplete(null); // null = 正常完成
    }, 200);
}

// 挂载取消回调供全局 ESC 调用
overlay._onCancel = () => {
    overlay.classList.remove('visible');
    setTimeout(() => {
        overlay.remove();
        if (onComplete) onComplete(false); // false = 用户取消
    }, 200);
};
```

ESC 和遮罩点击改为调用 `_onCancel`：

```javascript
// ESC 关闭
overlay.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') overlay._onCancel?.();
});

// 点击遮罩关闭
overlay.addEventListener('click', (e) => {
    if (e.target === overlay) overlay._onCancel?.();
});
```

#### 3. 改造 `showImportResults` 的冲突回调

区分"完成"和"取消"：

```javascript
showImportConflictDialog(conflicts, (result) => {
    if (result === false) {
        // 用户取消了冲突弹窗
        const msg = `导入已取消: 新建 ${successCount} 个`;
        nm.show(msg, 'warning', 3000);
        if (successCount > 0 || updatedCount > 0) {
            loadNotes().then(() => {
                loadNotebooks();
                flashNoteCards(importedNoteIds);
            });
        }
        return;
    }
    // result === null，正常完成
    // ... 现有逻辑不变
});
```

#### 4. 确认框 ESC 时阻止冒泡到冲突弹窗

确认框的 ESC 处理已有 `e.stopPropagation()`，且全局 ESC 链在确认框可见时直接 return，这确保了：

* 确认框打开时 → ESC 只关闭确认框，不关闭冲突弹窗

* 确认框关闭后 → ESC 关闭冲突弹窗

无需额外修改。

## 验证步骤

1. 拖入多个冲突文件，弹出冲突弹窗
2. 按 ESC → 应显示"导入已取消"通知，不显示"导入完成"
3. 点击遮罩空白处 → 同上
4. 点击某个项的"覆盖"按钮 → 确认框弹出，此时按 ESC → 只关闭确认框，冲突弹窗仍在
5. 确认框关闭后再按 ESC → 关闭冲突弹窗，显示取消通知
6. 逐个处理完所有冲突项 → 弹窗自动关闭，显示正常的导入完成通知

