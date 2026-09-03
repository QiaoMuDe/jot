# AI 消息分叉功能方案

## 概述

在 AI 助手模块中，右键 AI 消息时通过"分叉"菜单项，复制该消息之前的所有消息到新会话，新会话标题自动递增编号。

## 当前状态分析

### 后端已有 API（无需改动）

| API                             | 用途                     |
| ------------------------------- | ---------------------- |
| `CreateAISession()`             | 创建新会话，返回 ID            |
| `RenameAISession(id, title)`    | 重命名会话（标题限制 20 字符）      |
| `LoadAISessionMessages(id)`     | 加载会话全部消息（ASC 顺序）       |
| `SaveAIMessages(id, messages)`  | 批量保存消息到指定会话            |
| `LoadSessionConfig(id)`         | 加载会话配置（模型、模式、引用笔记、技能等） |
| `SaveSessionConfig(id, config)` | 保存会话配置                 |

### SessionConfig 结构

```go
type SessionConfig struct {
    ModelName         string `json:"model_name"`
    EnableThinking    bool   `json:"enable_thinking"`
    ReferencedNotes   string `json:"referenced_notes"`
    EnabledSkills     string `json:"enabled_skills"`
    RoleplayNotes     string `json:"roleplay_notes"`
    RecallNotebookIDs string `json:"recall_notebook_ids"`
    Mode              string `json:"mode"`
}
```

### 前端关键状态

* `sessions` — 会话列表，每个元素有 `id`、`title` 等字段

* `activeSessionId` — 当前活跃会话 ID

* `_contextMsgEl` — 右键消息元素，通过 `dataset.msgId` 获取消息 ID

* `chatHistory` — 当前会话消息缓存（但可能未加载全部，需走后端 API）

### 消息元素

`addMessage()` 在 [L3644](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3644) 设置 `el.dataset.msgId = msgId`，右键菜单可通过 `_contextMsgEl.dataset.msgId` 获取选中消息 ID。

## 修改方案

### 文件：`frontend/src/js/ai-chat.js`

#### 1. 添加分叉图标常量

在文件顶部图标常量区域，新增分叉 SVG 图标（使用 git-branch 风格图标）。

#### 2. 右键菜单添加"分叉"项

**位置**: `showAiMsgContextMenu()` — [L4195-L4201](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4195-L4201)

在 assistant 角色的菜单项 `save` 之后、`regen` 之前，插入：

```javascript
items.push({ action: 'fork', label: '分叉' });
```

并在 `actionIcons` 映射中添加 `fork` 图标。

#### 3. 菜单点击处理新增 fork 分支

**位置**: 右键菜单点击事件监听 — [L1278 起](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1278)

在 `action === 'save'` 和 `action === 'regen'` 之间新增：

```javascript
} else if (action === 'fork') {
    forkSession();
}
```

#### 4. 新增 `forkSession()` 函数

核心逻辑：

```
 1. 获取选中消息 ID  →  _contextMsgEl?.dataset.msgId
 2. 获取当前会话标题 →  sessions.find(s => s.id === activeSessionId)?.title
 3. 检查前置条件：无消息 ID / 无标题 / 流式输出中 → 跳过
 4. 加载全部消息     →  window.go.main.App.LoadAISessionMessages(activeSessionId)
 5. 过滤消息         →  取 ID <= 选中消息 ID 的消息（保留 ASC 顺序）
 6. 加载会话配置     →  window.go.main.App.LoadSessionConfig(activeSessionId)
 7. 计算新标题       →  parseForkTitle(originalTitle)
 8. 创建新会话       →  window.go.main.App.CreateAISession()
 9. 重命名新会话     →  window.go.main.App.RenameAISession(newId, newTitle)
10. 保存消息         →  window.go.main.App.SaveAIMessages(newId, filteredMsgs)
11. 保存会话配置     →  window.go.main.App.SaveSessionConfig(newId, config)
12. 切换到新会话     →  switchSession(newId)
13. 提示成功         →  showNotification('已分叉', 'success')
```

#### 5. 新增 `parseForkTitle()` 辅助函数

标题递增逻辑：

```javascript
function parseForkTitle(title) {
    // 匹配开头的 "(N) " 或 "(N)" 模式
    const match = title.match(/^\((\d+)\)\s*/);
    if (match) {
        const num = parseInt(match[1], 10) + 1;
        return `(${num}) ${title.substring(match[0].length)}`;
    }
    return `(1) ${title}`;
}
```

示例：

* `"测试对话"` → `"(1) 测试对话"`

* `"(1) 测试对话"` → `"(2) 测试对话"`

* `"(3) 测试对话"` → `"(4) 测试对话"`

### 边界处理

* **流式输出中**：`isStreaming` 为 true 时跳过（同其他菜单项行为）

* **无消息 ID**：`_contextMsgEl?.dataset.msgId` 不存在时跳过

* **无消息**：`LoadAISessionMessages` 返回空时跳过

* **选中消息不在加载结果中**：兜底全部复制（不阻塞）

* **标题超长**：后端 `RenameAISession` 限制 20 字符，前缀 `(1) `  占 4 字符，剩余 16 字给原标题

* **会话配置全量复制**：`LoadSessionConfig` 获取当前会话全部配置 → `SaveSessionConfig` 写入新会话，包含模型、模式、引用笔记、技能、角色扮演笔记、召回笔记本等

## 验证步骤

1. 编译通过：`go build ./...`
2. 启动应用，进入 AI 助手
3. 在已有会话中，右键 AI 消息 → 菜单出现"分叉"项
4. 点击"分叉" → 新会话创建成功，标题为 `(1) 原标题`
5. 新会话中只包含选中消息及之前的所有消息（不包含之后的消息）
6. 再次分叉 → 新会话标题为 `(2) 原标题`
7. 流式输出中右键 → 分叉不可用

