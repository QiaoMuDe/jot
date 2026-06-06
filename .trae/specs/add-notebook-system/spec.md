# 笔记本层级分组与内容隔离系统 Spec

## Why
当前 Jot 所有笔记平铺展示，随着笔记数量增长，用户难以按主题/项目/领域进行归类整理。需要一个笔记本（Notebook）体系将笔记分组管理，每个笔记本作为独立的知识容器，切换时只展示该笔记本下的内容。

## What Changes

### 核心交互逻辑
- **默认笔记本** — 首次启动时自动创建「默认笔记本」（id=1）
- **初始状态** — 应用启动后默认处于「默认笔记本」下，只显示该笔记本的笔记
- **内容严格隔离** — 当前激活哪个笔记本，页面就只显示该笔记本下的笔记，没有跨笔记本的「全部笔记」视图
- **首页 = 当前笔记本** — 回到首页时仍然在当前笔记本下，不切换
- **新建归属** — 新建笔记自动归入当前激活的笔记本
- **删除迁移** — 删除笔记本时，其下笔记自动迁入默认笔记本（不丢失数据）

### 新增内容
- **Notebook 模型** — 新增 `notebooks` 表，存储笔记本实体
- **Note.notebook_id** — Note 模型新增外键字段
- **数据库迁移** — AutoMigrate 新增 Notebook 模型 + 首次启动自动创建「默认笔记本」
- **存量迁移** — 所有存量笔记 notebook_id=0 → 自动归入「默认笔记本」
- **NotebookService** — 笔记本 CRUD 业务逻辑层
- **App 绑定** — 6 个新绑定方法
- **前端笔记本侧栏** — 主界面左侧新增笔记本列表面板
- **前端筛选逻辑** — 严格按 activeNotebookId 过滤，无「全部」模式
- **前端数据种子** — Seed 工具新增笔记本和笔记关联

### 模型设计

```
notebooks 表:
  - id          uint   (PK, AUTO_INCREMENT)
  - name        string (size:100, NOT NULL)
  - sort_order  int    (default:0)
  - created_at  time
  - updated_at  time
  - deleted_at  gorm.DeletedAt (软删除)

notes 表新增:
  - notebook_id uint   (gorm:"default:0;index")
  - 外键: 不强制 FK 约束
```

### 兼容性
- **首次启动**：Notebook 表为空时自动创建「默认笔记本」（id=1）
- **存量笔记**：所有 notebook_id=0 的笔记自动更新为 1（默认笔记本）
- **删除笔记本**：笔记本删除后，其下笔记 notebook_id 自动改为默认笔记本的 id，不丢失数据
- 不涉及现有字段变更，仅新增字段

## Impact
- **Affected specs**: add-card-note-app（基础 CRUD）
- **Affected code**:
  - `internal/models/notebook.go` — 新文件
  - `internal/models/note.go` — 新增 `NotebookID` 字段
  - `internal/services/notebook_service.go` — 新文件
  - `internal/services/note_service.go` — 扩展查询方法 + 存量迁移逻辑
  - `app.go` — 新增绑定 + 初始化 NotebookService + 默认笔记本初始化
  - `frontend/index.html` — 新增笔记本侧栏
  - `frontend/src/main.js` — 笔记本状态管理 + 筛选 + 操作
  - `frontend/src/style.css` — 侧栏样式
  - `frontend/wailsjs/` — 重新生成绑定

## ADDED Requirements

### Requirement: 默认笔记本自动创建
The system SHALL create a default notebook on first launch and migrate existing notes.

#### Scenario: 首次启动
- **WHEN** 应用首次启动且 notebooks 表为空
- **THEN** 自动创建名为「默认笔记本」的笔记本（id=1）
- **AND** 所有 notebook_id=0 的存量笔记更新为 notebook_id=1

#### Scenario: 后续启动
- **WHEN** 应用再次启动
- **THEN** 跳过创建，直接加载已有笔记本列表，默认选中「默认笔记本」

