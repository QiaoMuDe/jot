# AI 聊天工具栏模型名溢出优化方案

## 现状分析

AI 聊天输入框上方的工具栏使用 flexbox 水平排列：

```
[引用] [模型名（可很长...）] [深度思考 ○] [联网搜索 ○] [卡片召回 ○] [更多技能 ↓]
```

**根因**：`.ai-chat-model-trigger` 没有设置宽度限制，当模型名称（如 `gpt-4-turbo-preview`、`deepseek-chat` 等）超出可用空间时，会将右侧的开关按钮（深度思考、联网搜索、卡片召回）挤到下一行，导致工具栏垂直错乱。

## 改动方案

仅修改 `frontend/src/css/components/ai-chat.css` 一个文件，三处改动：

### 1. 模型触发器添加截断

给 `.ai-chat-model-trigger` 添加 `max-width` + `text-overflow: ellipsis`，长模型名自动截断显示 `...`：

```css
.ai-chat-model-trigger {
    /* ... existing styles ... */
    max-width: 140px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
```

140px 宽度足够显示大部分常见模型名（如 `gpt-4` 等短名完整显示），超长名称自动截断。

### 2. 切换开关禁止收缩

给 `.ai-chat-search-toggle`（深度思考、联网搜索、卡片召回）添加 `flex-shrink: 0`，确保它们不会在空间不足时被压缩：

```css
.ai-chat-search-toggle {
    /* ... existing styles ... */
    flex-shrink: 0;
}
```

### 3. 工具栏添加折行兜底

给 `.ai-chat-toolbar` 添加 `flex-wrap: wrap` 作为极端情况下的兜底（如窗口极窄时内容自然换行）：

```css
.ai-chat-toolbar {
    /* ... existing styles ... */
    flex-wrap: wrap;
    row-gap: 6px;
}
```

## 涉及文件

| 文件 | 改动 |
|------|------|
| `frontend/src/css/components/ai-chat.css` | 3 处 CSS 属性追加 |

## 验证

1. 正常宽度窗口下工具栏保持单行，模型名正常显示
2. 较窄窗口下模型名过长时自动截断为 `...`
3. 极端窄窗口下工具栏可折行显示
4. 鼠标悬停/点击模型触发器时仍可正常展开下拉菜单选择模型
