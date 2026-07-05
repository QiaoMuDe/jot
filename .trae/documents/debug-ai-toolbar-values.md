# AI 工具栏 toggle 值传递调试方案

## 静态分析结论

通读全部源码后，`enableThinking / enableWebSearch / enableCardRecall` 的值传递链路**无误**：

| 环节 | 位置 | 状态 |
|------|------|------|
| 变量声明 | `ai-chat.js:51/55/59`，初始 `false` | ✅ |
| `loadSettings` 设置 toolbar DOM | `main.js:7034-7041` | ✅ |
| `initAIChat` 从 DOM 读入变量 | `ai-chat.js:232/236/240` | ✅ |
| 用户点击 toolbar 时更新变量 | `ai-chat.js:673/687/701` | ✅ |
| 发送消息时传入 `CallAIStream` | `ai-chat.js:2001` | ✅ |
| 后端接收并使用 3 个 bool | `app.go:763/933/965/971/1002` + 搜索/召回逻辑 | ✅ |
| `saveSettings` 读设置页 DOM | `main.js:7098-7100` | ✅ |
| 后端 `SaveAllSettings` 存库 | `types.go:145-147` | ✅ |
| `loadSettings` 从后端读回 | `types.go:93-95` | ✅ |

用户"在 AI 助手模块启用"这一操作确保变量在内存中为 `true`。发送消息时直接引用该变量，不经过后端。**理论上功能应该正常工作。**

## 如果实际不正常，可能的原因

**1. 配置问题**（最常见）
- 联网搜索需要 Tavily API Key（后端 `app.go:800` 检查 `cfg.TavilyAPIKey != ""`，为空则静默跳过）
- 卡片召回需要笔记库中存在与问题相关的笔记内容
- 深度思考需要调用模型支持 `enable_thinking` 参数（如 Qwen3），且请求正确传递该参数 (`openai.go:41-43`)

**2. 变量被后续调用覆盖**
- 用户在 toggle 之后、发送之前，是否调用了别的功能导致 `initAIChat` 被重入？
- 可通过加 log 排查运行时是否存在变量被重置

## 建议的调试步骤

### 1. 在点击发送时添加 `console.log`

在 `ai-chat.js:2001` 之前加一行：

```javascript
console.log('[AI Send] enableThinking=%o enableWebSearch=%o enableCardRecall=%o',
            enableThinking, enableWebSearch, enableCardRecall);
```

### 2. 在 toolbar click handler 中添加 `console.log`

在 `ai-chat.js:673/687/701` 处 `enableXxx = ...` 之后各加：

```javascript
console.log('[Toolbar] enableThinking=%o', enableThinking);
```

### 3. 在 `saveSettings` 中添加 `console.log`

在 `main.js:7105` 之前加：

```javascript
console.log('[saveSettings] cfg=%o', cfg);
```

### 4. 运行时观察

1. 打开浏览器开发者工具 (F12) → Console
2. 在 AI 助手模块点击 toggle 启用所有三个功能
3. 发送一条消息
4. 观察 Console 输出，确认：
   - `[Toolbar]` 显示 `enableThinking=true`
   - `[saveSettings]` 显示 `ai_thinking_enabled: true, ai_web_search_enabled: true, ai_card_recall_enabled: true`
   - `[AI Send]` 显示 `enableThinking=true enableWebSearch=true enableCardRecall=true`

如果 log 显示值正确，说明值传递链路没问题，需要排查配置（Tavily Key）或模型能力（是否支持 thinking）。
