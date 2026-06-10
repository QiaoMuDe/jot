# 搜索筛选功能增强 Spec

## Why

当前搜索功能仅支持关键词模糊搜索，缺少常用的筛选条件。用户希望在搜索结果中快速按搜索范围（标题/内容）、更新时间范围和标签进行过滤，提升搜索效率和精确度。

## What Changes

- **前端**：在搜索结果视图（#viewSearch）的标题栏下方新增一行筛选栏，包含三个下拉筛选器：搜索范围、时间筛选、标签筛选
- **后端**：新增 API 用于获取时间筛选选项和笔记本关联标签；修改 SearchNotes API 以支持三个筛选参数
- **UI/UX**：筛选栏仅在搜索视图可见，风格与现有设计系统一致（温暖极简主题），交互流畅

## Impact

- Affected specs: 搜索功能
- Affected code:
  - `frontend/src/main.js` — 搜索逻辑、筛选事件绑定
  - `frontend/src/style.css` — 筛选栏样式
  - `frontend/index.html` — 搜索视图 HTML 结构
  - `app.go` — SearchNotes API 签名扩展
  - `internal/services/note_service.go` — Search/SearchByNotebook 方法扩展
  - `internal/services/note_service.go` — 新增 GetSearchFilterOptions 等方法

## ADDED Requirements

### Requirement: 搜索范围筛选

系统 SHALL 提供搜索范围下拉选择器，选项包括：
- **全部**（默认）—— 搜索标题 + 内容 + 标签
- **标题** —— 仅搜索标题
- **内容** —— 仅搜索内容

#### Scenario: 选择搜索范围
- **WHEN** 用户在搜索视图从"搜索范围"下拉中选择"标题"
- **THEN** 系统立即使用当前关键词 + "标题"范围重新执行搜索
- **THEN** 搜索结果仅匹配标题包含关键词的笔记

### Requirement: 时间筛选

系统 SHALL 提供时间范围下拉选择器，选项包括：
- **全部时间**（默认）
- **最近 7 天**
- **最近 30 天**
- **最近 90 天**
- **最近 1 年**

选项中的天数值为固定选项（非动态生成），由后端在查询时根据当前时间计算日期范围。

#### Scenario: 选择时间范围
- **WHEN** 用户在搜索视图从"时间筛选"下拉中选择"最近 7 天"
- **THEN** 系统立即使用当前关键词 + 7 天时间范围重新执行搜索
- **THEN** 搜索结果仅包含 updated_at 在最近 7 天内的笔记

### Requirement: 标签筛选

系统 SHALL 提供标签下拉选择器，选项包括：
- **全部标签**（默认）
- 当前笔记本下所有已使用的标签列表（从后端获取）

#### Scenario: 选择标签
- **WHEN** 用户在搜索视图从"标签筛选"下拉中选择一个具体标签
- **THEN** 系统立即使用当前关键词 + 选中标签重新执行搜索
- **THEN** 搜索结果仅包含关联了该标签的笔记

#### Scenario: 切换笔记本后标签更新
- **WHEN** 用户切换当前笔记本
- **THEN** 标签筛选下拉的选项自动更新为当前笔记本关联的标签
- **THEN** 如果当前选中的标签不在新笔记本中，自动重置为"全部标签"

### Requirement: 后端 API

系统 SHALL 提供以下后端 API：

1. **SearchNotes 扩展**：`SearchNotes(keyword string, page, pageSize int, notebookID uint, scope string, days int, tagID uint)`

   - `scope`: `"all"` | `"title"` | `"content"`，默认为 `"all"`
   - `days`: 时间筛选天数，`0` 表示全部时间，`7`/`30`/`90`/`365` 分别对应各时间范围
   - `tagID`: 标签 ID，`0` 表示全部标签
   
2. **GetNotebookTags**：`GetNotebookTags(notebookID uint)` — 获取指定笔记本下所有笔记关联的标签列表

#### Scenario: 后端搜索带筛选参数
- **WHEN** 后端接收到带 scope="title", days=7, tagID=3 的搜索请求
- **THEN** 生成 SQL：`WHERE title LIKE '%keyword%' AND updated_at >= NOW() - 7 days AND id IN (SELECT note_id FROM note_tags WHERE tag_id = 3)`

### Requirement: 筛选栏 UI

筛选栏 SHALL：
- 位于搜索视图标题栏下方，搜索结果列表上方
- 以水平排列的三个小型下拉选择器展示
- 每个下拉选择器左侧带有图标/标签提示
- 样式与温暖极简主题一致
- 带有平滑的出现/隐藏动画
- 选择任一筛选条件后自动触发搜索（带 300ms 防抖）
- 在筛选栏右侧显示"重置"按钮，一键重置所有筛选条件

#### 搜索范围下拉 UI
- 标签："搜索范围："
- 选项：全部 / 标题 / 内容
- 默认选中："全部"
- 宽度紧凑（约 120px）

#### 时间筛选下拉 UI
- 标签："时间："
- 选项：全部时间 / 最近 7 天 / 最近 30 天 / 最近 90 天 / 最近 1 年
- 默认选中："全部时间"
- 宽度紧凑（约 140px）

#### 标签筛选下拉 UI
- 标签："标签："
- 选项："全部标签" + 从后端获取的标签列表（显示标签名称和颜色圆点）
- 默认选中："全部标签"
- 宽度适中（约 160px），支持滚动

### Requirement: 筛选状态保持

- 筛选栏的选择状态在搜索视图内保持
- 退出搜索视图（返回网格视图）时重置所有筛选条件为默认值
- 切换笔记本时重置所有筛选条件为默认值

## MODIFIED Requirements

### Requirement: 现有搜索流程

搜索流程修改为：用户输入关键词 → 按 Enter 或自动触发搜索 → 进入搜索视图 → 显示筛选栏和搜索结果 → 用户可通过调整筛选条件缩小/扩大搜索范围 → 每次筛选变更自动重新搜索。

- `SearchNotes` 的 Go 方法签名新增 `scope`, `days`, `tagID` 三个参数
- `searchNotes()` 前端函数新增 `scope`, `days`, `tagID` 三个参数，默认值分别为 `'all'`, `0`, `0`
- `renderSearchResults()` 保持不变（仅渲染搜索结果列表）