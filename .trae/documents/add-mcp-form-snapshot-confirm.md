# MCP 服务器表单：打开时快照 + 关闭时未保存修改确认

## 摘要

为新增/编辑 MCP 服务器对话框（`mcpServerFormDialog`）增加「打开时快照」机制：打开表单时记录所有字段的初始值；关闭时对比当前值，若发现未保存修改，弹出确认对话框询问「有未保存的修改，确定放弃并关闭吗？」，确认后才关闭。实现完全复用项目预设弹窗（presetModal）的既有模式。

## 现状分析

参照实现（预设弹窗，main.js）：

* `let presetModalInitial = { name, url, key }`（3079）——打开时记录快照（3189-3194、3217-3222）

* `hasPresetModalChanges()`（3240-3245）——对比当前值与快照

* `async function closePresetModal(force = false)`（3229-3237）——非字面量 `true` 且有修改时 `showConfirmDialog('有未保存的修改，确定放弃并关闭吗？')`，确认后才关闭

* 保存成功后 `closePresetModal(true)` 跳过确认（3277）

* 全局 Esc 分支中 `confirmDialog` 打开时直接 `return`（6482-6484），防止确认框弹出期间重复触发关闭

MCP 服务器表单现状：

* 模块级状态：`mcpFormMode / mcpFormEditId / mcpFormSaving / mcpFormEnabled / mcpFormTransport`（9067-9076）

* `openMCPServerForm(srv)`（9244-9292）：填充字段后显示对话框

* `closeMCPServerForm()`（9297-9307）：**无确认逻辑**，直接关闭

* `showConfirmDialog(msg, okText, cancelText)` 返回 `Promise<boolean>`（1133-1164）

* `closeMCPServerForm()` 调用点共 4 处：

  * 6489 全局 Esc 分支（应走确认）

  * 9469 保存成功后（应跳过确认）

  * 9497 backdrop 点击（应走确认）

  * 9858 切换设置面板（应走确认）

表单可编辑字段（index.html）：`mcpServerNameInput`（名称）、传输方式（`mcpFormTransport` 变量 + `setMCPFormTransport`）、`mcpServerCommandInput`（命令）、`mcpServerArgsInput`（参数）、`mcpServerEnvInput`（环境变量）、`mcpServerUrlInput`（URL）、`mcpServerHeadersInput`（请求头）。

## 修改方案

全部改动位于 `frontend/src/main.js`，共 4 处，逻辑与预设弹窗逐行对齐：

### 1. 新增快照变量（9062-9076 区域，`MCP_TRANSPORT_OPTIONS` 之前）

```js
/** MCP 服务器表单打开时的初始值快照（用于关闭时判断是否有未保存修改） */
let mcpFormInitial = {
    name: '', transport: 'stdio', command: '', args: '', env: '', url: '', headers: '',
};
```

### 2. openMCPServerForm 末尾记录快照（9286 行 `mcpFormEnabled = true;` 之后、显示对话框之前）

```js
// 记录表单初始快照，用于关闭时判断是否有未保存修改
mcpFormInitial = {
    name: nameInput.value,
    transport: mcpFormTransport,
    command: commandInput.value,
    args: argsInput.value,
    env: envInput.value,
    url: urlInput.value,
    headers: headersInput.value,
};
```

### 3. 新增 hasMCPServerFormChanges()（放在 closeMCPServerForm 之前）

```js
// 判断 MCP 服务器表单是否相对初始快照有修改
function hasMCPServerFormChanges() {
    const g = (id) => (document.getElementById(id)?.value ?? '');
    return g('mcpServerNameInput') !== mcpFormInitial.name
        || mcpFormTransport !== mcpFormInitial.transport
        || g('mcpServerCommandInput') !== mcpFormInitial.command
        || g('mcpServerArgsInput') !== mcpFormInitial.args
        || g('mcpServerEnvInput') !== mcpFormInitial.env
        || g('mcpServerUrlInput') !== mcpFormInitial.url
        || g('mcpServerHeadersInput') !== mcpFormInitial.headers;
}
```

### 4. closeMCPServerForm 增加 force 参数（9294-9307）

```js
/**
 * 关闭 MCP 服务器表单对话框
 * @param {boolean} force - true 时跳过未保存修改确认（保存成功后使用）
 */
async function closeMCPServerForm(force = false) {
    const dialog = document.getElementById('mcpServerFormDialog');
    if (!dialog || dialog.style.display === 'none') return;
    // force 必须是字面量 true 才跳过确认（防御：避免事件对象等 truthy 值误传入跳过确认）
    if (force !== true && hasMCPServerFormChanges()) {
        const ok = await showConfirmDialog('有未保存的修改，确定放弃并关闭吗？');
        if (!ok) return;
    }
    dialog.classList.remove('visible');
    // 等关闭过渡结束后隐藏 DOM；期间若重新打开（重新加回 visible）则不隐藏
    setTimeout(() => {
        if (!dialog.classList.contains('visible')) {
            dialog.style.display = 'none';
        }
    }, 220);
}
```

### 5. 保存成功调用改为跳过确认（9469）

```js
closeMCPServerForm(true);
```

其余 3 处调用（6489 Esc、9497 backdrop、9858 切面板）保持无参，自动走确认。

## 行为与边界

* 打开后无任何修改直接关闭 → 不弹确认，直接关

* 打开后修改任意字段（含切换传输方式、输入后清空）→ 关闭时弹出确认；点「取消」保持表单打开且修改保留

* 保存成功 → 跳过确认直接关闭（调用点 9469 传 `true`）

* 确认框弹出期间再按 Esc → 全局分支 6482-6484 检测 confirmDialog visible 直接 return，不会递归触发（现成机制）

* 确认框 z-index 10001 > MCP 表单 10000，确认框期间无法误触 backdrop

* `closeMCPServerForm` 变为 async：3 处无参调用点无需 await（fire-and-forget，原同步行为不变）

## 验证步骤

1. `cd frontend && npm run build` —— 构建通过。
2. `wails build` —— 重新编译 `build\bin\jot.exe`。
3. 重启应用手工验证：

   * 打开 MCP 新增表单 → 直接按 Esc / 点 backdrop / 切设置面板 → 不弹确认直接关闭

   * 输入名称或改动任意字段 → Esc / backdrop / 切面板 → 弹出「有未保存的修改」，取消后表单保持

   * 确认后表单关闭

   * 编辑已有服务器、改动后保存成功 → 直接关闭且无确认弹窗

