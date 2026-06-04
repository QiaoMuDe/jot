# 笔记排序设置 + 按需加载 Spec

## Why

当前笔记列表在应用启动时一次性加载全部笔记（`GetNotes(1, 100)`），每次 CRUD 后也全量刷新。当笔记数量增多时会有性能问题。同时排序方式硬编码为 `updated_at DESC`，用户无法自定义。

## What Changes

### 1. 设置页新增「笔记排序」和「分页大小」选项
- 在设置页「标签管理」下方新增「笔记排序」分区
- 支持三种排序方式：按更新时间（默认）、按创建时间、按名称（A-Z）
- 在排序下方新增「分页大小」选项：水平滑块（`<input type="range">`），范围 9~54，步长 9，当前值实时显示
- 切换后立即生效并保存到后端 Setting

### 2. 笔记列表全局改为按需加载（懒加载）
- **所有触发场景**（启动、CRUD 后、刷新等）均采用分页加载，只加载第 1 页
- 滚动到卡片网格底部附近时自动加载下一页，追加到已有列表
- 全部加载完成后在底部显示「共 X 条笔记」提示

### 3. 搜索/回收站由后端处理
- 搜索视图：后端 `SearchNotes` 已在数据库层面执行 `LIKE` 查询，前端仅渲染结果
- 回收站视图：后端 `GetTrashNotes` 已在数据库层面查询 `deleted_at IS NOT NULL`，前端仅渲染结果
- 这两个视图保持现有实现不变，不涉及全量加载到前端过滤

### 4. 后端支持动态排序
- `GetAll` 新增 `sortBy` 参数，动态生成 ORDER BY 子句
- `GetByTag` 同样支持动态排序

## Impact

- Affected specs: 无
- Affected code:
  - `internal/services/note_service.go` — `GetAll`/`GetByTag` 新增 `sortBy` 参数
  - `app.go` — `GetNotes`/`GetNotesByTag` 透传 `sortBy`，新增 `GetSortOrder`/`SetSortOrder` 绑定
  - `frontend/index.html` — 设置页新增排序选项和分页大小滑块 DOM
  - `frontend/src/style.css` — 排序选项 + 滑块样式
  - `frontend/src/main.js` — 懒加载逻辑 + 排序/分页切换事件

## ADDED Requirements

### Requirement: 排序设置
The system SHALL 提供在设置页中选择笔记排序方式的功能。

#### Scenario: 切换排序
- **WHEN** 用户在设置页选择新的排序方式
- **THEN** 笔记列表立即按新排序重新加载，且排序偏好保存到后端

#### Scenario: 排序选项
- **WHEN** 用户打开设置页
- **THEN** 应看到三种排序选项：按更新时间、按创建时间、按名称

#### Scenario: 持久化
- **WHEN** 用户关闭应用后重新打开
- **THEN** 笔记列表应沿用上次选择的排序方式

### Requirement: 分页大小设置
The system SHALL 提供在设置页中自定义每次查询笔记数量的功能。

#### Scenario: 设置分页大小
- **WHEN** 用户拖动水平滑块
- **THEN** 当前值实时显示（如「每次加载 18 条」），滑块释放后笔记列表立即按新大小重新加载，且偏好保存到后端

#### Scenario: 滑块范围
- **WHEN** 用户操作滑块
- **THEN** 最小值为 9，最大值为 54，步长为 9（即可选值：9/18/27/36/45/54）

#### Scenario: 默认值
- **WHEN** 用户首次使用或未设置过分页大小
- **THEN** 默认分页大小为 18

### Requirement: 按需加载
The system SHALL 在所有触发场景均采用分页懒加载方式加载笔记列表。

#### Scenario: 首次加载
- **WHEN** 应用启动
- **THEN** 只加载第 1 页（按用户设置的分页大小）笔记

#### Scenario: CRUD 后刷新
- **WHEN** 用户创建/更新/删除/置顶/恢复笔记
- **THEN** 重置分页状态，重新从第 1 页加载

#### Scenario: 滚动加载更多
- **WHEN** 用户滚动到卡片网格底部附近（距底部 < 200px）
- **THEN** 自动加载下一页笔记，追加到列表末尾

#### Scenario: 全部加载完毕
- **WHEN** 所有笔记已加载完毕
- **THEN** 底部显示「共 X 条笔记」提示，不再触发加载

### Requirement: 搜索/回收站后端处理
搜索和回收站视图的查询 SHALL 在数据库层面完成。

#### Scenario: 搜索
- **WHEN** 用户在搜索框输入关键词
- **THEN** 后端执行 SQL `WHERE title LIKE '%kw%' OR content LIKE '%kw%'`，返回分页结果

#### Scenario: 回收站
- **WHEN** 用户进入回收站视图
- **THEN** 后端执行 SQL `WHERE deleted_at IS NOT NULL`，返回分页结果

## MODIFIED Requirements

### Requirement: 后端排序
`note_service.go` 的 `GetAll` 方法 SHALL 支持动态 `sortBy` 参数。

- `updated_at` → `ORDER BY pinned DESC, updated_at DESC`
- `created_at` → `ORDER BY pinned DESC, created_at DESC`
- `title` → `ORDER BY pinned DESC, title ASC`
