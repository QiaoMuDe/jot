# 移除快速笔记功能 Spec

## Why

快速笔记功能（启动时自动打开全屏新建编辑器）存在启动闪烁问题，且反复修复未能彻底解决。用户决定直接移除该功能，简化代码，消除维护负担。

## What Changes

- 移除设置页中的"快速笔记"开关 UI（checkbox + 提示文字）
- 移除前端 JS 中所有快速笔记相关逻辑（DOM 引用、事件监听、启动触发、设置加载/保存）
- 移除后端 Go `SettingsConfig` 结构体中的 `QuickNoteEnabled` 字段及相关读写
- 移除 TypeScript 模型中的 `quick_note_enabled` 字段
- 移除数据库默认值中的 `quick_note_enabled` 条目
- **不修改** CSS 文件：`.quick-note-hint` 类同时被"代码语法高亮"的提示使用，只需移除对应的 HTML 元素，CSS 保留不受影响

## Impact

- 影响设置页 UI（减少一行设置项）
- 影响应用启动行为（不再自动打开编辑器）
- 影响后端设置结构体（减少一个字段）
- 影响数据库默认值（减少一条 KV 记录）

## REMOVED Requirements

### Requirement: 快速笔记

**Reason**: 功能存在启动闪烁问题，用户决定移除
**Migration**: 用户如需快速记录，可手动点击 "+" 按钮或使用 Ctrl+N 快捷键

