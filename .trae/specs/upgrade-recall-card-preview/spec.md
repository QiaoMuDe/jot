# 召回卡片预览升级为完整笔记查看 Spec

## Why

AI 消息中的卡片召回预览目前使用独立的简陋浮层（`aiCardPreviewModal`），渲染能力和 UI 远不如笔记首页的 `openEditor` 只读视图——缺少代码块复制按钮、语言标签、标签展示、查找替换、字数统计等功能。复用已有的 `openEditor` 可以实现零新增组件的完整笔记查看体验。

## What Changes

- **修改**：点击召回卡片时，改为调用 `window.openEditor(card.id, true, false)` 在只读模式下打开完整编辑器视图
- **新增**：`openEditor` 新增 `hideEditBtn` 参数，控制是否隐藏"编辑"按钮，从召回卡片打开时隐藏
- **移除**：移除 `openCardPreview` 函数
- **移除**：移除 `aiCardPreviewModal` 相关 HTML 结构
- **移除**：移除卡片预览相关 CSS（`ai-card-preview-modal` 等）
- **移除**：移除 `_cardPreviewWorker` 变量

## Impact

- Affected specs: `add-card-recall`
- Affected code: `frontend/src/js/ai-chat.js`, `frontend/src/main.js`, `frontend/index.html`, `frontend/src/css/components/ai-chat.css`

---

## ADDED Requirements

### Requirement: `openEditor` 支持隐藏编辑按钮

The system SHALL support a `hideEditBtn` parameter in `openEditor` to control visibility of the "编辑" entry button.

#### Scenario: 从召回卡片打开
- **WHEN** `openEditor(noteId, true, false, true)` is called
- **THEN** `els.editorEditBtn.style.display` is set to `'none'`
- **AND** no clickable entry point to enter edit mode is shown

#### Scenario: 从笔记列表打开（只读模式，默认行为不变）
- **WHEN** `openEditor(noteId, true, false)` is called (without `hideEditBtn`)
- **THEN** `els.editorEditBtn.style.display` is set to `''` (visible, current behavior)

### Requirement: 召回卡片点击改用 `openEditor`

The system SHALL replace `openCardPreview(card)` with `window.openEditor(card.id, true, false, true)` when clicking a recall card item.

#### Scenario: 点击召回卡片
- **WHEN** user clicks a recall card item in the AI chat message
- **THEN** `window.openEditor(card.id, true, false, true)` is called
- **AND** the full editor view opens in read-only mode with edit button hidden
- **AND** the note is displayed with full rendering (Markdown preview, code copy buttons, language badges, tags, etc.)

---

## MODIFIED Requirements

### Requirement: `openEditor`（已存在）

Update the function signature to accept an optional 4th parameter: `(noteId, readOnly, startFullscreen, hideEditBtn)`.

---

## REMOVED Requirements

### Requirement: `openCardPreview` 函数

**Reason**: No longer needed — replaced by `openEditor` which provides a superior viewing experience.

### Requirement: `aiCardPreviewModal` HTML 结构

**Reason**: The custom preview modal is replaced by the built-in editor view.

### Requirement: 卡片预览相关 CSS

**Reason**: The `.ai-card-preview-modal`, `.ai-card-preview-overlay`, `.ai-card-preview-panel`, `.ai-card-preview-header`, `.ai-card-preview-title`, `.ai-card-preview-close`, `.ai-card-preview-content` CSS rules are no longer used.

### Requirement: `_cardPreviewWorker` 变量

**Reason**: Only used by `openCardPreview`, no longer needed.
