# 修复设置页 localStorage 加载和预设配置问题

## 问题分析

### 问题 1：设置页从 localStorage 加载数据

AI 设置页的部分开关/输入项在从后端读取不到值时，会回退到 `localStorage` 读取。这是早期版本迁移留下的兼容代码，现在后端数据库已经是唯一持久化存储，localStorage 回退反而会导致数据不一致（如重置后意外恢复旧值）。

涉及 `loadAISettings()` (`frontend/src/main.js`) 中的 4 项：

* `ai_thinking_enabled`（深度思考）— 第 1686-1691 行

* `ai_web_search_enabled`（联网搜索）— 第 1704-1708 行

* `ai_card_recall_enabled`（卡片召回）— 第 1718-1722 行

* `ai_card_recall_limit`（卡片召回数量）— 第 1729-1738 行

### 问题 2：Tavily API Key 输入框在重置后未清空

`loadAISettings()`（第 1696-1699 行）中，Tavily Key 输入框只在 `cfg.tavily_api_key` 非空时才赋值：

```javascript
if (tavilyKey && cfg.tavily_api_key) {
    tavilyKey.value = cfg.tavily_api_key;
}
```

重置后后端返回空字符串，条件不成立，输入框**保留旧值**。其他字段（如 `aiBaseURL`）使用 `|| ''` 始终赋值，Tavily Key 是唯一一个条件性赋值的字段，因此重置后只有它会残留。

### 问题 3：预设配置选择下拉不刷新

后端 `SaveAIConfig()` 在无预设时会自动创建"默认配置"预设。但 URL 和 Key 的 `change` 事件处理（第 2081-2105 行）只调用了 `SaveAIConfig()`，**没有调用** **`loadProfiles()`** **刷新下拉菜单**。管理列表（点"管理"时重新获取数据）能看到，但选择下拉（没有刷新）仍显示"无预设配置"。

### 问题 3：无预设时下拉菜单可点击

预设触发器 `presetTrigger` 的点击事件（第 2296-2300 行）没有判断是否有预设数据，即使 `profiles.length === 0` 时点击也会展开一个空的下拉列表，交互无意义。

***

## 变更计划

### 变更 1：移除 localStorage 回退

**文件：`frontend/src/main.js`**

修改 `loadAISettings()` 中的 4 处 localStorage 回退，改为纯后端读取 + 使用默认值：

**① 深度思考（第 1682-1693 行）**

```javascript
// 修改前
let enabled = false;
try {
    const val = await window.go.main.App.GetSetting('ai_thinking_enabled');
    if (val !== '') enabled = val === 'true';
    else enabled = localStorage.getItem('ai_thinking_enabled') === 'true';
} catch (_) {
    enabled = localStorage.getItem('ai_thinking_enabled') === 'true';
}

// 修改后
let enabled = false;
try {
    const val = await window.go.main.App.GetSetting('ai_thinking_enabled');
    enabled = val === 'true';
} catch (_) { /* 保持默认 false */ }
```

**② 联网搜索（第 1700-1711 行）**

```javascript
// 修改前
let webSearchEnabled = false;
try {
    const val = await window.go.main.App.GetSetting('ai_web_search_enabled');
    if (val !== '') webSearchEnabled = val === 'true';
    else webSearchEnabled = localStorage.getItem('ai_web_search_enabled') === 'true';
} catch (_) {
    webSearchEnabled = localStorage.getItem('ai_web_search_enabled') === 'true';
}

// 修改后
let webSearchEnabled = false;
try {
    const val = await window.go.main.App.GetSetting('ai_web_search_enabled');
    webSearchEnabled = val === 'true';
} catch (_) { /* 保持默认 false */ }
```

**③ 卡片召回启用（第 1713-1725 行）**

```javascript
// 修改前
let cardRecallEnabled = false;
try {
    const val = await window.go.main.App.GetSetting('ai_card_recall_enabled');
    if (val !== '') cardRecallEnabled = val === 'true';
    else cardRecallEnabled = localStorage.getItem('ai_card_recall_enabled') === 'true';
} catch (_) {
    cardRecallEnabled = localStorage.getItem('ai_card_recall_enabled') === 'true';
}

// 修改后
let cardRecallEnabled = false;
try {
    const val = await window.go.main.App.GetSetting('ai_card_recall_enabled');
    cardRecallEnabled = val === 'true';
} catch (_) { /* 保持默认 false */ }
```

