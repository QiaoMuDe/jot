# 拆分数据管理页面代码 Spec

## Why

`frontend/src/main.js` 已膨胀至难以维护（超过 237KB），所有页面逻辑都集中在一个文件中。需要按页面维度拆分代码，优先将数据管理页面的相关逻辑提取到独立文件，提高可维护性而不影响现有功能。

## What Changes

- 在 `frontend/src/js/` 目录下新建 `data-management.js` 文件
- 将 `main.js` 中数据管理页面的所有函数定义移至新文件
- 保留 `main.js` 中对数据管理函数的调用入口（事件绑定、视图切换）
- 通过 `window` 全局对象作为 JS 模块间通信的桥梁（当前架构模式）
- 不修改 HTML 和 CSS，不影响现有功能

## Impact

- Affected specs: 无（纯重构，无功能变更）
- Affected code:
  - `frontend/src/main.js` — 移除数据管理函数定义，通过 `window` 引用外部函数
  - `frontend/src/js/data-management.js` — **新文件**，包含数据管理所有函数
- No breaking changes

## ADDED Requirements

### Requirement: 新增 data-management.js 文件

系统应新增 `frontend/src/js/data-management.js` 文件。

#### Scenario: 文件内容正确性

- **WHEN** 加载 data-management.js
- **THEN** 所有数据管理函数应被正确导出到 `window` 对象

### Requirement: 函数迁移完整性

以下函数应完整迁移到新文件：

| 函数名 | 行号（原） | 说明 |
|---|---|---|
| `animateCountUp` | ~1880 | 数字递增动画（辅助函数） |
| `loadDataStats` | ~1900 | 加载数据统计概览 |
| `resetDatabase` | ~1954 | 恢复出厂设置 |
| `vacuumDatabase` | ~1997 | 数据库瘦身 |
| `openDataDir` | ~2015 | 打开数据目录 |
| `exportData` | ~2031 | 导出数据 |
| `importData` | ~2050 | 导入数据 |
| `loadBackupInfo` | ~2075 | 加载备份信息 |
| `backupToDir` | ~2097 | 一键备份 |
| `restoreFromDir` | ~2125 | 一键还原 |

#### Scenario: 函数可调用

- **WHEN** 用户点击数据管理页面的操作按钮
- **THEN** 对应函数应正常执行，不影响原有功能

### Requirement: 依赖关系处理

新文件应通过 `window` 对象引用以下外部依赖：

- `nm`（NotificationManager 实例）— 用于显示通知
- `els`（DOM 引用对象）— 用于操作 DOM 元素
- `SVGS`（SVG 图标常量）— 用于生成图标
- `showConfirmDialog` — 用于确认对话框
- `loadNotes`, `loadTags`, `loadNotebooks`, `switchView` — 用于页面切换和数据刷新
- `state`（应用状态）— 用于读取/更新状态

#### Scenario: 依赖可用

- **WHEN** data-management.js 中的函数执行
- **THEN** 应能通过 `window` 正常访问上述依赖

## MODIFIED Requirements

### Requirement: main.js 变更

修改前：包含数据管理页面的完整函数定义（~1872~2157 行）
修改后：移除这些函数定义，仅保留事件绑定和视图切换中的调用

## REMOVED Requirements

无
