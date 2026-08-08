# 数据管理页 / 设置页滚动条贴窗口右缘修复计划

## Summary

数据管理页（`#viewData`）和设置页（`#viewSettings`）的滚动条目前位于内层面板容器（`.data-panels` / `.settings-panels`）的右缘，被 `.view` 的 `padding-right: 32px` 推离窗口右缘 32px。目标是让滚动条统一贴在窗口右缘，与主区页面（`#mainContent` 滚动条）一致。

方案采用项目已有的 **todo 列表先例**：给滚动容器加 `margin-right: -32px` 抵消 `.view` 的右 padding，使容器右缘延伸到窗口右缘，滚动条随之贴窗。

## Current State Analysis

**布局层级**（滚动条位置根因）：

```
窗口 ─ #mainContent（无 padding，flex:1）
        └─ .view（padding: 24px 32px ← 右 32px 把内容整体内推）
             ├─ .view-header（返回按钮 + 标题 + 空 .view-controls）
             └─ .data-content / .settings-content（flex:1）
                  └─ .data-panels / .settings-panels（overflow-y: auto → 滚动条在这里）
```

* 主区页面（笔记列表等）滚动条属于 `#mainContent`（无 padding）→ 贴窗口右缘

* 数据管理/设置页滚动条属于内部面板容器，被 `.view` 右 padding 包裹 → 距窗口右缘 32px

**已确认的事实**：

1. `.view` 定义于 [main-content.css:14-18](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\main-content.css)：`padding: 24px 32px`
2. `.data-panels`（[data-view.css](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\data-view.css#L59-L66)）：`flex:1; overflow-y:auto; scrollbar-gutter:stable; padding:20px 24px` → 内容右间距 24px，滚动条 5px 不会遮挡内容
3. `.settings-panels`（[settings-panel.css:213-221](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\settings-panel.css#L213-L221)）：`flex:1; overflow-y:auto; scrollbar-gutter:stable; padding:0`；内容右间距由 `.settings-panel` 的 `padding:20px 24px` 提供（settings-panel.css:223-226）
4. 两个页面的 `view-header` 均只有左对齐元素（返回按钮、标题）+ **空的** `.view-controls`，无右对齐元素依赖右 padding（index.html:907-911、271-276）
5. **项目已有先例**：

   * [todo.css:210-221](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\todo.css)：`.todo-list-wrap` 用 `margin-right:-32px; padding-right:32px` 抵消 `.view` 右 padding，使滚动条贴窗口（与本场景完全同构）

   * [calendar.css:3-7](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\calendar.css)：`#viewCalendar.view { padding: 24px 0 24px 32px }` 直接覆盖 view 右 padding

## Proposed Changes

采用 todo 先例的**负 margin 方案**（只改滚动容器自身，不动 `.view` 全局行为，影响面最小）。

### 1. [data-view.css](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\data-view.css) — `.data-panels`

在 `.data-panels` 规则中添加：

```css
/* 右缘延伸到窗口（抵消 .view 的 padding-right），使滚动条贴窗口 */
margin-right: -32px;
```

* 容器右缘从"距窗口 32px"变为"贴窗口右缘"，滚动条随之贴窗

* 保留现有 `padding: 20px 24px`：内容仍距窗口 24px，5px 宽滚动条落在 padding 区域内，不遮挡内容

* 负 margin 恰好落回 `.view` 的 32px 空白区，不溢出 `#mainContent`

### 2. [settings-panel.css](d:\资源池\下水道\Dev\本地项目\jot\frontend\src\css\components\settings-panel.css) — `.settings-panels`

在 `.settings-panels` 规则中添加：

```css
/* 右缘延伸到窗口（抵消 .view 的 padding-right），使滚动条贴窗口 */
margin-right: -32px;
```

* 与数据页一致；内容右间距由 `.settings-panel` 的 `padding: 20px 24px` 提供，滚动条不遮内容

## Assumptions & Decisions

1. **两个页面一起改**：用户已确认"数据页 + 设置页一起改"，保证两页行为一致，且都与主区页面（滚动条贴窗）统一
2. **采用负 margin 而非覆盖** **`.view`** **padding**：沿用 todo 先例，只影响滚动容器自身；calendar 的 `:has()` 方案依赖较新选择器且改动 `.view` 全局，不采用
3. **不调整内容 padding**：`.data-panels` 的 24px 与 `.settings-panel` 的 24px 均大于滚动条宽度 5px，已保证滚动条不遮内容；卡片（`max-width: 580px` 居中）视觉位置基本不变
4. 无需改 JS / HTML / scrollbar.css

## Verification

1. `npm run build`（frontend 目录）通过，无报错
2. Wails 运行后检查：

   * 数据管理页任意面板内容超长出现滚动条 → 滚动条贴窗口右缘

   * 设置页同理

   * 左侧导航固定、面板滚动行为不变

   * 两个页面的头部（返回按钮、标题）位置正常

   * 卡片内容不被滚动条遮挡

   * 主区笔记列表页滚动条仍贴窗口右缘（回归确认）

