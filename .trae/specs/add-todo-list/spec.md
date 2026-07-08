# 待办清单功能 Spec

## Why
用户需要一个轻量级的待办清单功能来管理日常任务。作为笔记应用 Jot 的一部分，待办清单应与应用的温暖极简美学保持一致，并方便地从"更多"菜单快速访问。

## What Changes
- **后端新增**: Todo 数据模型 + TodoService（CRUD）+ app.go 绑定方法 + 数据库自动迁移
- **前端新增**: 待办清单页面 (`#viewTodo`) 作为一个新视图
- **菜单位置**: 在"更多"菜单 (`#moreMenu`) 中"AI 助手"下方添加"待办清单"入口
- 支持：添加、完成/取消完成、删除、编辑、筛选（全部/待办/已完成）

## Impact
- Affected specs: 更多菜单交互、视图切换系统
- Affected code:
  - `internal/models/todo.go` — 新增 Todo 模型
  - `internal/services/todo_service.go` — 新增 TodoService
  - `internal/database/db.go` — AutoMigrate 注册 Todo 模型
  - `app.go` — 新增 Todo CRUD 绑定方法
  - `frontend/index.html` — 新增 `#viewTodo` 视图和菜单项
  - `frontend/src/main.js` — 添加视图切换、事件处理、通过 Wails 调用后端
  - `frontend/src/css/components/todo.css` — 待办清单样式（新增）
  - `frontend/src/css/index.css` — 引入新样式文件

## ADDED Requirements

### Requirement: 后端数据模型
The system SHALL provide a Todo model persisted in SQLite via GORM.

#### Scenario: Todo 模型定义
- **WHEN** 应用启动
- **THEN** `Todo` 表通过 AutoMigrate 自动创建
- **AND** 包含字段: ID, Text, Done, CreatedAt, UpdatedAt

#### Scenario: CRUD 操作
- **WHEN** 前端调用 `CreateTodo(text)`
- **THEN** 后端创建新 Todo 记录并返回
- **WHEN** 前端调用 `ListTodos()`
- **THEN** 后端返回所有 Todo 列表，按创建时间倒序
- **WHEN** 前端调用 `ToggleTodo(id)`
- **THEN** 后端切换 Todo 的 Done 状态
- **WHEN** 前端调用 `DeleteTodo(id)`
- **THEN** 后端删除指定 Todo
- **WHEN** 前端调用 `UpdateTodo(id, text)`
- **THEN** 后端更新指定 Todo 的文本内容

### Requirement: 待办清单视图
The system SHALL provide a todo list view accessible from the "更多" menu.

#### Scenario: 打开待办清单
- **WHEN** 用户点击"更多"菜单中的"待办清单"
- **THEN** 应用切换到 `#viewTodo` 视图
- **AND** 通过 Wails 调用后端加载所有待办项

#### Scenario: 添加待办项
- **WHEN** 用户在输入框中输入内容并按 Enter
- **THEN** 调用后端 `CreateTodo` 创建新的待办项
- **AND** 待办项被添加到列表顶部
- **AND** 输入框被清空

#### Scenario: 完成/取消完成待办项
- **WHEN** 用户点击待办项左侧的复选框
- **THEN** 调用后端 `ToggleTodo` 切换状态
- **AND** 已完成项显示删除线样式

#### Scenario: 删除待办项
- **WHEN** 用户点击待办项右侧的删除按钮
- **THEN** 调用后端 `DeleteTodo` 删除
- **AND** 该待办项从列表中移除

#### Scenario: 编辑待办项
- **WHEN** 用户双击待办项文本
- **THEN** 文本变为可编辑状态
- **AND** 用户可修改内容
- **AND** 按 Enter 或失焦调用后端 `UpdateTodo` 保存修改

#### Scenario: 筛选待办项
- **WHEN** 用户点击筛选按钮（全部/待办/已完成）
- **THEN** 列表仅显示对应状态的待办项

#### Scenario: 统计信息
- **WHEN** 待办列表加载
- **THEN** 显示总项数和未完成项数

### 菜单位置
待办清单入口放在"更多"菜单的"AI 助手"下方，作为最后一项：
```
...
<div class="dropdown-divider"></div>
<div data-action="ai-chat">AI 助手</div>
<div data-action="todo">待办清单</div>  ← 新增
```

### Design Direction

采用 **温暖手写感** 设计方向：
- 色调：使用 Jot 默认主题的琥珀色 (`#D97706`) 作为品牌色，搭配亚麻白背景
- 字体：保持系统字体，但通过间距和粗细营造层次
- 复选框：使用 Jot 品牌色填充的圆形复选框，独特且温暖
- 空状态：使用柔和插图风格的提示文字
- 已完成项：使用柔和的淡出效果，而非生硬的隐藏
- 整体氛围：像一本精致的纸质笔记本中的待办页

## Design Identity

### 设计定位
- **产品类型**: Productivity - 轻量待办工具
- **目标用户**: Jot 笔记应用的现有用户，需要快速记录和管理任务
- **风格关键词**: warm, minimalist, organic, handcrafted, cozy

### 视觉风格
- **风格**: 温暖极简主义 (Warm Minimalism)
- **配色系统**:
  - 主色: 琥珀色 (`#D97706`) — 与 Jot 默认主题一致
  - 背景: 卡片白 (`#FFFFFF`) / 暖灰 (`#F7F5F0`)
  - 文本: 深褐 (`#2D2A24`) / 次级灰褐 (`#8B867C`)
  - 已完成: 柔和绿 (`#16A34A`) 的淡色背景
  - 强调: 使用 `--accent` CSS 变量自动适配所有主题

- **版式**:
  - 标题: 字号 1.25rem, 加粗 600
  - 待办文本: 字号 0.875rem, 常规 400
  - 统计信息: 字号 0.75rem, 次级色
  - 复选框: 自定义圆形，20×20px

- **交互细节**:
  - 添加待办: Enter 键 + 微弹入动画
  - 完成切换: 平滑过渡 200ms
  - 删除: 淡出 + 上移 200ms
  - 双击编辑: 边框高亮 + 自动聚焦
  - 悬停: 背景微亮 + 删除按钮显现

- **响应式**:
  - 与现有视图一致，使用 #mainContent 的滚动容器
  - 最大宽度 640px，居中对齐
  - 在小屏上左右边距自动收缩

### 动画系统
所有动画使用 CSS 过渡和关键帧：
1. **添加待办**: `todoEnter` — 从上方 12px 滑入 + 淡入 (200ms ease-out)
2. **完成待办**: 透明度 + 文本装饰线过渡 (200ms)
3. **删除待办**: `todoExit` — 淡出 + 向上收缩 (200ms ease-in)
4. **复选框切换**: 背景色 + 勾选图标过渡 (150ms)
5. **视图进入**: 复用 `.view-enter` 动画

## REMOVED Requirements
None.
