# AI 引用笔记选择器全选功能 Spec

## Why
AI 引用笔记选择器目前只能逐一点选，当需要引用大量笔记或某分类下全部笔记时操作效率极低。用户需要在各种筛选场景（全部笔记本、指定笔记本、标签筛选、搜索关键词）下都能一键全选当前过滤出的所有条目。

## What Changes
- **新增**: AI 引用笔记选择器浮层底部 footer 区域添加「全选」按钮
- **新增**: 后端 `SearchNoteIDs` API — 按当前筛选条件（关键词、笔记本、标签）返回所有匹配笔记 ID（不分页）
- **修改**: 前端 `loadNoteList` / `renderNoteList` / `appendToList` — 选中态与全选状态联动
- **修改**: 全选状态下滚动加载更多时，新追加条目自动选中

## Impact
- Affected specs: 无（全新功能，不修改现有 spec）
- Affected code:
  - `frontend/index.html`：全选复选框 DOM
  - `frontend/src/js/ai-chat.js`：全选逻辑、UI 联动、选中状态管理
  - `frontend/src/css/components/ai-chat.css`：全选复选框样式
  - `app.go`：新增 `SearchNoteIDs` 绑定方法
  - `internal/services/note_service.go`：新增 `SearchNoteIDs` / `SearchNoteIDsByNotebook`
  - `frontend/wailsjs/go/main/App.js`：Wails 自动生成绑定

## ADDED Requirements

### Requirement: 全选按钮
AI 引用笔记选择器浮层 SHALL 在底部 footer 区域提供全选按钮，让用户一键选中/取消选中当前筛选条件下的所有笔记。

#### Scenario: 全选 — 全部笔记本（无筛选）
- **WHEN** 用户在"全部笔记本"下打开引用笔记选择器
- **AND** 点击全选复选框
- **THEN** 调用 `GetAllNoteIDs()` 获取所有未删除笔记 ID
- **AND** 将所有 ID 加入 `_refTempSelected`
- **AND** 所有列表条目显示选中态
- **AND** 底部计数显示"已选 N 篇"（N 为总笔记数）

#### Scenario: 全选 — 按笔记本筛选
- **WHEN** 用户已选择一个具体笔记本
- **AND** 点击全选复选框
- **THEN** 调用 `SearchNoteIDs`（带 notebookID 参数）获取该笔记本下所有匹配笔记 ID
- **AND** 将所有 ID 加入 `_refTempSelected`

#### Scenario: 全选 — 按标签筛选
- **WHEN** 用户已选择 1 个或多个标签
- **AND** 点击全选复选框
- **THEN** 调用 `SearchNoteIDs`（带 tagIDs 参数）获取匹配标签的所有笔记 ID
- **AND** 将所有 ID 加入 `_refTempSelected`

#### Scenario: 全选 — 有搜索关键词
- **WHEN** 用户输入了搜索关键词
- **AND** 点击全选复选框
- **THEN** 调用 `SearchNoteIDs`（带 keyword 参数）获取匹配搜索的所有笔记 ID
- **AND** 将所有 ID 加入 `_refTempSelected`

#### Scenario: 全选 + 组合筛选
- **WHEN** 用户同时使用了笔记本 + 标签 + 搜索关键词的组合筛选
- **AND** 点击全选复选框
- **THEN** 调用 `SearchNoteIDs` 传入所有筛选参数，获取完全匹配的所有笔记 ID

#### Scenario: 取消全选
- **WHEN** 全选已激活（所有条目已选中）
- **AND** 用户再次点击全选复选框
- **THEN** 清空 `_refTempSelected`
- **AND** 所有列表条目取消选中态
- **AND** 底部计数显示"已选 0 篇"

#### Scenario: 全选状态下滚动加载更多
- **WHEN** 全选已激活（用户点击了全选）
- **AND** 用户滚动到底部加载了更多笔记
- **THEN** 新追加的条目自动显示选中态
- **AND** 底部计数随之增加

#### Scenario: 全选状态下手动逐条取消
- **WHEN** 全选已激活
- **AND** 用户点击了某一条笔记取消选中
- **THEN** 全选按钮自动变为"未全选"状态（取消勾选）
- **AND** 底部计数减少

#### Scenario: 手动逐条选完所有条目后全选按钮自动变为全选
- **WHEN** 用户逐条选中了当前筛选条件下的所有笔记
- **THEN** 全选按钮自动变为"已全选"状态（勾选）

#### Scenario: 筛选条件变更时全选状态重置
- **WHEN** 全选已激活
- **AND** 用户切换笔记本/标签/输入搜索关键词/清除搜索
- **THEN** 全选按钮重置为"未全选"状态
- **AND** `_refTempSelected` **不清除**（保持已有选中，用户可在此基础上增量选择）

### Requirement: 后端 SearchNoteIDs API
系统 SHALL 提供后端 API，按筛选条件返回所有匹配笔记 ID（不分页，一次性返回全部符合条件的结果）。

#### Scenario: SearchNoteIDs 调用
- **WHEN** 前端调用 `SearchNoteIDs(keyword, notebookID, tagIDs)`
- **THEN** 后端按 notebookID > 0 判断是否限定笔记本
- **AND** 按 keyword 非空添加 LIKE 搜索条件
- **AND** 按 tagIDs 非空添加标签 AND 过滤子查询
- **AND** 返回 `[]uint` 类型的所有匹配笔记 ID
- **AND** 结果不受分页影响，返回全部 ID

### Requirement: 全选按钮 UI
全选按钮 SHALL 展示在浮层底部 footer 区域（`ai-note-ref-count` 左侧），作为一个带复选框样式的按钮。

#### Scenario: 全选按钮位置
- **WHEN** 浮层打开
- **THEN** 在 footer 的 `已选 N 篇` 文字左侧显示全选按钮
- **AND** 按钮包含复选框图标 + "全选" 文字

#### Scenario: 全选按钮样式
- **WHEN** 浮层打开
- **THEN** 全选按钮使用内联 flex 布局，与计数文字在同一行
- **AND** 选中时复选框图标显示对勾 SVG
- **AND** 未选中时复选框图标显示空心圆/方框
