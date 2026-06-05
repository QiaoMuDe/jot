# 统一通知系统 Spec

## Why
当前应用内存在多种通知样式：底部 `undo-toast` 堆叠通知、通用 `showToast` 复用 undo-toast UI、`showConfirmDialog` 自定义确认弹窗。通知方式不统一（位置、动画、关闭逻辑、视觉风格各异），造成用户困惑且维护困难。需重新设计一套从**右上角浮动弹出**的统一通知组件，替换所有旧有通知调用点。

## What Changes
- 设计全局 `NotificationManager` 组件：纯 JS 类，操作 DOM 而非框架绑定
- 通知类型：`success` / `error` / `warning` / `info` / `undo`
- 通知位置：**右上角固定**，纵向堆叠，单条自动消失（可配时长）
- `undo` 类型带操作按钮，保持 5s
- 替换所有旧有 `showToast` / `showUndoToast` / `showConfirmDialog` 调用点
- 删除旧有的 `animateToast` / `showToast` / `hideToast` / `showUndoToast` / `hideUndoToast` 函数及相关 DOM
- `showConfirmDialog` 保留（模态确认框非通知类），但统一其视觉风格（使用通知系统的色板变量）

## Impact
- Affected specs: 所有涉及用户通知的现有功能（CRUD 操作反馈、导入导出、备份还原、标签管理、回收站、批量删除、数据重置）
- Affected code: `frontend/src/main.js`（重写通知函数）、`frontend/src/style.css`（统一通知样式，删除旧 toast 样式）、`frontend/index.html`（替换 toast DOM 结构）

## ADDED Requirements

### Requirement: NotificationManager 类
The system SHALL provide a singleton `NotificationManager` class with the following API:
- `NotificationManager.show(message, type, duration?)` — 显示通知
- `NotificationManager.showUndo(message, onUndo, duration?)` — 显示可撤销通知
- 自动销毁超时通知，支持 `destroy()` 手动关闭

#### Scenario: 右上角浮动通知
- **WHEN** 用户执行操作触发通知
- **THEN** 通知从右上角滑入，带弹性动画，堆叠显示不影响其他通知
- **AND** 3s 后自动向左滑出消失（undo 类型 5s）

#### Scenario: 通知类型视觉区分
- **WHEN** 通知类型为 `success` / `error` / `warning` / `info`
- **THEN** 左侧有色标条 + 对应图标（✓ / ✕ / ⚠ / ℹ）+ 颜色区分
- **AND** 点击通知可手动关闭

#### Scenario: 可撤销通知
- **WHEN** `showUndo()` 被调用
- **THEN** 通知右侧出现「撤销」按钮，点击执行回调并关闭通知
- **AND** 通知 5s 后自动消失

### Requirement: 通知容器
The system SHALL 在 `body` 内维护一个 `.notification-container` 元素，position fixed，top: 16px，right: 16px，z-index: 2000

## MODIFIED Requirements

### Requirement: 现有通知调用点全部替换
将以下旧调用点统一替换为新通知 API：

| 旧调用点 | 新类型 | 说明 |
|----------|--------|------|
| `showToast('该标签已存在')` | `warning` | 标签重名提示 |
| `showToast('笔记已恢复')` | `success` | 回收站恢复 |
| `showToast(result.message)` (导入成功) | `success` | 导入结果 |
| `showToast(msg)` (导出成功) | `success` | 导出结果 |
| `showToast('创建标签失败')` | `error` | 标签创建失败 |
| `showToast('导入功能不可用')` | `error` | 后端未绑定 |
| `showToast('还原失败：' + err)` | `error` | 还原失败 |
| `showUndoToast(msg, noteIds)` | `undo` | 批量删除可撤销 |

### Requirement: showConfirmDialog 视觉同步
`showConfirmDialog` 保留模态确认框行为，但色板变量统一使用通知系统的色板，保持视觉一致性。

## REMOVED Requirements

### Requirement: 旧 Toast 函数
**Reason**: 被统一 NotificationManager 替代
**Migration**: 所有调用点替换为新 `nm.show()` / `nm.showUndo()`

### Requirement: 旧 Toast DOM 元素
**Reason**: 不再需要底部堆叠结构
**Migration**: 删除 `#undoToast` / `#undoToastMsg` / `#undoToastBtn` DOM 元素及相关样式
