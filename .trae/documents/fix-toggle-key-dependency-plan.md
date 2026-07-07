# 修复搜索开关与密钥的实时联动逻辑

## 问题描述

两个需要修复的行为：

1. **清空密钥 → 立即关闭并禁用对应开关**：当用户清空 Tavily API Key 或知乎 Access Secret 输入框后保存，对应的搜索开关应立即关闭（移除 active）并禁用（添加 disabled 样式），而不需要重新加载设置页。

2. **点击启用开关 → 检查密钥是否存在**：当用户点击搜索开关尝试开启时，应检查对应的密钥是否已配置，如果未配置则弹出通知提示并取消开启操作。

## 现状

当前只在 `loadSettings()` 中根据数据库值一次性设置禁用状态。密钥输入框的 `change` 事件处理器只调用 `saveSettings()`，没有后续的禁用逻辑。搜索开关点击处理器在有 `disabled` class 时直接 `return`，不会弹出任何通知。

## 修改方案

### 修改 1：CSS — 移除 `pointer-events: none`

**文件：** `frontend/src/css/components/settings-panel.css`（第 692-696 行）

移除 `.ai-setting-toggle-row.disabled` 中的 `pointer-events: none`，使被禁用的开关行仍然能捕获点击事件，从而触发 JS 通知。

```css
/* 修改前 */
.ai-setting-toggle-row.disabled {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
}

/* 修改后 */
.ai-setting-toggle-row.disabled {
    opacity: 0.5;
    cursor: not-allowed;
}
```

**文件：** `frontend/src/css/components/ai-chat.css`（第 920-923 行）

同理，移除 `.ai-chat-search-source-item.disabled` 中的 `pointer-events: none`。

```css
/* 修改前 */
.ai-chat-search-source-item.disabled {
    opacity: 0.4;
    cursor: not-allowed;
    pointer-events: none;
}

/* 修改后 */
.ai-chat-search-source-item.disabled {
    opacity: 0.4;
    cursor: not-allowed;
}
```

---

### 修改 2：密钥 `change` 事件处理器 — 清空时立即禁用开关

**文件：** `frontend/src/main.js`

#### 2a. Tavily API Key change 事件（第 1946-1953 行）

当前代码：
```javascript
tavilyKey.addEventListener('change', async () => {
    await saveSettings();
    nm.show(tavilyKey.value.trim() ? 'Tavily API Key 已保存' : 'Tavily API Key 已清除', 'success');
});
```

改为：
```javascript
tavilyKey.addEventListener('change', async () => {
    if (!tavilyKey.value.trim()) {
        // 清空 Key → 禁用 Tavily 搜索开关
        const line = document.getElementById('aiSettingTavilySearchLine');
        if (line) {
            line.classList.add('disabled');
            const toggle = line.querySelector('.ai-chat-toggle-switch');
            if (toggle) toggle.classList.remove('active');
        }
        // 同步聊天栏复选框
        const cb = document.getElementById('aiChatTavilySearch');
        if (cb) {
            cb.disabled = true;
            cb.checked = false;
            const label = cb.closest('.ai-chat-search-source-item');
            if (label) label.classList.add('disabled');
        }
    }
    await saveSettings();
    nm.show(tavilyKey.value.trim() ? 'Tavily API Key 已保存' : 'Tavily API Key 已清除', 'success');
});
```

**关键点：** 在 `saveSettings()` 之前先修改 DOM 状态，这样 `saveSettings()` 读取 DOM 时会读到正确的 toggle 状态（`active: false`）。

#### 2b. 知乎 Access Secret change 事件（第 1997-2003 行）

当前代码：
```javascript
zhihuSecretInput.addEventListener('change', async () => {
    await saveSettings();
    nm.show(zhihuSecretInput.value.trim() ? 'Access Secret 已保存' : 'Access Secret 已清除', 'success');
});
```

