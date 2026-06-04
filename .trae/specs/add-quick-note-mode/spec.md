# 快速笔记模式 Spec

## Why
用户希望在启动应用时直接进入全屏编辑模式，快速记录想法，省去先打开软件再点新建的步骤。

## What Changes
- 设置页新增「快速笔记」开关
- 启动时加载设置，若开启则自动打开全屏空白编辑器
- 保存/关闭编辑器后回到正常网格视图

## Impact
- Affected code: `frontend/index.html`, `frontend/src/main.js`, `frontend/src/style.css`
- Affected specs: 设置页 UI、启动流程

## ADDED Requirements

### Requirement: 快速笔记开关
系统在设置页 SHALL 提供一个「快速笔记」开关。

#### Scenario: 开关操作
- **WHEN** 用户进入设置页
- **THEN** 看到「快速笔记」开关和说明文字
- **WHEN** 用户切换开关
- **THEN** 设置持久化到后端 Setting 表

### Requirement: 启动时自动进入全屏编辑
系统在启用快速笔记后，启动时 SHALL 自动打开全屏编辑器。

#### Scenario: 启动流程
- **WHEN** 应用启动且 `quick_note_enabled = true`
- **THEN** 在所有设置和笔记加载完成后，自动调用 `openEditor(null)` 打开空白编辑器，并调用 `toggleEditorFullscreen()` 进入全屏模式
- **WHEN** 用户保存或取消编辑
- **THEN** 关闭编辑器，回到网格首页视图
