# AI 会话侧栏折叠动画优化实施计划

## 摘要
修复 AI 助手左侧会话侧栏在折叠/展开时动画不丝滑的问题。当前折叠过程中，header、列表、footer 瞬间消失，而搜索框会滞后，造成不同步的"抽动"感。

## Current State Analysis

### 问题根因
当 `.collapsed` 类被添加时，同时发生以下行为：

| 元素 | 效果 | 动画方式 |
|------|------|----------|
| `.ai-session-sidebar` (父容器) | `width: 230px → 0` | **平滑过渡 0.25s** |
| `.ai-session-sidebar-header` | `display: none` | **立即生效，不可动画** |
| `.ai-session-list` | `display: none` | **立即生效，不可动画** |
| `.ai-session-sidebar-footer` | `display: none` | **立即生效，不可动画** |
| `.ai-session-search-wrap` | **无任何变化** | 被 `overflow: hidden` 裁剪 |

`display: none` 不是可动画属性，会导致：
- **折叠时**：header、list、footer 瞬间消失，而 search-wrap 没有设置 `display: none`，会随着父容器宽度缩小被 `overflow: hidden` 逐步裁剪——比其他内容消失得慢
- **展开时**：header、list、footer 瞬间出现，而 search-wrap 随宽度展开逐渐出现

### 相关 CSS 规则
文件：`frontend/src/css/components/ai-chat.css`

```css
/* 第 1905-1915 行 — 基础样式 */
.ai-session-sidebar {
    width: 230px;
    transition: width 0.25s ease;
    overflow: hidden;
    /* ... */
}

/* 第 1918-1927 行 — 折叠状态 */
.ai-session-sidebar.collapsed {
    width: 0;
    border-right: none;
}

.ai-session-sidebar.collapsed .ai-session-sidebar-header,
.ai-session-sidebar.collapsed .ai-session-list,
.ai-session-sidebar.collapsed .ai-session-sidebar-footer {
    display: none;
}
```

### 涉及文件
- `frontend/src/css/components/ai-chat.css` — 唯一需要修改的文件

## Proposed Changes

### 改动: 移除子元素的 `display: none`，仅依赖父容器 overflow: hidden 裁剪

**文件**: `frontend/src/css/components/ai-chat.css`（第 1918-1927 行）

**What**: 删除 `.ai-session-sidebar.collapsed` 中对三个子元素的 `display: none` 规则。

**Why**: 
- 父容器已经有 `overflow: hidden` 和 `width` 过渡动画
- 删除 `display: none` 后，所有子元素在视觉上完全同步——折叠时全部被父容器的宽度收缩逐步裁剪，展开时全部逐步露出
- 视觉效果平滑统一，无抽动

**How**:
将：
```css
.ai-session-sidebar.collapsed {
    width: 0;
    border-right: none;
}

.ai-session-sidebar.collapsed .ai-session-sidebar-header,
.ai-session-sidebar.collapsed .ai-session-list,
.ai-session-sidebar.collapsed .ai-session-sidebar-footer {
    display: none;
}
```

修改为：
```css
.ai-session-sidebar.collapsed {
    width: 0;
    border-right: none;
}
```

**无其他改动**。JS 代码、HTML 结构、其他 CSS 规则均保持不变。

## Assumptions & Decisions
- `overflow: hidden` 本就存在于 `.ai-session-sidebar` 上，无需新增
- 移除 `display: none` 后，子元素仍然会因父容器 `width: 0` + `overflow: hidden` 而不可见，功能无影响
- 不需要改动 JS（`classList.toggle('collapsed')` 继续正常工作）

## Verification
1. 检查 `ai-chat.css` 中不再有 `.ai-session-sidebar.collapsed` 的子元素 `display: none` 规则
2. 肉眼验证折叠/展开动画：所有子元素同步、平滑地被裁剪/露出
3. `npm run build` 构建通过
