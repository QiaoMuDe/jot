# 修复 AI 工具栏 toggle 切换到设置页后不同步

## 根因

`ai-chat.js` 中 3 个 toolbar toggle 的 click 事件处理顺序有误：

```
❌ 当前顺序：
1. enableXxx = xxxToggle.classList.toggle('active')     // 反 toolbar DOM
2. await window.saveSettings()                           // 读设置页 DOM → 仍是旧值，保存错
3. settingToggle.classList.toggle('active', enableXxx)   // 同步设置页 → 太晚了
```

`saveSettings()` 会从设置页 DOM 读取 `aiSettingXxxToggle` 的 `active` 类来决定 `ai_xxx_enabled`。但此时设置页 toggle 尚未更新（第 3 步还没执行），所以 `saveSettings()` 保存的是**旧值**。

用户切回设置页时 → `loadSettings()` 从后端读回旧值 → 覆盖了第 3 步的 DOM 同步 → 看起来就像没同步。

## 修复方案

### 改: ai-chat.js — 3 个 toolbar toggle 先同步再保存

将每个 handler 中的 `saveSettings()` 和 sync 代码交换位置：

```javascript
// 改前
enableThinking = searchToggle.classList.toggle('active');
try { await window.saveSettings(); } catch (_) {}
const settingToggle = document.getElementById('aiSettingSearchToggle');
if (settingToggle) settingToggle.classList.toggle('active', enableThinking);

// 改后
enableThinking = searchToggle.classList.toggle('active');
// 先同步设置页 toggle，再保存（保证 saveSettings 读到最新值）
const settingToggle = document.getElementById('aiSettingSearchToggle');
if (settingToggle) settingToggle.classList.toggle('active', enableThinking);
try { await window.saveSettings(); } catch (_) {}
```

同样处理 webSearchToggle (687-693) 和 cardRecallToggle (701-707)。

## 影响范围

仅 `frontend/src/js/ai-chat.js`，每处只调换 2 行顺序，不影响逻辑。

## 验证

1. AI 对话页开启卡片召回 → 切到设置页 → 卡片召回开关显示为开
2. AI 对话页关闭深度思考 → 切到设置页 → 深度思考开关显示为关
3. 设置页修改 → 切到 AI 对话页 → 工具栏同步
4. 重启应用 → 状态保持一致
