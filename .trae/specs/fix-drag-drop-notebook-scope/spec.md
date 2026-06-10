# 拖拽导入按当前笔记本作用域修正 Spec

## Why
拖拽导入的文件始终写入默认笔记本（notebook_id=1），与用户当前所在的笔记本不一致。用户期望文件导入到当前正在浏览的笔记本中。

## What Changes
- 前端 `handleFileDropPaths()` 调用 `ImportFiles` 时传入当前 `state.activeNotebookId`
- 后端 `ImportFiles` 新增 `notebookID` 参数，提取标题和类型后调用 `CreateWithNotebook(nil, title, content, noteType, notebookID)`
- 前端导入完成后刷新列表时确保 `activeNotebookId` 上下文正确

## Impact
- Affected specs: `add-drag-drop-import`
- Affected code:
  - `app.go` — `ImportFiles` 方法签名变更
  - `frontend/src/main.js` — `handleFileDropPaths()` 调用变更

## MODIFIED Requirements

### Requirement: ImportFiles 接受笔记本参数

#### Scenario: 在默认笔记本中拖拽导入
- **WHEN** 用户在默认笔记本（ID=1）中拖拽文件导入
- **THEN** 笔记创建到笔记本 1，侧边栏笔记本 1 的 badge 计数 +1

#### Scenario: 在指定笔记本中拖拽导入
- **WHEN** 用户切换到指定笔记本（如 ID=2）后拖拽文件导入
- **THEN** 笔记创建到笔记本 2，侧边栏笔记本 2 的 badge 计数 +1，笔记本 1 计数不变

#### Scenario: 导入成功后刷新列表
- **WHEN** 导入完成
- **THEN** `loadNotebooks()` 刷新 badge 计数，`loadNotes()` 显示当前笔记本的新笔记
