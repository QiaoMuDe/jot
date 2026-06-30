# 追问引用栏 — 实现计划

## Summary

追问功能 UI 重新设计：点击追问按钮后，不在输入框内注入引用文本，而是在输入区域顶部显示一个引用栏，展示被追问消息的缩略内容 + 右侧 X 取消按钮。发送时将引用内容注入到 system message 中，AI 能感知被追问的上文。

## Current State

当前追问按钮的逻辑：

* 点击后将 AI 回复内容截取前 100 字符，以 `> excerpt...` 格式直接填入 `#aiChatInput` 文本框

* 文本框是单行 `rows="1"`，填入多行内容后不会自动展开，用户看不到完整引用

* 用户需要手动在引用后面打字，体验割裂

## Proposed Changes

### 文件 1: `frontend/index.html`

在 `#aiChatInputArea` 内、toolbar 上方新增引用栏：

```html
<!-- 追问引用栏 -->
<div id="aiChatFollowUpBar" class="ai-chat-followup-bar" style="display:none;">
    <span id="aiChatFollowUpText" class="ai-chat-followup-text"></span>
    <button id="aiChatFollowUpClose" class="ai-chat-followup-close" title="取消追问">&times;</button>
</div>
```

放在 `div.ai-chat-toolbar` 前面（即工具栏上方）。

### 文件 2: `frontend/src/css/components/ai-chat.css`

新增 `.ai-chat-followup-bar` 样式：

* Flex 行，左右排列

* 左侧文本：单行截断（`text-overflow: ellipsis`），灰色文字，左侧带引用图标或竖线装饰

* 右侧关闭按钮：圆形的 `&times;`，hover 变红色

* 底部内边距 + 分隔线（与 toolbar 区分）

### 文件 3: `frontend/src/js/ai-chat.js`

新增 `let followUpRef = ''` 变量存储被引用的 AI 回复完整内容。

**追问按钮点击处理**（`createMsgActions` 内）：

```javascript
// 改为：
const excerpt = safeContent.replace(/\s+/g, ' ').trim().slice(0, 80);
followUpRef = safeContent;  // 存完整内容用于注入 system message
const bar = document.getElementById('aiChatFollowUpBar');
const text = document.getElementById('aiChatFollowUpText');
if (bar && text) {
    text.textContent = '引用: ' + excerpt + (safeContent.length > 80 ? '…' : '');
    bar.style.display = 'flex';
}
```

**X 按钮事件**（在初始化或 document ready 中）：

```javascript
document.getElementById('aiChatFollowUpClose').addEventListener('click', () => {
    followUpRef = '';
    document.getElementById('aiChatFollowUpBar').style.display = 'none';
});
```

**`onSend`** **中注入引用到上下文**：
在 `systemContext` 构建完成后，如果 `followUpRef` 非空，追加到 `systemContext`：

```javascript
if (followUpRef) {
    const refText = '用户正在追问以下内容：\n' + followUpRef.slice(0, 500);
    systemContext = systemContext
        ? systemContext + '\n\n' + refText
        : refText;
}
```

**发送后清理**：发送完成后自动隐藏引用栏并清空 `followUpRef`。

### 文件 4: `AGENTS.md`

新增 六十二、新增记忆点（追问引用栏重构）条目。

## Assumptions & Decisions

* 引用栏放置在 toolbar 上方（用户要求"工具栏的上面"），作为输入区域最顶部元素

* 引用文本截取 80 字符显示，超过加 `…`；完整内容（最多 500 字符）注入 system message

* 引用栏样式类似于一个轻提示条（chip bar 风格），与现有的 `#aiChatRefBar` 视觉区分

* 每次只能有一个追问引用（点击新的追问替换旧的）

* 追问引用和笔记引用可以共存，都在 `systemContext` 中拼接

## Verification

1. 点击 AI 回复的追问按钮 → 输入区域顶部出现引用栏，显示缩略文本 + X 按钮
2. 点击 X → 引用栏消失，`followUpRef` 清空
3. 点击追问 A，再点击追问 B → 引用栏更新为 B 的内容
4. 有引用时发送消息 → system message 中包含引用文本，AI 能正确感知上下文
5. 发送完成后引用栏自动隐藏