**④ 卡片召回数量（第 1726-1740 行）**

```javascript
// 修改前
try {
    const val = await window.go.main.App.GetSetting('ai_card_recall_limit');
    if (val) {
        cardRecallLimit.value = val;
    } else {
        const saved = localStorage.getItem('ai_card_recall_limit');
        if (saved) cardRecallLimit.value = saved;
    }
} catch (_) {
    const saved = localStorage.getItem('ai_card_recall_limit');
    if (saved) cardRecallLimit.value = saved;
}

// 修改后
try {
    const val = await window.go.main.App.GetSetting('ai_card_recall_limit');
    if (val) cardRecallLimit.value = val;
} catch (_) { /* 保持 HTML 默认值 */ }
```

> **说明**：这些设置的**保存逻辑**中仍保留了 `localStorage.setItem()` 调用（第 2114、2143、2167 等行），这是为了向后兼容旧版本。只移除读取时的回退，保留写入时的兼容写入，不会造成破坏。

### 变更 2：URL/Key 保存后刷新预设下拉

**文件：`frontend/src/main.js`**

在 `aiBaseURL` 和 `aiAPIKey` 的 `change` 事件处理末尾，追加 `loadProfiles()` 调用：

**URL change 事件（第 2081-2092 行）**

```javascript
els.aiBaseURL.addEventListener('change', async () => {
    const url = els.aiBaseURL.value.trim();
    try {
        const cfg = await window.go.main.App.GetAIConfig();
        cfg.base_url = url;
        cfg.provider = getActiveProvider();
        await window.go.main.App.SaveAIConfig(cfg);
        nm.show('AI 配置已保存', 'success');
        await loadProfiles();  // ← 新增：刷新预设下拉（可能自动创建了默认配置）
    } catch (e) {
        nm.show('保存配置失败: ' + e, 'error');
    }
});
```

**Key change 事件（第 2094-2105 行）**

```javascript
els.aiAPIKey.addEventListener('change', async () => {
    const key = els.aiAPIKey.value.trim();
    try {
        const cfg = await window.go.main.App.GetAIConfig();
        cfg.api_key = key;
        cfg.provider = getActiveProvider();
        await window.go.main.App.SaveAIConfig(cfg);
        nm.show('AI 配置已保存', 'success');
        await loadProfiles();  // ← 新增：刷新预设下拉（可能自动创建了默认配置）
    } catch (e) {
        nm.show('保存配置失败: ' + e, 'error');
    }
});
```

### 变更 3：Tavily Key 输入框改为始终赋值

**文件：`frontend/src/main.js`** 第 1696-1699 行

```javascript
// 修改前
if (tavilyKey && cfg.tavily_api_key) {
    tavilyKey.value = cfg.tavily_api_key;
}

// 修改后
if (tavilyKey) {
    tavilyKey.value = cfg.tavily_api_key || '';
}
```

### 变更 4：无预设时禁用下拉菜单

**文件：`frontend/src/main.js`**

修改 `presetTrigger` 的点击事件（第 2296-2300 行），先判断是否有预设数据：

```javascript
presetTrigger.addEventListener('click', async (e) => {
    e.stopPropagation();
    // 无预设时禁用下拉展开
    const profiles = await window.go.main.App.GetProfiles();
    if (profiles.length === 0) return;
    presetTrigger.classList.toggle('open');
    presetDropdown.classList.toggle('open');
});
```

***

## 变更文件清单

| 文件                                      | 变更内容                           |
| --------------------------------------- | ------------------------------ |
| `frontend/src/main.js`（\~第 1686-1738 行） | 移除 4 处 localStorage 回退读取       |
| `frontend/src/main.js`（\~第 2090、2102 行） | URL/Key 保存后追加 `loadProfiles()` |
| `frontend/src/main.js`（\~第 1696-1699 行） | Tavily Key 改为始终赋值，重置后清空        |
| `frontend/src/main.js`（\~第 2296 行）      | 无预设时禁用下拉展开                     |

***

## 验证步骤

1. 打开 AI 设置页，确认深度思考/联网搜索/卡片召回开关正常从后端读取
2. 在 localStorage 中设置旧值（如 `ai_thinking_enabled=true`），确认页面加载后不读取该值
3. 在 URL 输入框输入新地址，确认下拉选择中可看到自动创建的"默认配置"
4. 删除所有预设配置后，点击预设下拉菜单确认不会展开

