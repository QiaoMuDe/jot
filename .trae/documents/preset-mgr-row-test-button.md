# 预设配置管理列表：新增「测试连接」按钮

## Summary
在 AI 设置 →「对话连接 / 嵌入连接」的「配置预设管理」列表每一条目上，新增一个「测试」按钮，用于即时验证该预设的 API 连通性。逻辑与编辑对话框中的「测试」完全一致（调用 `App.TestAIConnection(baseURL, apiKey)`，校验地址非空、加载态、成功/失败通知）。纯前端改动，不涉及后端。

## Current State Analysis
- 预设管理列表由 `renderPresetMgrList(anchorRow)` 动态创建（`frontend/src/main.js:3316`），每行 DOM 由 `createPresetRowElement(p)`（`main.js:3380`）构建。
- 「对话连接」(`presetMgrBtn`) 与「嵌入连接」(`aiEmbedPresetMgrBtn`) 两入口共用 `renderPresetMgrList` → `createPresetRowElement`，因此只需改一处即对两处生效。
- 行数据 `p` 含 `id` / `name` / `base_url` / `api_key`（编辑按钮 `openEditProfileModal(p.id, p.name, p.base_url, p.api_key)` 直接使用，说明 `api_key` 已可作为明文用于测试）。
- 编辑对话框测试逻辑（`main.js:3019-3040`）：
  - baseURL 空 → `nm.show('请先填写 API 地址','warning')`
  - `setBtnLoading(btn, true)` 加载态（`main.js:2354`，签名 `(btn, loading)`，会自动禁用按钮并加 spinner）
  - `await App.TestAIConnection(baseURL, apiKey)` 返回 boolean
  - 成功 → green 提示「X 连接成功」；失败 → 提示地址/Key 问题；异常 → red 提示
  - `finally { setBtnLoading(btn, false) }`
- 后端已存在 `App.TestAIConnection(baseURL, apiKey string) (bool, error)`（`app.go`），复用即可，无需改动后端。

## Proposed Changes
仅修改 `frontend/src/main.js`。

### 1) 抽出可复用的行内测试函数
在 `createPresetRowElement` 之前新增顶层 `async` 函数 `testPresetProfileConnection(p, btn)`，逻辑与编辑页测试一致：

```js
// 测试预设配置连通性（管理列表行内按钮，逻辑与编辑对话框一致）
async function testPresetProfileConnection(p, btn) {
    const baseURL = (p.base_url || '').trim();
    const apiKey = (p.api_key || '').trim();
    if (!baseURL) {
        nm.show('请先填写 API 地址', 'warning');
        return;
    }
    setBtnLoading(btn, true);
    try {
        const ok = await window.go.main.App.TestAIConnection(baseURL, apiKey);
        if (ok) {
            nm.show(`「${p.name}」连接成功`, 'success');
        } else {
            nm.show(`「${p.name}」连接失败，请检查地址和 Key 是否匹配`, 'warning');
        }
    } catch (e) {
        nm.show(`「${p.name}」连接失败: ${e.message || e}`, 'error');
    } finally {
        setBtnLoading(btn, false);
    }
}
```

### 2) 在行操作区新增「测试」按钮
在 `createPresetRowElement` 的 `actions` 区中，`editBtn` 之前插入 `testBtn`（与其它按钮一致的内联样式，`width:52px` 防加载态文案抖动，与编辑对话框测试按钮对齐）：

```js
const testBtn = document.createElement('button');
testBtn.className = 'btn btn-sm btn-save';
testBtn.textContent = '测试';
testBtn.title = '测试该配置的 API 连通性';
testBtn.style.cssText = 'width:52px;white-space:nowrap;flex-shrink:0;';
testBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    testPresetProfileConnection(p, testBtn);
});
actions.appendChild(testBtn);
// 现有 editBtn / delBtn 依次 append（顺序：测试 → 编辑 → 删除）
```

## Assumptions & Decisions
- 复用编辑页同一接口 `App.TestAIConnection(baseURL, apiKey)`，不做任何后端改动。
- `p.api_key` 由 `GetProfiles` 返回、可直接用于连通性测试（依据：编辑对话框回填同名值）。
- 按钮位置放在「编辑」之前（操作顺序：测试 / 编辑 / 删除），与 MCP 条目操作区顺序（分享/测试/编辑/删除）风格一致。
- 保持行内按钮固定宽度 `width:52px`，加载态时 spinner 替换文案不会引起按钮宽度跳动。
- 不修改 `renderPresetMgrList`、`loadProfiles`、编辑对话框既有逻辑，避免回归。

## Verification
1. `cd frontend && npm run build` 成功。
2. `wails build` 生成新 `jot.exe`。
3. 手动验证：
   - 进入 设置 → AI 设置 → 对话连接，点「管理」展开列表，每行应出现「测试」按钮。
   - 点「测试」：加载态显示 spinner 且按钮禁用；成功显示绿色「X 连接成功」，失败显示警示，异常显示红色错误。
   - 嵌入连接入口的「管理」列表同样生效。
   - 搬到 MCP 服务器分组后页面布局不受影响。