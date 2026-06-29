# Plan: 联网搜索 → 深度思考

## 概要

将 AI 对话中的「联网搜索」toggle 功能替换为「深度思考」toggle，控制是否启用 DeepSeek 思考模式（`thinking: { type: enabled/disabled }`）。

## 当前状态分析

`SearchEnabled bool \`json:"search\_enabled,omitempty"\``被发送到`POST /v1/chat/completions`，但 DeepSeek 的 `  search\_enabled\` 参数仅对特定「联网搜索版」应用生效，用户普通 API Key 不支持该功能。参考文档：<https://api-docs.deepseek.com/zh-cn/guides/thinking_mode>

### 技术链路

```
前端 toggle → localStorage('ai_search_enabled') → CallAIStream(messages, enableSearch)
    → chatRequest.SearchEnabled → JSON "search_enabled": true/false
```

## 改动文件

### 1. `internal/services/ai_service.go` — 后端

| 改动                                                                | 位置                  |
| ----------------------------------------------------------------- | ------------------- |
| 删除 `SearchEnabled bool` 字段                                        | `chatRequest` 结构体   |
| 新增 `Thinking *thinkingParam` 字段 + `thinkingParam` 内嵌类型            | `chatRequest` 结构体附近 |
| 修改 `CallAIStream` 签名 `enableSearch bool` → `thinkingEnabled bool` | 函数定义                |
| 构建请求 body 时，根据 `thinkingEnabled` 设置 `Thinking`                    | 函数内 marshaling 前    |

**JSON 输出时机：**

* `thinkingEnabled: true` → `"thinking": {"type": "enabled"}`

* `thinkingEnabled: false` → `"thinking": {"type": "disabled"}`

* 始终显式发送

### 2. `app.go` — 绑定层

| 改动                                                | 位置                        |
| ------------------------------------------------- | ------------------------- |
| 修改签名 `enableSearch bool` → `thinkingEnabled bool` | `CallAIStream` binding 方法 |
| 透传给 `aiService.CallAIStream`                      | 函数体内                      |

### 3. `frontend/index.html` — 工具栏 + 设置页

| 区域           | 改动                                            |
| ------------ | --------------------------------------------- |
| 工具栏 toggle 行 | 文本「联网搜索」→「深度思考」，svg 放大镜图标 → 大脑/灯泡图标           |
| 设置页 toggle 行 | 文本「联网搜索」→「深度思考」，描述「发送消息时启用联网搜索」→「发送消息时启用深度思考」 |

**不改动结构**（CSS class/id 暂时保持不变，只改文本内容）

### 4. `frontend/src/css/components/ai-chat.css` — 工具栏样式（不修改）

CSS class 名不更改，`.ai-chat-search-toggle` 等选择器继续复用，只改 HTML 文本内容。

### 5. `frontend/src/js/ai-chat.js` — 前端核心

| 改动                                                               | 位置                     |
| ---------------------------------------------------------------- | ---------------------- |
| 变量 `enableSearch` → `enableThinking`                             | 模块顶部                   |
| localStorage key `'ai_search_enabled'` → `'ai_thinking_enabled'` | 初始化 + 切换事件             |
| `searchToggle` 变量命名不变（只改逻辑语义）                                    | —                      |
| `document.getElementById('aiChatSearchToggle')` id 不变            | —                      |
| 切换事件中的同步代码 `enableSearch = ...` → `enableThinking = ...`         | click handler          |
| 传给 `CallAIStream` 的参数 `enableSearch` → `enableThinking`          | `startStreaming()` 调用处 |

### 6. `frontend/src/main.js` — 设置页同步

| 改动                                                               | 位置      |
| ---------------------------------------------------------------- | ------- |
| localStorage key `'ai_search_enabled'` → `'ai_thinking_enabled'` | 加载 + 切换 |
| toggle click handler 内保存逻辑同步                                     | 事件绑定代码  |

## 假设与决策

1. 由于 CSS class 和 DOM id 不变，CSS 和 HTML 结构**不做任何改动**，只改文本内容和 JS 逻辑语义
2. 旧用户 localStorage 中的 `ai_search_enabled` 值被忽略（新 key `ai_thinking_enabled`），默认 thinking 为关闭
3. DeepSeek 思考模式默认启用，但为了前端行为明确，toggle ON 时显式发送 `type: enabled`
4. `thinking` 参数通过 JSON body 直接发送，`extra_body` 包装不需要（非 OpenAI SDK 场景）
5. 思考模式开启后 response 中的 `reasoning_content` 现有前端逻辑已正确处理

## 验证清单

* [ ] Vite build 零错误

* [ ] Wails build 零错误

* [ ] 工具栏 toggle 文本显示「深度思考」

* [ ] 设置页 toggle 文本/描述显示「深度思考」

* [ ] 开启 toggle → JSON body 含 `"thinking": {"type": "enabled"}`

* [ ] 关闭 toggle → JSON body 含 `"thinking": {"type": "disabled"}`

* [ ] 关闭思考模式后 response 不再有 `reasoning_content`（模型不输出思维链）

* [ ] 开启思考模式后思维链正常显示

* [ ] 工具栏 ↔ 设置页状态双向同步

* [ ] 状态持久化到 localStorage（`ai_thinking_enabled`）

