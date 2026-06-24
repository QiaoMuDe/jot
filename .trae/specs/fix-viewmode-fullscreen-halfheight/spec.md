# 修复查看模式全屏后编辑器半高问题 Spec

## Why

用户在纯文本笔记的**查看模式**下点击全屏按钮后，编辑器区域（CM6）只占面板一半高度，下半部分被空白占据。而在**新建/编辑模式**下全屏正常，编辑器撑满可用高度。这是一个明显的回归/不一致 bug，影响查看笔记的体验。

## 根因分析

### 关键差异：`data-mode` 属性是否设置

| 步骤 | 新建/编辑模式 | 查看模式（纯文本） |
|------|--------------|-------------------|
| `openEditor()` 中 | 调用 `switchEditorMode('edit')` → **设置 `data-mode="edit"`** | **未设置 `data-mode`**（line 2537-2569 分支无 `switchEditorMode` 调用） |
| 初始状态 | `data-mode="edit"` → CSS 生效：`.md-rendered { display: none !important }` | `data-mode` 未定义 → 无 `[data-mode]` CSS 匹配，`.md-rendered` 靠 HTML 内联 `style="display:none"` 隐藏 |
| 全屏切换 Phase 3 | `mdRendered.style.display = ''` → CSS `!important` 接管 → `.md-rendered` 保持隐藏 | `mdRendered.style.display = ''` → **内联样式被清除** → `.md-rendered` 恢复为 `display: block`（浏览器默认），且带有 `flex: 1` |

### 为什么查看模式会出现半高

在 `toggleEditorFullscreen()` 的三阶段方案中，**Phase 3（350ms 后）** 执行：

```js
if (mdRendered) mdRendered.style.display = '';  // 清除内联样式
```

清除内联样式后，`.md-rendered` 的显示状态完全由 CSS 决定：

- **新建/编辑模式**（`data-mode="edit"`）：CSS `.editor-overlay[data-mode="edit"] .md-rendered { display: none !important }` → **隐藏，没问题**
- **查看模式**（`data-mode` 未定义）：无 `[data-mode]` 选择器命中 → `.md-rendered` 使用基类样式 `flex: 1`，浏览器默认 `display: block` → **可见，且占据 `flex: 1` 的空间**

由于 `.md-rendered` 和 `.editor-textarea`（CM6 容器）同为 `.editor-panes` 的 flex 子项，两者都有 `flex: 1`，在 `flex-direction: column` 布局下各占一半高度 → **编辑器只剩半高**。

### 完整的 flex 高度传播链

```
.editor-panel.fullscreen (height: calc(100vh - 56px), display: flex)
  └─ .editor-body (flex: 1, display: flex, flex-direction: column)
       └─ .editor-panes (flex: 1, display: flex, flex-direction: column)
            ├─ .editor-textarea (flex: 1, display: flex) → CM6
            └─ .md-rendered (flex: 1, display: block) ← 本应隐藏，但查看模式下可见
```

## What Changes

1. **在查看模式纯文本分支中设置 `data-mode="edit"`** — 在 `openEditor()` 的 view mode + text note 分支（line 2539-2541），在设置 `els.mdRendered.style.display = 'none'` 的同时，确保 `.editor-overlay` 的 `data-mode` 被设为 `'edit'`，使 CSS `display: none !important` 生效。

2. **（可选加固）设置后不再依赖内联样式** — 由于 CSS 已有 `!important`，设置 `data-mode="edit"` 后，`els.mdRendered.style.display = 'none'` 内联样式实际上是冗余的（`!important` 比内联优先级高），但保留不影响功能，可安全保留。

## Impact

- **Affected code**: `frontend/src/main.js` — `openEditor()` 函数中的 view mode 纯文本分支
- **Affected specs**: 无（这是一个纯 bug 修复，无关 spec 变更）
- **Behavior change**: 查看模式纯文本笔记全屏后，编辑器区域从半高变为全高

## ADDED Requirements

### Requirement: 查看模式全屏正确铺满
The system SHALL ensure that when a plain-text note is opened in view-only mode and the user toggles fullscreen, the CM6 editor area occupies 100% of the available height within `.editor-panel`.

#### Scenario: View mode text note → fullscreen
- **WHEN** user opens a plain-text note in view mode (read-only)
- **AND** clicks the fullscreen button
- **THEN** the CM6 editor area fills the entire available height below the editor header

#### Scenario: New note → fullscreen (regression check)
- **WHEN** user creates a new plain-text note
- **AND** clicks the fullscreen button
- **THEN** the CM6 editor area still fills the entire available height (no regression)

#### Scenario: Edit mode text note → fullscreen (regression check)
- **WHEN** user edits an existing plain-text note
- **AND** clicks the fullscreen button
- **THEN** the CM6 editor area still fills the entire available height (no regression)

#### Scenario: Markdown note view mode → fullscreen
- **WHEN** user opens a Markdown note in view mode
- **AND** clicks the fullscreen button
- **THEN** the preview content fills the entire available height (no regression)

## MODIFIED Requirements

无 — 这是一个纯 bug 修复，不修改任何既有需求。

## REMOVED Requirements

无。
