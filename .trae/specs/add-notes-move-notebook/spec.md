# 笔记跨笔记本迁移 Spec

## Why

当前笔记本中的笔记无法迁移到另一个笔记本。用户只能把笔记本 A 整个删除（此时其中的笔记全部迁入默认笔记本），缺乏精确、灵活的笔记跨笔记本迁移能力。需要支持**单条笔记**和**批量笔记**迁移到任意目标笔记本。

## What Changes

### 核心交互逻辑

- **右键单条笔记 →「移动到...」** — 在右键菜单中新增「移动到...」入口，点击弹出目标笔记本选择器，选中即执行迁移
- **批量模式 →「移动到」** — 批量操作栏新增「移动到」按钮，选中多条笔记后弹出同样的选择器，批量迁移
- **目标笔记本选择器** — 轻量模态弹窗，列出所有可用笔记本（排除当前笔记本），radio 列表选择，确定后执行
- **迁移后行为** — 留在当前笔记本视图不切换，已迁走的笔记从列表消失，源/目标笔记本 badge 计数同时更新
- **排除自身** — 选择器自动过滤当前笔记本，不能移到自身

### 新增内容

- **后端 `MoveToNotebook`** — 单条笔记迁移，直接 `UpdateColumn("notebook_id", targetID)` 不更新时间
- **后端 `BatchMoveToNotebook`** — 批量迁移，遍历 noteIDs 逐个更新
- **App 绑定** — `MoveNoteToNotebook` / `BatchMoveNotesToNotebook` 两个新绑定方法
- **前端右键菜单项** — 在「编辑」和「置顶」之间新增「移动到...」
- **前端批量工具栏按钮** — 在「-标签」和「批量删除」之间新增「移动到」
- **前端目标笔记本选择器弹窗** — 模态弹窗 + notebook radio 列表 + 确定/取消
- **前端 badge 同步更新** — 迁移完成后同时更新源/目标笔记本的计数

### 兼容性

- 不修改现有笔记字段或数据模型
- 不改变现有笔记本删除逻辑
- 不影响现有右键菜单和批量模式其他功能

## Target Notebook Selector Dialog 设计

### 设计定位

与 Jot 现有设计语言保持一致：温暖极简、轻量毛玻璃、弹簧动效。选择器作为临时操作弹窗，不喧宾夺主，聚焦「快速选择目标」的核心任务。

### 视觉风格

- **遮罩层**：半透明深色遮罩 + `backdrop-filter: blur(4px)`，与应用的 confirm-overlay 风格统一
- **弹窗卡片**：白/暗色圆角卡片（`border-radius: 14px`），轻薄阴影（`box-shadow`），轻微边框（`border: 1px solid var(--border)`）
- **入场动画**：`elasticScale` 弹簧缩放动画（`cubic-bezier(0.16, 1, 0.3, 1)`），与编辑器/关于页弹窗动画一致
- **标题区**：左上角「移动到...」+ 右上角 `✕` 关闭按钮
- **笔记本列表**：radio 式列表，每行一个笔记本，左侧 radio 圆圈 + 笔记本名称 + 右侧笔记数 badge
  - radio 圆圈使用自定义样式（accent 色填充 + 外圈），隐藏原生 radio
  - 每行 hover 时浅色底色
  - 已选项有选中态视觉（accent 色 radio 填充 + 稍深底色）
- **底部操作栏**：分隔线 + 「取消」和「移动到」两个按钮
  - 「取消」：透明背景，hover 变浅灰
  - 「移动到」：accent 色填充按钮，hover 加深，关键操作视觉强调
- **最大宽度** `360px`，随内容自适应高度
- **暗色主题**：遮罩更暗，卡片用 `--card-bg`，所有变量跟随主题系统

### 交互流程

```
右键笔记 / 批量选中 →
  弹出目标笔记本选择器（弹簧动画入场）
  → 用户点击某个笔记本（radio 选中）
  → 点击「移动到」按钮
  → 选择器关闭（淡出）
  → 执行后端 API
  → Toast 通知「已移动 N 条笔记到「xxx」」
  → 刷新笔记列表 + badge 计数
```

