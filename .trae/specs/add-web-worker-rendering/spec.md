# Web Worker 离线程预览渲染 Spec

## Why

大文件 Markdown 预览时 `marked.parse()` + DOM 操作在**主线程同步执行**，UI 被阻塞无法响应用户操作（滚动、切换按钮、编辑器输入等）。需要将 CPU 密集的 Markdown 解析移到 Web Worker 线程。

## What Changes

- 新增 `frontend/src/preview-worker.js` Web Worker 文件，负责 `marked.parse()` 离线程解析
- 修改 `frontend/src/main.js` 的 `updatePreview()` 函数，改为向 Worker postMessage 触发异步渲染
- 新增加载状态指示器（`.md-rendered` 内显示"渲染中…"脉动提示）

## Impact

- Affected specs: Markdown 渲染预览
- Affected code: `frontend/src/main.js`（updatePreview 函数）、`frontend/src/style.css`（加载状态样式）

## ADDED Requirements

### Requirement: Web Worker 离线程渲染
The system SHALL use a Web Worker to perform Markdown parsing in a background thread.

#### Scenario: 大文件预览
- **WHEN** 用户打开大文件（>50KB Markdown）并切换到预览模式
- **THEN** UI 保持流畅可响应，预览区域显示加载状态后平滑显示渲染结果

#### Scenario: 小文件预览
- **WHEN** 用户预览小文件（<50KB Markdown）
- **THEN** Worker 几乎瞬间返回，加载状态一闪而过或不显示

### Requirement: 内容哈希缓存
The system SHALL skip re-rendering when content hasn't changed.

#### Scenario: 相同内容不重渲染
- **WHEN** `updatePreview()` 被重复调用且内容未变化
- **THEN** 直接 return，不触发 Worker 和 DOM 操作

### Requirement: 加载状态指示
The system SHALL show a loading indicator during Worker rendering.

#### Scenario: 渲染中状态
- **WHEN** Worker 正在解析 Markdown
- **THEN** `.md-rendered` 内显示脉动的"渲染中…"提示，Worker 返回后隐藏

## MODIFIED Requirements

### Requirement: updatePreview 函数

**修改前**：同步执行 `marked.parse()` + `innerHTML` + `hljs` 高亮 + 按钮/标签 DOM 创建

**修改后**：检查内容哈希 → 无变化 return → 显示加载状态 → postMessage(content) 到 Worker → Worker 内 `marked.parse()` → postMessage(html) 回主线程 → `innerHTML` + `hljs` 高亮 + 按钮/标签 DOM 创建 → 隐藏加载状态

## REMOVED Requirements

无
