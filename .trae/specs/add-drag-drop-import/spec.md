# 拖拽导入文件功能 Spec

## Why

用户需要将本地文件快速导入 Jot 成为笔记。通过拖拽文件到应用窗口，省去"新建 → 粘贴内容 → 改标题"的手动流程，提升内容录入效率。

## What Changes

- **后端新增** `ImportFile(paths []string)` 方法 — 批量处理拖拽文件，创建笔记
- **前端新增** `OnFileDrop` 回调注册 — 监听文件拖入事件，调用后端
- 拖拽成功后打开编辑器显示刚创建的笔记

## Impact

- Affected specs: 笔记创建、编辑器打开
- Affected code:
  - `app.go` — 新增 `ImportFile` 方法
  - `frontend/src/main.js` — 注册 `OnFileDrop` 回调 + 处理结果

## ADDED Requirements

### Requirement: 拖拽导入文件

The system SHALL allow users to drag files from file explorer into the application window to create notes.

#### Scenario: 成功导入单个文件
- **WHEN** 用户从文件管理器拖拽一个文件到 Jot 窗口
- **THEN** 系统读取文件内容，以文件名（去后缀）为标题创建笔记
- **AND** `.md` 文件类型设为 `markdown`，其他文件类型设为 `text`
- **AND** 笔记归入默认笔记本（id=1）
- **AND** 创建成功后打开编辑器显示该笔记

#### Scenario: 成功导入多个文件
- **WHEN** 用户同时拖拽多个文件到 Jot 窗口
- **THEN** 系统逐个处理每个文件，全部创建成功后
- **AND** 打开编辑器显示最后一个导入的笔记
- **AND** 侧边栏笔记数 badge 正确更新

#### Scenario: 拖入目录
- **WHEN** 用户拖拽一个文件夹到 Jot 窗口
- **THEN** 系统弹窗提示"不支持导入目录，请选择文件"
- **AND** 不创建任何笔记

#### Scenario: 拖入超大文件
- **WHEN** 用户拖入的文件大小超过 10MB
- **THEN** 系统弹窗提示"文件过大（超过 10MB），无法导入"

#### Scenario: 文件读取失败
- **WHEN** 拖入的文件无法读取（权限不足、被占用等）
- **THEN** 系统跳过该文件并显示通知提示具体文件名和失败原因

#### Scenario: 混合拖入文件和目录
- **WHEN** 用户同时拖入文件 + 目录
- **THEN** 系统仅处理其中的文件，忽略目录
- **AND** 显示通知提示有多少个目录被跳过

## REMOVED Requirements

N/A — 全新功能，不涉及移除。
