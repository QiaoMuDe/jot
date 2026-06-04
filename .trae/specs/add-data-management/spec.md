# 数据管理功能 Spec

## Why
用户目前无法查看笔记应用的统计数据，也没有数据备份和迁移的能力。需要提供一个数据管理页面，集中展示统计信息并支持数据导入导出。

## What Changes
- 更多菜单增加「数据管理」入口选项
- 新增数据管理视图，包含数据统计概览 + 数据导入导出功能
- 后端新增统计查询接口和导入导出接口

## Impact
- Affected specs: 更多菜单导航、视图切换
- Affected code:
  - `frontend/index.html` — 新增数据管理视图 DOM、更多菜单选项
  - `frontend/src/main.js` — 视图切换、渲染函数、事件绑定、导入导出交互
  - `frontend/src/style.css` — 数据管理页面样式
  - `app.go` — 新增绑定方法
  - `services/note_service.go` — 统计查询、导出、导入方法
  - `services/tag_service.go` — 统计查询方法
  - `models/note.go` — 可能需要辅助方法
  - `wailsjs/go/main/App.js` — 构建后自动更新

## ADDED Requirements

### Requirement: 数据管理入口
The system SHALL provide a "数据管理" entry in the 更多菜单下拉列表。

#### Scenario: 从菜单进入
- **WHEN** 用户点击顶部☰按钮展开更多菜单
- **THEN** 菜单中显示「数据管理」选项
- **WHEN** 用户点击「数据管理」
- **THEN** 切换到数据管理视图

### Requirement: 数据统计概览
The system SHALL display data statistics on the data management page。

#### Scenario: 查看统计
- **WHEN** 用户进入数据管理页面
- **THEN** 显示以下统计数据：
  - 笔记总数（未删除）
  - 标签总数
  - 回收站笔记数
  - 已置顶笔记数

### Requirement: 数据导出
The system SHALL support exporting all notes in JSON format。

#### Scenario: 导出笔记
- **WHEN** 用户点击「导出数据」按钮
- **THEN** 弹窗询问导出格式（JSON），确认后下载包含所有笔记（含标签）的 JSON 文件
- **THEN** 导出内容包括：标题、内容、颜色、置顶状态、标签、创建/更新时间

### Requirement: 数据导入
The system SHALL support importing notes from a JSON file。

#### Scenario: 导入笔记
- **WHEN** 用户点击「导入数据」按钮
- **THEN** 弹出文件选择对话框，接受 `.json` 文件
- **WHEN** 选择合法 JSON 文件
- **THEN** 解析并逐条创建笔记（含标签），显示导入结果（成功数/失败数）
- **WHEN** 选择无效格式文件
- **THEN** 提示格式错误，不执行导入
