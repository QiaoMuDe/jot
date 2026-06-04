# 批量模式标签/FAB显隐/回收站滚动条/恢复反馈 修复计划

## 概要
修复 4 个独立问题：
1. 批量管理模式下标签不应触发搜索跳转
2. FAB 悬浮按钮（`+` / `↑`）仅在首页（grid 视图）显示
3. 回收站列表滚动条改为窗口最右侧（取消居中宽度限制）
4. 单条恢复笔记增加 Toast 成功提示

## 当前状态分析

### 问题 1：batchMode + 标签点击
- `renderCardGrid()` 中标签 chip 的 `onclick` 始终执行 `window.searchByTag('...')`，**不受 batchMode 控制**
- batchMode 只控制了卡片的左击/右键/复选框

### 问题 2：FAB 按钮显隐
- `#fabNewNote` 和 `#backToTopBtn` 在所有视图中始终可见（`index.html` 全局挂载）
- `switchView()` 仅控制了顶部栏按钮的显隐，未处理 FAB

### 问题 3：回收站滚动条位置
- `.trash-list` 有 `max-width: 680px; margin: 0 auto;` 居中，滚动条出现在内容区右侧，而非窗口最右侧
- 需要的效果：回收站内容左对齐，滚动条贴窗口右边缘（与 `#mainContent` 一致）

### 问题 4：恢复缺少反馈
- `restoreNote(id)` → 静默恢复，无任何 Toast
- `restoreAllNotes()` → 已有 Toast + 可撤销
- `deleteNote(id)` → 有 Toast + 可撤销（对比不一致）

## 变更清单

### 变更 1：批量模式下阻止标签搜索
**文件**：`frontend/src/main.js`
- 在 `renderCardGrid()` 中构建标签 `onclick` 时，加条件判断：
  ```js
  onclick: state.batchMode ? undefined : `event.stopPropagation(); window.searchByTag('${escapeAttr(tag.name)}')`
  ```
  或简化为内联写法，利用 `event.stopPropagation()` 已经存在，在外层再加一层 batchMode 检查的逻辑。

  实际上更干净的做法是让标签点击函数自己检查 batchMode。修改标签 chip 的渲染逻辑：
  - batchMode 时标签 chip 不绑 `onclick`（或加一个 no-op），变成纯展示
  - 非 batchMode 时保持现有行为

**为什么**：批量模式下用户期望的操作是勾选/取消勾选笔记，标签点击跳转搜索会打断批量操作流程。

### 变更 2：FAB 按钮仅在 grid 视图显示
**文件**：`frontend/src/main.js`
- 在 `switchView()` 中增加 FAB 显隐逻辑：
  ```js
  els.fabGroup.style.display = view === 'grid' ? '' : 'none';
  ```
- `fabGroup` 是包裹两个 FAB 按钮的容器，统一显隐

**为什么**：数据管理、回收站、设置页面不需要新建笔记或回到顶部，FAB 按钮在这些页面会造成干扰。

### 变更 3：回收站滚动条改为窗口最右侧（内容保持居中）
**文件**：`frontend/style.css`、`frontend/index.html`
- `.trash-list` 改为全宽滚动容器（`width: 100%; flex: 1; overflow-y: auto;`）
- 在 `.trash-list` 内部新增一个 `.trash-list-inner` 做居中：
  ```css
  .trash-list-inner {
    max-width: 680px;
    width: 100%;
    margin: 0 auto;
    padding: 20px;
  }
  ```
- `index.html`：`.trash-list` 内包一层 `div.trash-list-inner`，JS 渲染追加到 `.trash-list-inner`

**为什么**：滚动条跟随全宽容器贴在窗口右边缘，同时内容仍视觉居中保持 680px 宽度。

### 变更 4：单条恢复增加 Toast
**文件**：`frontend/src/main.js`
- 在 `restoreNote()` 成功恢复后，增加 `showToast('笔记已恢复')` 调用
- 不需要撤销按钮（回收站中可重新删除），使用纯提示的 `showToast`（3秒自动消失）

**为什么**：与 `deleteNote` 的反馈保持一致，用户操作后得到明确确认。

## 假设与决策
- 回收站内容宽度沿用 `680px` 但改为 `padding-left` 方式实现视觉居中，让滚动条在窗口最右侧
- FAB 按钮通过父容器 `#fabGroup` 统一显隐，不单独控制每个按钮
- 恢复操作的 toast 使用纯提示（无撤销），因为回收站中恢复不可逆但用户可再次删除

## 验证方式
1. 批量模式下点击标签 → 不跳转搜索
2. 切换到数据管理/回收站/设置 → FAB 按钮隐藏；回到首页 → FAB 显示
3. 回收站内容滚动 → 滚动条在窗口最右侧边缘
4. 单条恢复笔记 → 底部出现「笔记已恢复」提示，3 秒消失
