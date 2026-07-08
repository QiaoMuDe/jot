# AI助手模块 — 联网搜索 & 更多技能 菜单宽度缩减15% 计划

## 概要

将 AI 助手工具栏中两个下拉菜单（联网搜索多源选择菜单、更多技能菜单）的宽度减少 15%。

## 当前状态分析

**涉及文件：**
- [ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css) — 所有下拉菜单样式集中在此文件

### 当前 CSS 结构

| 元素 | CSS 类/ID | 当前宽度控制 |
|------|-----------|-------------|
| **联网搜索下拉菜单** | `.ai-chat-search-sources-dropdown` | `min-width: 160px`（L898），项 `padding: 7px 12px`（L941） |
| **更多技能下拉菜单** | `.ai-chat-skills-dropdown` | `min-width: 160px`（L974），项 `padding: 7px 12px`（L1025） |

两个菜单的宽度均受 `min-width: 160px` + 子项 `padding: 0 12px` 共同决定。实际宽度取两者较大值，当前 160px 起决定作用。

## 变更方案

### 规则
- 仅修改 CSS，不涉及 HTML/JS 改动
- 减少幅度：15%（160px → 136px，即减少 24px）
- 配合微调项内边距以保持视觉一致性

### 具体修改

#### 1. `ai-chat.css` — `.ai-chat-search-sources-dropdown`（联网搜索菜单）

```css
/* 当前 */
min-width: 160px;

/* 改为 */
min-width: 136px;
```

**行号：** L898

#### 2. `ai-chat.css` — `.ai-chat-skills-dropdown`（更多技能菜单）

```css
/* 当前 */
min-width: 160px;

/* 改为 */
min-width: 136px;
```

**行号：** L974

#### 3. `ai-chat.css` — 两项菜单中的 `gap` 微调（可选优化）

当前 `.ai-chat-search-source-item` 和 `.ai-chat-skills-item` 的 `gap: 6px` 保持不变（`6px` 已较小，减少会影响布局紧凑性）。

当前项 `padding: 7px 12px` — 左右 `12px` 不调整，因为菜单变窄后如果左右 padding 也减少会导致内容贴边。

### 影响评估

| 影响点 | 说明 |
|--------|------|
| 布局 | 菜单变窄后，内容（复选框+文字/SVG+文字）仍能完整显示，不会溢出 |
| 响应式 | 极小宽度下菜单会提前触发换行，但 136px 仍远大于最小内容宽度（约 100px） |
| 其他 | 不涉及其他元素，改动完全隔离在两个 CSS 类中 |

## 验证步骤

1. 打开 AI 助手界面
2. 点击"联网搜索"按钮，确认下拉菜单宽度减少约 15%，内容完整可读
3. 点击"更多技能"按钮，确认下拉菜单宽度减少约 15%，内容完整可读
4. 分别勾选/选择功能，确认功能正常