### 动画

- **入场**：`elasticScale` 0.35s spring
- **出场**：`fadeOut` 0.15s ease
- **遮罩**：opacity 过渡 0.2s

### 6 主题适配

使用 CSS 变量体系（`--bg`/`--card-bg`/`--accent`/`--border`/`--text-primary`/`--text-muted`），无需为各主题硬编码颜色。通过 `color-mix()` 处理 hover/active 态。

## Impact

- **Affected specs**: add-notebook-system（笔记本系统基础）
- **Affected code**:
  - `internal/services/note_service.go` — 新增 `MoveToNotebook` / `BatchMoveToNotebook`
  - `app.go` — 新增 2 个绑定方法
  - `frontend/index.html` — 右键菜单 +1 项，批量栏 +1 按钮，新增选择器弹窗 HTML
  - `frontend/src/main.js` — 选择器逻辑 + 迁移函数 + badge 同步
  - `frontend/src/style.css` — 选择器弹窗样式

## ADDED Requirements

### Requirement: 后端笔记迁移
The system SHALL support moving notes between notebooks.

#### Scenario: 单条笔记迁移
- **WHEN** 用户右键单条笔记 → 移动到 → 选择目标笔记本 → 确认
- **THEN** 后端执行 `UpdateColumn("notebook_id", targetID)` — UpdatedAt 不变
- **AND** 返回成功

#### Scenario: 批量笔记迁移
- **WHEN** 用户在批量模式下选中多条笔记 → 移动到 → 选择目标笔记本 → 确认
- **THEN** 后端遍历所有 noteIDs 逐个更新 `notebook_id`
- **AND** 返回成功（所有都成功或全部回滚？全部回滚过于复杂，先实现逐个更新，失败时返回错误信息，已迁的不回滚）

#### Scenario: 校验
- **WHEN** 目标笔记本不存在（已被删除）
- **THEN** 返回错误「目标笔记本不存在」
- **WHEN** 目标笔记本与当前笔记本相同
- **THEN** 选择器中该笔记本不显示（前端过滤）

### Requirement: 目标笔记本选择器 UI
The system SHALL display a modal dialog for selecting the target notebook.

#### Scenario: 打开选择器
- **WHEN** 用户点击右键菜单「移动到...」或批量模式「移动到」
- **THEN** 遮罩 + 弹窗弹簧动画入场
- **AND** 弹窗列出所有笔记本（排除当前笔记本），每个显示名称 + 笔记数
- **AND** 默认无选中项，「移动到」按钮禁用

#### Scenario: 选择后确认
- **WHEN** 用户点击某个笔记本（radio 选中）
- **THEN** 该行高亮 + radio 填充 accent 色
- **AND** 「移动到」按钮变为可用
- **WHEN** 用户点击「移动到」
- **THEN** 弹窗淡出 → 执行迁移 → Toast 通知
- **AND** 刷新笔记列表（`loadNotes()`）
- **AND** 更新源/目标笔记本的 badge 计数（`renderNotebookList()`）

#### Scenario: 取消
- **WHEN** 用户点击「取消」/ `✕` / 遮罩层
- **THEN** 弹窗淡出关闭，不执行任何操作

#### Scenario: 空状态
- **WHEN** 当前只有一个笔记本（仅默认笔记本），无其他可选
- **THEN** 弹窗提示「没有其他笔记本可供选择」+ 仅一个「关闭」按钮

### Requirement: badge 计数同步
The system SHALL update notebook note counts after moving notes.

#### Scenario: 迁移后刷新
- **WHEN** 迁移操作成功完成
- **THEN** `loadNotebooks()` 重新获取各笔记本笔记数
- **AND** `renderNotebookList()` 重新渲染侧栏，源笔记本 badge -N，目标笔记本 badge +N
