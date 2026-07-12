# 修复角色扮演 UI 三个 Bug

## Summary
修复角色扮演技能的三个 UI 问题：选择器的布局位置不对、取消技能后选择器不隐藏、笔记选择器确认/取消逻辑不完整。

## Current State Analysis

### Issue 1: 选择器独占一行
- **问题**：角色档案选择器（`#aiChatRoleplaySelector`）是独立 `<div>`，放在技能指示条（`#aiChatSkillBar`）和上传文件条（`#aiChatFileBar`）之间
- **期望**：它应该作为 roleplay skill 的附带配置，出现在 `#aiChatSkillChips` 行内，与 skill chips 在同一行
- **原因**：实现时未考虑复用 skill bar 的布局，直接新增了独立一行

### Issue 2: 取消技能后选择器不隐藏
- **文件**：`frontend/src/js/ai-chat.js` `renderSkillChips()` 函数
- **位置**：第 1733-1737 行
- **问题**：当 roleplay 是唯一的技能时，删除 `activeSkills.roleplay` 后，`renderSkillChips()` 检测到 `keys.length === 0` 立即 return（第 1736 行），跳过了第 1873 行的 `updateRoleplaySelector()` 调用
- **结果**：`roleplaySelector.style.display` 仍为 `''`，选择器保持可见

### Issue 3: 笔记选择器确认后不更新
- **文件**：`frontend/src/js/ai-chat.js` `openRoleplaySelector()` 函数
- **位置**：第 1970-1976 行
- **问题**：确认按钮的 handler 中，`selectedIds.length === 0`（用户把所有已选笔记取消勾选后确认）时直接 return，不更新 `roleplayNotes = []`，也不关闭模态框
- **场景**：用户已选 1 篇 → 打开选择器 → 取消勾选该篇 → 点击确认 → 无反应，`roleplayNotes` 仍为旧值

## Proposed Changes

### Change 1: 将选择器移入 skill chips 行内

**文件**：`frontend/index.html`
- **修改**：移除第 1067-1078 行的独立 `#aiChatRoleplaySelector` 元素

**文件**：`frontend/src/js/ai-chat.js`
- **修改**：在 `renderSkillChips()` 中，渲染完所有 skill chips 后，如果 `activeSkills.roleplay` 为 true，追加渲染角色档案选择器作为 chip 行内的最后一个元素
- **放弃独立的 DOM 元素**：不再通过 `document.getElementById('aiChatRoleplaySelector')` 控制显示/隐藏，而是像 `updateRefChips()` 那样直接用 innerHTML 渲染
- **`updateRoleplaySelector()` 改为 `renderRoleplayChip()`**：渲染一个特殊的 chip，包含面具图标 + "角色档案: N 篇" + 点击打开选择器 + 清除按钮

**文件**：`frontend/src/css/components/ai-chat.css`
- **修改**：将 `.ai-chat-roleplay-selector` 和 `.roleplay-selector-*` 样式调整为适配 skill chip 行内的样式（与 `.ai-chat-skill-chip` 风格一致）

### Change 2: 修复渲染排序——确保选择器跟随技能开关

**文件**：`frontend/src/js/ai-chat.js`
- **修改**：在 `renderSkillChips()` 的早期 return 之前（第 1736 行），添加 `updateRoleplaySelector()` 的清理逻辑
- **方案**：在 `if (keys.length === 0)` 的 return 之前，额外检查并隐藏 roleplay 选择器

更简洁的方案：将 `updateRoleplaySelector()` 和 `clearRoleplayNotes()` 移到 `renderSkillChips()` 顶部，在早期 return 之前执行。

### Change 3: 修复笔记选择器确认后不更新

**文件**：`frontend/src/js/ai-chat.js`
- **修改**：第 1972 行的 `if (selectedIds.length === 0) return;` 改为允许空选择——设置 `roleplayNotes = []` 并继续执行关闭和保存逻辑
- 注意：因为 `getSelectedNotesInfo([])` 会返回 `[]`，`ids.length === 0` 时可以直接 `roleplayNotes = []`，跳过后端调用

## Assumptions & Decisions
- 角色档案选择器将复用 skill chip 的视觉风格（背景色、圆角、字号），而不是独立样式
- 取消笔记选择时，不保存修改（保持原有 roleplayNotes）
- 确认时若 0 篇选中 = 清空角色档案

## Verification
1. `npm run dev` 或 `npx vite build` 前端构建通过
2. 激活角色扮演 → 选择器出现在 skill chips 行内
3. 点击 chip X 取消角色扮演 → 选择器随 chip 一起消失
4. 打开笔记选择器，选 1-3 篇确认 → 选择器显示正确数量和 tooltip
5. 打开笔记选择器，取消/关闭 → 选择器保持原有状态不变
6. 打开笔记选择器，取消所有勾选后确认 → 角色档案清空