改为：
```javascript
zhihuSecretInput.addEventListener('change', async () => {
    if (!zhihuSecretInput.value.trim()) {
        // 清空 Secret → 禁用知乎搜索和知乎全网搜索开关
        ['aiSettingZhihuSearchLine', 'aiSettingZhihuGlobalSearchLine'].forEach(lineId => {
            const line = document.getElementById(lineId);
            if (line) {
                line.classList.add('disabled');
                const toggle = line.querySelector('.ai-chat-toggle-switch');
                if (toggle) toggle.classList.remove('active');
            }
        });
        // 同步聊天栏复选框
        ['aiChatZhihuSearch', 'aiChatZhihuGlobalSearch'].forEach(cbId => {
            const cb = document.getElementById(cbId);
            if (cb) {
                cb.disabled = true;
                cb.checked = false;
                const label = cb.closest('.ai-chat-search-source-item');
                if (label) label.classList.add('disabled');
            }
        });
    }
    await saveSettings();
    nm.show(zhihuSecretInput.value.trim() ? 'Access Secret 已保存' : 'Access Secret 已清除', 'success');
});
```

---

### 修改 3：搜索开关点击处理器 — 开启前验证密钥

**文件：** `frontend/src/main.js`

三个开关（第 2030-2067 行）做相同的修改模式：

#### 3a. 知乎搜索切换（第 2030-2041 行）

当前：
```javascript
settingZhihuSearchLine.addEventListener('click', async () => {
    if (settingZhihuSearchLine.classList.contains('disabled')) return;
    const toggleSwitch = document.getElementById('aiSettingZhihuSearchToggle');
    if (!toggleSwitch) return;
    const isActive = toggleSwitch.classList.toggle('active');
    await saveSettings();
    nm.show(isActive ? '知乎搜索已开启' : '知乎搜索已关闭', isActive ? 'success' : 'info');
});
```

改为：
```javascript
settingZhihuSearchLine.addEventListener('click', async () => {
    const toggleSwitch = document.getElementById('aiSettingZhihuSearchToggle');
    if (!toggleSwitch) return;
    const willBeActive = !toggleSwitch.classList.contains('active');
    // 尝试开启 → 检查是否有知乎 Access Secret
    if (willBeActive) {
        const secret = document.getElementById('aiZhihuAccessSecret')?.value || '';
        if (!secret.trim()) {
            nm.show('请先在设置中配置知乎 Access Secret', 'warning');
            return;
        }
    }
    toggleSwitch.classList.toggle('active');
    await saveSettings();
    nm.show(willBeActive ? '知乎搜索已开启' : '知乎搜索已关闭', willBeActive ? 'success' : 'info');
});
```

#### 3b. 知乎全网搜索切换（第 2043-2054 行）

同样的模式 — 开启前检查 `aiZhihuAccessSecret`。

#### 3c. Tavily 搜索切换（第 2056-2067 行）

同样的模式 — 开启前检查 `aiTavilyApiKey`。

**三个处理器的行为一致：**
- 开启前（`willBeActive === true`）：读取当前密钥输入框的值，为空则弹窗提示并 `return`（不切换 toggle）
- 关闭前（`willBeActive === false`）：始终允许关闭，无需检查

注：去掉了原来的 `if (el.classList.contains('disabled')) return;` 检查，因为现在由更细粒度的密钥值检查替代。

---

## 涉及文件清单

| 文件 | 修改内容 |
|------|----------|
| `frontend/src/css/components/settings-panel.css` | 移除 `.ai-setting-toggle-row.disabled` 的 `pointer-events: none` |
| `frontend/src/css/components/ai-chat.css` | 移除 `.ai-chat-search-source-item.disabled` 的 `pointer-events: none` |
| `frontend/src/main.js` | ① Tavily Key change 事件增加清空后的禁用逻辑；② 知乎 Secret change 事件增加清空后的禁用逻辑；③ 三个搜索开关点击处理器改为开启前检查密钥值 |

## 验证步骤

1. **清空 Tavily Key** → 保存 → Tavily 搜索开关自动关闭、变灰 → 聊天栏 Tavily 复选框也变灰不可选
2. **点击已禁用的 Tavily 开关** → 弹出警告通知「请先配置 Tavily API Key」，开关不切换
3. **清空知乎 Secret** → 保存 → 知乎搜索和全网搜索开关均自动关闭变灰 → 聊天栏对应复选框变灰
4. **点击已禁用的知乎搜索开关** → 弹出警告通知「请先配置知乎 Access Secret」
5. **填入 Tavily Key 并保存** → 重新加载设置页（目前依赖 `loadSettings` 恢复启用）
6. **有密钥时正常开启/关闭开关** → 行为不变
