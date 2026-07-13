# 修复快速笔记启动触发丢失问题

## 问题 Summary

快速笔记开关在设置页可正常保存和加载（checkbox 状态正确），但重启后编辑器不再自动弹出。原因是「unify-settings-load-save」重构中，旧 `loadQuickNoteSetting()` 函数被删除，但其**启动时触发编辑器**的逻辑未被迁移到 `init()` 中。

## 当前状态分析

- **设置保存** ✅ — `saveSettings()` → `SaveAllSettings()` 正确写入 DB 的 `quick_note_enabled` 字段
- **设置加载** ✅ — `loadSettings()` → `GetAllSettings()` 正确读取并更新 checkbox 状态
- **启动触发** ❌ — `init()` 中 `loadSettings()` 后**没有检查 `quick_note_enabled` 并调用 `openEditor()`**

旧 `loadQuickNoteSetting()` 的行为记录了在 [direct-fullscreen-on-quicknote.md](file:///d:/资源池/下水道/Dev/本地项目/jot/.trae/documents/direct-fullscreen-on-quicknote.md)：调用 `openEditor(null, false, true)` 以全屏模式打开空白编辑器（第三个参数 `startFullscreen=true` 自动跳过入场动画）。

## 修改方案

仅修改一个文件：`frontend/src/main.js`

### 改动：`init()` 函数中追加快速笔记启动检测

**位置**：[main.js#L7003-L7025](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L7003-L7025)，在 `await loadSettings()` 之后插入

**代码**：
```javascript
// 快速笔记：启用时自动打开全屏编辑器
if (els.quickNoteToggle?.checked) {
    await openEditor(null, false, true);
}
```

**为什么可以直接读 checkbox**：因为上一行 `await loadSettings()` 已经完成了 `els.quickNoteToggle.checked = cfg.quick_note_enabled`，DOM 状态已与后端同步。

**为什么不改 `loadSettings()` 返回 cfg**：`loadSettings` 被 `data-management.js` 的 `reloadSettings()` 通过 `window.loadSettings()` 调用，增加返回值可能引入未预期的影响。在 `init()` 中直接读 checkbox 是侵入最小、最安全的方式。

## 不变的文件

- 后端 Go 代码：不需要改动（设置已正确保存/读取）
- `index.html`：不需要改动
- 所有 CSS 文件：不需要改动
- `data-management.js`、`ai-chat.js` 等：不需要改动

## 验证步骤

1. 启用快速笔记 → 重启程序 → 编辑器以全屏模式直接打开，不闪现悬浮卡片
2. 禁用快速笔记 → 重启程序 → 正常主界面，不打开编辑器
3. 正常点击新建笔记 → 仍然以悬浮卡片打开
4. 全屏切换按钮 → 正常工作
5. 关闭编辑器后 → 回到正常网格视图
