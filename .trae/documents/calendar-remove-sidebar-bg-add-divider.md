# 日历视图：移除左侧背景块，添加分割线

## 概述

移除左侧 `.calendar-sidebar` 的卡片背景和圆角，改为使用分割线来区分左右面板，消除视觉割裂感。

## 当前状态分析

`.calendar-sidebar` 当前样式：

```css
.calendar-sidebar {
    width: 340px;
    flex-shrink: 0;
    padding: 20px 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    background: var(--card-bg);     /* ← 要去掉 */
    border-radius: 12px;            /* ← 要去掉 */
}
```

左右面板由 `.calendar-layout` 的 `gap: 12px` 分隔，但无任何分割线。

## 修改方案

**文件：** `d:\峡谷\Dev\本地项目\jot\frontend\src\css\components\calendar.css`

### 改动 1：移除背景 + 圆角，添加右边框分割线

在 `.calendar-sidebar` 中：

* 删除 `background: var(--card-bg)` — 移除卡片背景

* 删除 `border-radius: 12px` — 移除卡片圆角

* 添加 `border-right: 1px solid var(--border)` — 右侧分割线

```css
.calendar-sidebar {
    width: 340px;
    flex-shrink: 0;
    padding: 20px 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    border-right: 1px solid var(--border);
}
```

### 改动 2：调整 gap 为 padding 作用域

由于添加了 `border-right` 分割线，原有的 `gap: 12px` 中分割线到右侧面板之间仍有间距。保持 `gap: 12px` 不变，这样视觉上形成：**左侧面板内容 → border-right → 12px gap → 右侧面板内容**，层次清晰且不拥挤。

## 设计决策

| 选项                     | 选择理由                                        |
| ---------------------- | ------------------------------------------- |
| **border-right 而非伪元素** | 更简洁，直接附着在 sidebar 上，无需额外元素                  |
| **保留 gap: 12px**       | 分割线贴在 sidebar 右侧边缘，gap 提供呼吸空间，避免分割线与右侧内容贴太近 |
| **不增加右侧面板背景**          | 保持右侧"开放"的视觉特性，统一底色后两侧自然融合                   |

## 验证

1. 切换所有主题（浅色/深色），确认 `var(--border)` 分割线在各主题下均可见且协调
2. 确认去掉背景后，`.calendar-sidebar` 内的元素（导航按钮、网格、统计项）仍有足够的视觉层次（stat-item 自身有 `background: var(--hover-bg)` 支撑）
3. 确认入场动画 `calendarSidebarIn` 仍正常工作

