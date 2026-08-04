# 修复：笔记首页右键菜单"复制"只复制卡片摘要的问题

## 摘要
笔记首页（卡片视图）右键菜单点击"复制"时，目前复制的是卡片上展示的截断摘要（前 200 字符），而非笔记完整正文。修复方案：`copyNote` 改为根据笔记 id 调用后端已有的 `GetNoteContent` API 获取完整正文后再复制（仅复制正文，不包含标题），仅在后端不可用/失败时降级使用列表中的截断内容。

## 现状分析
- 后端列表查询 [note_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L180-L181) 的 `noteThinSelect = "id, title, SUBSTR(content, 1, 200) AS content, ..."` 明确将 content 截断为前 200 字符用于卡片预览。
- 首页通过 `GetNotes` 加载数据存入 `state.notes`（见 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L745-L750)），因此 `state.notes` 中每条 note 的 `content` 都是截断摘要。
- [copyNote](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L1038-L1052) 直接从 `state.notes` 取值拼接 `title + content` 复制，导致复制的只有摘要。
- 后端已有现成 API：[GetNoteContent](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L511-L520)（App 绑定）→ [NoteService.GetNoteContent](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/note_service.go#L102-L116)（按 id 查 `deleted_at IS NULL` 的完整 content）。前端其他位置（如编辑器按需加载 [main.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L3872-L3873)）已有相同调用模式。

## 变更内容
仅修改一个文件：`frontend/src/main.js` 中的 `copyNote` 函数（约 1041-1052 行）。

### 修改逻辑
1. 仍从 `state.notes` 查找 note（作为降级数据源；不再使用 title）。
2. 复制前，尝试调用 `window.go.main.App.GetNoteContent(id)` 获取完整正文。
   - 绑定存在且调用成功 → 使用完整正文。
   - 后端未绑定（降级模拟数据模式）或调用失败 → 记录错误日志并降级使用 `note.content`（截断摘要），保证原有兜底行为不破坏。
3. 仅将完整正文写入剪贴板（不拼接标题），成功/失败提示逻辑保持不变。

### 伪代码
```javascript
async function copyNote(id) {
    const note = state.notes.find((n) => n.id === id);
    if (!note) return;
    // 列表查询仅返回截断的前 200 字符，需按 id 获取完整正文再复制（仅复制正文，不含标题）
    let content = note.content || '';
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetNoteContent) {
            const fullContent = await window.go.main.App.GetNoteContent(id);
            if (fullContent != null) content = fullContent;
        }
    } catch (err) {
        console.error('获取完整笔记内容失败，降级使用列表截断内容:', err);
    }
    try {
        await navigator.clipboard.writeText(content);
        nm.show('已复制到剪贴板', 'success');
    } catch (err) {
        console.error('复制失败:', err);
        nm.show('复制失败', 'error');
    }
}
```

## 假设与决策
- 后端无需改动：`GetNoteContent` API 已存在且返回完整 content。
- 复制内容仅包含正文，不包含标题。
- 保持降级路径：后端未绑定或接口报错时不阻塞复制，回退到原截断内容。

## 验证步骤
1. 启动应用，进入笔记首页，右键任意笔记 → 点击"复制"。
2. 粘贴到任意文本编辑器，确认复制的是该笔记的完整正文（超出 200 字符的正文应完整出现），且不包含标题行。
3. 对空正文笔记复制，确认行为正常（剪贴板为空）。
4. 确认复制成功/失败 toast 提示正常。
