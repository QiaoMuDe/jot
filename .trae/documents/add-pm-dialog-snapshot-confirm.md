# 计划：密码管理新建/编辑对话框增加快照 + 未保存确认机制

## 变更概览

为密码管理页的添加/编辑对话框增加表单快照与脏检查机制，关闭时如有未保存修改则弹出确认提示，防止误操作丢失数据。完全复用设置页 API 预设对话框的快照模式。

## 当前状态

### 密码管理对话框（无任何快照机制）

* `openPmEditDialog`：打开时清空/填充表单，无快照

* `closePsetModal`：直接隐藏，无确认

* 4 个关闭入口（×按钮 / 取消 / 遮罩层 / ESC）全部直接关闭

### 参考实现：API 预设对话框（有完整快照机制）

* `presetModalInitial`：打开时记录快照 `{ name, url, key }`

* `hasPresetModalChanges()`：逐字段对比当前值与快照

* `closePresetModal(force)`：`force !== true` 且有修改时弹 `showConfirmDialog()`

* 保存成功后 `closePresetModal(true)` 跳过确认

### 已有可复用组件

* `showConfirmDialog(msg)` — 全局确认弹窗，返回 `Promise<boolean>`

***

## 改动清单

仅改动 `password-manager.js` 一个文件，约 25 行。

### 改动 1：新增快照变量

**位置**：模块级变量区（`editingId` 附近，约第 33 行）

```js
let pmEditSnapshot = { name: '', username: '', password: '', url: '', note: '' };
```

### 改动 2：新增脏检查函数

**位置**：`closePmEditDialog` 函数附近

```js
function hasPmEditChanges() {
    return pmFieldName.value.trim() !== pmEditSnapshot.name
        || pmFieldUsername.value.trim() !== pmEditSnapshot.username
        || pmFieldPassword.value !== pmEditSnapshot.password
        || pmFieldUrl.value.trim() !== pmEditSnapshot.url
        || pmFieldNote.value !== pmEditSnapshot.note;
}
```

### 改动 3：`openPmEditDialog` 打开时创建快照

在 `fillPmEditForm` 调用之后、`pmEditOverlay.style.display = 'flex'` 之前，记录快照：

```js
pmEditSnapshot = {
    name: pmFieldName.value.trim(),
    username: pmFieldUsername.value.trim(),
    password: pmFieldPassword.value,
    url: pmFieldUrl.value.trim(),
    note: pmFieldNote.value,
};
```

### 改动 4：改写 `closePmEditDialog` 支持 force 参数

```js
async function closePmEditDialog(force) {
    if (force !== true && hasPmEditChanges()) {
        const ok = await showConfirmDialog('有未保存的修改，确定放弃并关闭吗？');
        if (!ok) return;
    }
    pmEditOverlay.style.display = 'none';
    editingId = null;
}
```

### 改动 5：保存成功后强制关闭

在 `savePmRecord` 的 try 块中，将 `closePmEditDialog()` 改为 `closePmEditDialog(true)`，跳过确认。

### 改动 6：ESC 键入口同步更新

`window.pmHandleEscape` 中的关闭调用也需要经过快照检查（当前已调用 `closePmEditDialog()`，改写后自动生效）。

***

## 涉及文件

| 文件                                    | 改动                                                                                                  |
| ------------------------------------- | --------------------------------------------------------------------------------------------------- |
| `frontend/src/js/password-manager.js` | 新增快照变量 + hasPmEditChanges + openPmEditDialog 记录快照 + closePmEditDialog 加 force + savePmRecord 传 true |

总计 **\~25 行**，单文件改动。

***

## 验证步骤

1. **新建对话框无修改 → 关闭**：点击新建 → 不输入任何内容 → 点取消/×/遮罩/ESC → 直接关闭，无弹窗
2. **新建对话框有修改 → 关闭**：点击新建 → 输入名称 → 点取消 → 弹出确认"有未保存的修改" → 点取消 → 对话框保持打开 → 点确定 → 对话框关闭
3. **编辑对话框无修改 → 关闭**：编辑已有记录 → 不改任何内容 → 点取消 → 直接关闭
4. **编辑对话框有修改 → 关闭**：编辑已有记录 → 修改密码 → 点遮罩层 → 弹出确认 → 确定后关闭
5. **保存后无确认**：修改内容 → 点保存 → 对话框直接关闭，不弹确认
6. **ESC 键层级**：编辑对话框打开时按 ESC → 弹出确认 → 再按 ESC → 确认框关闭，编辑对话框保持打开

