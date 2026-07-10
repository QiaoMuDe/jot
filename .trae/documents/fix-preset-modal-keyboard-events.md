# 修复预设弹窗键盘事件问题

## 问题概述

1. **ESC 需要先点击弹窗才能生效**：打开新增/编辑预设弹窗后，直接按 ESC 无效，必须先点击弹窗区域再按 ESC 才能关闭。
2. **回车保存需要按两次**：打开弹窗后直接按 Enter，光标会跳到标题/其他元素，需再按一次 Enter 才能保存。

## 根因分析

### 问题1：ESC 不生效

当前代码在 `presetModalOverlay` 元素上注册了 `keydown` 监听（bubble 阶段），但焦点实际在弹窗内部的 `<input>` 上。按理说 input 上的键盘事件应该冒泡到 overlay，但经过排查发现：

- 弹窗打开时先调了 `overlay.focus()`，随即又调了 `presetModalName.focus()`，焦点最终在 input 上
- 在 Wails/WebView2 环境中，ESBuild 打包后的事件冒泡链可能不稳定，或者 input 的默认行为（Enter）先于 bubble 阶段被处理
- 全局 ESC 处理器（`handleKeyboardNavigation`）在 document 的 bubble 阶段拦截了事件

**真正的解决方案**：参考搜索弹窗的已有模式，使用 **capture 阶段**（`addEventListener(..., true)`）来拦截键盘事件。搜索弹窗已有成功实践（`handleSearchModalKeydown`），可以避免与全局处理器冲突。

### 问题2：回车需要按两次

当前 Enter 处理也在 overlay 的 bubble 阶段监听。当焦点在 input 上时，Enter 键的默认行为（某些环境下会触发表单提交或移动焦点）可能先于 overlay 的 bubble 阶段执行，导致第一次按 Enter 时焦点被移到其他元素上，第二次按键才能真正触发 `savePresetModal()`。

**解决方案**：同样使用 capture 阶段 + `e.stopPropagation()`，确保在事件达到任何目标元素之前就拦截处理。

## 修改方案

### 修改点1：移除 overlay 的 bubble 阶段 keydown 监听，改用 capture 阶段

**文件：** `frontend/src/main.js`

**当前代码（L2390-L2402）：**
```js
document.getElementById('presetModalOverlay')?.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        e.stopPropagation();
        closePresetModal();
    } else if (e.key === 'Enter') {
        const providerDd = document.getElementById('presetModalProviderDropdown');
        if (providerDd && providerDd.classList.contains('open')) return;
        e.preventDefault();
        savePresetModal();
    }
});
```

**改为：**
```js
document.getElementById('presetModalOverlay')?.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        e.stopPropagation();
        closePresetModal();
    } else if (e.key === 'Enter') {
        // 如果服务商下拉菜单打开，不触发保存
        const providerDd = document.getElementById('presetModalProviderDropdown');
        if (providerDd && providerDd.classList.contains('open')) return;
        e.preventDefault();
        e.stopPropagation();
        savePresetModal();
    }
}, true);  // capture 阶段拦截
```

关键改动：
- 添加第三个参数 `true`（capture 阶段），在事件到达目标元素之前拦截
- Enter 处理也加上 `e.stopPropagation()`，防止事件继续传播触发其他行为

### 修改点2：移除全局 ESC 处理器中的预设弹窗空检查

**文件：** `frontend/src/main.js`

**当前代码（L5323-L5327）：**
```js
// 预设弹窗打开时，由弹窗自己的 ESC 处理
const presetOverlay = document.getElementById('presetModalOverlay');
if (presetOverlay && presetOverlay.classList.contains('visible')) {
    return;
}
```

**改为：**
删除这段代码。

原因：使用 capture 阶段后，overlay 的 ESC 处理器在 capture 阶段就拦截了事件并 `stopPropagation()`，事件根本到不了全局处理器。这段检查不再需要。

### 修改点3：移除 open 函数中不必要的 overlay.focus()

**文件：** `frontend/src/main.js`

**`openAddProfileModal`（L2558-L2560）和 `openEditProfileModal`（L2586-L2588）中：**

当前代码在显示弹窗后先 `overlay.focus()` 再 `presetModalName.focus()`。使用 capture 阶段后，不必再依赖 overlay 获取焦点，可以直接只 focus 到输入框。

改为：
```js
overlay.classList.add('visible');
document.getElementById('presetModalName').focus();
```

移除 `overlay.focus({ preventScroll: true })` 这行。

### 修改点4：移除 HTML 中不必要的 tabindex

**文件：** `frontend/index.html`

将 `tabindex="-1"` 移除。使用 capture 阶段后不再需要 overlay 可聚焦。

## 影响范围

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `frontend/src/main.js` | 修改 | overlay keydown 监听改为 capture 阶段，添加 `stopPropagation` |
| `frontend/src/main.js` | 删除 | 移除全局 ESC 处理器中的预设弹窗空检查 |
| `frontend/src/main.js` | 删除 | 移除 `openAddProfileModal`/`openEditProfileModal` 中的 `overlay.focus()` |
| `frontend/index.html` | 删除 | 移除 `presetModalOverlay` 的 `tabindex="-1"` |

## 验证步骤

1. 打开设置 → 配置预设 → 点击新增/编辑
2. 不点击弹窗任何区域，直接按 `ESC` → 弹窗应关闭
3. 再次打开弹窗，直接按 `Enter` → 如果名称为空，应显示错误提示；如果已填写内容，应保存成功
4. 确保原有功能正常：点击遮罩关闭、点击保存/取消按钮、服务商下拉切换
