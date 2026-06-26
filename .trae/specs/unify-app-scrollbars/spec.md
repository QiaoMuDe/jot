# 统一应用滚动条样式 Spec

## Why
目前应用中有多套滚动条样式并存：
1. `app.css` 全局 8px 滚动条带自动淡出效果（滑块太浅）
2. `style.css` 中散落着多个自定义滚动条规则（颜色/宽度不统一）
3. MD 语法手册页面使用了独立的 6px 滚动条样式
4. 用户希望所有滚动条统一为细条、贴边、可见的风格，且颜色跟随主题

## What Changes
- 更新各主题的 `--scrollbar-thumb` / `--scrollbar-thumb-hover` 变量值，提高可见度
- 全局 `::-webkit-scrollbar` 宽度从 8px → 6px
- 保持自动淡出机制但让可见状态更明显
- `style.css` 中散落的滚动条规则统一使用 `--scrollbar-thumb` / `--scrollbar-thumb-hover` 变量
- 为缺少滚动条样式的可滚动区域添加统一规则

## Impact
- Affected specs: 视觉风格统一
- Affected code: `frontend/src/app.css`（变量值 + 全局滚动条宽度）和 `frontend/src/style.css`（统一各区域滚动条样式）

## ADDED Requirements

### Requirement: 主题变量统一
各主题 `--scrollbar-thumb` 和 `--scrollbar-thumb-hover` SHALL 使用更高可见度的值。

#### 各主题变量值对照表
| 主题 | `--scrollbar-thumb` (当前) | `--scrollbar-thumb` (改为) | `--scrollbar-thumb-hover` (当前→改为) |
|------|--------------------------|--------------------------|--------------------------------------|
| default | `rgba(0,0,0,0.18)` | `rgba(0,0,0,0.28)` | `0.32→0.45` |
| light | `rgba(0,0,0,0.14)` | `rgba(0,0,0,0.25)` | `0.26→0.40` |
| nord | `rgba(0,0,0,0.14)` | `rgba(0,0,0,0.25)` | `0.26→0.40` |
| dark | `rgba(255,255,255,0.18)` | `rgba(255,255,255,0.28)` | `0.32→0.45` |
| monokai-pro | `rgba(255,255,255,0.18)` | `rgba(255,255,255,0.28)` | `0.32→0.45` |
| tokyo-night | `rgba(255,255,255,0.18)` | `rgba(255,255,255,0.28)` | `0.32→0.45` |

### Requirement: 全局滚动条宽度统一
`::-webkit-scrollbar` 宽度 SHALL 从 8px 改为 6px，`scrollbar-width: thin` 保持不变。

### Requirement: 所有可滚动区域使用统一滚动条样式
以下区域 SHALL 使用 `--scrollbar-thumb` / `--scrollbar-thumb-hover` / `--scrollbar-track` 变量：

- `#mainContent`（已有，使用变量，保持）
- `.search-results`（已有，使用变量，保持）
- `.data-content`（已有，使用变量，保持）
- `.cm-scroller`（style.css 中硬编码颜色 → 改为用变量）
- `.md-rendered`（style.css 中硬编码/fallback → 改为用变量）
- `.sidebar-notebook-list`（style.css 中 3px + var(--border) → 改为 6px + 变量）
- `.shortcuts-body`（新增滚动条样式）
- `.search-modal-results`（新增）
- `.search-modal-filter-dropdown`（已有但颜色硬编码 → 改为用变量）
- `.batch-tag-body`（新增）
- `.move-notebook-body`（已有但颜色用 var(--border) → 改为用变量）
- `.font-family-options`（已有但颜色用 var(--divider) → 改为用变量）

## REMOVED Requirements
无
