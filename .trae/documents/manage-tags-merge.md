# 右键菜单标签管理合并 实施计划

## Summary

将笔记右键菜单的「添加标签」「移除标签」两个入口合并为一个「管理标签」，打开同一个弹窗的新 `manage` 模式：芯片即开关（已挂标签初始选中），点击切换期望状态，确认时按差量分别调用批量添加/移除 API。批量模式（多选）的两个入口与 add/remove 模式原样保留。

## Current State

* 菜单两项：\[index.html#L2362-L2363] `data-action="add-tag"` / `remove-tag`

* 可用性禁用逻辑：\[main.js#L5245-L5258]（满 3 禁添加、无标签禁移除）

* 分发：\[main.js#L5319-L5324] `handleContextAction` 两个 case → `openBatchTagPicker('add'|'remove', [id])`

* 弹窗：\[main.js#L5617-L5789] `openBatchTagPicker` / `closeBatchTagPicker` / `renderBatchTagList` / `onBatchTagClick` / `confirmBatchTagAction`，模块级状态 `batchTagAction` / `batchTagNoteIds` / `batchTagAddLimit`；复用元素 `batchTagTitle/List/Footer/ConfirmBtn/Overlay`

* 工具类：`getTagIdsInNotes(ids)` 返回笔记集合中标签 id 的 Set

* 批量模式入口：\[main.js#L6278-L6279] 仍调用 add/remove 模式，不动

## Changes

### 1. index.html — 菜单项合并

* 删除 `remove-tag` 项；`add-tag` 项改为 `data-action="manage-tags"`，文案「管理标签」

### 2. main.js — 5 处修改

* **模块级变量**：新增 `let batchTagInitialSelected = new Set();`（manage 模式打开时的已挂标签快照）；`closeBatchTagPicker` 中重置

* **showContextMenu（L5245-L258）**：删除 add/remove 两项的禁用逻辑（管理标签永远可点），仅保留置顶文本更新

* **handleContextAction**：`add-tag`/`remove-tag` 两个 case 替换为 `case 'manage-tags': openBatchTagPicker('manage', [id]);`

* **openBatchTagPicker**：新增 `manage` 分支——要求 `batchTagNoteIds` 为单笔记；标题「管理标签」；跳过添加限额/移除空标签前置拦截；记录 `batchTagInitialSelected = getTagIdsInNotes(ids)`；确认按钮文案「保存」

* **renderBatchTagList**：`manage` 分支——所有芯片均可点，初始 `selected` 类 = `batchTagInitialSelected.has(tag.id)`；无任何标签时沿用现有空态文案

* **onBatchTagClick**：`manage` 分支——点击即 toggle；若点亮后选中数 >3 则拒绝并提示「一篇笔记最多 3 个标签，请先取消一个」

* **confirmBatchTagAction**：`manage` 分支——差量计算：`added = 当前选中 − initial`，`removed = initial − 当前选中`；两者皆空 → 关弹窗 + `nm.show('未做任何修改', 'info')`；否则先逐个 `BatchAddTagToNotes` 再逐个 `BatchRemoveTagFromNotes`，成功提示按非零部分拼「已添加 x 个标签、已移除 y 个标签」

### 3. CSS

* 零改动（复用现有 `batch-tag-chip` 的 selected/disabled 样式）

## Assumptions & Decisions

* 仅单笔记右键菜单走 manage；批量模式语义含糊（各笔记标签状态不一致），保留原 add/remove 两入口

* 差量提交顺序：先加后删；单标签失败即中断报错（与现有 confirm 逻辑一致）

* 成功后 `loadNotes()` 刷新，行为与现有一致

## Verification

* `npm run build` + `wails build -skipbindings` 通过

* 手动验证：右键单笔记 → 管理标签 → 同弹窗内勾选新标签 + 取消已有标签 → 保存 → 卡片标签更新、提示计数正确

* 边界：无标签笔记（全可点）、满 3 个后再点亮被拦、无修改保存提示「未做任何修改」、应用无任何标签显示空态

* 回归：批量模式「批量添加/批量移除标签」行为不变

