# 在 FAB 组添加批量管理按钮 — 计划

## 概述

在笔记首页右下角的浮动操作按钮组（FAB Group）中新增一个"批量管理"按钮，提供批量管理功能的快捷入口，同时保持现有的滚动显隐逻辑不变。

## 当前状态分析

### FAB 组（index.html #L1526-L1530）

```html
<div id="fabGroup" class="fab-group">
  <button id="fabNewNote" class="fab fab-add" title="新建笔记">...</button>
  <button id="fabAI" class="fab fab-ai" title="AI 助手">...</button>
  <button id="backToTopBtn" class="fab fab-top" title="回到顶部">...</button>
</div>
```

* 固定定位，右下角（bottom: 12px, right: 28px）

* flex 垂直排列，gap: 12px

* 三个按钮：新建笔记（始终显示）、AI 助手（始终显示）、回到顶部（滚动 > 300px 后显示）

### 滚动逻辑（main.js #L5273-L5280）

* `els.mainContent.scrollTop > 300` 时：`fabGroup` 加 `.scrolled` 类（bottom: 12px → 64px），`backToTopBtn` 加 `.visible` 类

* `.scrolled` 类只改变 `bottom` 值，不影响内部按钮

### 批量管理功能

* 完整功能已存在（`toggleBatchMode()`、`#batchBar`、卡片勾选等）

* 当前入口：更多菜单（`data-action="batch-mode"`）、快捷键 `Ctrl+3`

* 点击时切换 grid 视图 + 进入批量模式

## 修改方案

### 文件 1：frontend/index.html

* 在 `fabAI` 与 `backToTopBtn` 之间插入批量管理按钮的 HTML

* 按钮样式：圆角矩形（44×44）、图标使用勾选列表 SVG，title="批量管理"

### 文件 2：frontend/src/css/components/main-content.css

* 新增 `.fab-batch` 样式类（与 `.fab-add` / `.fab-ai` 风格一致）

* 使用与 `.fab-add` 相同的 accent 背景色，但可以微调 hover 效果以示区分

### 文件 3：frontend/src/main.js

* <br />

  1. 在 `els` 对象中注册 `fabBatch` 引用（使用已有的 `$('fabBatch')` 模式）

* <br />

  1. 绑定 click 事件：`switchView('grid'); toggleBatchMode();`（与更多菜单的 batch-mode 行为一致，见 main.js #L4684-L4686）

### 不修改的

* `.scrolled` 类逻辑不动——新按钮自然继承 `.fab-group` 的 `.scrolled` 位移

* `toggleBatchMode()` 函数不动

* 现有的更多菜单和快捷键入口不动

## 决策

* **图标选择**：使用一个 checkbox 列表/勾选方块的 SVG，视觉上与"批量选择"概念匹配

* **按钮位置**：放在 fabAI 下面、backToTopBtn 上面——即有滚动时才出现的按钮保持在最下方

* **颜色**：与 fab-add/fab-ai 一致的 accent 主色，统一视觉层级

## 验证方式

1. 确保按钮在 grid 视图下显示（`fabGroup` 的 display 控制）
2. 点击按钮 → 切换到 grid 视图 + 进入批量模式 + 顶部显示 batch-bar
3. 滚动页面 → 按钮组整体上移（`.scrolled`），回到顶部按钮出现
4. 切换到非 grid 视图 → FAB 组隐藏（已有逻辑），不影响

