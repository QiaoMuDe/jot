# 一键备份添加确认弹窗

## 现状分析

| 操作 | 文件 | 行号 | 有确认弹窗？ |
|------|------|------|:-----------:|
| `backupToDir()` | `data-management.js` | L421-445 | ❌ 无，直接执行 |
| `restoreFromDir()` | `data-management.js` | L450-483 | ✅ 有，L456 |

`restoreFromDir()` 的确认弹窗实现模式：
```js
const { els, nm, showConfirmDialog } = window;
// ...
const confirmed = await showConfirmDialog('确定要从最新备份恢复数据吗？当前所有笔记将被替换为备份内容，此操作不可撤销。');
if (!confirmed) return;
```

`showConfirmDialog` 已在 `main.js:8896` 注册到 `window` 上，`data-management.js` 中多处（清空AI会话/清空待办/清理图片/重置数据库等）均使用此模式。

## 修改方案

只改一个文件：`data-management.js`

### 修改内容

在 `backupToDir()` 函数中，在按钮 disabled 逻辑之前添加确认弹窗：

```js
export async function backupToDir() {
    const { els, nm, showConfirmDialog } = window;
    const btn = els.backupBtn;
    const labelEl = btn.querySelector('.dar-label');
    const origText = labelEl ? labelEl.textContent : '';
    
    // 新增：确认弹窗
    const confirmed = await showConfirmDialog('一键备份将覆盖上次备份内容，确定继续吗？');
    if (!confirmed) return;
    
    btn.disabled = true;
    // ... 原有逻辑不变
}
```

与 `restoreFromDir()` 完全一致的模式，仅确认文案不同。

### 确认文案

「一键备份将覆盖上次备份内容，确定继续吗？」— 明确告知用户"覆盖"风险。

## 涉及文件

- `frontend/src/js/data-management.js` — `backupToDir()` 函数首部添加确认弹窗

## 验证

- 点击「一键备份」→ 弹出确认弹窗 → 取消 = 不执行
- 点击「一键备份」→ 弹出确认弹窗 → 确认 = 原有逻辑正常执行
- 构建通过
