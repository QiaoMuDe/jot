# 点击标签筛选笔记 Spec

## Why

当前点击笔记卡片或搜索结果中的标签时，只是把标签名填入搜索弹窗做关键词搜索。用户希望点击标签直接按标签 ID 筛选笔记，得到更精确的结果。

## What Changes

- **前端**：`window.searchByTag(tagName)` 改为接受 tag ID，调用后端 `GetNotesByTag` API 获取笔记列表，在搜索视图展示结果
- **前端**：卡片标签和搜索结果标签的 `onclick` 传递标签 ID 和名称
- **前端**：搜索视图顶部显示当前筛选的标签 chip，支持点击清除筛选回到网格
- **后端**：无变更。`GetNotesByTag` 接口已存在，直接复用
- **搜索弹窗**：不受影响，原有逻辑不变

## Impact

- Affected specs: 笔记标签交互、搜索功能
- Affected code:
  - `frontend/src/main.js` — `searchByTag()` 函数重写、卡片/搜索结果标签 click handler、新增 tag-filter 状态与 UI

## ADDED Requirements

### Requirement: 标签筛选

系统 SHALL 支持通过标签 ID 筛选笔记。

#### Scenario: 点击卡片标签筛选
- **WHEN** 用户在卡片网格视图中点击笔记卡片上的标签 chip
- **THEN** 系统调用 `GetNotesByTag(tagId, 1, 100, sortBy)` 获取该标签的笔记
- **THEN** 系统切换到搜索视图，展示筛选结果
- **THEN** 搜索视图顶部显示当前筛选的标签 chip（含颜色圆点和名称）
- **THEN** 标签 chip 旁有关闭按钮（×），点击清空筛选并回到网格首页

#### Scenario: 点击搜索结果标签筛选
- **WHEN** 用户在搜索视图中点击搜索结果中的标签 chip
- **THEN** 系统调用 `GetNotesByTag(tagId, 1, 100, sortBy)` 获取该标签的笔记
- **THEN** 系统刷新搜索视图，展示新的筛选结果
- **THEN** 搜索视图顶部显示当前筛选的标签 chip

#### Scenario: 清除标签筛选
- **WHEN** 用户在标签筛选状态下点击标签 chip 旁的关闭按钮（×）
- **THEN** 系统清空筛选状态，调用 `switchView('grid')` 回到网格首页

#### Scenario: 批量模式不触发筛选
- **WHEN** 用户在批量模式下点击卡片上的标签 chip
- **THEN** 现有行为不变：仅 `event.stopPropagation()`，不触发任何搜索或筛选

## MODIFIED Requirements

### Requirement: 现有 `searchByTag` 函数

函数签名从 `searchByTag(tagName)` 改为 `searchByTag(tagId, tagName, tagColor)`。

- 不再打开搜索弹窗
- 不再设搜索框值
- 改为调用 `GetNotesByTag(tagId, 1, 100, state.sortBy)`（若后端不可用则本地 `state.notes.filter(n => n.tags.some(t => t.id === tagId))` 降级）
- 设置 `state.filterTag = {id: tagId, name: tagName, color: tagColor}`
- 调用 `renderSearchResults(results, '', state.filterTag)`（空 keyword 避免高亮）
- 调用 `switchView('search')` 切换到搜索视图

### Requirement: `renderSearchResults` 函数

新增参数 `filterTag`（可选），传入 `{id, name, color}`：
- 非空时在搜索结果列表上方渲染 `.tag-filter-indicator`，包含标签 chip + 关闭按钮
- 空时隐藏该指示器

### Requirement: 卡片渲染标签 onclick

`renderCardGrid()` 中标签的 onclick 从 `window.searchByTag('${escapeHtml(tag.name)}')` 改为 `window.searchByTag(${tag.id}, '${escapeHtml(tag.name)}', '${escapeHtml(tag.color || '#6366f1')}')`。

### Requirement: 搜索结果渲染标签 onclick

`renderSearchResults()` 中标签的 onclick 同样改为传递 tagId。

## REMOVED Requirements

无。