### Requirement: 笔记本 CRUD
The system SHALL support creating, renaming, and deleting notebooks.

#### Scenario: 新建笔记本
- **WHEN** 用户点击侧栏「✚」新建按钮
- **THEN** 弹出输入框 → 输入名称 → 创建成功 → 侧栏刷新 → 自动切换到该笔记本

#### Scenario: 重命名笔记本
- **WHEN** 用户右键笔记本 → 重命名
- **THEN** 该条目变为内联编辑输入框 → 修改名称 → 回车/失焦保存 → 侧栏更新

#### Scenario: 删除笔记本
- **WHEN** 用户右键笔记本 → 删除（非默认笔记本）
- **THEN** 确认弹窗「删除该笔记本后，其中的笔记将移至「默认笔记本」」→ 确认 → 笔记本软删除 → 笔记 notebook_id 改为默认笔记本的 id → 若当前激活的是被删笔记本则自动切到默认笔记本 → 侧栏刷新
- **AND** 不允许删除「默认笔记本」

### Requirement: 内容严格隔离
The system SHALL ONLY show notes belonging to the active notebook. 不存在跨笔记本查看全部笔记的功能。

#### Scenario: 切换笔记本
- **WHEN** 用户点击侧栏中的某个笔记本
- **THEN** 页面清除搜索、重置到第 1 页 → 仅加载该 notebook_id 的笔记
- **AND** 侧栏中该笔记本高亮显示
- **AND** 仅刷新笔记网格，不影响标签/设置等其他视图

#### Scenario: 新建笔记归属
- **WHEN** 用户在当前笔记本下新建笔记
- **THEN** 创建时自动使用当前 activeNotebookId
- **AND** 保存后该笔记出现在当前笔记本中，只能在该笔记本下看到

#### Scenario: 回到首页
- **WHEN** 用户按 ESC 或在子视图点「← 返回」回到首页
- **THEN** 仍然在当前笔记本下，不切换笔记本

### Requirement: 笔记本侧栏 UI
The system SHALL display a notebook sidebar on the main page.

#### Scenario: 侧栏展开态
- **WHEN** 用户进入主页面
- **THEN** 左侧展示笔记本侧栏（宽度 200px），包含：
  - 顶部标题（如「笔记本」或「📓」）
  - 所有用户创建的笔记本列表（名称 + 笔记数 badge），**默认笔记本始终排在最前面**
  - 底部「✚ 新建笔记本」按钮
  - 右上角「◀ 折叠」按钮

#### Scenario: 侧栏折叠态
- **WHEN** 用户点击折叠按钮
- **THEN** 侧栏宽度变为 44px（仅显示 📓 图标）
- **AND** 笔记网格区域自动扩展填满空白
- **AND** 折叠状态通过 `localStorage` 持久化

#### Scenario: 视觉指示
- **WHEN** 用户处于某个笔记本下
- **THEN** 该笔记本在侧栏中高亮（accent 色背景 + 加粗名称）
- **AND** 笔记本名称旁显示该笔记本下笔记数量 badge

### Requirement: 「默认笔记本」不可删除
The default notebook SHALL NOT be deletable.

#### Scenario: 右键菜单
- **WHEN** 用户右键「默认笔记本」
- **THEN** 菜单中「删除」选项灰色禁用或隐藏
- **AND** 仅「重命名」选项可用

## MODIFIED Requirements
（无现有需求被修改）

## REMOVED Requirements
- ~~「全部笔记」跨笔记本查看功能~~ — 用户明确要求始终只显示当前笔记本的笔记，无跨笔记本视图
- ~~编辑器笔记本选择器~~ — 新建自动归当前笔记本，无需选择器
- ~~「未分类」虚拟笔记本~~ — 存量笔记归入默认笔记本，删除笔记自动迁回默认笔记本
