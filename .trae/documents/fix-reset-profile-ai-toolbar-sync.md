# 修复重置出厂设置后 Profile 未清空 + AI 工具栏开关未同步

## 现状分析

### Issue 1: 重置出厂设置后 preset 下拉仍显示旧选项

**流程**：`resetDatabase()` (data-management.js:164-206) → 后端 DropTable → AutoMigrate → InitDefaultSettings → reconnectDB → 前端 `loadDataStats()` → `window.loadNotebooks()` → `reloadSettings()` (即 `window.loadSettings()`)

**问题**：`loadProfiles()` 从未在 reset 流程中调用，且前端 `loadProfiles` 函数未暴露到 `window`（main.js:7099-7113 的 window 导出列表中没有它）。

**效果**：后端 `APIProfile` 表已清空，但前端的 profile 下拉 DOM 仍然显示旧的选项。

### Issue 2: AI 工具栏开关状态未同步

**初始加载时**：

* `loadSettings()` (main.js:6937-7057) 只同步了设置页的 3 个 AI toggle DOM（aiSettingSearchToggle 等），**没有同步 AI 工具栏的 3 个 toggle**（aiChatSearchToggle 等）

* `initAIChat()` (ai-chat.js:208) 在 `loadSettings()` 之后执行，但 `enableThinking/enableWebSearch/enableCardRecall` 三变量硬编码为 `false`（ai-chat.js:232/236/240），不读 DOM

**交互时**（已正常工作）：

* 设置页 toggle → 工具栏 toggle 同步（main.js \~1892-1996）

* 工具栏 toggle → 设置页 toggle 同步（ai-chat.js \~671-709）

## 变更方案

### 改 1: main.js — 暴露 loadProfiles 到 window

**文件**：`d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`

**位置**：约 7112 行附近（window 导出区块）

**变更**：在 `window.loadSettings = loadSettings` 后添加一行：

```javascript
window.loadProfiles = loadProfiles;
```

### 改 2: data-management.js — resetDatabase 末尾调用 loadProfiles

**文件**：`d:\峡谷\Dev\本地项目\jot\frontend\src\js\data-management.js`

**位置**：约 194 行（`reloadSettings()` 调用之后）

**变更**：在 `reloadSettings();` 后添加：

```javascript
window.loadProfiles();
```

### 改 3: main.js — loadSettings 中同步 AI 工具栏 toggle

**文件**：`d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`

**位置**：约 7036 行（设置页 toggle 同步代码之后）

**变更**：在 3 个 `aiSettingXxxToggle` 同步之后，追加同步 AI 工具栏的 3 个 toggle：

```javascript
// 同步 AI 聊天工具栏 toggle
const chatSearchToggle = document.getElementById('aiChatSearchToggle');
if (chatSearchToggle) chatSearchToggle.classList.toggle('active', cfg.ai_thinking_enabled);

const chatWebSearchToggle = document.getElementById('aiChatWebSearchToggle');
if (chatWebSearchToggle) chatWebSearchToggle.classList.toggle('active', cfg.ai_web_search_enabled);

const chatCardRecallToggle = document.getElementById('aiChatCardRecallToggle');
if (chatCardRecallToggle) chatCardRecallToggle.classList.toggle('active', cfg.ai_card_recall_enabled);
```

### 改 4: ai-chat.js — 从 DOM 读取初始 toggle 状态

**文件**：`d:\峡谷\Dev\本地项目\jot\frontend\src\js\ai-chat.js`

**位置**：约 232/236/240 行（`initAIChat` 函数中）

**变更**：将：

```javascript
enableThinking = false;
```

改为：

```javascript
enableThinking = searchToggle?.classList.contains('active') || false;
```

同理处理 `enableWebSearch` 和 `enableCardRecall`。

**原理**：`loadSettings()` 在 `initAIChat()` 之前执行，已把正确状态同步到 DOM toggle 的 `active` 类上。此时从 DOM 读取可避免硬编码。

## 影响范围

| 文件                   | 改动量                                | 风险                             |
| -------------------- | ---------------------------------- | ------------------------------ |
| `main.js`            | +4 行（window 导出 + 3 行 toolbar sync） | 低，纯新增                          |
| `data-management.js` | +1 行                               | 低，纯新增                          |
| `ai-chat.js`         | 3 行修改（false → 读 DOM）               | 中，需确认 `initAIChat` 执行时 DOM 已渲染 |

## 验证步骤

1. `go vet ./...` 通过
2. 重置出厂设置 → preset 下拉显示"无预设配置"
3. 设置页启用"深度思考" → 打开 AI 对话页，工具栏开关应同步为开启
4. AI 对话页开启"联网搜索" → 回到设置页，开关应同步为开启
5. 重新启动应用 → 设置和工具栏状态保持一致

