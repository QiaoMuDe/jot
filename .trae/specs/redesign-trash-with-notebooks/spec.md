# 回收站支持笔记本条目 Spec

## Why

当前回收站只支持笔记的恢复和永久删除，笔记本删除后不进入回收站，"删除笔记本并清空所有笔记"会硬删除笔记（不可还原）。需要将回收站改造为同时支持笔记和笔记本两种条目，统一回收站管理，让用户能还原误删的笔记本和笔记。

## What Changes

### 后端 - NotebookService 修改
- **`DeleteWithNotes` 改为软删除笔记（进回收站）**：笔记本软删除的同时，其下所有笔记也软删除（进回收站），而非当前硬删除
- **新增回收站操作方法**：`GetTrash`, `RestoreFromTrash`, `PermanentDeleteFromTrash`, `RestoreAllFromTrash`, `EmptyTrash`，与 NoteService 的回收站操作对标

### 后端 - App 绑定
- 新增对应的 Wails 绑定方法：`GetTrashNotebooks`, `RestoreTrashNotebook`, `PermanentDeleteTrashNotebook`, `RestoreAllTrashNotebooks`, `EmptyTrashNotebooks`

### 前端 - 回收站页面重构
- `trash-page.js` 同时加载回收站的笔记和笔记本两种条目
- 渲染时区分条目类型：笔记显示标题+删除时间，笔记本显示名称+删除时间，各自有不同的图标
- 两种条目各自有"恢复"和"永久删除"按钮
- "全部恢复"同时恢复所有笔记和笔记本
- "全部清空"同时永久删除所有笔记和笔记本
- 还原笔记本时无需处理笔记引用（笔记本恢复后，回收站中引用它的笔记在单独恢复时自动关联）
- 永久删除笔记本时，回收站中引用该笔记本的笔记自动迁到默认笔记本

### 前端 - 笔记本侧栏联动
- 删除笔记本后，若当前激活笔记本是被删除的笔记本，自动切到默认笔记本

## Impact

- Affected specs: 笔记本系统、回收站页面
- Affected code:
  - `internal/services/notebook_service.go` — DeleteWithNotes 修改 + 新增回收站方法
  - `app.go` — 新增绑定方法
  - `frontend/src/js/trash-page.js` — 重构渲染逻辑
  - `frontend/src/css/components/main-content.css` — 笔记本条目样式（徽章/图标）
- No model changes needed (Notebook 已有 `gorm.DeletedAt`)

## ADDED Requirements

### Requirement: 笔记本回收站操作
The system SHALL support viewing, restoring, and permanently deleting notebooks from the recycle bin.

#### Scenario: 查看回收站笔记本
- **WHEN** 用户打开回收站页面
- **THEN** 页面同时展示已软删除的笔记列表和已软删除的笔记本列表，按删除时间倒序排列，两种条目混合展示，用不同类型的图标区分

#### Scenario: 恢复笔记本
- **WHEN** 用户在回收站中对笔记本条目点击"恢复"
- **THEN** 笔记本取消软删除，重新出现在笔记本侧栏中，弹出成功提示

#### Scenario: 永久删除笔记本
- **WHEN** 用户在回收站中对笔记本条目点击"永久删除"
- **THEN** 笔记本被硬删除（从数据库中彻底移除），回收站中引用该笔记本的笔记自动迁到默认笔记本，弹出提示

#### Scenario: 全部恢复（含笔记本）
- **WHEN** 用户在回收站点击"全部恢复"
- **THEN** 回收站中所有笔记和笔记本都被恢复

#### Scenario: 全部清空（含笔记本）
- **WHEN** 用户在回收站点击"全部清空"
- **THEN** 回收站中所有笔记和笔记本都被永久删除（引用已删除笔记本的笔记先迁到默认笔记本）

## MODIFIED Requirements

### Requirement: 删除笔记本并清空所有笔记
Before: 笔记本软删除，其下笔记硬删除（永久丢失）
After: 笔记本软删除，其下笔记也软删除（进入回收站，可还原）

#### Scenario: 删除笔记本并清空所有笔记
- **WHEN** 用户在删除笔记本对话框中勾选"同时永久删除该笔记本中的 N 条笔记"并确认
- **THEN** 笔记本被软删除（进回收站），其下所有笔记也被软删除（进回收站）
- 对话框文案从"同时永久删除该笔记本中的 N 条笔记（不进回收站）"改为"同时将该笔记本中的 N 条笔记移入回收站"

### Requirement: 回收站状态栏联动
- **WHEN** 回收站中的笔记本恢复后
- **THEN** 加载笔记本列表（`loadNotebooks()`）
- **WHEN** 回收站中的笔记恢复后
- **THEN** 加载笔记列表（`loadNotes()`）+ 加载笔记本列表（`loadNotebooks()`）