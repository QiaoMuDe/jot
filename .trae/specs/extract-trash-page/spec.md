# 提取回收站页面模块 Spec

## Why
`frontend/src/main.js` 文件已膨胀至超过 227KB，难以维护。按照页面维度拆分代码，首先将回收站（Trash）页面的相关逻辑提取到独立的 JS 模块文件中，降低 main.js 的复杂度。

## What Changes
- 新建 `frontend/src/js/trash-page.js`，包含回收站页面的所有逻辑
- 从 `main.js` 中移除回收站相关的函数定义、部分 state 字段、部分 DOM 引用
- 将回收站相关的函数和状态导出为全局可访问的 API（通过 `window` 对象）
- `main.js` 通过 import 引入 `trash-page.js`
- 保证所有原有功能不受影响：视图切换、事件监听、键盘快捷键、菜单导航、数据加载

## Impact
- Affected specs: 无（新模块提取）
- Affected code:
  - `frontend/src/main.js` — 移除回收站函数定义，保留 import 和事件绑定关联
  - `frontend/src/js/trash-page.js` — **新文件**，回收站所有逻辑
  - `frontend/index.html` — 无需修改

## ADDED Requirements

### Requirement: 回收站模块提取
系统 SHALL 将回收站相关逻辑提取到独立文件中。

#### Scenario: 模块结构
- **WHEN** 用户访问回收站页面
- **THEN** 所有回收站功能（加载、渲染、恢复、永久删除、清空、全部恢复）由 `trash-page.js` 提供

#### Scenario: 全局 API
- **WHEN** 模板中的 `onclick` 调用 `window.restoreNote()` 或 `window.permanentDeleteNote()`
- **THEN** 这些函数必须在 `trash-page.js` 中挂载到 `window` 对象上

## MODIFIED Requirements

### Requirement: main.js 瘦身
main.js SHALL 移除以下回收站相关内容：
- `state.trashNotes` 定义
- `els.trashList`, `els.trashListInner`, `els.trashBackBtn`, `els.restoreAllBtn`, `els.emptyTrashBtn` DOM 引用
- 函数：`loadTrashNotes`, `restoreNote`, `restoreAllNotes`, `emptyTrash`, `permanentDeleteNote`, `renderTrashList`
- 保留 `switchView` 中对 `'trash'` 的 case 处理（可简化为调用模块函数）
- 保留事件绑定中调用回收站函数的部分（改为调用模块导出的函数）

### Requirement: 无功能回退
系统 SHALL 确保拆分后所有回收站功能正常：
- `trash-back-btn` 返回网格视图
- `restore-all-btn` 全部恢复
- `empty-trash-btn` 全部清空
- 每条笔记的「恢复」「永久删除」按钮
- Ctrl+5 快捷键导航到回收站
- 更多菜单"回收站"项导航
- visibilitychange 事件中回收站视图刷新
- 统计面板 `statTrashedNotes` 数据更新
