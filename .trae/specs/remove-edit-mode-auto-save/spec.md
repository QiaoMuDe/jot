# 移除编辑模式自动保存

## Why

编辑已有笔记时，每次停止输入 3 秒就会自动调 `UpdateNote()` 触发后端写入，但：
1. 用户已有手动保存（按钮/Ctrl+S）+ `nm.show('笔记已更新')` 通知，不需要自动保存来确认内容安全
2. 编辑过程中频繁（每分钟十几次）触发不必要的后端写入
3. 底部 "已保存 ✓" 指示器与右上角通知重复，容易混淆

新建笔记的草稿自动保存（`SaveDraft()`）保留，作为新建内容的安全网（防止关闭/崩溃丢数据）。

## What Changes

- **REMOVED**：编辑模式（`state.editingNoteId` 不为 null）下 `startAutoSave()` 不再触发 `UpdateNote()`
- **保留**：新建笔记（`state.editingNoteId` 为 null）的草稿自动保存逻辑不变
- 自动保存定时器（`autoSaveTimer`）机制保留，仅用于新建笔记的草稿保存
- 查看模式下的 `clearTimeout(autoSaveTimer)` 逻辑保留不变

## Impact

- Affected code: `frontend/src/main.js` — `startAutoSave()` 函数

## REMOVED Requirements

### Requirement: 编辑模式自动保存

**Reason**：手动保存已有明确反馈，自动保存带来不必要的后端写入和 UI 混淆。

**Migration**：
- 编辑笔记时 `startAutoSave()` 仍会被调用（由 CM6 `updateListener` 触发），但会在内部判断 `state.editingNoteId` 不为 null 时直接 return
- 底部 `autoSaveIndicator` 在编辑模式下不再显示 "已保存 ✓"
