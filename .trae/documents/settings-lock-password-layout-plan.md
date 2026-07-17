# 锁屏密码设置卡片三栏布局改造计划

## 总结
将"锁屏密码"卡片中的两个设置项改为三栏布局：标签在左、描述在中间、控件在右。

## 当前状态分析

**Row 1 — 启用密码**（index.html:719-727）
```html
<div class="ai-setting-item">
    <span class="ai-setting-label">启用密码</span>
    <div class="ai-setting-toggle-row" id="screenLockToggleLine">
        <span class="ai-setting-toggle-desc">启动应用时需要输入密码</span>
        <div class="ai-chat-toggle-switch" id="screenLockToggle">
            <div class="ai-chat-toggle-knob"></div>
        </div>
    </div>
</div>
```
- 使用旧式 `.ai-setting-toggle-row` 容器，描述和开关在同一行
- 点击事件绑定在 `screenLockToggleLine` 上

**Row 2 — 设置密码**（index.html:728-744）
```html
<div class="ai-setting-item screen-lock-pwd-item" id="screenLockPasswordRow" style="display:none;">
    <span class="ai-setting-label">设置密码</span>
    <div class="ai-setting-control" style="max-width:380px;">
        <input type="password" id="screenLockPasswordInput" ... />
        <button id="screenLockPwdToggleBtn">👁</button>
    </div>
</div>
```
- 使用 `max-width:380px` 限制控制组宽度，与右侧未对齐
- 无描述文本

**JS 引用**（main.js）
- `screenLockToggleLine` — 点击切换锁屏开关（~2256）
- `screenLockToggle` — 开关 DOM 元素（~2260）
- `screenLockPasswordRow` — 密码行显示/隐藏（~2263）
- `screenLockPasswordInput` — 密码输入框（~2282）
- `screenLockPwdToggleBtn` — 密码显示/隐藏按钮（~2296）
- `reloadSettings`（~7912-7926）— 加载锁屏开关状态和密码提示

## 目标布局

```
Row 1: [启用密码]  [启动应用时需要输入密码]  [toggle switch ── → 右]
Row 2: [设置密码]  [────────── input ──────────] [👁]  (占满整行)
```

## 改动内容

### 1. index.html — 锁屏密码行结构调整

**Row 1（启用密码）:**
- 移除 `.ai-setting-toggle-row#screenLockToggleLine` 容器
- 将 `.ai-setting-toggle-desc` 改为 `.font-setting-desc`，内容不变
- 将 toggle 开关放入 `.ai-setting-control[style="flex:none;margin-left:auto;"]`

```html
<div class="ai-setting-item">
    <span class="ai-setting-label">启用密码</span>
    <span class="font-setting-desc">启动应用时需要输入密码</span>
    <div class="ai-setting-control" style="flex:none;margin-left:auto;">
        <div class="ai-chat-toggle-switch" id="screenLockToggle">
            <div class="ai-chat-toggle-knob"></div>
        </div>
    </div>
</div>
```

**Row 2（设置密码）:**
- `.ai-setting-control` 的 `max-width:380px` → `flex:1;max-width:none;`
- 移除 `.screen-lock-pwd-item` 类引用（CSS 中检查是否还有相关样式）

```html
<div class="ai-setting-item" id="screenLockPasswordRow" style="display:none;">
    <span class="ai-setting-label">设置密码</span>
    <div class="ai-setting-control" style="flex:1;max-width:none;">
        <input type="password" id="screenLockPasswordInput" class="settings-input" placeholder="输入锁屏密码后失焦自动保存" />
        <button id="screenLockPwdToggleBtn" class="btn btn-sm btn-save" style="flex-shrink:0;width:40px;height:36px;display:flex;align-items:center;justify-content:center;padding:0;" title="显示/隐藏">
            ...
        </button>
    </div>
</div>
```

### 2. main.js — 点击事件迁移

**将 click 事件从 `screenLockToggleLine` 迁移到 `screenLockToggle`：**

当前（~2256-2278）：
```js
const screenLockToggleLine = document.getElementById('screenLockToggleLine');
screenLockToggleLine.addEventListener('click', ...);
```

改为（~2256-2278）：
```js
const screenLockToggle = document.getElementById('screenLockToggle');
if (screenLockToggle) {
    screenLockToggle.addEventListener('click', async (e) => {
        e.stopPropagation();
        const isActive = screenLockToggle.classList.toggle('active');
        const pwdRow = document.getElementById('screenLockPasswordRow');
        if (pwdRow) {
            pwdRow.style.display = isActive ? '' : 'none';
        }
        if (!isActive) {
            const pwdInput = document.getElementById('screenLockPasswordInput');
            if (pwdInput) pwdInput.value = '';
        } else {
            const pwdInput = document.getElementById('screenLockPasswordInput');
            if (pwdInput) pwdInput.placeholder = '输入锁屏密码后失焦自动保存';
        }
        await saveSettings();
        nm.show(isActive ? '锁屏密码已启用' : '锁屏密码已关闭', 'info');
    });
}
```

其余 JS 代码不变（`screenLockPasswordInput` change 事件、`screenLockPwdToggleBtn` 点击事件、`reloadSettings` 中的锁屏加载逻辑均不受影响）。

### 3. CSS — 清理/补充

**settings-panel.css:**
- `.ai-setting-item .ai-setting-control .ai-chat-toggle-switch` 系列样式已存在（之前添加的），**无需新增**
- `screen-lock-pwd-item` 类在 CSS 中搜索确认是否可移除

## 验证步骤
1. 运行 `npx vite build` 确认前端构建无错误
2. 打开设置页 → 锁屏密码卡片
3. 点击 toggle 开关 → 锁屏密码行显示/隐藏、通知正确
4. 输入密码失焦 → 提示已保存、占位符更新
5. 关闭锁屏 → 密码行隐藏、密码被清空
6. 刷新页面 → 状态正确恢复
