# 回退到本轮对话发起前（Rollback）功能

## 概述

为用户消息右键菜单新增"回退到本轮对话发起前"功能，将对话回退到用户发送该消息之前的状态——删除本条及后续所有消息，把消息文本填入输入框，恢复技能/引用笔记等 Meta 状态。

## 当前状态分析

### 相关函数/变量

| 名称                           | 位置                                                                                   | 说明                                |
| ---------------------------- | ------------------------------------------------------------------------------------ | --------------------------------- |
| `handleDeleteMsg`            | [ai-chat.js#L5173](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5173) | 删除单条消息的 DOM 操作模式                  |
| `TruncateAISessionAtMessage` | [app.go#L2836](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2836)                         | 后端删除本条及之后所有消息                     |
| `buildUserMessageMeta`       | [ai-chat.js#L2360](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2360) | 从工具栏状态派生 Meta，含 ref/skill/file 三类 |
| `rerenderUserMessageChips`   | [ai-chat.js#L3537](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3537) | 更新用户消息气泡内的 chip 显示                |
| `updateRefChips`             | [ai-chat.js#L5882](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5882) | 渲染输入区的引用笔记芯片                      |
| `renderSkillChips`           | [ai-chat.js#L1928](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1928) | 渲染输入区的技能芯片                        |
| `referencedNotes`            | [ai-chat.js#L73](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L73)     | 全局变量：当前引用笔记列表                     |
| `activeSkills`               | [ai-chat.js#L129](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L129)   | 全局变量：当前激活技能                       |
| `uploadedFiles`              | [ai-chat.js#L82](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L82)     | 全局变量：当前上传文件列表（不回退）                |
| `inputEl`                    | [ai-chat.js#L9](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L9)       | 输入框元素 (`#aiChatInput`)            |
| `updateContextUsage`         | 已有                                                                                   | 更新右上角上下文使用率                       |

### 用户消息 Meta 结构

用户消息保存时，`meta` 字段为 JSON 字符串，格式为数组：

```json
[
  {"type": "ref", "id": 1, "title": "笔记标题", "notebook": "笔记本名"},
  {"type": "skill", "skillId": "translate", ...config},
  {"type": "file", "name": "文件.txt", "truncated": false}
]
```

### 输入框芯片状态管理

- `referencedNotes` 全局变量 → `updateRefChips()` 渲染输入区引用笔记芯片

- `activeSkills` 全局变量 → `renderSkillChips()` 渲染输入区技能芯片

- `uploadedFiles` 全局变量 → `renderFileChips()` 渲染输入区文件芯片（不回退）

***

## 修改方案

### 1. 新增 `ROLLBACK_ICON` SVG 图标

在 [ai-chat.js#L4177](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4177) 图标区新增回退箭头图标：

```js
const ROLLBACK_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>';
```

### 2. 菜单项 + 图标映射

在 [ai-chat.js#L3984-L3986](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3984-L3986) 用户消息的 `edit` 后追加 `rollback` 菜单项：

```js
if (role === 'user') {
    items.push({ type: 'divider' });
    items.push({ action: 'edit', label: '编辑' });
    items.push({ action: 'rollback', label: '回退到本轮对话发起前' });
}
```

在 [ai-chat.js#L4002](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L4002) 图标映射中追加：

```js
const actionIcons = {
    copy: COPY_ICON,
    edit: EDIT_ICON,
    rollback: ROLLBACK_ICON,  // 新增
    save: SAVE_ICON,
    regen: REGEN_ICON,
    followUp: FOLLOWUP_ICON,
    delete: DELETE_ICON,
};
```

### 3. 新增 `handleRollback` 函数

在 `handleRegenerate` 函数之后（[ai-chat.js#L5298](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5298)）新增 `handleRollback` 函数：

```js
/**
 * 回退到本轮对话发起前：删除本条及之后所有消息，将文本填入输入框，恢复技能/引用笔记
 */
async function handleRollback(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    const msgId = +msgEl.dataset.msgId || 0;
    if (!msgId) return;

    // 读取消息内容
    const contentDiv = msgEl.querySelector('.msg-content');
    if (!contentDiv) return;
    const textEl = contentDiv.querySelector('.ai-msg-text');
    const content = textEl ? textEl.textContent : (contentDiv.textContent || '');

    // 读取消息的 Meta（从 chatHistory 或 DOM）
    let msgMeta = '';
    const idx = chatHistory.findIndex(m => m.id === msgId);
    if (idx >= 0 && chatHistory[idx].meta) {
        msgMeta = chatHistory[idx].meta;
    }

    // 移除本条及之后所有消息的 DOM
    let nextEl = msgEl;
    while (nextEl) {
        const toRemove = nextEl;
        nextEl = nextEl.nextElementSibling;
        if (toRemove.classList.contains('ai-msg')) {
            toRemove.remove();
        }
    }

    // 后端截断（删除本条及之后的消息）
    if (activeSessionId !== null) {
        try {
            await window.go.main.App.TruncateAISessionAtMessage(activeSessionId, msgId);
        } catch (_) { /* 静默 */ }
    }

    // 截断 chatHistory 缓冲区
    if (idx >= 0) {
        chatHistory = chatHistory.slice(0, idx);
    }

    // 恢复 Meta 到工具栏状态（仅 ref 和 skill，不恢复 file）
    const refNotes = [];
    const skills = {};
    if (msgMeta) {
        try {
            const items = JSON.parse(msgMeta);
            if (Array.isArray(items)) {
                for (const it of items) {
                    if (!it || typeof it !== 'object') continue;
                    if (it.type === 'ref') {
                        refNotes.push({ id: it.id, title: it.title || '', notebook_name: it.notebook || '' });
                    } else if (it.type === 'skill') {
                        // 从 meta 还原 skill 配置
                        const cfg = { ...it };
                        delete cfg.type;
                        delete cfg.skillId;
                        skills[it.skillId] = cfg;
                    }
                }
            }
        } catch (_) { /* 静默 */ }
    }
    referencedNotes = refNotes;
    activeSkills = skills;
    // 不回退上传文件（用户明确要求）
    uploadedFiles = [];
    // 更新输入区芯片
    updateRefChips();
    renderSkillChips();
    renderFileChips();

    // 填入输入框并聚焦
    if (inputEl) {
        inputEl.value = content;
        inputEl.focus();
        // 触发 input 事件以更新自动撑高
        inputEl.dispatchEvent(new Event('input', { bubbles: true }));
    }

    // 更新上下文使用率
    updateContextUsage();
}
```

### 4. 新增 `rollback` action 处理分支

在 [ai-chat.js#L1269](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1269) 的 `edit` 分支和 `delete` 分支之间追加：

```js
} else if (action === 'rollback') {
    if (isStreaming) return;
    if (msgEl) handleRollback(msgEl);
} else if (action === 'delete') {
```

### 5. 不需要后端改动

`TruncateAISessionAtMessage` 已存在，无需新增后端接口。

***

## 验证步骤

1. `go build` 编译通过
2. `npm run build` 前端无报错
3. 手动测试：

   - 有一条用户消息带引用笔记和技能 → 右键 → "回退到本轮对话发起前" → 消息被删除 → 文本填入输入框 → 引用笔记和技能芯片恢复 → 输入区聚焦

   - 有后续对话 → 回退后后续对话消失

   - 上传文件 → 回退后文件不恢复

   - 无后续对话 → 正常回退到输入框

   - 回退后 → 右上角上下文使用率更新

