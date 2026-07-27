# 修复启动器侧栏切换项标签不更新问题

## 问题描述

启动器菜单（Ctrl+P）中的"展开侧栏"项的标签和图标始终固定显示为"展开侧栏"和展开图标，即使用户已经展开了侧边栏也没有更新为"折叠侧栏"和折叠图标。

## 当前状态分析

- **启动器定义**: `frontend/src/js/launcher.js` 第 10-14 行，`launcherItems` 数组中 `sidebar-toggle` 项的 `label` 和 `svg` 是硬编码的 `'展开侧栏'`
- **渲染**: `renderLauncherItems()`（第 282-290 行）在初始化时一次性渲染所有卡片，之后不再刷新
- **对比「更多菜单」**: `main.js` 中的 `updateSidebarMenuItem()` 函数（第 6993-7001 行）在每次侧栏切换后动态更新菜单项的标签和 SVG 图标——这正是启动器缺少的逻辑
- **打开启动器**: `openLauncher()`（第 295-312 行）只负责显示、重置搜索和过滤，不检查侧栏的当前状态

## 修复方案

在 `openLauncher()` 函数中，渲染完成后、过滤之前，根据当前侧栏的折叠状态动态更新 `sidebar-toggle` 项的 DOM 元素（标签文字、图标、`data-label`）。

### 修改文件

**`frontend/src/js/launcher.js`**

在 `openLauncher()` 函数中，`filterItems('')` 调用之后，添加一段逻辑：
1. 查询 `[data-action="sidebar-toggle"]` 的 DOM 元素
2. 通过 `window.els.notebookSidebar?.classList.contains('collapsed')` 获取当前状态
3. 更新标签文本（"展开侧栏" ↔ "折叠侧栏"）
4. 更新图标 SVG（`x1="9"` ↔ `x1="15"`）
5. 更新 `data-label` 属性（用于搜索过滤）

### SVG 图标对照

| 状态 | SVG（line x1） | 文字 |
|------|---------------|------|
| 折叠（collapsed） | `x1="9"`（左侧竖线→展开） | 展开侧栏 |
| 展开（非collapsed） | `x1="15"`（右侧竖线→折叠） | 折叠侧栏 |

### 验证步骤

1. 打开启动器（Ctrl+P），侧栏默认折叠 → 应显示"展开侧栏"
2. 展开侧栏后再次打开启动器 → 应显示"折叠侧栏"
3. 在启动器中输入搜索关键字过滤 → data-label 更新后应能正确匹配
